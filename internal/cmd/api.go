package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/marketcalls/openalgo-cli/internal/cmdutil"
	"github.com/marketcalls/openalgo-cli/internal/config"
	"github.com/marketcalls/openalgo-cli/internal/output"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api [METHOD] <path>",
	Short: "Raw API access",
	Long: `Make raw API calls to any OpenAlgo endpoint. Paths are relative to /api/v1;
a leading /api/v1 is accepted and stripped.

METHOD defaults to POST, since almost every OpenAlgo endpoint is POST, or to
GET when only --query is given. The API key is added
automatically: to the JSON body for POST, to the apikey query parameter
for GET. Raw calls are never retried on server errors.

The body comes from --body (a JSON object, @file, or - for stdin) or from
piped or redirected stdin. Piped stdin that stays silent for 2 seconds is
treated as no body, so an inherited open pipe never blocks the command.
With --csv and no --jq, the response's data field is rendered as rows.`,
	Example: `  openalgo api POST /funds
  openalgo api POST /quotes --body '{"symbol":"RELIANCE","exchange":"NSE"}'
  openalgo api GET /instruments --query "exchange=NSE"
  openalgo api /api/v1/ping
  echo '{"symbol":"SBIN","exchange":"NSE"}' | openalgo api POST /depth
  openalgo api POST /quotes --body @quote.json --csv`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		bodyFlag := cmdutil.Str(cmd, "body")
		queryFlag := strings.TrimPrefix(cmdutil.Str(cmd, "query"), "?")

		if len(args) == 2 && !validMethods[strings.ToUpper(args[0])] {
			return fmt.Errorf("unknown HTTP method %q (use GET, POST, PUT, PATCH or DELETE)", args[0])
		}
		method, path := parseMethodPath(args, queryFlag != "" && !cmdutil.Changed(cmd, "body"))
		if strings.Contains(path, "://") {
			return fmt.Errorf("api takes a path relative to /api/v1 (e.g. /funds), not a full URL\nHint: the host comes from the active profile or OPENALGO_HOST")
		}

		var body any
		switch {
		case cmdutil.Changed(cmd, "body"):
			raw, err := readBodyFlag(bodyFlag)
			if err != nil {
				return err
			}
			m, err := decodeJSONObject(raw)
			if err != nil {
				return fmt.Errorf("invalid JSON in --body: %w", err)
			}
			body = m
		case methodSupportsBody(method) && stdinHasData():
			raw, err := readImplicitStdin(os.Stdin, stdinWait)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			if len(bytes.TrimSpace(raw)) > 0 {
				m, err := decodeJSONObject(raw)
				if err != nil {
					return fmt.Errorf("invalid JSON on stdin: %w", err)
				}
				body = m
			}
		}

		if body != nil && !methodSupportsBody(method) {
			return fmt.Errorf("%s requests cannot carry a body; use --query or POST", method)
		}
		fullURL := cfg.BaseURL + apiPath(path)
		if queryFlag != "" {
			sep := "?"
			if strings.Contains(fullURL, "?") {
				sep = "&"
			}
			fullURL += sep + queryFlag
		}

		data, err := apiClient.RawRequest(method, fullURL, body)
		if err != nil {
			return err
		}
		out := any(voidOrData(data))
		// --csv has no op schema here: render the envelope's data field
		// (an object as one row, an array as rows) rather than the envelope.
		if getOutput() == output.FormatCSV && jqFlag == "" && json.Valid(data) {
			if rows := rowsAtPath(out, "data"); rows != nil {
				out = rows
			}
		}
		return renderData(cmd, out)
	},
}

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
}

// parseMethodPath splits the positional arguments. Without an explicit
// method it is POST, or GET when queryOnly (only --query was given).
func parseMethodPath(args []string, queryOnly bool) (method, path string) {
	if len(args) == 2 {
		if upper := strings.ToUpper(args[0]); validMethods[upper] {
			return upper, args[1]
		}
	}
	if queryOnly {
		return "GET", args[len(args)-1]
	}
	return http.MethodPost, args[len(args)-1]
}

// apiPath normalizes a user-supplied path so it is relative to /api/v1:
// "funds", "/funds" and "/api/v1/funds" all become "/funds".
func apiPath(p string) string {
	p = strings.TrimSpace(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if rest, ok := strings.CutPrefix(p, config.APIPrefix); ok && (rest == "" || rest[0] == '/' || rest[0] == '?') {
		p = rest
		if p == "" || p[0] == '?' {
			p = "/" + p
		}
	}
	return p
}

func decodeJSONObject(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(cmdutil.NormalizeText(raw)))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("body must be a JSON object")
	}
	if dec.More() {
		return nil, fmt.Errorf("unexpected data after the JSON object")
	}
	return m, nil
}

// voidOrData keeps stdout valid JSON when the server returns an empty body.
func voidOrData(data json.RawMessage) any {
	if len(data) == 0 {
		return json.RawMessage("{}")
	}
	return data
}

// stdinHasData reports whether stdin is a pipe or a redirected file, the
// only cases where a body may be waiting. A terminal or /dev/null is not.
func stdinHasData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeNamedPipe != 0 || stat.Mode().IsRegular()
}

// stdinWait bounds how long piped stdin may stay silent before the command
// proceeds without a body. Tests shorten it.
var stdinWait = 2 * time.Second

// readImplicitStdin reads a body from stdin that was not asked for
// explicitly. Agent harnesses and CI often hand the process an open pipe
// that never delivers data or EOF; waiting for the first bytes with a
// deadline keeps such a call from blocking forever. Once data arrives the
// rest is read to EOF.
func readImplicitStdin(f *os.File, wait time.Duration) ([]byte, error) {
	if stat, err := f.Stat(); err == nil && stat.Mode().IsRegular() {
		return io.ReadAll(f)
	}
	type chunk struct {
		b   []byte
		err error
	}
	first := make(chan chunk, 1)
	go func() {
		buf := make([]byte, 32*1024)
		n, err := f.Read(buf)
		first <- chunk{buf[:n], err}
	}()
	select {
	case c := <-first:
		if c.err != nil {
			if c.err == io.EOF {
				return c.b, nil
			}
			return nil, c.err
		}
		rest, err := io.ReadAll(f)
		return append(c.b, rest...), err
	case <-time.After(wait):
		return nil, nil
	}
}

// readBodyFlag returns the --body value: "-" reads stdin to EOF, "@path"
// reads a file, anything else is the JSON text itself.
func readBodyFlag(v string) ([]byte, error) {
	switch {
	case v == "-":
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("--body: reading stdin: %w", err)
		}
		return b, nil
	case strings.HasPrefix(v, "@"):
		b, err := os.ReadFile(v[1:])
		if err != nil {
			return nil, fmt.Errorf("--body: %w", err)
		}
		return b, nil
	}
	return []byte(v), nil
}

func methodSupportsBody(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}

func init() {
	apiCmd.Flags().String("body", "", "JSON request body: an object, @file, or - for stdin (apikey is added automatically)")
	apiCmd.Flags().String("query", "", "Query string to append (e.g. \"exchange=NSE&format=json\")")
}
