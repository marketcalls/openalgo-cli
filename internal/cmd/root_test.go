package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/marketcalls/openalgo-cli/internal/cmdutil"
	"github.com/marketcalls/openalgo-cli/internal/config"
	"github.com/spf13/cobra"
)

// setupTestOutput resets the package-level output state that renderData
// reads, restoring it when the test ends.
func setupTestOutput(t *testing.T) {
	t.Helper()
	oldCfg, oldCSV, oldJQ, oldNoColor := cfg, csvFlag, jqFlag, color.NoColor
	cfg, csvFlag, jqFlag, color.NoColor = &config.Resolved{Output: "json"}, false, "", true
	t.Cleanup(func() { cfg, csvFlag, jqFlag, color.NoColor = oldCfg, oldCSV, oldJQ, oldNoColor })
}

// runCmd executes a standalone command (not the shared root) and returns
// its stdout.
func runCmd(t *testing.T, c *cobra.Command, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs(args)
	if err := c.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	return buf.String()
}

// opCmd builds a command annotated with the given generated op, as fetchCmd
// would, so renderData can find its RowsPath and schema.
func opCmd(opName string) (*cobra.Command, *bytes.Buffer) {
	c := &cobra.Command{Use: "test"}
	cmdutil.RegisterFlags(c, nil, opName, nil)
	var buf bytes.Buffer
	c.SetOut(&buf)
	return c, &buf
}

const orderbookResponse = `{"status":"success","data":{"orders":[
  {"orderid":"1","symbol":"SBIN","quantity":"1","price":801.5},
  {"orderid":"2","symbol":"INFY","quantity":"5","price":0}
],"statistics":{"total_buy_orders":2}}}`

func TestRenderData_JSON(t *testing.T) {
	setupTestOutput(t)
	c, buf := opCmd("GetOrderbook")
	if err := renderData(c, json.RawMessage(orderbookResponse)); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, buf.String())
	}
	if m["status"] != "success" {
		t.Errorf("envelope not preserved: %s", buf.String())
	}
}

func TestRenderData_CSVRowsPath(t *testing.T) {
	setupTestOutput(t)
	op, ok := api.OpByName("GetOrderbook")
	if !ok || op.RowsPath == "" {
		t.Skip("GetOrderbook has no RowsPath in the generated descriptions")
	}
	csvFlag = true
	c, buf := opCmd("GetOrderbook")
	if err := renderData(c, json.RawMessage(orderbookResponse)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("want header + 2 rows, got:\n%s", buf.String())
	}
	// Columns follow the schema order, whether or not rows exist.
	if !strings.HasPrefix(lines[0], "orderid,symbol,exchange,action,quantity,filled_quantity,pending_quantity,price,") {
		t.Errorf("header = %q", lines[0])
	}
	// Prices keep their full text (801.5, not 801.50).
	if !strings.HasPrefix(lines[1], "1,SBIN,,,1,,,801.5,") {
		t.Errorf("row 1 = %q", lines[1])
	}
}

func TestRenderData_CSVEmptyRowsUsesSchemaHeaders(t *testing.T) {
	setupTestOutput(t)
	op, ok := api.OpByName("GetOrderbook")
	if !ok || op.RowsPath == "" {
		t.Skip("GetOrderbook has no RowsPath in the generated descriptions")
	}
	csvFlag = true
	c, buf := opCmd("GetOrderbook")
	if err := renderData(c, json.RawMessage(`{"status":"success","data":{"orders":[]}}`)); err != nil {
		t.Fatal(err)
	}
	header := strings.TrimSpace(buf.String())
	if !strings.Contains(header, "orderid") || !strings.Contains(header, "symbol") {
		t.Errorf("header = %q, want schema item fields", header)
	}
}

func TestRenderData_JQ(t *testing.T) {
	setupTestOutput(t)
	jqFlag = ".data.orders[].symbol"
	c, buf := opCmd("GetOrderbook")
	if err := renderData(c, json.RawMessage(orderbookResponse)); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(strings.Fields(buf.String()), ""); got != `["SBIN","INFY"]` {
		t.Errorf("jq output = %s", buf.String())
	}
}

func TestRenderData_NonJSONPassthrough(t *testing.T) {
	setupTestOutput(t)
	raw := "symbol,exchange\nRELIANCE,NSE"
	for _, csv := range []bool{false, true} {
		csvFlag = csv
		c, buf := opCmd("GetInstruments")
		if err := renderData(c, json.RawMessage(raw)); err != nil {
			t.Fatal(err)
		}
		if buf.String() != raw+"\n" {
			t.Errorf("csv=%v: output = %q, want verbatim", csv, buf.String())
		}
	}
}

