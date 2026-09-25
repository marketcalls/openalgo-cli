package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/marketcalls/openalgo-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

const dryRunUsage = "Print the request body without sending it; the API key is never printed"

// fetchCmd creates a command that fetches data and renders it.
// All OAS flags (path, query, body) are auto-registered from the Op, and
// mutating ops get a --dry-run flag.
func fetchCmd(use string, op api.Op, fetch func(cmd *cobra.Command, args []string) (any, error), configure ...func(*cobra.Command)) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Short:   op.Summary,
		Long:    op.Long,
		Example: op.Example,
		Args:    cobra.NoArgs,
	}
	wireOp(cmd, op, fetch, configure)
	return cmd
}

// attachCmd mirrors fetchCmd but operates on an existing command. Used for
// parent group commands that are also directly runnable (e.g. `ping`).
func attachCmd(cmd *cobra.Command, op api.Op, fetch func(cmd *cobra.Command, args []string) (any, error), configure ...func(*cobra.Command)) {
	if op.Long != "" {
		cmd.Long = op.Long
	}
	if op.Example != "" {
		cmd.Example = op.Example
	}
	if cmd.Short == "" {
		cmd.Short = op.Summary
	}
	wireOp(cmd, op, fetch, configure)
}

func wireOp(cmd *cobra.Command, op api.Op, fetch func(cmd *cobra.Command, args []string) (any, error), configure []func(*cobra.Command)) {
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := requireFlags(cmd, op); err != nil {
			return err
		}
		data, err := fetch(cmd, args)
		if err != nil {
			return err
		}
		return renderData(cmd, data)
	}
	for _, fn := range configure {
		fn(cmd)
	}
	cmdutil.RegisterFlags(cmd, op.Flags, op.Name, nil)
	if op.Mutating && cmd.Flags().Lookup("dry-run") == nil {
		cmd.Flags().Bool("dry-run", false, dryRunUsage)
	}
}

// requireFlags validates that all OAS-required flags have been provided,
// that enum flags hold one of their allowed values, and any per-operation
// rule in opChecks. A CLI-supplied default (e.g. strategy=openalgo-cli)
// satisfies Required.
func requireFlags(cmd *cobra.Command, op api.Op) error {
	var req []string
	for _, f := range op.Flags {
		if f.Required && !f.CLIDefault {
			req = append(req, f.Name)
		}
	}
	if len(req) > 0 {
		if err := cmdutil.RequireAll(cmd, req...); err != nil {
			return err
		}
	}
	for _, f := range op.Flags {
		if err := cmdutil.CheckEnum(cmd, f); err != nil {
			return err
		}
	}
	if check := opChecks[op.Name]; check != nil {
		return check(cmd)
	}
	return nil
}

// opChecks holds client-side rules the spec cannot express, keyed by
// operation name. They run before the request (and before --dry-run
// prints), so mistakes surface without a round trip.
var opChecks = map[string]func(cmd *cobra.Command) error{
	"WhatsAppNotify":  checkWhatsAppNotify,
	"PlaceOrder":      checkTriggerPrice,
	"PlaceSmartOrder": checkTriggerPrice,
	"SplitOrder":      checkTriggerPrice,
	"ModifyOrder":     checkTriggerPrice,
	"OptionsOrder":    checkTriggerPrice,
}

// checkTriggerPrice requires an explicit --trigger-price above 0 for SL and
// SL-M orders. The 0 default only makes sense for MARKET and LIMIT; the
// server accepts 0 and would forward it to the broker unchecked.
func checkTriggerPrice(cmd *cobra.Command) error {
	pt := cmdutil.Str(cmd, "pricetype")
	if !strings.EqualFold(pt, "SL") && !strings.EqualFold(pt, "SL-M") {
		return nil
	}
	if !cmd.Flags().Changed("trigger-price") {
		return fmt.Errorf("--trigger-price is required when --pricetype is %s", strings.ToUpper(pt))
	}
	s := strings.TrimSpace(cmdutil.Str(cmd, "trigger-price"))
	if v, err := strconv.ParseFloat(s, 64); err != nil || v <= 0 {
		return fmt.Errorf("--trigger-price must be above 0 when --pricetype is %s, got %q", strings.ToUpper(pt), s)
	}
	return nil
}

// checkWhatsAppNotify enforces exactly one recipient form and some content.
func checkWhatsAppNotify(cmd *cobra.Command) error {
	var set []string
	for _, name := range []string{"self", "username", "phone", "phones"} {
		// --self=false is the same as leaving --self out.
		if cmd.Flags().Changed(name) && (name != "self" || cmdutil.Bool(cmd, name)) {
			set = append(set, "--"+name)
		}
	}
	switch len(set) {
	case 0:
		return fmt.Errorf("one recipient is required: --self, --username, --phone, or --phones")
	case 1:
	default:
		return fmt.Errorf("use exactly one recipient form, got %s", strings.Join(set, " and "))
	}
	for _, name := range []string{"message", "image-path", "document-path"} {
		if cmd.Flags().Changed(name) {
			return nil
		}
	}
	return fmt.Errorf("nothing to send: pass --message, --image-path, or --document-path")
}

// expiryDashed matches DD-MMM-YY, the form `openalgo symbol expiry` prints.
var expiryDashed = regexp.MustCompile(`^\d{2}-[A-Za-z]{3}-\d{2}$`)

