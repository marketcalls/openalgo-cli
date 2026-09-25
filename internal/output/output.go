package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

func Render(w io.Writer, format Format, data any) error {
	switch format {
	case FormatCSV:
		return CSV(w, data)
	default:
		return JSON(w, data)
	}
}

func JSON(w io.Writer, data any) error {
	if data != nil {
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Slice && v.IsNil() {
			_, err := io.WriteString(w, "[]\n")
			return err
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func CSV(w io.Writer, data any) error {
	return CSVWithHeaders(w, data, nil)
}

// CSVWithHeaders renders data as CSV. An array of objects becomes one row
// per object, a single object one row, an array of scalars one column, and a
// lone scalar one cell. Nested objects are flattened into dotted columns
// (data.ltp, ce.ltp) and arrays are written as JSON text, so every cell can
// be parsed back. Numbers keep the exact text the server sent.
//
// headers (usually from the response schema) fix the leading column order,
// and any extra keys found in the rows are appended in sorted order. For an
// array of scalars or a lone scalar, headers[0] names the column.
func CSVWithHeaders(w io.Writer, data any, headers []string) error {
	v, err := Normalize(data)
	if err != nil {
		return fmt.Errorf("--csv: %w", err)
	}

	var rows []map[string]any
	switch val := v.(type) {
	case nil:
		rows = nil
	case map[string]any:
		rows = []map[string]any{flatten(val)}
	case []any:
		objects, scalars := 0, 0
		for _, item := range val {
			switch item.(type) {
			case map[string]any:
				objects++
			case []any:
			default:
				scalars++
			}
		}
		switch {
		case objects == len(val):
			for _, item := range val {
				rows = append(rows, flatten(item.(map[string]any)))
			}
		case scalars == len(val):
			return writeColumn(w, val, columnName(headers))
		default:
			return fmt.Errorf("--csv needs an array of objects or of plain values; use --jq to select rows")
		}
	default:
		return writeColumn(w, []any{val}, columnName(headers))
	}

	keys := csvHeaders(rows, headers)
	if len(keys) == 0 {
		return nil
	}
	cw := csv.NewWriter(w)
	_ = cw.Write(keys)
	for _, row := range rows {
		vals := make([]string, len(keys))
		for i, k := range keys {
			vals[i] = rawField(row, k)
		}
		_ = cw.Write(vals)
	}
	cw.Flush()
	return cw.Error()
}

// Normalize converts data into the untyped representation (map[string]any,
// []any, json.Number, string, bool, nil). Numbers are kept as json.Number so
// the text the server sent (4-decimal prices, large volumes) is preserved.
func Normalize(data any) (any, error) {
	var b []byte
	switch d := data.(type) {
	case json.RawMessage:
		b = d
	case []byte:
		b = d
	default:
		var err error
		if b, err = json.Marshal(d); err != nil {
			return nil, err
		}
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func columnName(headers []string) string {
	if len(headers) > 0 && headers[0] != "" {
		return headers[0]
	}
	return "value"
}

func writeColumn(w io.Writer, vals []any, header string) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{header})
	for _, v := range vals {
		_ = cw.Write([]string{cell(v)})
	}
	cw.Flush()
	return cw.Error()
}

// flatten turns nested objects into dotted keys: {"data":{"ltp":1}} becomes
// {"data.ltp":1}. Arrays are left in place and written as JSON. An empty
// object stays as a single key so the column is not lost.
func flatten(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if sub, ok := v.(map[string]any); ok && len(sub) > 0 {
				walk(key, sub)
				continue
			}
			out[key] = v
		}
	}
	walk("", m)
	return out
}

// csvHeaders orders columns: the known headers first (a header naming a
// nested object is replaced by its dotted sub-columns found in the rows),
// then every other key found in any row, sorted.
func csvHeaders(rows []map[string]any, known []string) []string {
	present := map[string]bool{}
	for _, row := range rows {
		for k := range row {
			present[k] = true
		}
	}
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for _, h := range known {
		if present[h] || len(rows) == 0 {
			add(h)
			continue
		}
		var sub []string
		for k := range present {
			if strings.HasPrefix(k, h+".") {
				sub = append(sub, k)
			}
		}
		if len(sub) == 0 {
			add(h)
			continue
		}
		sort.Strings(sub)
		for _, k := range sub {
			add(k)
		}
	}
	var extra []string
	for k := range present {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		add(k)
	}
	return keys
}

func rawField(row map[string]any, field string) string {
	v, ok := row[field]
	if !ok {
		return ""
	}
	return cell(v)
}

// cell formats one CSV value. Numbers are written in full (never rounded or
// in exponent form), null as an empty cell, and arrays or objects as JSON.
func cell(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case json.Number:
		return val.String()
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	case []any, map[string]any:
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(val); err != nil {
			return fmt.Sprint(val)
		}
		return strings.TrimSuffix(buf.String(), "\n")
	default:
		return fmt.Sprint(val)
	}
}