func TestRenderData_JQOnNonJSONFails(t *testing.T) {
	setupTestOutput(t)
	jqFlag = "."
	c, _ := opCmd("GetTicker")
	if err := renderData(c, json.RawMessage("2024-01-01,1,2,3,4")); err == nil {
		t.Error("expected an error applying --jq to a non-JSON body")
	}
}

func TestRowsAtPath(t *testing.T) {
	data := json.RawMessage(`{"data":{"orders":[{"a":1}]},"list":[1]}`)
	if got := marshal(t, rowsAtPath(data, "data.orders")); got != `[{"a":1}]` {
		t.Errorf("data.orders = %s", got)
	}
	if rowsAtPath(data, "data.missing") != nil {
		t.Error("missing path should be nil")
	}
	if rowsAtPath(data, "list.x") != nil {
		t.Error("walking into a non-object should be nil")
	}
	if got := marshal(t, rowsAtPath(map[string]any{"data": []int{1}}, "data")); got != `[1]` {
		t.Errorf("map input = %s", got)
	}
}

func TestRowHeaders(t *testing.T) {
	fields := []api.ResponseField{
		{Name: "status", Type: "string"},
		{Name: "data", Type: "object", Fields: []api.ResponseField{
			{Name: "orders", Type: "[]object", Fields: []api.ResponseField{
				{Name: "orderid", Type: "string"},
				{Name: "symbol", Type: "string"},
			}},
		}},
	}
	if got := strings.Join(rowHeaders(fields, "data.orders"), ","); got != "orderid,symbol" {
		t.Errorf("headers = %q", got)
	}
	if rowHeaders(fields, "data.nope") != nil {
		t.Error("unknown path should give no headers")
	}
}

func TestSplitHint(t *testing.T) {
	msg, hint := splitHint("authentication required\nHint: run `openalgo profile login`")
	if msg != "authentication required" || hint != "run `openalgo profile login`" {
		t.Errorf("got %q / %q", msg, hint)
	}
	if msg, hint := splitHint("plain"); msg != "plain" || hint != "" {
		t.Errorf("got %q / %q", msg, hint)
	}
}

func TestAuthRequiredErrorUnwraps(t *testing.T) {
	inner := errors.New("authentication required")
	var target *authRequiredError
	if !errors.As(error(&authRequiredError{inner}), &target) || !errors.Is(target, inner) {
		t.Error("authRequiredError should be detectable and unwrap")
	}
}

func TestNeedsAuth(t *testing.T) {
	find := func(path ...string) *cobra.Command {
		c, _, err := rootCmd.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		return c
	}
	for _, path := range [][]string{{"version"}, {"doctor"}, {"update"}, {"profile", "login"}, {"profile", "list"}} {
		if needsAuth(find(path...)) {
			t.Errorf("%v should not need auth", path)
		}
	}
	for _, path := range [][]string{{"api"}, {"ping"}, {"order", "place"}, {"account", "funds"}} {
		if !needsAuth(find(path...)) {
			t.Errorf("%v should need auth", path)
		}
	}
	if needsAuth(rootCmd) {
		t.Error("bare root prints help and should not need auth")
	}
}

func TestNeedsAuth_DryRunSkipsAuth(t *testing.T) {
	parent := &cobra.Command{Use: "openalgo"}
	child := &cobra.Command{Use: "place", RunE: func(*cobra.Command, []string) error { return nil }}
	child.Flags().Bool("dry-run", false, "")
	parent.AddCommand(child)
	if !needsAuth(child) {
		t.Fatal("without --dry-run the command should need auth")
	}
	if err := child.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}
	if needsAuth(child) {
		t.Error("--dry-run never calls the API and should not need auth")
	}
}

func TestEnvBool(t *testing.T) {
	for v, want := range map[string]bool{"": false, "0": false, "false": false, "no": false, "1": true, "true": true, "yes": true} {
		t.Setenv("OPENALGO_TEST_BOOL", v)
		if got := envBool("OPENALGO_TEST_BOOL"); got != want {
			t.Errorf("envBool(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestRootGroups(t *testing.T) {
	want := map[string]string{
		"order": "trading", "gtt": "trading", "position": "trading", "option": "trading", "strategy": "trading",
		"account": "account", "data": "account", "symbol": "account", "market": "account", "analyzer": "account",
		"telegram": "notify", "whatsapp": "notify",
		"chart": "util", "ping": "util", "profile": "util", "api": "util", "doctor": "util", "update": "util", "version": "util",
	}
	for name, group := range want {
		c, _, err := rootCmd.Find([]string{name})
		if err != nil || c.Name() != name {
			t.Errorf("command %q not registered", name)
			continue
		}
		if c.GroupID != group {
			t.Errorf("%s group = %q, want %q", name, c.GroupID, group)
		}
	}
}
