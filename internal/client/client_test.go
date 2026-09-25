package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marketcalls/openalgo-cli/internal/config"
)

const testKey = "test-api-key-0123456789abcdef"

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Client{
		HTTP:      &http.Client{Timeout: 5 * time.Second},
		BaseURL:   srv.URL + "/api/v1",
		APIKey:    testKey,
		UserAgent: "openalgo-cli/test",
	}
}

// captureStderr runs fn with os.Stderr redirected to a pipe and returns what
// was written. The client writes debug/verbose output straight to os.Stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stderr = old
	return <-done
}

func TestPostInjectsAPIKeyInBody(t *testing.T) {
	var gotBody map[string]any
	var gotPath, gotContentType, gotUA string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotUA = r.Header.Get("User-Agent")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		if r.URL.Query().Get("apikey") != "" {
			t.Error("apikey must not be sent in the query for POST")
		}
		_, _ = w.Write([]byte(`{"status":"success","orderid":"123"}`))
	})

	_, err := c.Post("/placeorder", map[string]any{"symbol": "RELIANCE", "quantity": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/v1/placeorder" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["apikey"] != testKey {
		t.Errorf("apikey not injected into body: %v", gotBody)
	}
	if gotBody["symbol"] != "RELIANCE" {
		t.Errorf("caller fields lost: %v", gotBody)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if gotUA != "openalgo-cli/test" {
		t.Errorf("User-Agent = %q", gotUA)
	}
}

func TestPostWithoutBodySendsAPIKeyObject(t *testing.T) {
	var gotBody map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	})
	if _, err := c.Post("/funds", nil); err != nil {
		t.Fatal(err)
	}
	if len(gotBody) != 1 || gotBody["apikey"] != testKey {
		t.Errorf("body = %v, want only apikey", gotBody)
	}
}

func TestPostDoesNotMutateCallerBody(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	body := map[string]any{"symbol": "SBIN"}
	_, _ = c.Post("/placeorder", body)
	if _, ok := body["apikey"]; ok {
		t.Error("caller's body map was modified; dry-run output could leak the key")
	}
}

func TestPostKeepsExplicitAPIKey(t *testing.T) {
	var gotBody map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	_, _ = c.Post("/ping", map[string]any{"apikey": "other"})
	if gotBody["apikey"] != "other" {
		t.Errorf("explicit apikey overwritten: %v", gotBody["apikey"])
	}
}

func TestGetInjectsAPIKeyInQuery(t *testing.T) {
	var gotKey, gotExchange, gotMethod string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotKey = r.URL.Query().Get("apikey")
		gotExchange = r.URL.Query().Get("exchange")
		if r.ContentLength > 0 {
			t.Error("GET must not carry a body")
		}
		_, _ = w.Write([]byte(`{"status":"success","data":[]}`))
	})

	_, err := c.Get("/instruments", map[string][]string{"exchange": {"NSE"}})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s", gotMethod)
	}
	if gotKey != testKey {
		t.Errorf("apikey query = %q", gotKey)
	}
	if gotExchange != "NSE" {
		t.Errorf("exchange query = %q", gotExchange)
	}
}

func TestRawRequestAppendsQuery(t *testing.T) {
	var gotQuery string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	_, err := c.RawRequest("GET", c.BaseURL+"/instruments?exchange=NSE", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "exchange=NSE") || !strings.Contains(gotQuery, "apikey=") {
		t.Errorf("query = %q, want both exchange and apikey", gotQuery)
	}
}

func TestSuccessReturnsBody(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"success","data":{"broker":"zerodha","message":"pong"}}`))
	})
	data, err := c.Post("/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Data struct {
			Broker string `json:"broker"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Data.Broker != "zerodha" {
		t.Errorf("broker = %q", m.Data.Broker)
	}
}