// normalizeExpiry converts DD-MMM-YY (27-OCT-26) to the DDMMMYY form
// (27OCT26) the option endpoints expect, so expiry output can be chained
// straight into option commands. Other values are left alone.
func normalizeExpiry(v any) any {
	s, ok := v.(string)
	if !ok || !expiryDashed.MatchString(s) {
		return v
	}
	return strings.ToUpper(strings.ReplaceAll(s, "-", ""))
}

// normalizeExpiries applies normalizeExpiry to expiry_date at the body root
// and inside arrays of objects (option legs).
func normalizeExpiries(body map[string]any) {
	for k, v := range body {
		if k == "expiry_date" {
			body[k] = normalizeExpiry(v)
			continue
		}
		if items, ok := v.([]any); ok {
			for _, item := range items {
				if m, ok := item.(map[string]any); ok {
					if e, ok := m["expiry_date"]; ok {
						m["expiry_date"] = normalizeExpiry(e)
					}
				}
			}
		}
	}
}

// flagIncluded reports whether a flag's value should be sent: either the
// user set it, or it carries a CLI-supplied default.
func flagIncluded(cmd *cobra.Command, f api.FlagDef) bool {
	return cmd.Flags().Lookup(f.Name) != nil && (cmd.Flags().Changed(f.Name) || f.CLIDefault)
}

// queryFromFlags builds url.Values from cobra flags using FlagDef metadata,
// replacing per-endpoint *ParamsFromFlags boilerplate with a single runtime helper.
func queryFromFlags(cmd *cobra.Command, op api.Op) url.Values {
	v := url.Values{}
	for _, f := range op.Flags {
		if f.Source != "query" || !flagIncluded(cmd, f) {
			continue
		}
		switch f.Type {
		case "int":
			v.Set(f.OASName, strconv.Itoa(cmdutil.Int(cmd, f.Name)))
		case "bool":
			v.Set(f.OASName, strconv.FormatBool(cmdutil.Bool(cmd, f.Name)))
		default:
			v.Set(f.OASName, cmdutil.CanonicalEnum(cmdutil.Str(cmd, f.Name), f.Completions))
		}
	}
	return v
}

// bodyFromFlags builds the JSON request body from Source=="body" flags.
// Values keep their JSON types: numbers go out as json.Number (so the exact
// text the user typed is sent, never a rounded float), and json/object flags
// are parsed from a literal, an @file path, or "-" for stdin. An object flag
// is merged into the body root; explicitly named flags win on conflict.
func bodyFromFlags(cmd *cobra.Command, op api.Op) (map[string]any, error) {
	body := map[string]any{}
	var merges []map[string]any
	for _, f := range op.Flags {
		if f.Source != "body" || !flagIncluded(cmd, f) {
			continue
		}
		switch f.Type {
		case "int":
			body[f.OASName] = cmdutil.Int(cmd, f.Name)
		case "bool":
			body[f.OASName] = cmdutil.Bool(cmd, f.Name)
		case "number":
			s := strings.TrimSpace(cmdutil.Str(cmd, f.Name))
			if _, err := strconv.ParseFloat(s, 64); err != nil || !json.Valid([]byte(s)) {
				return nil, fmt.Errorf("--%s: %q is not a number", f.Name, s)
			}
			body[f.OASName] = json.Number(s)
		case "json":
			val, err := parseJSONFlag(cmd, f.Name)
			if err != nil {
				return nil, err
			}
			body[f.OASName] = val
		case "array":
			val, err := parseJSONFlag(cmd, f.Name)
			if err != nil {
				return nil, err
			}
			if _, ok := val.([]any); !ok {
				return nil, fmt.Errorf("--%s must be a JSON array, e.g. '[...]' (literal, @file, or - for stdin)", f.Name)
			}
			body[f.OASName] = val
		case "object":
			val, err := parseJSONFlag(cmd, f.Name)
			if err != nil {
				return nil, err
			}
			obj, ok := val.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("--%s must be a JSON object, e.g. '{\"key\":\"value\"}'", f.Name)
			}
			merges = append(merges, obj)
		default:
			body[f.OASName] = cmdutil.CanonicalEnum(cmdutil.Str(cmd, f.Name), f.Completions)
		}
	}
	for _, obj := range merges {
		for k, v := range obj {
			if _, exists := body[k]; !exists {
				body[k] = v
			}
		}
	}
	normalizeExpiries(body)
	return body, nil
}

// stdinReader is swapped out by tests.
var stdinReader io.Reader = os.Stdin

// parseJSONFlag reads a json/object flag value. "@path" reads a file, "-"
// reads stdin, anything else is parsed as literal JSON. Numbers are decoded
// as json.Number to round-trip exactly.
func parseJSONFlag(cmd *cobra.Command, name string) (any, error) {
	raw := strings.TrimSpace(cmdutil.Str(cmd, name))
	var data []byte
	switch {
	case raw == "-":
		b, err := io.ReadAll(stdinReader)
		if err != nil {
			return nil, fmt.Errorf("--%s: reading stdin: %w", name, err)
		}
		data = b
	case strings.HasPrefix(raw, "@"):
		b, err := os.ReadFile(raw[1:])
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", name, err)
		}
		data = b
	default:
		data = []byte(raw)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("--%s: invalid JSON: %w", name, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("--%s: invalid JSON: unexpected data after the first value", name)
	}
	return v, nil
}
