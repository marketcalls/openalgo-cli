// Command generate reads api/specs/openalgo-api.json and writes the generated
// API client, operation descriptions, and cobra command tree.
//
// Unlike a typed client generator, request bodies are map[string]any built
// at runtime from flag metadata (bodyFromFlags) and responses are passed
// through as json.RawMessage, so no request/response structs are emitted.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const specFile = "openalgo-api.json"

// maxResponseDepth caps how deep nested response objects are expanded.
const maxResponseDepth = 4

// compSchemas holds components/schemas of the loaded spec for $ref resolution.
var compSchemas map[string]any

func main() {
	root := findProjectRoot()
	apiDir := filepath.Join(root, "internal", "api")
	cmdDir := filepath.Join(root, "internal", "cmd")
	for _, d := range []string{apiDir, cmdDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatalf("creating output dir: %v", err)
		}
	}

	// Phase 1: load the spec and extract endpoints.
	spec := loadSpec(filepath.Join(root, "api", "specs", specFile))
	compSchemas = mapGet(mapGet(spec, "components"), "schemas")
	endpoints := extractEndpoints(spec)

	epByOp := make(map[string]*endpointInfo, len(endpoints))
	for _, ep := range endpoints {
		if _, dup := epByOp[ep.goName]; dup {
			log.Fatalf("duplicate operationId %q", ep.operationID)
		}
		epByOp[ep.goName] = ep
	}

	// Phase 2: derive flag/response descriptions and validate the registry
	// before writing anything, so a failed run leaves the old files intact.
	ops := collectDescriptions(endpoints)
	checkExhaustive(epByOp, ops)

	writeGo(filepath.Join(apiDir, "openalgo_client.gen.go"), genClient(endpoints))
	writeGo(filepath.Join(apiDir, "descriptions.gen.go"), writeDescriptionsFile(ops))
	writeGo(filepath.Join(cmdDir, "commands.gen.go"), genCommands(epByOp))

	fmt.Printf("Generated %d endpoints, %d commands, %d skipped\n",
		len(endpoints), len(cmdRegistry), len(cmdSkip))
}

// --- Spec loading ---

// keyOrder records the source order of keys for every JSON object decoded
// from the spec, keyed by the map's identity. Go maps are unordered, but
// response fields (and therefore CSV columns) read best in spec order.
var keyOrder = map[uintptr][]string{}

func loadSpec(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("reading spec: %v", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeOrdered(dec)
	if err != nil {
		log.Fatalf("parsing spec: %v", err)
	}
	spec, ok := v.(map[string]any)
	if !ok {
		log.Fatal("parsing spec: top-level value is not an object")
	}
	return spec
}

// decodeOrdered decodes the next JSON value, recording object key order in
// keyOrder. Numbers are converted to float64 to match encoding/json defaults.
func decodeOrdered(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			m := map[string]any{}
			var keys []string
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := kt.(string)
				val, err := decodeOrdered(dec)
				if err != nil {
					return nil, err
				}
				if _, seen := m[key]; !seen {
					keys = append(keys, key)
				}
				m[key] = val
			}
			if _, err := dec.Token(); err != nil { // closing '}'
				return nil, err
			}
			keyOrder[reflect.ValueOf(m).Pointer()] = keys
			return m, nil
		case '[':
			arr := []any{}
			for dec.More() {
				val, err := decodeOrdered(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil { // closing ']'
				return nil, err
			}
			return arr, nil
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		// string, bool, or nil
		return t, nil
	}
}

// orderedKeys returns m's keys in spec order when known, else sorted.
func orderedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	if keys, ok := keyOrder[reflect.ValueOf(m).Pointer()]; ok && sameKeys(keys, m) {
		return keys
	}
	return sortedKeys(m)
}

// sameKeys guards against a stale keyOrder entry: once a decoded map is
// garbage collected, a new map (e.g. one built in a test) can reuse its
// address, so the recorded keys must match the map's actual keys.
func sameKeys(keys []string, m map[string]any) bool {
	if len(keys) != len(m) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			log.Fatal("could not find project root (no go.mod)")
		}
		dir = parent
	}
}

// --- Endpoint extraction ---

type endpointInfo struct {
	method      string
	path        string
	operationID string
	goName      string
	summary     string
	description string // OAS operation description (longer than summary)
	tags        []string
	mutating    bool // x-mutating: true - the op has side effects
	pathParams  []paramInfo
	queryParams []paramInfo
	hasBody     bool           // operation declares an application/json requestBody
	bodySchema  map[string]any // resolved request body schema
	respSchema  map[string]any // resolved 200 response schema
}