func TestNonJSONPassthrough(t *testing.T) {
	csvBody := "symbol,exchange\nRELIANCE,NSE\n"
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(csvBody))
	})
	data, err := c.Get("/instruments", map[string][]string{"format": {"csv"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != csvBody {
		t.Errorf("body = %q, want verbatim CSV", string(data))
	}
}

func TestErrorEnvelope400(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"status":"error","message":"Invalid symbol"}`))
	})

	_, err := c.Post("/placeorder", map[string]any{"symbol": "NOPE"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Message != "Invalid symbol" {
		t.Errorf("got %d %q", apiErr.StatusCode, apiErr.Message)
	}
	if apiErr.ExitCode() != ExitAPIError {
		t.Errorf("exit code = %d, want %d", apiErr.ExitCode(), ExitAPIError)
	}
	if apiErr.Method != "POST" || apiErr.Path != "/api/v1/placeorder" {
		t.Errorf("method/path = %s %s", apiErr.Method, apiErr.Path)
	}
	if apiErr.Hint() == "" {
		t.Error("expected a hint for 400")
	}
}

func TestAuthErrorsExitCode2(t *testing.T) {
	for _, status := range []int{401, 403} {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"status":"error","message":"Invalid openalgo apikey"}`))
		})
		_, err := c.Post("/funds", nil)
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("%d: expected *APIError, got %T", status, err)
		}
		if apiErr.ExitCode() != ExitAuthError {
			t.Errorf("%d: exit code = %d, want %d", status, apiErr.ExitCode(), ExitAuthError)
		}
		if !strings.Contains(apiErr.Hint(), "profile login") {
			t.Errorf("%d: hint = %q", status, apiErr.Hint())
		}
	}
}

// TestStatusErrorIn200 covers OpenAlgo endpoints that report failures with
// HTTP 200 and {"status":"error"}: the CLI must still exit non-zero.
func TestStatusErrorIn200(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"error","message":"Order not found"}`))
	})
	_, err := c.Post("/orderstatus", map[string]any{"orderid": "1"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.Message != "Order not found" {
		t.Errorf("message = %q", apiErr.Message)
	}
	if apiErr.ExitCode() != ExitAPIError {
		t.Errorf("exit code = %d", apiErr.ExitCode())
	}
}

func TestHTMLErrorGetsHostHint(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Not Found</body></html>"))
	})
	_, err := c.Post("/ping", nil)
	apiErr := err.(*APIError)
	if apiErr.Message != "Not Found" {
		t.Errorf("message = %q, want status text instead of HTML", apiErr.Message)
	}
	if !strings.Contains(apiErr.Hint(), "doctor") {
		t.Errorf("hint = %q", apiErr.Hint())
	}
}

func TestRetryReadOn5xx(t *testing.T) {
	var attempts atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 2 {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"status":"error","message":"busy"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	if _, err := c.Post("/funds", nil); err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

// TestNoRetryWriteOn5xx guards against duplicate orders: OpenAlgo has no
// idempotency key, so a 5xx on a mutating call must never be replayed.
func TestNoRetryWriteOn5xx(t *testing.T) {
	var attempts atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"status":"error","message":"broker timeout"}`))
	})
	_, err := c.DoWrite("POST", "/placeorder", nil, map[string]any{"symbol": "SBIN"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retry for writes)", got)
	}
}

func TestRawRequestNoRetryOn5xx(t *testing.T) {
	var attempts atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(502)
	})
	_, _ = c.RawRequest("POST", c.BaseURL+"/placeorder", nil)
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1", got)
	}
}

// TestWriteTimeoutOutcomeUnknown guards against duplicate orders: a write
// that times out after it was sent must not claim the server was unreachable.
func TestWriteTimeoutOutcomeUnknown(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	})
	c.HTTP.Timeout = 50 * time.Millisecond
	_, err := c.DoWrite("POST", "/placeorder", nil, map[string]any{"symbol": "SBIN"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T %v", err, err)
	}
	if strings.Contains(apiErr.Message, "could not reach") || !strings.Contains(apiErr.Message, "outcome unknown") {
		t.Errorf("message = %q, want outcome unknown", apiErr.Message)
	}
	if !strings.Contains(apiErr.Hint(), "order list") {
		t.Errorf("hint = %q, want a check-before-retry hint", apiErr.Hint())
	}
}

func TestWriteGatewayErrorOutcomeUnknown(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(504)
	})
	_, err := c.DoWrite("POST", "/placeorder", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok || !strings.Contains(apiErr.Message, "outcome unknown") || !strings.Contains(apiErr.Hint(), "order list") {
		t.Fatalf("got %T %v", err, err)
	}
}

