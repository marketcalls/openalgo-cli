package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/marketcalls/openalgo-cli/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const smokeKey = "smoke-test-key-0123456789"

// smokeRequest is what the mock server saw for one call.
type smokeRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

// setupMockClients starts an httptest server and pre-injects apiClient and
// openalgoClient pointing at it; PersistentPreRunE leaves an injected client
// alone. That's how the mock server URL gets into the CLI without any
// test-only env var. The returned func lists the requests received so far.
func setupMockClients(t *testing.T, status int, response string) func() []smokeRequest {
	t.Helper()
	var mu sync.Mutex
	var seen []smokeRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		req := smokeRequest{Method: r.Method, Path: r.URL.Path}
		_ = json.Unmarshal(raw, &req.Body)
		mu.Lock()
		seen = append(seen, req)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)

	isolateConfig(t)
	t.Setenv("OPENALGO_API_KEY", smokeKey)

	oldAPI, oldClient := apiClient, openalgoClient
	apiClient = &client.Client{
		HTTP:      &http.Client{Timeout: 5 * time.Second},
		BaseURL:   srv.URL + "/api/v1",
		APIKey:    smokeKey,
		UserAgent: "openalgo-cli/test",
		Timeout:   5 * time.Second,
	}
	openalgoClient = api.NewClient(apiClient)
	t.Cleanup(func() { apiClient, openalgoClient = oldAPI, oldClient })

	return func() []smokeRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]smokeRequest(nil), seen...)
	}
}

// resetFlags puts every flag in the shared command tree back to its default
// and clears Changed, so one Execute does not leak --csv or --dry-run into
// the next.
func resetFlags(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

// runRoot executes args through the shared rootCmd, as main does, and
// returns stdout and the error Execute would map to an exit code.
func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	setupTestOutput(t)
	root := Root()
	resetFlags(root)
	t.Cleanup(func() { resetFlags(root) })
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	t.Cleanup(func() { root.SetOut(nil); root.SetErr(nil); root.SetArgs(nil) })
	err := root.Execute()
	return out.String(), err
}

// exitCodeOf mirrors Execute's error-to-exit-code mapping.
func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ExitCode()
	}
	var authErr *authRequiredError
	if errors.As(err, &authErr) {
		return client.ExitAuthError
	}
	return exitAPIError
}

func TestCommandSmoke(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		status   int
		response string
		wantPath string
		wantOut  string
		wantBody map[string]any
		wantExit int
	}{
		{
			name:     "order place",
			args:     []string{"order", "place", "--symbol", "SBIN", "--exchange", "nse", "--action", "buy", "--quantity", "1", "--product", "CNC"},
			status:   200,
			response: `{"status":"success","orderid":"250408000989443","mode":"analyze"}`,
			wantPath: "/api/v1/placeorder",
			wantOut:  `"orderid": "250408000989443"`,
			// apikey injected, CLI strategy default applied, enum case
			// canonicalized, number sent as a JSON number.
			wantBody: map[string]any{"apikey": smokeKey, "strategy": "openalgo-cli", "exchange": "NSE", "action": "BUY", "quantity": float64(1), "product": "CNC"},
		},
		{
			name:     "order list csv",
			args:     []string{"order", "list", "--csv"},
			status:   200,
			response: orderbookResponse,
			wantPath: "/api/v1/orderbook",
			wantOut:  "SBIN",
			wantBody: map[string]any{"apikey": smokeKey},
		},
		{
			name:     "data quote",
			args:     []string{"data", "quote", "--symbol", "RELIANCE", "--exchange", "NSE"},
			status:   200,
			response: `{"status":"success","data":{"ltp":2800.5,"open":2790}}`,
			wantPath: "/api/v1/quotes",
			wantOut:  `"ltp": 2800.5`,
			wantBody: map[string]any{"apikey": smokeKey, "symbol": "RELIANCE", "exchange": "NSE"},
		},
		{
			name:     "status error in 200",
			args:     []string{"account", "funds"},
			status:   200,
			response: `{"status":"error","message":"broker session expired"}`,
			wantPath: "/api/v1/funds",
			wantExit: 1,
		},
		{
			name:     "invalid key",
			args:     []string{"account", "funds"},
			status:   403,
			response: `{"status":"error","message":"Invalid openalgo apikey"}`,
			wantPath: "/api/v1/funds",
			wantExit: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := setupMockClients(t, tt.status, tt.response)
			out, err := runRoot(t, tt.args...)
			if got := exitCodeOf(err); got != tt.wantExit {
				t.Fatalf("exit = %d, want %d (err %v)", got, tt.wantExit, err)
			}
			if tt.wantOut != "" && !strings.Contains(out, tt.wantOut) {
				t.Errorf("stdout missing %q:\n%s", tt.wantOut, out)
			}
			reqs := requests()
			if len(reqs) != 1 {
				t.Fatalf("server saw %d requests, want 1", len(reqs))
			}
			if reqs[0].Method != "POST" || reqs[0].Path != tt.wantPath {
				t.Errorf("request = %s %s, want POST %s", reqs[0].Method, reqs[0].Path, tt.wantPath)
			}
			for k, want := range tt.wantBody {
				if got := reqs[0].Body[k]; got != want {
					t.Errorf("body[%s] = %#v, want %#v (body %v)", k, got, want, reqs[0].Body)
				}
			}
		})
	}
}