type paramInfo struct {
	name     string
	required bool
	schema   map[string]any // raw parameter schema (may be a $ref)
	desc     string         // parameter-level description
}

func extractEndpoints(spec map[string]any) []*endpointInfo {
	paths := mapGet(spec, "paths")

	var result []*endpointInfo
	for path, methods := range paths {
		methodMap, ok := methods.(map[string]any)
		if !ok {
			continue
		}
		// Path-level parameters (shared by all methods on this path)
		pathLevelParams, _ := methodMap["parameters"].([]any)

		for method, opRaw := range methodMap {
			if method == "parameters" {
				continue
			}
			op, ok := opRaw.(map[string]any)
			if !ok {
				continue
			}
			opID, _ := op["operationId"].(string)
			if opID == "" {
				continue
			}
			ep := &endpointInfo{
				method:      strings.ToUpper(method),
				path:        path,
				operationID: opID,
				goName:      toGoName(opID),
			}
			ep.summary, _ = op["summary"].(string)
			ep.description, _ = op["description"].(string)
			ep.mutating, _ = op["x-mutating"].(bool)
			if tags, ok := op["tags"].([]any); ok {
				for _, t := range tags {
					if s, ok := t.(string); ok {
						ep.tags = append(ep.tags, s)
					}
				}
			}

			// Merge path-level + operation-level parameters
			var allParams []any
			allParams = append(allParams, pathLevelParams...)
			opParams, _ := op["parameters"].([]any)
			allParams = append(allParams, opParams...)

			for _, pRaw := range allParams {
				p, ok := pRaw.(map[string]any)
				if !ok {
					continue
				}
				if _, isRef := p["$ref"]; isRef {
					log.Fatalf("%s %s: parameter $ref is not supported; inline the parameter", ep.method, path)
				}
				pi := paramInfo{schema: mapGet(p, "schema")}
				pi.name, _ = p["name"].(string)
				pi.required, _ = p["required"].(bool)
				pi.desc, _ = p["description"].(string)
				switch in, _ := p["in"].(string); in {
				case "path":
					pi.required = true
					ep.pathParams = append(ep.pathParams, pi)
				case "query":
					ep.queryParams = append(ep.queryParams, pi)
				default:
					log.Fatalf("%s %s: unsupported parameter location %q for %q", ep.method, path, in, pi.name)
				}
			}

			if rb := mapGet(op, "requestBody"); rb != nil {
				if schema := mapGet(mapGet(mapGet(rb, "content"), "application/json"), "schema"); schema != nil {
					ep.hasBody = true
					ep.bodySchema = resolveRef(schema)
				}
			}

			if resp := mapGet(mapGet(op, "responses"), "200"); resp != nil {
				if schema := mapGet(mapGet(mapGet(resp, "content"), "application/json"), "schema"); schema != nil {
					ep.respSchema = resolveRef(schema)
				}
			}

			result = append(result, ep)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].path == result[j].path {
			return methodOrder(result[i].method) < methodOrder(result[j].method)
		}
		return result[i].path < result[j].path
	})
	return result
}

func methodOrder(m string) int {
	switch m {
	case "GET":
		return 0
	case "POST":
		return 1
	default:
		return 2
	}
}

func (ep *endpointInfo) hasTag(tag string) bool {
	for _, t := range ep.tags {
		if t == tag {
			return true
		}
	}
	return false
}

// bodyProps returns the request body's properties (nil for free-form bodies).
func (ep *endpointInfo) bodyProps() map[string]any {
	return mapGet(ep.bodySchema, "properties")
}

// isFreeFormBody reports whether the body is an object with no declared
// properties that accepts arbitrary keys (additionalProperties).
func (ep *endpointInfo) isFreeFormBody() bool {
	if ep.bodySchema == nil || len(ep.bodyProps()) > 0 {
		return false
	}
	switch ap := ep.bodySchema["additionalProperties"].(type) {
	case bool:
		return ap
	case map[string]any:
		return true
	}
	return false
}

// --- Schema helpers ---

// schemaType returns the non-null JSON Schema type and whether null is allowed.
// Handles OAS 3.0 (`type: "string", nullable: true`) and OAS 3.1
// (`type: ["string", "null"]`).
func schemaType(schema map[string]any) (typ string, nullable bool) {
	nullable, _ = schema["nullable"].(bool)
	switch t := schema["type"].(type) {
	case string:
		return t, nullable
	case []any:
		var primary string
		for _, v := range t {
			s, _ := v.(string)
			if s == "null" {
				nullable = true
				continue
			}
			if primary == "" {
				primary = s
			}
		}
		return primary, nullable
	default:
		return "", nullable
	}
}

