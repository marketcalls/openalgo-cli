package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/marketcalls/openalgo-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// testOp is a hand-built op covering every FlagDef type so these tests do
// not depend on the generated descriptions.
var testOp = api.Op{
	Name:     "TestOp",
	Method:   "POST",
	Path:     "/test",
	Mutating: true,
	Flags: []api.FlagDef{
		{Name: "symbol", OASName: "symbol", Type: "string", Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "int", Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Default: "0", Source: "body"},
		{Name: "enabled", OASName: "enabled", Type: "bool", Source: "body"},
		{Name: "orders", OASName: "orders", Type: "json", Source: "body"},
		{Name: "preferences", OASName: "preferences", Type: "object", Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Required: true, CLIDefault: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Source: "query"},
		{Name: "limit", OASName: "limit", Type: "int", Source: "query"},
		{Name: "format", OASName: "format", Type: "string", Default: "json", Source: "query"},
	},
}

func newOpCmd(t *testing.T, op api.Op, set map[string]string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	cmdutil.RegisterFlags(cmd, op.Flags, "", nil)
	for k, v := range set {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatalf("set --%s: %v", k, err)
		}
	}
	return cmd
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBodyFromFlags_TypeConversions(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{
		"symbol":   "RELIANCE",
		"quantity": "10",
		"price":    "2450.05",
		"enabled":  "true",
		"orders":   `[{"symbol":"SBIN","quantity":1}]`,
	})
	body, err := bodyFromFlags(cmd, testOp)
	if err != nil {
		t.Fatal(err)
	}

	got := marshal(t, body)
	want := `{"enabled":true,"orders":[{"quantity":1,"symbol":"SBIN"}],"price":2450.05,"quantity":10,"strategy":"openalgo-cli","symbol":"RELIANCE"}`
	if got != want {
		t.Errorf("body =\n  %s\nwant\n  %s", got, want)
	}
	if _, ok := body["price"].(json.Number); !ok {
		t.Errorf("price is %T, want json.Number", body["price"])
	}
	if _, ok := body["exchange"]; ok {
		t.Error("query flag leaked into the body")
	}
}

func TestBodyFromFlags_OmitsUnchanged(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{"symbol": "SBIN"})
	body, err := bodyFromFlags(cmd, testOp)
	if err != nil {
		t.Fatal(err)
	}
	// Only the explicit flag and the CLI default; server defaults (price=0)
	// are left to the server.
	if got := marshal(t, body); got != `{"strategy":"openalgo-cli","symbol":"SBIN"}` {
		t.Errorf("body = %s", got)
	}
}

func TestBodyFromFlags_CLIDefaultOverride(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{"symbol": "SBIN", "strategy": "my-algo"})
	body, _ := bodyFromFlags(cmd, testOp)
	if body["strategy"] != "my-algo" {
		t.Errorf("strategy = %v", body["strategy"])
	}
}

func TestBodyFromFlags_NumberPreservesText(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{"price": "0.1"})
	body, _ := bodyFromFlags(cmd, testOp)
	if got := marshal(t, body["price"]); got != "0.1" {
		t.Errorf("price = %s, want 0.1", got)
	}
}

func TestBodyFromFlags_InvalidNumber(t *testing.T) {
	for _, v := range []string{"abc", "1,000", "NaN", "Inf", "0x10"} {
		cmd := newOpCmd(t, testOp, map[string]string{"price": v})
		if _, err := bodyFromFlags(cmd, testOp); err == nil || !strings.Contains(err.Error(), "--price") {
			t.Errorf("price=%q: err = %v, want a --price error", v, err)
		}
	}
}

func TestBodyFromFlags_InvalidJSON(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{"orders": `[{"symbol":`})
	if _, err := bodyFromFlags(cmd, testOp); err == nil || !strings.Contains(err.Error(), "--orders") {
		t.Errorf("err = %v", err)
	}
	cmd = newOpCmd(t, testOp, map[string]string{"orders": `[] []`})
	if _, err := bodyFromFlags(cmd, testOp); err == nil {
		t.Error("expected error for trailing data")
	}
}

