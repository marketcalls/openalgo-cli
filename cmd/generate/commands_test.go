package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// loadRealSpec loads api/specs/openalgo-api.json and returns its endpoints.
func loadRealSpec(t *testing.T) (map[string]*endpointInfo, []*opDesc) {
	t.Helper()
	prev := compSchemas
	t.Cleanup(func() { compSchemas = prev })
	spec := loadSpec(filepath.Join("..", "..", "api", "specs", specFile))
	compSchemas = mapGet(mapGet(spec, "components"), "schemas")
	endpoints := extractEndpoints(spec)
	epByOp := map[string]*endpointInfo{}
	for _, ep := range endpoints {
		epByOp[ep.goName] = ep
	}
	return epByOp, collectDescriptions(endpoints)
}

func TestRegistryExhaustiveAgainstSpec(t *testing.T) {
	epByOp, ops := loadRealSpec(t)
	if errs := registryErrors(epByOp, ops); len(errs) > 0 {
		t.Fatalf("registry errors:\n  %s", strings.Join(errs, "\n  "))
	}
	if len(epByOp) != len(cmdRegistry)+len(cmdSkip) {
		t.Errorf("spec has %d ops, registry+skip covers %d", len(epByOp), len(cmdRegistry)+len(cmdSkip))
	}
}

func TestRegistryErrorsDetectsDrift(t *testing.T) {
	epByOp, ops := loadRealSpec(t)

	// An operation missing from the registry must be reported.
	saved := cmdRegistry["GetFunds"]
	delete(cmdRegistry, "GetFunds")
	errs := registryErrors(epByOp, ops)
	cmdRegistry["GetFunds"] = saved
	if !containsErr(errs, `unregistered operation "GetFunds"`) {
		t.Errorf("missing registry entry not reported: %v", errs)
	}

	// Empty examples must be reported.
	def := cmdRegistry["GetFunds"]
	def.examples = ""
	cmdRegistry["GetFunds"] = def
	errs = registryErrors(epByOp, ops)
	cmdRegistry["GetFunds"] = saved
	if !containsErr(errs, `cmdRegistry["GetFunds"] has empty examples`) {
		t.Errorf("empty examples not reported: %v", errs)
	}

	// An order op without the strategy CLI default must be reported.
	saved = cmdRegistry["CancelOrder"]
	def = saved
	def.defaults = nil
	cmdRegistry["CancelOrder"] = def
	epOps := collectDescriptions(sortedEndpoints(epByOp))
	errs = registryErrors(epByOp, epOps)
	cmdRegistry["CancelOrder"] = saved
	if !containsErr(errs, `cmdRegistry["CancelOrder"]: strategy body field needs defaults`) {
		t.Errorf("missing strategy default not reported: %v", errs)
	}
}

func sortedEndpoints(m map[string]*endpointInfo) []*endpointInfo {
	var eps []*endpointInfo
	for _, k := range sortedKeys(m) {
		eps = append(eps, m[k])
	}
	return eps
}

