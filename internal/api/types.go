package api

// Op describes a generated API operation. Passed to fetchCmd/attachCmd
// for automatic flag registration, help text, and required-flag validation.
type Op struct {
	Name     string
	Method   string // HTTP method: "GET" or "POST"
	Path     string // path relative to /api/v1, e.g. "/placeorder" or "/ticker/{symbol}"
	Summary  string
	Long     string
	Example  string
	Mutating bool // true when the operation has side effects (orders, toggles, notifications)
	// RowsPath is the dotted path (e.g. "data" or "data.orders") to the array
	// of records used for --csv output. Empty when the response has no table.
	RowsPath string
	Flags    []FlagDef
	Response []ResponseField
}

// FlagDef describes a CLI flag derived from the OpenAPI spec.
type FlagDef struct {
	Name        string // kebab-case CLI flag name
	OASName     string // original OAS property/parameter name
	Type        string // "string", "int", "number", "bool", "json" (object value), "array" (JSON array value), "object" (free-form body merged at root)
	Default     string
	Description string
	Completions []string // enum values for shell completion
	Required    bool     // true if OAS marks this parameter as required
	Source      string   // "path", "query", or "body"
	// CLIDefault marks a default supplied by the CLI rather than the server
	// (e.g. strategy=openalgo-cli). It is sent even when the flag is unset
	// and satisfies Required.
	CLIDefault bool
}

// ResponseField describes a field in an API response. Nested objects and
// arrays of objects carry their members in Fields.
type ResponseField struct {
	Name        string
	Type        string // "string", "number", "integer", "boolean", "object", "any", "[]string", "[]object", ...; nullable adds "|null"
	Description string
	EnumValues  []string
	Fields      []ResponseField
}