// A refused connection means nothing was sent, so the plain wording stays.
func TestWriteDialErrorCouldNotReach(t *testing.T) {
	c := &Client{HTTP: &http.Client{Timeout: time.Second}, BaseURL: "http://127.0.0.1:1/api/v1", APIKey: testKey}
	_, err := c.DoWrite("POST", "/placeorder", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok || !strings.Contains(apiErr.Message, "could not reach") || !strings.Contains(apiErr.Hint(), "doctor") {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestWriteRetriesOn429(t *testing.T) {
	var attempts atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			return
		}
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	c.Quiet = true
	if _, err := c.DoWrite("POST", "/placeorder", nil, nil); err != nil {
		t.Fatalf("expected success after 429 retry, got %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

func TestNoRetryOn400(t *testing.T) {
	var attempts atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"status":"error","message":"bad"}`))
	})
	_, _ = c.Post("/quotes", nil)
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1", got)
	}
}

func TestDebugScrubsAPIKey(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Echo the key back to make sure response bodies are scrubbed too.
		_, _ = w.Write([]byte(`{"status":"success","echo":"` + testKey + `"}`))
	})
	c.Debug = true

	out := captureStderr(t, func() {
		_, _ = c.Post("/ping", nil)
		_, _ = c.Get("/instruments", nil)
	})
	if strings.Contains(out, testKey) {
		t.Errorf("API key leaked in debug output:\n%s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in debug output:\n%s", out)
	}
}

func TestErrorPathScrubsAPIKey(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"status":"error","message":"bad"}`))
	})
	_, err := c.Get("/instruments", nil)
	apiErr := err.(*APIError)
	if strings.Contains(apiErr.Path, testKey) || strings.Contains(apiErr.Error(), testKey) {
		t.Errorf("API key leaked in error: %+v", apiErr)
	}
}

func TestConnectionErrorScrubsAPIKey(t *testing.T) {
	c := &Client{
		HTTP:    &http.Client{Timeout: time.Second},
		BaseURL: "http://127.0.0.1:1/api/v1",
		APIKey:  testKey,
	}
	_, err := c.Get("/instruments", nil)
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), testKey) {
		t.Errorf("API key leaked in connection error: %v", err)
	}
	apiErr, ok := err.(*APIError)
	if !ok || !strings.Contains(apiErr.Hint(), "doctor") {
		t.Errorf("expected APIError with doctor hint, got %T %v", err, err)
	}
}

func TestScrub(t *testing.T) {
	c := &Client{APIKey: "abc+def/ghi"}
	in := "key=abc+def/ghi and escaped=abc%2Bdef%2Fghi"
	out := c.scrub(in)
	if strings.Contains(out, "abc+def/ghi") || strings.Contains(out, "abc%2Bdef%2Fghi") {
		t.Errorf("scrub left key material: %s", out)
	}
	if (&Client{}).scrub("plain text") != "plain text" {
		t.Error("empty key should not alter text")
	}
}

func TestNewClient(t *testing.T) {
	c := New(&config.Resolved{BaseURL: "http://127.0.0.1:5000/api/v1/", APIKey: "k"})
	if c.BaseURL != "http://127.0.0.1:5000/api/v1" {
		t.Errorf("trailing slash not trimmed: %s", c.BaseURL)
	}
	if c.Timeout != 30*time.Second {
		t.Errorf("Timeout = %s", c.Timeout)
	}
	if re := regexp.MustCompile(`^\S+/\S* \S+/\S+`); !re.MatchString(c.UserAgent) {
		t.Errorf("UserAgent = %q", c.UserAgent)
	}
	c.SetTimeout(5 * time.Second)
	if c.HTTP.Timeout != 5*time.Second {
		t.Errorf("SetTimeout did not apply to HTTP client")
	}
}

func TestErrorString(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{"empty message", APIError{StatusCode: 500}, "API error (HTTP 500)"},
		{"message and status", APIError{StatusCode: 400, Message: "invalid"}, "invalid (HTTP 400)"},
		{"no status", APIError{Message: "something"}, "something"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