// scalarTypes returns every non-null type of an OAS 3.1 type array, in spec
// order. A plain string type yields a single entry.
func scalarTypes(schema map[string]any) []string {
	list, ok := schema["type"].([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, v := range list {
		if s, _ := v.(string); s != "" && s != "null" {
			out = append(out, s)
		}
	}
	return out
}

// resolveRef follows a components/schemas $ref. Sibling keywords next to the
// $ref (OAS 3.1 allows description/default there) override the target's.
func resolveRef(schema map[string]any) map[string]any {
	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}
	target, ok := compSchemas[refBaseName(ref)].(map[string]any)
	if !ok {
		log.Fatalf("unresolved $ref %q", ref)
	}
	target = resolveRef(target)
	if len(schema) == 1 {
		return target
	}
	merged := make(map[string]any, len(target)+len(schema))
	for k, v := range target {
		merged[k] = v
	}
	for k, v := range schema {
		if k != "$ref" {
			merged[k] = v
		}
	}
	return merged
}

// flagType maps a resolved property schema to a FlagDef.Type.
func flagType(s map[string]any) string {
	switch t, _ := schemaType(s); t {
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "number":
		return "number"
	case "array":
		return "array"
	case "object":
		return "json"
	default:
		return "string"
	}
}

// schemaEnums returns enum values of a resolved schema, falling back to the
// enum of array items. Order follows the spec (it is often meaningful, e.g.
// intervals 1s..Y) and null entries are dropped.
func schemaEnums(s map[string]any) []string {
	if vals := extractEnumValues(s); len(vals) > 0 {
		return vals
	}
	if t, _ := schemaType(s); t == "array" {
		if items := mapGet(s, "items"); items != nil {
			return extractEnumValues(resolveRef(items))
		}
	}
	return nil
}

func extractEnumValues(schema map[string]any) []string {
	raw, ok := schema["enum"].([]any)
	if !ok {
		return nil
	}
	var vals []string
	for _, v := range raw {
		if v == nil {
			continue
		}
		if s := scalarString(v); s != "" {
			vals = append(vals, s)
		}
	}
	return vals
}

func extractRequired(schema map[string]any) map[string]bool {
	reqMap := map[string]bool{}
	if reqList, ok := schema["required"].([]any); ok {
		for _, r := range reqList {
			if s, ok := r.(string); ok {
				reqMap[s] = true
			}
		}
	}
	return reqMap
}

// extractDefault renders a schema default as a flag default string. Floats
// print in shortest form (0.0 -> "0"); objects and arrays render as JSON.
func extractDefault(schema map[string]any) string {
	dv, ok := schema["default"]
	if !ok || dv == nil {
		return ""
	}
	switch v := dv.(type) {
	case map[string]any, []any:
		b, err := json.Marshal(v)
		if err != nil {
			log.Fatalf("marshaling default %v: %v", v, err)
		}
		return string(b)
	default:
		return scalarString(v)
	}
}

func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

// --- Naming utilities ---

var abbreviations = map[string]string{
	"id": "ID", "url": "URL", "api": "API", "http": "HTTP",
	"json": "JSON", "gtt": "GTT", "pnl": "Pnl",
}

func toGoName(s string) string {
	if s == "" {
		return s
	}
	parts := splitIdent(s)
	var result strings.Builder
	for _, part := range parts {
		lower := strings.ToLower(part)
		if abbr, ok := abbreviations[lower]; ok && part == lower {
			result.WriteString(abbr)
		} else {
			result.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return result.String()
}

// splitIdent splits snake_case / kebab-case identifiers. CamelCase input
// (operationIds such as PlaceGTTOrder) is kept whole so acronyms survive.
func splitIdent(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '/' || r == '{' || r == '}'
	})
}

// kebab converts an OAS snake_case name to a CLI flag name.
func kebab(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", "-"))
}

// goIdent converts an OAS parameter name to a Go parameter identifier.
func goIdent(s string) string {
	return lcFirst(toGoName(s))
}

func refBaseName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

func mapGet(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]any)
	return v
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func lcFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// --- Client generation ---

