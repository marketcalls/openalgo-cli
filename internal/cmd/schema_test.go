package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/spf13/cobra"
)

func TestWriteSchemaObject_Nested(t *testing.T) {
	old := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = old })

	fields := []api.ResponseField{
		{Name: "status", Type: "string", EnumValues: []string{"success", "error"}, Description: "request status"},
		{Name: "data", Type: "object", Description: "payload", Fields: []api.ResponseField{
			{Name: "orders", Type: "[]object", Fields: []api.ResponseField{
				{Name: "orderid", Type: "string"},
				{Name: "price", Type: "number|null", Description: "limit price\nsecond line"},
			}},
			{Name: "tags", Type: "[]string"},
			{Name: "extra", Type: "any"},
		}},
	}
	var buf bytes.Buffer
	writeSchemaObject(&buf, fields, 0)

	want := `{
  status: "success" | "error"; // request status
  data: {
    orders: {
      orderid: string;
      price: number | null; // limit price
    }[];
    tags: string[];
    extra: unknown;
  }; // payload
}`
	if got := buf.String(); got != want {
		t.Errorf("schema =\n%s\nwant\n%s", got, want)
	}
}

func TestTsTypePlain(t *testing.T) {
	cases := map[string]string{
		"integer":              "number",
		"[]integer":            "number[]",
		"map[string]integer":   "Record<string, number>",
		"[][]number":           "number[][]",
		"somethingUnknownType": "somethingUnknownType",
	}
	for in, want := range cases {
		if got := tsTypePlain(in); got != want {
			t.Errorf("tsTypePlain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitNullable(t *testing.T) {
	if base, ok := splitNullable("integer|null"); base != "integer" || !ok {
		t.Errorf("got %q %v", base, ok)
	}
	if base, ok := splitNullable("string"); base != "string" || ok {
		t.Errorf("got %q %v", base, ok)
	}
}

func TestPrintCommandSchema_GeneratedOp(t *testing.T) {
	old := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = old })

	if _, ok := api.ResponseSchema("Ping"); !ok {
		t.Skip("Ping schema not generated")
	}
	c := &cobra.Command{Use: "ping", Annotations: map[string]string{"op": "Ping"}}
	var buf bytes.Buffer
	c.SetOut(&buf)
	if err := printCommandSchema(c); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "broker: string;") || !strings.HasSuffix(out, "}\n") {
		t.Errorf("unexpected schema output:\n%s", out)
	}
}

func TestPrintCommandSchema_NoOp(t *testing.T) {
	c := &cobra.Command{Use: "profile"}
	if err := printCommandSchema(c); err == nil {
		t.Error("expected error for a command without an op")
	}
}
