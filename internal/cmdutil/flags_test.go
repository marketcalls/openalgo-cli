package cmdutil

import (
	"strings"
	"testing"

	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("name", "default", "")
	cmd.Flags().Bool("verbose", false, "")
	cmd.Flags().Int("count", 0, "")
	return cmd
}

func TestStr(t *testing.T) {
	cmd := newTestCmd()
	if got := Str(cmd, "name"); got != "default" {
		t.Errorf("Str(name) = %q, want %q", got, "default")
	}
}

func TestBool(t *testing.T) {
	cmd := newTestCmd()
	if got := Bool(cmd, "verbose"); got != false {
		t.Errorf("Bool(verbose) = %v, want false", got)
	}
}

func TestInt(t *testing.T) {
	cmd := newTestCmd()
	if got := Int(cmd, "count"); got != 0 {
		t.Errorf("Int(count) = %d, want 0", got)
	}
}

func TestChanged(t *testing.T) {
	cmd := newTestCmd()
	if got := Changed(cmd, "name"); got != false {
		t.Errorf("Changed(name) = %v, want false (not set)", got)
	}
}

func TestRegisterFlags_TypeDispatch(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	defs := []api.FlagDef{
		{Name: "name", OASName: "name", Type: "string", Default: "hello", Description: "A name"},
		{Name: "count", OASName: "count", Type: "int", Default: "42", Description: "A count"},
		{Name: "active", OASName: "active", Type: "bool", Default: "true", Description: "Active flag"},
		{Name: "price", OASName: "price", Type: "number", Default: "9.99", Description: "A price"},
		{Name: "orders", OASName: "orders", Type: "json", Description: "Orders array"},
		{Name: "preferences", OASName: "preferences", Type: "object", Description: "Free-form object"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", CLIDefault: true},
	}
	RegisterFlags(cmd, defs, "", nil)

	tests := []struct {
		flag     string
		wantType string
		wantDef  string
	}{
		// Spec defaults the CLI does not send are server defaults: they go
		// into the usage text, not the cobra default.
		{"name", "string", ""},
		{"count", "int", "0"},
		{"active", "bool", "false"},
		{"price", "string", ""},
		{"orders", "string", ""},
		{"preferences", "string", ""},
		{"strategy", "string", "openalgo-cli"},
	}
	for _, tt := range tests {
		f := cmd.Flags().Lookup(tt.flag)
		if f == nil {
			t.Errorf("flag %q not registered", tt.flag)
			continue
		}
		if f.Value.Type() != tt.wantType {
			t.Errorf("flag %q type = %s, want %s", tt.flag, f.Value.Type(), tt.wantType)
		}
		if f.DefValue != tt.wantDef {
			t.Errorf("flag %q default = %s, want %s", tt.flag, f.DefValue, tt.wantDef)
		}
	}
}

func TestRegisterFlags_DefaultOverride(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	defs := []api.FlagDef{
		{Name: "timeframe", OASName: "timeframe", Type: "string", Default: "1Min", Description: "Timeframe"},
	}
	RegisterFlags(cmd, defs, "", &FlagOpts{
		Defaults: map[string]string{"timeframe": "1Day"},
	})

	f := cmd.Flags().Lookup("timeframe")
	if f == nil {
		t.Fatal("timeframe not registered")
	}
	if f.DefValue != "1Day" {
		t.Errorf("default = %s, want 1Day (overridden from 1Min)", f.DefValue)
	}
}

func TestRegisterFlags_Completions(t *testing.T) {
	cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
	defs := []api.FlagDef{
		{
			Name:        "status",
			OASName:     "status",
			Type:        "string",
			Description: "Status filter",
			Completions: []string{"active", "inactive"},
		},
		{
			Name:        "name",
			OASName:     "name",
			Type:        "string",
			Description: "A name (no completions)",
		},
	}
	RegisterFlags(cmd, defs, "", nil)

	if cmd.Flags().Lookup("status") == nil {
		t.Error("status flag not registered")
	}
	if cmd.Flags().Lookup("name") == nil {
		t.Error("name flag not registered")
	}
	if cmd.Flags().Lookup("status").Usage != "Status filter (one of: active, inactive)" {
		t.Errorf("status description = %q", cmd.Flags().Lookup("status").Usage)
	}
}

func TestRegisterFlags_NilOpts(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	defs := []api.FlagDef{
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "Symbol"},
	}
	RegisterFlags(cmd, defs, "", nil)

	if cmd.Flags().Lookup("symbol") == nil {
		t.Error("flag not registered with nil opts")
	}
}

func TestRegisterFlags_EmptyDefs(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd, nil, "", nil)
	RegisterFlags(cmd, []api.FlagDef{}, "", nil)

	if cmd.Flags().HasFlags() {
		t.Error("expected no flags with empty defs")
	}
}