func containsErr(errs []string, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

func TestCheckFlagsReservedCollision(t *testing.T) {
	testSchemas(t)
	ep := &endpointInfo{
		goName:  "Fake",
		hasBody: true,
		bodySchema: map[string]any{"type": "object", "properties": map[string]any{
			"timeout": map[string]any{"type": "integer"},
			"symbol":  map[string]any{"type": "string"},
		}},
	}

	op := &opDesc{goName: "Fake", flags: opFlags(ep, cmdDef{})}
	errs := checkFlags("Fake", cmdDef{}, ep, op)
	if !containsErr(errs, "flag --timeout collides with a global flag") {
		t.Errorf("reserved collision not reported: %v", errs)
	}

	def := cmdDef{flagAliases: map[string]string{"timeout": "order-timeout"}}
	op = &opDesc{goName: "Fake", flags: opFlags(ep, def)}
	if errs := checkFlags("Fake", def, ep, op); len(errs) > 0 {
		t.Errorf("aliased flag still reported: %v", errs)
	}
	var aliased *flagDesc
	for _, f := range op.flags {
		if f.oasName == "timeout" {
			aliased = f
		}
	}
	if aliased == nil || aliased.flagName != "order-timeout" {
		t.Fatalf("alias not applied: %#v", aliased)
	}

	def = cmdDef{flagAliases: map[string]string{"nope": "x"}, defaults: map[string]string{"missing": "1"}}
	errs = checkFlags("Fake", def, ep, &opDesc{flags: opFlags(ep, def)})
	if !containsErr(errs, `flagAliases key "nope" matches no flag`) || !containsErr(errs, `defaults key "missing" is not a body field`) {
		t.Errorf("stale alias/default not reported: %v", errs)
	}
}

func TestOpFlagsOrderingAndCLIDefault(t *testing.T) {
	testSchemas(t)
	ep := &endpointInfo{
		hasBody: true,
		bodySchema: map[string]any{
			"type":     "object",
			"required": []any{"symbol", "strategy", "exchange"},
			"properties": map[string]any{
				"symbol":   map[string]any{"type": "string"},
				"price":    map[string]any{"type": "number", "default": 0.0},
				"strategy": map[string]any{"type": "string"},
				"exchange": map[string]any{"$ref": "#/components/schemas/Exchange"},
				"action":   map[string]any{"type": "string"},
			},
		},
	}
	flags := opFlags(ep, cmdDef{defaults: strategyDefault})
	var names []string
	for _, f := range flags {
		names = append(names, f.flagName)
	}
	if got, want := strings.Join(names, ","), "exchange,strategy,symbol,action,price"; got != want {
		t.Errorf("flag order = %s, want %s (required first, then alpha)", got, want)
	}
	for _, f := range flags {
		if f.oasName == "strategy" {
			if !f.cliDefault || f.defaultVal != "openalgo-cli" || !f.required {
				t.Errorf("strategy flag = %#v, want CLIDefault openalgo-cli and required", f)
			}
		} else if f.cliDefault {
			t.Errorf("%s unexpectedly has CLIDefault", f.flagName)
		}
	}
}

func TestFreeFormObjectBody(t *testing.T) {
	testSchemas(t)
	ep := &endpointInfo{hasBody: true, bodySchema: map[string]any{"type": "object", "additionalProperties": true}}
	if !ep.isFreeFormBody() {
		t.Fatal("isFreeFormBody() = false")
	}
	flags := opFlags(ep, cmdDef{objectFlag: "preferences"})
	if len(flags) != 1 {
		t.Fatalf("got %d flags, want 1", len(flags))
	}
	if f := flags[0]; f.flagName != "preferences" || f.flagType != "object" || f.source != "body" || !f.required {
		t.Errorf("unexpected object flag: %#v", f)
	}

	closed := &endpointInfo{hasBody: true, bodySchema: map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"type": "string"}}}}
	if closed.isFreeFormBody() {
		t.Error("schema with properties treated as free-form")
	}
}