// TestCommandSmoke_NoRequest covers commands that must fail or finish
// before any HTTP request is sent.
func TestCommandSmoke_NoRequest(t *testing.T) {
	t.Run("dry-run prints body without key", func(t *testing.T) {
		requests := setupMockClients(t, 200, `{}`)
		out, err := runRoot(t, "order", "place", "--symbol", "SBIN", "--exchange", "NSE", "--action", "BUY", "--quantity", "1", "--product", "CNC", "--dry-run")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, smokeKey) || strings.Contains(out, "apikey") {
			t.Errorf("dry-run leaked the API key:\n%s", out)
		}
		if !strings.Contains(out, `"strategy": "openalgo-cli"`) {
			t.Errorf("dry-run body missing strategy default:\n%s", out)
		}
		if n := len(requests()); n != 0 {
			t.Errorf("dry-run sent %d requests", n)
		}
	})

	for _, args := range [][]string{
		{"order", "modify", "--orderid", "1", "--symbol", "SBIN", "--exchange", "NSE", "--action", "BUY", "--product", "CNC", "--pricetype", "SL", "--price", "781", "--quantity", "1", "--dry-run"},
		{"order", "place", "--symbol", "SBIN", "--exchange", "NSE", "--action", "BUY", "--quantity", "1", "--product", "CNC", "--pricetype", "sl-m", "--trigger-price", "0"},
		{"order", "place", "--symbol", "SBIN", "--exchange", "NSE", "--action", "HOLD", "--quantity", "1"},
		{"order", "place", "--exchange", "NSE", "--action", "BUY", "--quantity", "1"},
	} {
		t.Run(strings.Join(args[:2], " ")+" rejected", func(t *testing.T) {
			requests := setupMockClients(t, 200, `{}`)
			if _, err := runRoot(t, args...); exitCodeOf(err) != 1 {
				t.Errorf("args %v: err = %v, want a validation error", args, err)
			}
			if n := len(requests()); n != 0 {
				t.Errorf("sent %d requests despite a validation error", n)
			}
		})
	}
}

func TestCompletionSkipsAuth(t *testing.T) {
	isolateConfig(t)
	out, err := runRoot(t, cobra.ShellCompRequestCmd, "order", "place", "--exchange", "")
	if err != nil {
		t.Fatalf("completion failed without credentials: %v", err)
	}
	if !strings.Contains(out, "NSE") || !strings.Contains(out, "NFO") {
		t.Errorf("completion output missing exchanges:\n%s", out)
	}
}