func TestRequireStr(t *testing.T) {
	cmd := newTestCmd()
	if _, err := RequireStr(cmd, "name"); err != nil {
		t.Errorf("expected no error for flag with default, got: %v", err)
	}

	cmd.Flags().Set("name", "")
	if _, err := RequireStr(cmd, "name"); err == nil {
		t.Error("expected error for empty required flag")
	}
}

func TestRequireAll(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("a", "", "")
	cmd.Flags().String("b", "", "")
	cmd.Flags().String("c", "filled", "")

	err := RequireAll(cmd, "a", "b", "c")
	if err == nil {
		t.Fatal("expected error when a and b are empty")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--a") || !strings.Contains(msg, "--b") {
		t.Errorf("error should list both missing flags, got: %s", msg)
	}
	if strings.Contains(msg, "--c") {
		t.Errorf("error should not list --c (has value), got: %s", msg)
	}
}

func TestRequireAll_AllPresent(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("x", "val", "")

	if err := RequireAll(cmd, "x"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRegisterFlags_OpAnnotation(t *testing.T) {
	cmd := &cobra.Command{Use: "test", Args: cobra.NoArgs}
	RegisterFlags(cmd, nil, "PlaceOrder", nil)
	if cmd.Annotations["op"] != "PlaceOrder" {
		t.Errorf("op annotation = %q", cmd.Annotations["op"])
	}

	// --schema must bypass the positional-argument validator.
	cmd.Flags().Bool("schema", false, "")
	_ = cmd.Flags().Set("schema", "true")
	if err := cmd.Args(cmd, []string{"extra"}); err != nil {
		t.Errorf("Args with --schema = %v, want nil", err)
	}
}

func TestRequireAll_NonStringFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Int("quantity", 0, "")
	cmd.Flags().Bool("mode", false, "")
	cmd.Flags().Int("lots", 1, "")

	err := RequireAll(cmd, "quantity", "mode", "lots")
	if err == nil {
		t.Fatal("expected error for unset int/bool flags")
	}
	if msg := err.Error(); !strings.Contains(msg, "--quantity") || !strings.Contains(msg, "--mode") || strings.Contains(msg, "--lots") {
		t.Errorf("unexpected missing list: %s", msg)
	}

	_ = cmd.Flags().Set("quantity", "0")
	_ = cmd.Flags().Set("mode", "false")
	if err := RequireAll(cmd, "quantity", "mode"); err != nil {
		t.Errorf("explicitly set zero values should satisfy required: %v", err)
	}
}

func TestRequireAll_UnknownFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	if err := RequireAll(cmd, "missing"); err == nil {
		t.Error("expected error for an unregistered flag")
	}
}

func TestFlagUsage(t *testing.T) {
	tests := []struct {
		def    api.FlagDef
		server string
		want   string
	}{
		{api.FlagDef{Description: "order action", Type: "string", Required: true, Completions: []string{"BUY", "SELL"}}, "", "order action (required; one of: BUY, SELL)"},
		{api.FlagDef{Description: "product type", Type: "string", Completions: []string{"MIS", "CNC"}}, "MIS", "product type (one of: MIS, CNC; server default: MIS)"},
		{api.FlagDef{Description: "strategy", Type: "string", Required: true, CLIDefault: true}, "", "strategy"},
		{api.FlagDef{Description: "legs", Type: "array", Completions: []string{"NSE"}}, "", "legs"},
	}
	for _, tt := range tests {
		if got := FlagUsage(tt.def, tt.server); got != tt.want {
			t.Errorf("FlagUsage() = %q, want %q", got, tt.want)
		}
	}
}

func TestRegisterFlags_ServerDefaultInUsage(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd, []api.FlagDef{{Name: "product", OASName: "product", Type: "string", Default: "MIS", Description: "product type"}}, "", nil)
	f := cmd.Flags().Lookup("product")
	if f.DefValue != "" || f.Usage != "product type (server default: MIS)" {
		t.Errorf("product default = %q usage = %q", f.DefValue, f.Usage)
	}
}

func TestCheckEnum(t *testing.T) {
	def := api.FlagDef{Name: "action", Type: "string", Completions: []string{"BUY", "SELL"}}
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd, []api.FlagDef{def}, "", nil)
	if err := CheckEnum(cmd, def); err != nil {
		t.Errorf("unset flag: %v", err)
	}
	_ = cmd.Flags().Set("action", "buy")
	if err := CheckEnum(cmd, def); err != nil {
		t.Errorf("case-insensitive match rejected: %v", err)
	}
	_ = cmd.Flags().Set("action", "FOO")
	if err := CheckEnum(cmd, def); err == nil || !strings.Contains(err.Error(), "BUY, SELL") {
		t.Errorf("CheckEnum(FOO) = %v, want error listing values", err)
	}
}
