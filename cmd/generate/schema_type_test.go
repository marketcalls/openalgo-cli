package main

import (
	"reflect"
	"strings"
	"testing"
)

// testSchemas installs a minimal components/schemas map for $ref resolution.
func testSchemas(t *testing.T) {
	t.Helper()
	prev := compSchemas
	compSchemas = map[string]any{
		"Exchange": map[string]any{"type": "string", "description": "exchange code", "enum": []any{"NSE", "BSE", "NFO"}},
		"Product":  map[string]any{"type": "string", "description": "product type", "enum": []any{"MIS", "CNC", "NRML"}},
		"Alias":    map[string]any{"$ref": "#/components/schemas/Product"},
	}
	t.Cleanup(func() { compSchemas = prev })
}

func TestSchemaType(t *testing.T) {
	tests := []struct {
		name         string
		schema       map[string]any
		wantType     string
		wantNullable bool
	}{
		{
			name:     "oas30 string",
			schema:   map[string]any{"type": "string"},
			wantType: "string",
		},
		{
			name:         "oas30 nullable string",
			schema:       map[string]any{"type": "string", "nullable": true},
			wantType:     "string",
			wantNullable: true,
		},
		{
			name:         "oas31 string or null",
			schema:       map[string]any{"type": []any{"string", "null"}},
			wantType:     "string",
			wantNullable: true,
		},
		{
			name:         "oas31 null first",
			schema:       map[string]any{"type": []any{"null", "integer"}},
			wantType:     "integer",
			wantNullable: true,
		},
		{
			name:   "missing type",
			schema: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotNullable := schemaType(tt.schema)
			if gotType != tt.wantType || gotNullable != tt.wantNullable {
				t.Fatalf("schemaType() = (%q, %v), want (%q, %v)",
					gotType, gotNullable, tt.wantType, tt.wantNullable)
			}
		})
	}
}

func TestSchemaFlag(t *testing.T) {
	testSchemas(t)

	tests := []struct {
		name     string
		prop     map[string]any
		wantType string
		wantDef  string
		wantDesc string
		wantEnum []string
	}{
		{
			name:     "string",
			prop:     map[string]any{"type": "string", "description": "trading symbol"},
			wantType: "string", wantDesc: "trading symbol",
		},
		{
			name:     "integer with default",
			prop:     map[string]any{"type": "integer", "default": float64(500)},
			wantType: "int", wantDef: "500", wantDesc: "limit",
		},
		{
			name:     "boolean default false",
			prop:     map[string]any{"type": "boolean", "default": false},
			wantType: "bool", wantDef: "false", wantDesc: "limit",
		},
		{
			name:     "number float default renders shortest",
			prop:     map[string]any{"type": "number", "default": 0.0},
			wantType: "number", wantDef: "0", wantDesc: "limit",
		},
		{
			name:     "number fractional default",
			prop:     map[string]any{"type": "number", "default": 6.5},
			wantType: "number", wantDef: "6.5", wantDesc: "limit",
		},
		{
			name:     "nullable integer maps by non-null type",
			prop:     map[string]any{"type": []any{"integer", "null"}, "default": nil},
			wantType: "int", wantDesc: "limit",
		},
		{
			name:     "nullable number",
			prop:     map[string]any{"type": []any{"number", "null"}},
			wantType: "number", wantDesc: "limit",
		},
		{
			name:     "nullable enum drops null",
			prop:     map[string]any{"type": []any{"string", "null"}, "enum": []any{"info", "warn", nil}},
			wantType: "string", wantDesc: "info, warn", wantEnum: []string{"info", "warn"},
		},
		{
			name:     "array is array with shape hint",
			prop:     map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
			wantType: "array", wantDesc: "limit; JSON array of objects; literal, @file, or - for stdin",
		},
		{
			name:     "object is json with JSON default",
			prop:     map[string]any{"type": "object", "default": map[string]any{}},
			wantType: "json", wantDef: "{}", wantDesc: "limit; JSON object; literal, @file, or - for stdin",
		},
		{
			name:     "array item enum becomes completions",
			prop:     map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Exchange"}},
			wantType: "array", wantDesc: "NSE, BSE, NFO; JSON array of strings; literal, @file, or - for stdin", wantEnum: []string{"NSE", "BSE", "NFO"},
		},
		{
			name:     "ref enum keeps spec order",
			prop:     map[string]any{"$ref": "#/components/schemas/Exchange"},
			wantType: "string", wantDesc: "exchange code", wantEnum: []string{"NSE", "BSE", "NFO"},
		},
		{
			name:     "ref with sibling description and default",
			prop:     map[string]any{"$ref": "#/components/schemas/Product", "description": "Product for the order.", "default": "MIS"},
			wantType: "string", wantDef: "MIS", wantDesc: "product for the order", wantEnum: []string{"MIS", "CNC", "NRML"},
		},
		{
			name:     "chained ref",
			prop:     map[string]any{"$ref": "#/components/schemas/Alias"},
			wantType: "string", wantDesc: "product type", wantEnum: []string{"MIS", "CNC", "NRML"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := schemaFlag("limit", "body", "", tt.prop, false)
			if f.flagType != tt.wantType {
				t.Errorf("flagType = %q, want %q", f.flagType, tt.wantType)
			}
			if f.defaultVal != tt.wantDef {
				t.Errorf("defaultVal = %q, want %q", f.defaultVal, tt.wantDef)
			}
			if f.description != tt.wantDesc {
				t.Errorf("description = %q, want %q", f.description, tt.wantDesc)
			}
			if !reflect.DeepEqual(f.enumValues, tt.wantEnum) {
				t.Errorf("enumValues = %v, want %v", f.enumValues, tt.wantEnum)
			}
			if f.flagName != "limit" || f.source != "body" {
				t.Errorf("unexpected flag identity: %#v", f)
			}
		})
	}
}