func TestBodyFromFlags_JSONFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orders.json")
	if err := os.WriteFile(path, []byte(`[{"symbol":"INFY","quantity":5}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newOpCmd(t, testOp, map[string]string{"orders": "@" + path})
	body, err := bodyFromFlags(cmd, testOp)
	if err != nil {
		t.Fatal(err)
	}
	if got := marshal(t, body["orders"]); got != `[{"quantity":5,"symbol":"INFY"}]` {
		t.Errorf("orders = %s", got)
	}

	cmd = newOpCmd(t, testOp, map[string]string{"orders": "@" + path + ".missing"})
	if _, err := bodyFromFlags(cmd, testOp); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestBodyFromFlags_JSONFromStdin(t *testing.T) {
	old := stdinReader
	stdinReader = strings.NewReader(`{"a":1}`)
	t.Cleanup(func() { stdinReader = old })

	cmd := newOpCmd(t, testOp, map[string]string{"orders": "-"})
	body, err := bodyFromFlags(cmd, testOp)
	if err != nil {
		t.Fatal(err)
	}
	if got := marshal(t, body["orders"]); got != `{"a":1}` {
		t.Errorf("orders = %s", got)
	}
}

func TestBodyFromFlags_ObjectMerge(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{
		"symbol":      "SBIN",
		"preferences": `{"theme":"dark","symbol":"IGNORED","layout":{"cols":2}}`,
	})
	body, err := bodyFromFlags(cmd, testOp)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body["preferences"]; ok {
		t.Error("object flag should be merged into the root, not nested")
	}
	if body["theme"] != "dark" {
		t.Errorf("theme = %v", body["theme"])
	}
	if body["symbol"] != "SBIN" {
		t.Errorf("explicit flag should win over merged object, symbol = %v", body["symbol"])
	}
	if got := marshal(t, body["layout"]); got != `{"cols":2}` {
		t.Errorf("layout = %s", got)
	}
}

func TestBodyFromFlags_ObjectMustBeObject(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{"preferences": `[1,2]`})
	if _, err := bodyFromFlags(cmd, testOp); err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Errorf("err = %v", err)
	}
}

func TestQueryFromFlags(t *testing.T) {
	cmd := newOpCmd(t, testOp, map[string]string{"exchange": "NSE", "limit": "0", "symbol": "SBIN"})
	v := queryFromFlags(cmd, testOp)
	if v.Get("exchange") != "NSE" {
		t.Errorf("exchange = %q", v.Get("exchange"))
	}
	if v.Get("limit") != "0" {
		t.Errorf("explicit zero int should be sent, limit = %q", v.Get("limit"))
	}
	if v.Has("format") {
		t.Error("unchanged server default should not be sent")
	}
	if v.Has("symbol") {
		t.Error("body flag leaked into the query")
	}
}

func TestRequireFlags(t *testing.T) {
	cmd := newOpCmd(t, testOp, nil)
	err := requireFlags(cmd, testOp)
	if err == nil || !strings.Contains(err.Error(), "--symbol") {
		t.Fatalf("err = %v, want --symbol missing", err)
	}
	if strings.Contains(err.Error(), "--strategy") {
		t.Error("CLI default should satisfy the required strategy flag")
	}

	cmd = newOpCmd(t, testOp, map[string]string{"symbol": "SBIN"})
	if err := requireFlags(cmd, testOp); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchCmd_DryRunRegistration(t *testing.T) {
	noop := func(*cobra.Command, []string) (any, error) { return nil, nil }

	c := fetchCmd("place", testOp, noop)
	if f := c.Flags().Lookup("dry-run"); f == nil || !strings.Contains(f.Usage, "API key is never printed") {
		t.Error("mutating op should register --dry-run")
	}
	if c.Annotations["op"] != "TestOp" {
		t.Errorf("op annotation = %q", c.Annotations["op"])
	}

	readOp := testOp
	readOp.Mutating = false
	if fetchCmd("get", readOp, noop).Flags().Lookup("dry-run") != nil {
		t.Error("read-only op should not register --dry-run")
	}

	parent := &cobra.Command{Use: "ping", Short: "Ping"}
	attachCmd(parent, testOp, noop)
	if parent.RunE == nil || parent.Flags().Lookup("dry-run") == nil {
		t.Error("attachCmd should make the command runnable with --dry-run")
	}
}

func TestFetchCmd_DryRunReturnsBodyWithoutKey(t *testing.T) {
	setupTestOutput(t)
	c := fetchCmd("place", testOp, func(cmd *cobra.Command, args []string) (any, error) {
		body, err := bodyFromFlags(cmd, testOp)
		if err != nil {
			return nil, err
		}
		if cmdutil.Bool(cmd, "dry-run") {
			return body, nil
		}
		t.Fatal("dry-run should not reach the API")
		return nil, nil
	})
	out := runCmd(t, c, "--symbol", "SBIN", "--quantity", "1", "--dry-run")
	if strings.Contains(out, "apikey") {
		t.Errorf("dry-run output contains apikey: %s", out)
	}
	if !strings.Contains(out, `"symbol": "SBIN"`) {
		t.Errorf("dry-run output = %s", out)
	}
}

func TestBodyFromFlags_ArrayMustBeArray(t *testing.T) {
	op := api.Op{Flags: []api.FlagDef{{Name: "phones", OASName: "phones", Type: "array", Source: "body"}}}
	cmd := newOpCmd(t, op, map[string]string{"phones": "919876543210"})
	if _, err := bodyFromFlags(cmd, op); err == nil || !strings.Contains(err.Error(), "must be a JSON array") {
		t.Errorf("bare number: err = %v, want JSON array error", err)
	}
	cmd = newOpCmd(t, op, map[string]string{"phones": `["919876543210"]`})
	body, err := bodyFromFlags(cmd, op)
	if err != nil || marshal(t, body) != `{"phones":["919876543210"]}` {
		t.Errorf("array: body = %v, err = %v", body, err)
	}
}

func TestBodyFromFlags_NormalizesExpiry(t *testing.T) {
	op := api.Op{Flags: []api.FlagDef{
		{Name: "expiry-date", OASName: "expiry_date", Type: "string", Source: "body"},
		{Name: "legs", OASName: "legs", Type: "array", Source: "body"},
	}}
	cmd := newOpCmd(t, op, map[string]string{
		"expiry-date": "27-oct-26",
		"legs":        `[{"offset":"ATM","expiry_date":"24-NOV-26"},{"offset":"OTM1","expiry_date":"27OCT26"}]`,
	})
	body, err := bodyFromFlags(cmd, op)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"expiry_date":"27OCT26","legs":[{"expiry_date":"24NOV26","offset":"ATM"},{"expiry_date":"27OCT26","offset":"OTM1"}]}`
	if got := marshal(t, body); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestRequireFlags_EnumAndOpChecks(t *testing.T) {
	op := api.Op{Name: "EnumOp", Flags: []api.FlagDef{
		{Name: "action", OASName: "action", Type: "string", Required: true, Completions: []string{"BUY", "SELL"}, Source: "body"},
	}}
	if err := requireFlags(newOpCmd(t, op, map[string]string{"action": "FOO"}), op); err == nil {
		t.Error("requireFlags accepted --action FOO")
	}
	if err := requireFlags(newOpCmd(t, op, map[string]string{"action": "sell"}), op); err != nil {
		t.Errorf("requireFlags rejected sell: %v", err)
	}

	wa, ok := api.OpByName("WhatsAppNotify")
	if !ok {
		t.Fatal("WhatsAppNotify op missing")
	}
	cases := []struct {
		set     map[string]string
		wantErr string
	}{
		{map[string]string{"message": "hi"}, "one recipient is required"},
		{map[string]string{"self": "true", "phone": "91", "message": "x"}, "exactly one recipient"},
		{map[string]string{"phone": "91"}, "nothing to send"},
		{map[string]string{"self": "0", "phone": "91", "message": "x"}, ""},
	}
	for _, c := range cases {
		err := requireFlags(newOpCmd(t, wa, c.set), wa)
		if c.wantErr == "" && err != nil {
			t.Errorf("%v: unexpected error %v", c.set, err)
		}
		if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
			t.Errorf("%v: err = %v, want %q", c.set, err, c.wantErr)
		}
	}
}