func genClient(endpoints []*endpointInfo) string {
	var body bytes.Buffer

	fmt.Fprintf(&body, "// Client provides one method per OpenAlgo API operation. Request bodies\n")
	fmt.Fprintf(&body, "// are plain maps and responses are returned as raw JSON.\n")
	fmt.Fprintf(&body, "type Client struct {\n\tRaw *client.Client\n}\n\n")
	fmt.Fprintf(&body, "// NewClient wraps a configured HTTP client.\n")
	fmt.Fprintf(&body, "func NewClient(raw *client.Client) *Client {\n\treturn &Client{Raw: raw}\n}\n\n")

	for _, ep := range endpoints {
		genEndpointMethod(&body, ep)
	}

	// Assemble with header - detect imports from generated body
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated from api/specs/%s by cmd/generate; DO NOT EDIT.\n\n", specFile)
	fmt.Fprintf(&buf, "package api\n\n")

	bodyStr := body.String()
	fmt.Fprintf(&buf, "import (\n")
	fmt.Fprintf(&buf, "\t\"encoding/json\"\n")
	if strings.Contains(bodyStr, "fmt.") {
		fmt.Fprintf(&buf, "\t\"fmt\"\n")
	}
	if strings.Contains(bodyStr, "url.") {
		fmt.Fprintf(&buf, "\t\"net/url\"\n")
	}
	fmt.Fprintf(&buf, "\n\t\"github.com/marketcalls/openalgo-cli/internal/client\"\n")
	fmt.Fprintf(&buf, ")\n\n")
	buf.Write(body.Bytes())

	return buf.String()
}

// clientArgs returns the Go parameter list for an endpoint method: path
// params first (as strings), then query params, then the request body.
func clientArgs(ep *endpointInfo) []string {
	var args []string
	for _, pp := range ep.pathParams {
		args = append(args, goIdent(pp.name)+" string")
	}
	if len(ep.queryParams) > 0 {
		args = append(args, "params url.Values")
	}
	if ep.hasBody {
		args = append(args, "body map[string]any")
	}
	return args
}

func genEndpointMethod(buf *bytes.Buffer, ep *endpointInfo) {
	fmt.Fprintf(buf, "// %s calls %s %s", ep.goName, ep.method, ep.path)
	if ep.summary != "" {
		fmt.Fprintf(buf, ": %s", strings.TrimRight(ep.summary, "."))
	}
	fmt.Fprintf(buf, ".\n")
	fmt.Fprintf(buf, "func (c *Client) %s(%s) (json.RawMessage, error) {\n", ep.goName, strings.Join(clientArgs(ep), ", "))

	paramsExpr := "nil"
	if len(ep.queryParams) > 0 {
		paramsExpr = "params"
	}
	bodyExpr := "nil"
	if ep.hasBody {
		bodyExpr = "body"
	}
	do := "Do"
	if ep.mutating {
		do = "DoWrite"
	}
	fmt.Fprintf(buf, "\treturn c.Raw.%s(%q, %s, %s, %s)\n", do, ep.method, buildPathExpr(ep.path, ep.pathParams), paramsExpr, bodyExpr)
	fmt.Fprintf(buf, "}\n\n")
}

func buildPathExpr(path string, pathParams []paramInfo) string {
	expr := path
	var fmtArgs []string
	for _, p := range pathParams {
		placeholder := "{" + p.name + "}"
		if strings.Contains(expr, placeholder) {
			expr = strings.Replace(expr, placeholder, "%s", 1)
			fmtArgs = append(fmtArgs, "url.PathEscape("+goIdent(p.name)+")")
		}
	}
	if len(fmtArgs) > 0 {
		return fmt.Sprintf("fmt.Sprintf(%q, %s)", expr, strings.Join(fmtArgs, ", "))
	}
	return fmt.Sprintf("%q", path)
}

// --- Description generation ---

type opDesc struct {
	goName   string
	method   string
	path     string
	summary  string
	long     string
	mutating bool
	rowsPath string
	flags    []*flagDesc
	response []*responseFieldDesc
}

type flagDesc struct {
	oasName     string
	flagName    string
	flagType    string
	defaultVal  string
	description string
	enumValues  []string
	source      string // "path", "query", or "body"
	required    bool
	cliDefault  bool
}

type responseFieldDesc struct {
	name       string
	jsonType   string
	desc       string
	enumValues []string
	fields     []*responseFieldDesc
}