func TestSchemaFlagKebabAndParamDesc(t *testing.T) {
	testSchemas(t)
	f := schemaFlag("trigger_price", "query", "Trigger price.", map[string]any{"type": "number", "description": "ignored"}, true)
	if f.flagName != "trigger-price" || f.oasName != "trigger_price" {
		t.Errorf("names = %q/%q", f.flagName, f.oasName)
	}
	if f.description != "trigger price" {
		t.Errorf("parameter description should win, got %q", f.description)
	}
	if !f.required {
		t.Error("required not propagated")
	}
}

func TestResponseType(t *testing.T) {
	testSchemas(t)
	tests := []struct {
		schema map[string]any
		want   string
	}{
		{map[string]any{"type": "string"}, "string"},
		{map[string]any{"type": []any{"number", "null"}}, "number|null"},
		{map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "[]string"},
		{map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{}}}, "[]object"},
		{map[string]any{"type": "array", "items": map[string]any{"type": "array"}}, "[][]any"},
		{map[string]any{"type": []any{"object", "null"}}, "object|null"},
		{map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Exchange"}}, "[]string"},
		{map[string]any{}, "any"},
		{map[string]any{"type": []any{"string", "number"}}, "string|number"},
		{map[string]any{"type": []any{"string", "number", "null"}}, "string|number|null"},
	}
	for _, tt := range tests {
		if got := responseType(tt.schema); got != tt.want {
			t.Errorf("responseType(%v) = %q, want %q", tt.schema, got, tt.want)
		}
	}
}

func TestResponseFieldsDepthCap(t *testing.T) {
	testSchemas(t)
	// Build a 6-level nested object: a.a.a.a.a.leaf
	leaf := map[string]any{"type": "object", "properties": map[string]any{"leaf": map[string]any{"type": "string"}}}
	s := leaf
	for i := 0; i < 5; i++ {
		s = map[string]any{"type": "object", "properties": map[string]any{"a": s}}
	}
	fields := responseFields(s, 1)
	depth := 0
	for len(fields) > 0 {
		depth++
		fields = fields[0].fields
	}
	if depth != maxResponseDepth {
		t.Errorf("response tree depth = %d, want %d", depth, maxResponseDepth)
	}
}

func TestDecodeOrderedPreservesKeyOrder(t *testing.T) {
	path := t.TempDir() + "/spec.json"
	writeFile(t, path, `{"properties":{"status":{},"message":{},"data":{},"alpha":{}},"n":1.50}`)
	spec := loadSpec(path)
	got := orderedKeys(mapGet(spec, "properties"))
	want := []string{"status", "message", "data", "alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("orderedKeys = %v, want %v", got, want)
	}
	if n, ok := spec["n"].(float64); !ok || n != 1.5 {
		t.Errorf("number decoded as %T %v, want float64 1.5", spec["n"], spec["n"])
	}
	// Maps not produced by the decoder fall back to sorted order.
	if got := orderedKeys(map[string]any{"b": 1, "a": 2}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("fallback orderedKeys = %v", got)
	}
}

func TestJSONValueHint(t *testing.T) {
	testSchemas(t)
	s := map[string]any{"type": "array", "items": map[string]any{
		"type":       "object",
		"required":   []any{"symbol", "exchange"},
		"properties": map[string]any{"symbol": map[string]any{"type": "string"}, "exchange": map[string]any{"type": "string"}, "note": map[string]any{"type": "string"}},
	}}
	got := jsonValueHint(s)
	if !strings.Contains(got, "JSON array of {") || !strings.Contains(got, "symbol") || !strings.Contains(got, "exchange") || strings.Contains(got, "note") {
		t.Errorf("jsonValueHint() = %q, want required item fields only", got)
	}
}