func TestRowsPath(t *testing.T) {
	testSchemas(t)
	obj := func(props map[string]any) map[string]any {
		return map[string]any{"type": "object", "properties": props}
	}
	arrOf := func(items map[string]any) map[string]any {
		return map[string]any{"type": "array", "items": items}
	}
	rec := obj(map[string]any{"symbol": map[string]any{"type": "string"}})

	tests := []struct {
		name string
		resp map[string]any
		want string
	}{
		{"data array of objects", obj(map[string]any{"data": arrOf(rec)}), "data"},
		{"data array of strings", obj(map[string]any{"data": arrOf(map[string]any{"type": "string"})}), "data"},
		{"data object with one record array", obj(map[string]any{"data": obj(map[string]any{
			"orders":     arrOf(rec),
			"statistics": obj(map[string]any{"total": map[string]any{"type": "number"}}),
			"tags":       arrOf(map[string]any{"type": "string"}),
		})}), "data.orders"},
		{"data object with two record arrays", obj(map[string]any{"data": obj(map[string]any{
			"bids": arrOf(rec), "asks": arrOf(rec),
		})}), ""},
		{"data plain object is one row", obj(map[string]any{"data": obj(map[string]any{"ltp": map[string]any{"type": "number"}})}), "data"},
		{"data object without properties", obj(map[string]any{"data": map[string]any{"type": "object"}}), ""},
		{"no data", obj(map[string]any{"results": arrOf(rec)}), ""},
		{"nullable data array", obj(map[string]any{"data": map[string]any{"type": []any{"array", "null"}, "items": rec}}), "data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rowsPath(tt.resp); got != tt.want {
				t.Errorf("rowsPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRowsPathRealSpec(t *testing.T) {
	_, ops := loadRealSpec(t)
	want := map[string]string{
		"GetOrderbook":    "data.orders",
		"GetHoldings":     "data.holdings",
		"GetPositionbook": "data",
		"GetHistory":      "data",
		"GetDepth":        "data", // registry override: one row, asks/bids as JSON
		"GetFunds":        "data",
		"PlaceOrder":      "",
		"BasketOrder":     "results",
		"SplitOrder":      "results",
		"GetMultiQuotes":  "results",
		"GetOptionChain":  "chain",
	}
	for _, op := range ops {
		if w, ok := want[op.goName]; ok && op.rowsPath != w {
			t.Errorf("%s RowsPath = %q, want %q", op.goName, op.rowsPath, w)
		}
	}
}

func TestBuildFetchBody(t *testing.T) {
	tests := []struct {
		name       string
		ep         *endpointInfo
		wantSnips  []string
		wantAbsent []string
	}{
		{
			name: "mutating body op supports dry-run",
			ep:   &endpointInfo{goName: "PlaceOrder", method: "POST", hasBody: true, mutating: true},
			wantSnips: []string{
				"body, err := bodyFromFlags(cmd, api.PlaceOrderOp)",
				`if cmdutil.Bool(cmd, "dry-run") {`,
				"return body, nil",
				"return openalgoClient.PlaceOrder(body)",
			},
		},
		{
			name:       "read-only body op has no dry-run",
			ep:         &endpointInfo{goName: "GetQuotes", method: "POST", hasBody: true},
			wantSnips:  []string{"return openalgoClient.GetQuotes(body)"},
			wantAbsent: []string{"dry-run"},
		},
		{
			name:       "no-body read",
			ep:         &endpointInfo{goName: "GetFunds", method: "POST"},
			wantSnips:  []string{"return openalgoClient.GetFunds()"},
			wantAbsent: []string{"bodyFromFlags", "dry-run"},
		},
		{
			name:      "no-body mutating op prints empty body on dry-run",
			ep:        &endpointInfo{goName: "TelegramStart", method: "POST", mutating: true},
			wantSnips: []string{"return map[string]any{}, nil", "return openalgoClient.TelegramStart()"},
		},
		{
			name: "GET with path and query params",
			ep: &endpointInfo{
				goName: "GetTicker", method: "GET",
				pathParams:  []paramInfo{{name: "symbol"}},
				queryParams: []paramInfo{{name: "from"}},
			},
			wantSnips: []string{`return openalgoClient.GetTicker(cmdutil.Str(cmd, "symbol"), queryFromFlags(cmd, api.GetTickerOp))`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFetchBody(tt.ep)
			for _, s := range tt.wantSnips {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q in:\n%s", s, got)
				}
			}
			for _, s := range tt.wantAbsent {
				if strings.Contains(got, s) {
					t.Errorf("unexpected %q in:\n%s", s, got)
				}
			}
		})
	}
}

func TestGenEndpointMethod(t *testing.T) {
	tests := []struct {
		ep   *endpointInfo
		want []string
	}{
		{
			ep: &endpointInfo{goName: "PlaceOrder", method: "POST", path: "/placeorder", hasBody: true, mutating: true},
			want: []string{
				"func (c *Client) PlaceOrder(body map[string]any) (json.RawMessage, error)",
				`c.Raw.DoWrite("POST", "/placeorder", nil, body)`,
			},
		},
		{
			ep: &endpointInfo{goName: "GetFunds", method: "POST", path: "/funds"},
			want: []string{
				"func (c *Client) GetFunds() (json.RawMessage, error)",
				`c.Raw.Do("POST", "/funds", nil, nil)`,
			},
		},
		{
			ep: &endpointInfo{
				goName: "GetTicker", method: "GET", path: "/ticker/{symbol}",
				pathParams:  []paramInfo{{name: "symbol"}},
				queryParams: []paramInfo{{name: "from"}},
			},
			want: []string{
				"func (c *Client) GetTicker(symbol string, params url.Values) (json.RawMessage, error)",
				`c.Raw.Do("GET", fmt.Sprintf("/ticker/%s", url.PathEscape(symbol)), params, nil)`,
			},
		},
	}
	for _, tt := range tests {
		var buf bytes.Buffer
		genEndpointMethod(&buf, tt.ep)
		for _, w := range tt.want {
			if !strings.Contains(buf.String(), w) {
				t.Errorf("%s: missing %q in:\n%s", tt.ep.goName, w, buf.String())
			}
		}
	}
}

func TestCheckExamples(t *testing.T) {
	op := &opDesc{goName: "PlaceOrder", mutating: true, flags: []*flagDesc{{flagName: "symbol"}, {flagName: "orders"}}}
	def := cmdDef{parent: "order", use: "place", examples: `  openalgo order place --symbol SBIN --dry-run --jq '.orderid'
  openalgo order place --orders '[{"--not-a-flag":1}]'`}
	if errs := checkExamples("PlaceOrder", def, op); len(errs) > 0 {
		t.Errorf("valid examples reported: %v", errs)
	}

	def.examples = `  openalgo order place --symbl SBIN
  openalgo order list
openalgo order place`
	errs := checkExamples("PlaceOrder", def, op)
	for _, want := range []string{"unknown flag --symbl", `does not start with "openalgo order place"`, "indented two spaces"} {
		if !containsErr(errs, want) {
			t.Errorf("expected error containing %q, got %v", want, errs)
		}
	}

	op.mutating = false
	def.examples = "  openalgo order place --dry-run"
	if errs := checkExamples("PlaceOrder", def, op); !containsErr(errs, "--dry-run is only available on mutating commands") {
		t.Errorf("dry-run on read op not reported: %v", errs)
	}
}

func TestCommandPath(t *testing.T) {
	tests := []struct {
		def  cmdDef
		want string
	}{
		{cmdDef{parent: "order", use: "place"}, "order place"},
		{cmdDef{parent: "telegramConfig", use: "get"}, "telegram config get"},
		{cmdDef{parent: "ping", self: true}, "ping"},
	}
	for _, tt := range tests {
		if got := commandPath(tt.def); got != tt.want {
			t.Errorf("commandPath(%+v) = %q, want %q", tt.def, got, tt.want)
		}
	}
}

func TestStripQuoted(t *testing.T) {
	tests := []struct{ in, want string }{
		{`a 'b --c' d`, "a  d"},
		{`--x "--y" --z`, "--x  --z"},
		{`no quotes`, "no quotes"},
	}
	for _, tt := range tests {
		if got := stripQuoted(tt.in); got != tt.want {
			t.Errorf("stripQuoted(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCmdVarNameAvoidsParentCollision(t *testing.T) {
	if got := cmdVarName("Ping", cmdDef{parent: "order", use: "check"}); got != "pingCheckCmd" {
		t.Errorf("cmdVarName collided with parent var: %q", got)
	}
	if got := cmdVarName("PlaceOrder", cmdDef{parent: "order", use: "place"}); got != "placeOrderCmd" {
		t.Errorf("cmdVarName = %q", got)
	}
}