// schemaFlag derives a flag from a (possibly $ref) property schema.
// paramDesc, when set, takes precedence over the schema description.
func schemaFlag(name, source, paramDesc string, raw map[string]any, required bool) *flagDesc {
	s := map[string]any{}
	if raw != nil {
		s = resolveRef(raw)
	}
	enums := schemaEnums(s)
	desc := paramDesc
	if desc == "" {
		desc, _ = s["description"].(string)
	}
	if desc == "" {
		desc = humanize(name, enums)
	}
	typ := flagType(s)
	desc = normalizeDesc(desc)
	if hint := jsonValueHint(s); hint != "" && (typ == "array" || typ == "json") {
		desc += "; " + hint + "; literal, @file, or - for stdin"
	}
	return &flagDesc{
		oasName:     name,
		flagName:    kebab(name),
		flagType:    typ,
		defaultVal:  extractDefault(s),
		description: desc,
		enumValues:  enums,
		source:      source,
		required:    required,
	}
}

// jsonValueHint describes the JSON shape a json/array flag expects, e.g.
// "JSON array of {symbol, exchange} objects", so --help says what to pass.
// Object items list their required fields (all fields when none are
// required, up to maxHintFields).
func jsonValueHint(s map[string]any) string {
	switch t, _ := schemaType(s); t {
	case "object":
		return "JSON object"
	case "array":
		items := mapGet(s, "items")
		if items == nil {
			return "JSON array"
		}
		items = resolveRef(items)
		it, _ := schemaType(items)
		if it != "object" {
			if it == "" {
				return "JSON array"
			}
			return "JSON array of " + it + "s"
		}
		props := orderedKeys(mapGet(items, "properties"))
		if len(props) == 0 {
			return "JSON array of objects"
		}
		req := extractRequired(items)
		var names []string
		for _, p := range props {
			if len(req) == 0 || req[p] {
				names = append(names, p)
			}
		}
		if len(names) > maxHintFields {
			names = append(names[:maxHintFields], "...")
		}
		return "JSON array of {" + strings.Join(names, ", ") + "} objects"
	}
	return ""
}

const maxHintFields = 6

// opFlags derives every flag for an endpoint, applying registry aliases and
// CLI defaults, sorted required-first then alphabetically.
func opFlags(ep *endpointInfo, def cmdDef) []*flagDesc {
	var flags []*flagDesc
	for _, p := range ep.pathParams {
		flags = append(flags, schemaFlag(p.name, "path", p.desc, p.schema, true))
	}
	for _, p := range ep.queryParams {
		flags = append(flags, schemaFlag(p.name, "query", p.desc, p.schema, p.required))
	}
	if ep.isFreeFormBody() {
		name := def.objectFlag
		if name == "" {
			name = "body"
		}
		desc, _ := ep.bodySchema["description"].(string)
		if desc == "" {
			desc = "JSON object (literal, @file, or - for stdin) whose keys are merged into the request body"
		}
		flags = append(flags, &flagDesc{
			oasName:     name,
			flagName:    name,
			flagType:    "object",
			description: normalizeDesc(desc),
			source:      "body",
			required:    true,
		})
	} else if props := ep.bodyProps(); props != nil {
		req := extractRequired(ep.bodySchema)
		for _, name := range orderedKeys(props) {
			ps, _ := props[name].(map[string]any)
			flags = append(flags, schemaFlag(name, "body", "", ps, req[name]))
		}
	}

	for _, f := range flags {
		if alias, ok := def.flagAliases[f.flagName]; ok {
			f.flagName = alias
		}
		if f.source == "body" {
			if v, ok := def.defaults[f.oasName]; ok {
				f.defaultVal = v
				f.cliDefault = true
			}
		}
	}

	sort.SliceStable(flags, func(i, j int) bool {
		if flags[i].required != flags[j].required {
			return flags[i].required
		}
		return flags[i].flagName < flags[j].flagName
	})
	return flags
}

func collectDescriptions(endpoints []*endpointInfo) []*opDesc {
	var ops []*opDesc
	for _, ep := range endpoints {
		def := cmdRegistry[ep.goName]
		op := &opDesc{
			goName:   ep.goName,
			method:   ep.method,
			path:     ep.path,
			summary:  normalizeSummary(ep.method, ep.summary),
			long:     normalizeOASDescription(ep.description),
			mutating: ep.mutating,
			flags:    opFlags(ep, def),
		}
		if def.long != "" {
			op.long = def.long
		}
		if ep.respSchema != nil {
			op.response = responseFields(ep.respSchema, 1)
			op.rowsPath = rowsPath(ep.respSchema)
		}
		if def.rowsPath != "" {
			op.rowsPath = def.rowsPath
		}
		ops = append(ops, op)
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].goName < ops[j].goName })
	return ops
}

