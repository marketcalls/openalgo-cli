package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	schemaComment = color.New(color.Faint)
	schemaType    = color.New(color.FgCyan)
	schemaEnum    = color.New(color.FgGreen)
)

// printCommandSchema prints the response schema of the command's op as a
// TypeScript-like tree, without calling the API. Nested objects and arrays
// of objects are expanded inline.
func printCommandSchema(cmd *cobra.Command) error {
	opName := cmd.Annotations["op"]
	if cmd.Parent() == streamCmd {
		if fields, ok := streamSchemas[cmd.Name()]; ok {
			w := cmd.OutOrStdout()
			schemaComment.Fprintln(w, "// one NDJSON message per line")
			writeSchemaObject(w, fields, 0)
			fmt.Fprintln(w)
			return nil
		}
	}
	if opName == "" {
		return fmt.Errorf("no response schema available for %q", cmd.CommandPath())
	}
	fields, ok := api.ResponseSchema(opName)
	if !ok {
		return fmt.Errorf("no response schema available for %q", cmd.CommandPath())
	}

	w := cmd.OutOrStdout()
	if op, ok := api.OpByName(opName); ok {
		if op.Summary != "" {
			schemaComment.Fprintf(w, "// %s\n", op.Summary)
		}
		if op.RowsPath != "" {
			schemaComment.Fprintf(w, "// --csv renders the rows at .%s\n", op.RowsPath)
		}
	}
	writeSchemaObject(w, fields, 0)
	fmt.Fprintln(w)
	return nil
}

// writeSchemaObject writes "{ ...fields }" with the closing brace at the
// given depth; the caller writes whatever follows it ("[]", ";", newline).
func writeSchemaObject(w io.Writer, fields []api.ResponseField, depth int) {
	fmt.Fprintln(w, "{")
	indent := strings.Repeat("  ", depth+1)
	for _, f := range fields {
		fmt.Fprintf(w, "%s%s: ", indent, f.Name)
		base, nullable := splitNullable(f.Type)
		isArray := strings.HasPrefix(base, "[]")
		elem := strings.TrimPrefix(base, "[]")

		if len(f.Fields) > 0 && (elem == "object" || elem == "any") {
			writeSchemaObject(w, f.Fields, depth+1)
			if isArray {
				fmt.Fprint(w, "[]")
			}
		} else {
			fmt.Fprint(w, tsTypeColorized(f, base))
		}
		if nullable {
			fmt.Fprint(w, " | ", schemaType.Sprint("null"))
		}
		fmt.Fprint(w, ";")
		if desc := firstLine(f.Description); desc != "" {
			fmt.Fprint(w, " ", schemaComment.Sprintf("// %s", desc))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprint(w, strings.Repeat("  ", depth), "}")
}

// splitNullable strips a "|null" suffix from a ResponseField type.
func splitNullable(t string) (string, bool) {
	if base, ok := strings.CutSuffix(t, "|null"); ok {
		return base, true
	}
	return t, false
}

var oasToTS = map[string]string{
	"string":    "string",
	"boolean":   "boolean",
	"integer":   "number",
	"number":    "number",
	"enum":      "string",
	"object":    "object",
	"any":       "unknown",
	"[]string":  "string[]",
	"[]integer": "number[]",
	"[]number":  "number[]",
	"[]boolean": "boolean[]",
	"[]object":  "object[]",
	"[]any":     "unknown[]",
	"[]enum":    "string[]",
}

func tsTypeColorized(f api.ResponseField, base string) string {
	if len(f.EnumValues) > 0 {
		parts := make([]string, len(f.EnumValues))
		for i, v := range f.EnumValues {
			parts[i] = schemaEnum.Sprintf("%q", v)
		}
		union := strings.Join(parts, " | ")
		if strings.HasPrefix(base, "[]") {
			return "(" + union + ")[]"
		}
		return union
	}
	return schemaType.Sprint(tsTypePlain(base))
}

func tsTypePlain(t string) string {
	if strings.Contains(t, "|") {
		// A union such as "string|number" (the type differs between live
		// brokers and analyzer mode).
		parts := strings.Split(t, "|")
		for i, p := range parts {
			parts[i] = tsTypePlain(p)
		}
		return strings.Join(parts, " | ")
	}
	if ts, ok := oasToTS[t]; ok {
		return ts
	}
	if strings.HasPrefix(t, "map[string]") {
		return "Record<string, " + tsTypePlain(t[len("map[string]"):]) + ">"
	}
	if strings.HasPrefix(t, "[]") {
		return tsTypePlain(t[2:]) + "[]"
	}
	return t
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