// responseFields expands an object schema's properties into a field tree.
// Nested objects and arrays of objects recurse up to maxResponseDepth.
func responseFields(schema map[string]any, depth int) []*responseFieldDesc {
	props := mapGet(resolveRef(schema), "properties")
	var fields []*responseFieldDesc
	for _, name := range orderedKeys(props) {
		raw, _ := props[name].(map[string]any)
		ps := resolveRef(raw)
		desc, _ := ps["description"].(string)
		f := &responseFieldDesc{
			name:       name,
			jsonType:   responseType(ps),
			desc:       normalizeDesc(desc),
			enumValues: schemaEnums(ps),
		}
		if depth < maxResponseDepth {
			if obj := objectSchema(ps); obj != nil {
				f.fields = responseFields(obj, depth+1)
			}
		}
		fields = append(fields, f)
	}
	return fields
}

// objectSchema returns the object schema carrying properties for s itself
// or, when s is an array, for its items. Nil when neither has properties.
func objectSchema(s map[string]any) map[string]any {
	if len(mapGet(s, "properties")) > 0 {
		return s
	}
	if t, _ := schemaType(s); t == "array" {
		if items := mapGet(s, "items"); items != nil {
			items = resolveRef(items)
			if len(mapGet(items, "properties")) > 0 {
				return items
			}
		}
	}
	return nil
}

// responseType renders a TypeScript-like type for a response field, e.g.
// "number", "[]object", "string|null".
func responseType(s map[string]any) string {
	t, nullable := schemaType(s)
	var out string
	if alts := scalarTypes(s); len(alts) > 1 {
		// A field whose type differs between live brokers and analyzer
		// mode, e.g. ["string", "number"] renders as "string|number".
		out = strings.Join(alts, "|")
		if nullable {
			out += "|null"
		}
		return out
	}
	switch t {
	case "string", "number", "integer", "boolean", "object":
		out = t
	case "array":
		out = "[]any"
		if items := mapGet(s, "items"); items != nil {
			out = "[]" + strings.TrimSuffix(responseType(resolveRef(items)), "|null")
		}
	default:
		if len(mapGet(s, "properties")) > 0 {
			out = "object"
		} else {
			out = "any"
		}
	}
	if nullable && out != "any" {
		out += "|null"
	}
	return out
}

// rowsPath locates the records used for --csv output: "data" when data is
// an array, "data.<field>" when data is an object with exactly one
// array-of-objects property, "data" (a single row) when data is a plain
// record with no array-of-objects property, else "".
func rowsPath(resp map[string]any) string {
	raw := mapGet(mapGet(resolveRef(resp), "properties"), "data")
	if raw == nil {
		return ""
	}
	data := resolveRef(raw)
	switch t, _ := schemaType(data); t {
	case "array":
		return "data"
	case "object":
		var found []string
		props := mapGet(data, "properties")
		for _, name := range orderedKeys(props) {
			ps, _ := props[name].(map[string]any)
			ps = resolveRef(ps)
			if pt, _ := schemaType(ps); pt != "array" {
				continue
			}
			if items := mapGet(ps, "items"); items != nil {
				if it, _ := schemaType(resolveRef(items)); it == "object" {
					found = append(found, name)
				}
			}
		}
		switch {
		case len(found) == 1:
			return "data." + found[0]
		case len(found) == 0 && len(props) > 0:
			return "data"
		}
	}
	return ""
}

func normalizeDesc(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.TrimRight(s, ".")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.TrimPrefix(s, "The ")
	if len(s) > 120 {
		s = firstSentence(s)
	}
	if len(s) > 120 {
		s = s[:117] + "..."
	}
	if len(s) > 1 && unicode.IsUpper(rune(s[0])) {
		if unicode.IsLower(rune(s[1])) {
			s = strings.ToLower(s[:1]) + s[1:]
		}
	}
	return s
}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i]
	}
	if i := strings.Index(s, "; "); i > 0 {
		return s[:i]
	}
	if i := strings.Index(s, "\n"); i > 0 {
		return s[:i]
	}
	return s
}

var imperativeVerbs = map[string]bool{
	"get": true, "list": true, "place": true, "modify": true, "cancel": true,
	"close": true, "calculate": true, "resolve": true, "search": true,
	"split": true, "download": true, "toggle": true, "ping": true,
	"start": true, "stop": true, "update": true, "send": true,
	"broadcast": true, "set": true, "delete": true, "check": true,
	"show": true, "fetch": true, "return": true, "returns": true,
}

// properNouns keep their capitalization when a title-cased summary is
// converted to sentence case.
var properNouns = map[string]bool{
	"OpenAlgo": true, "Telegram": true, "WhatsApp": true, "Greeks": true,
	"Historify": true, "Black-76": true,
}

func normalizeSummary(method, summary string) string {
	s := strings.TrimRight(strings.TrimSpace(summary), ".")
	if s == "" {
		return s
	}
	s = titleToSentence(s)

	words := strings.Fields(s)
	first := strings.ToLower(words[0])
	if !imperativeVerbs[first] {
		verb := httpMethodVerb(method)
		if len(s) > 1 && unicode.IsUpper(rune(s[0])) && unicode.IsUpper(rune(s[1])) {
			s = verb + " " + s
		} else {
			s = verb + " " + strings.ToLower(s[:1]) + s[1:]
		}
	}
	return s
}

// normalizeOASDescription cleans an OAS operation description for use as
// Cobra Long help text. Takes the first paragraph, strips markdown artifacts,
// and trims to a reasonable length at a sentence boundary.
func normalizeOASDescription(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	// Take only the first paragraph (split on blank line).
	if i := strings.Index(s, "\n\n"); i >= 0 {
		s = s[:i]
	}

	// Strip markdown blockquote markers.
	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimPrefix(line, "> ")
		line = strings.TrimPrefix(line, ">")
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	s = strings.Join(cleaned, " ")

	// Convert markdown links [text](url) to just text.
	s = stripMarkdownLinks(s)
	s = strings.ReplaceAll(s, "`", "")

	// Trim trailing period for consistency with Short.
	s = strings.TrimRight(s, ".")

	// Cap at ~300 chars at a sentence boundary.
	if len(s) > 300 {
		cut := 300
		if i := strings.LastIndex(s[:cut], ". "); i > 100 {
			s = s[:i]
		} else {
			s = s[:cut]
			if i := strings.LastIndex(s, " "); i > 0 {
				s = strings.TrimRight(s[:i], " .,")
			}
		}
	}

	return s
}

// stripMarkdownLinks replaces [text](url) with text.
func stripMarkdownLinks(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == '[' {
			close := strings.Index(s[i:], "](")
			if close < 0 {
				b.WriteByte(s[i])
				i++
				continue
			}
			linkText := s[i+1 : i+close]
			urlStart := i + close + 2
			urlEnd := strings.IndexByte(s[urlStart:], ')')
			if urlEnd < 0 {
				b.WriteByte(s[i])
				i++
				continue
			}
			b.WriteString(linkText)
			i = urlStart + urlEnd + 1
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// titleToSentence lower-cases Title-Case words after the first, keeping
// acronyms (GTT, P&L), mixed-case words and known proper nouns intact.
func titleToSentence(s string) string {
	words := strings.Fields(s)
	for i := 1; i < len(words); i++ {
		w := words[i]
		if properNouns[w] || len(w) < 2 {
			continue
		}
		rest := w[1:]
		if unicode.IsUpper(rune(w[0])) && strings.ToLower(rest) == rest {
			words[i] = strings.ToLower(w)
		}
	}
	return strings.Join(words, " ")
}

// httpMethodVerb picks the verb prefixed to a non-imperative summary.
// OpenAlgo uses POST for most reads, so the method alone cannot tell a read
// from a write; mutating ops carry imperative summaries in the spec.
func httpMethodVerb(method string) string {
	switch method {
	case "DELETE":
		return "Delete"
	default:
		return "Get"
	}
}

func humanize(name string, enumValues []string) string {
	if len(enumValues) > 0 {
		return strings.Join(enumValues, ", ")
	}
	s := strings.ReplaceAll(name, "_", " ")
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

func writeDescriptionsFile(ops []*opDesc) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated from api/specs/%s by cmd/generate; DO NOT EDIT.\n\n", specFile)
	fmt.Fprintf(&buf, "package api\n\n")

	// Per-op vars
	for _, op := range ops {
		def := cmdRegistry[op.goName]
		fmt.Fprintf(&buf, "var %sOp = Op{\n", op.goName)
		fmt.Fprintf(&buf, "\tName: %q, Method: %q, Path: %q,\n", op.goName, op.method, op.path)
		fmt.Fprintf(&buf, "\tSummary: %q,\n", op.summary)
		if op.long != "" && op.long != op.summary {
			fmt.Fprintf(&buf, "\tLong: %q,\n", op.long)
		}
		if def.examples != "" {
			fmt.Fprintf(&buf, "\tExample: %s,\n", backtickQuote(def.examples))
		}
		if op.mutating {
			fmt.Fprintf(&buf, "\tMutating: true,\n")
		}
		if op.rowsPath != "" {
			fmt.Fprintf(&buf, "\tRowsPath: %q,\n", op.rowsPath)
		}

		if len(op.flags) > 0 {
			fmt.Fprintf(&buf, "\tFlags: []FlagDef{\n")
			for _, p := range op.flags {
				fmt.Fprintf(&buf, "\t\t{Name: %q, OASName: %q, Type: %q", p.flagName, p.oasName, p.flagType)
				if p.defaultVal != "" {
					fmt.Fprintf(&buf, ", Default: %q", p.defaultVal)
				}
				fmt.Fprintf(&buf, ", Description: %q", p.description)
				if len(p.enumValues) > 0 {
					fmt.Fprintf(&buf, ", Completions: %s", stringSliceLit(p.enumValues))
				}
				if p.required {
					fmt.Fprintf(&buf, ", Required: true")
				}
				fmt.Fprintf(&buf, ", Source: %q", p.source)
				if p.cliDefault {
					fmt.Fprintf(&buf, ", CLIDefault: true")
				}
				fmt.Fprintf(&buf, "},\n")
			}
			fmt.Fprintf(&buf, "\t},\n")
		}

		if len(op.response) > 0 {
			fmt.Fprintf(&buf, "\tResponse: []ResponseField{\n")
			writeResponseFields(&buf, op.response, "\t\t")
			fmt.Fprintf(&buf, "\t},\n")
		}

		fmt.Fprintf(&buf, "}\n\n")
	}

	fmt.Fprintf(&buf, "// AllOps lists every generated Op for iteration in tests and tooling.\n")
	fmt.Fprintf(&buf, "var AllOps = []Op{\n")
	for _, op := range ops {
		fmt.Fprintf(&buf, "\t%sOp,\n", op.goName)
	}
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "var opByName map[string]Op\n\n")
	fmt.Fprintf(&buf, "func init() {\n")
	fmt.Fprintf(&buf, "\topByName = make(map[string]Op, len(AllOps))\n")
	fmt.Fprintf(&buf, "\tfor _, op := range AllOps {\n")
	fmt.Fprintf(&buf, "\t\topByName[op.Name] = op\n")
	fmt.Fprintf(&buf, "\t}\n")
	fmt.Fprintf(&buf, "}\n\n")
	fmt.Fprintf(&buf, "// OpByName returns the Op with the given name, if any.\n")
	fmt.Fprintf(&buf, "func OpByName(name string) (Op, bool) {\n")
	fmt.Fprintf(&buf, "\top, ok := opByName[name]\n")
	fmt.Fprintf(&buf, "\treturn op, ok\n")
	fmt.Fprintf(&buf, "}\n\n")
	fmt.Fprintf(&buf, "// ResponseSchema returns the response field tree for an operation.\n")
	fmt.Fprintf(&buf, "func ResponseSchema(name string) ([]ResponseField, bool) {\n")
	fmt.Fprintf(&buf, "\top, ok := opByName[name]\n")
	fmt.Fprintf(&buf, "\tif !ok || len(op.Response) == 0 {\n")
	fmt.Fprintf(&buf, "\t\treturn nil, false\n")
	fmt.Fprintf(&buf, "\t}\n")
	fmt.Fprintf(&buf, "\treturn op.Response, true\n")
	fmt.Fprintf(&buf, "}\n")

	return buf.String()
}

func writeResponseFields(buf *bytes.Buffer, fields []*responseFieldDesc, indent string) {
	for _, f := range fields {
		fmt.Fprintf(buf, "%s{Name: %q, Type: %q, Description: %q", indent, f.name, f.jsonType, f.desc)
		if len(f.enumValues) > 0 {
			fmt.Fprintf(buf, ", EnumValues: %s", stringSliceLit(f.enumValues))
		}
		if len(f.fields) > 0 {
			fmt.Fprintf(buf, ", Fields: []ResponseField{\n")
			writeResponseFields(buf, f.fields, indent+"\t")
			fmt.Fprintf(buf, "%s}", indent)
		}
		fmt.Fprintf(buf, "},\n")
	}
}

func stringSliceLit(vals []string) string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = strconv.Quote(v)
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}

// --- File writing ---

// writeGo gofmt's content and writes it. Generated code that fails to parse
// is a generator bug, so it is fatal rather than written unformatted.
func writeGo(path, content string) {
	formatted, err := format.Source([]byte(content))
	if err != nil {
		_ = os.WriteFile(path+".broken", []byte(content), 0o644)
		log.Fatalf("gofmt failed for %s: %v (unformatted output in %s.broken)", filepath.Base(path), err, filepath.Base(path))
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		log.Fatalf("writing %s: %v", path, err)
	}
}
