package stream

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/marketcalls/openalgo-cli/internal/client"
)

const testKey = "test-key-123"

// fakeServer is a minimal stand-in for OpenAlgo's WebSocket proxy. It
// records every client request and replies per the documented protocol;
// ticks are pushed after the subscribe ack.
type fakeServer struct {
	t          *testing.T
	ticks      int
	authReply  map[string]any
	subReply   map[string]any
	silentAuth bool

	mu       sync.Mutex
	requests []map[string]any
}

func (f *fakeServer) handler(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		f.t.Errorf("upgrade: %v", err)
		return
	}
	defer conn.Close()
	for {
		var req map[string]any
		if err := conn.ReadJSON(&req); err != nil {
			return
		}
		f.mu.Lock()
		f.requests = append(f.requests, req)
		f.mu.Unlock()

		switch req["action"] {
		case "authenticate":
			if f.silentAuth {
				continue
			}
			if f.authReply != nil {
				_ = conn.WriteJSON(f.authReply)
				continue
			}
			if req["api_key"] != testKey {
				_ = conn.WriteJSON(map[string]any{"status": "error", "code": "AUTHENTICATION_ERROR", "message": "Invalid API key"})
				continue
			}
			_ = conn.WriteJSON(map[string]any{"type": "auth", "status": "success", "broker": "zerodha"})
		case "subscribe":
			if f.subReply != nil {
				_ = conn.WriteJSON(f.subReply)
				continue
			}
			_ = conn.WriteJSON(map[string]any{"type": "subscribe", "status": "success", "request_id": req["request_id"]})
			for i := 0; i < f.ticks; i++ {
				_ = conn.WriteJSON(map[string]any{
					"type": "market_data", "symbol": "RELIANCE", "exchange": "NSE", "mode": 1,
					"data": map[string]any{"ltp": 1400.0 + float64(i)},
				})
			}
		case "subscribe_orders":
			_ = conn.WriteJSON(map[string]any{"type": "subscribe_orders", "status": "success"})
			for i := 0; i < f.ticks; i++ {
				_ = conn.WriteJSON(map[string]any{"type": "order_update", "orderid": "1", "order_status": "complete"})
			}
		case "unsubscribe":
			_ = conn.WriteJSON(map[string]any{"type": "unsubscribe", "status": "success"})
		case "unsubscribe_orders":
			_ = conn.WriteJSON(map[string]any{"type": "unsubscribe_orders", "status": "success"})
		}
	}
}

func (f *fakeServer) actions() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.requests {
		out = append(out, r["action"].(string))
	}
	return out
}

func startFake(t *testing.T, f *fakeServer) string {
	t.Helper()
	f.t = t
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// collect runs a session and returns the emitted frames' types.
func collect(t *testing.T, opts Options) ([]map[string]any, error) {
	t.Helper()
	var got []map[string]any
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Run(ctx, opts, func(m json.RawMessage) error {
		var v map[string]any
		if err := json.Unmarshal(m, &v); err != nil {
			t.Fatalf("emitted invalid JSON: %s", m)
		}
		got = append(got, v)
		return nil
	})
	return got, err
}

func TestRunLTPCountStopsAndUnsubscribes(t *testing.T) {
	f := &fakeServer{ticks: 5}
	u := startFake(t, f)
	got, err := collect(t, Options{
		URL: u, APIKey: testKey, Mode: ModeLTP, Count: 3,
		Symbols: []Instrument{{Symbol: "RELIANCE", Exchange: "NSE"}},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 data messages, got %d: %v", len(got), got)
	}
	for _, m := range got {
		if m["type"] != "market_data" {
			t.Errorf("non-raw run emitted control frame %v", m)
		}
	}
	want := []string{"authenticate", "subscribe", "unsubscribe"}
	if a := f.actions(); strings.Join(a, ",") != strings.Join(want, ",") {
		t.Errorf("actions = %v, want %v", a, want)
	}

	f.mu.Lock()
	sub := f.requests[1]
	f.mu.Unlock()
	if sub["mode"] != "LTP" {
		t.Errorf("subscribe mode = %v", sub["mode"])
	}
	if _, has := sub["depth"]; has {
		t.Errorf("LTP subscribe should not carry depth: %v", sub)
	}
	syms := sub["symbols"].([]any)
	if s := syms[0].(map[string]any); s["symbol"] != "RELIANCE" || s["exchange"] != "NSE" {
		t.Errorf("subscribe symbols = %v", syms)
	}
}

func TestRunRawEmitsAcks(t *testing.T) {
	f := &fakeServer{ticks: 1}
	u := startFake(t, f)
	got, err := collect(t, Options{
		URL: u, APIKey: testKey, Mode: ModeDepth, Depth: 5, Count: 1, Raw: true,
		Symbols: []Instrument{{Symbol: "RELIANCE", Exchange: "NSE"}},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var types []string
	for _, m := range got {
		types = append(types, m["type"].(string))
	}
	want := "auth,subscribe,market_data,unsubscribe"
	if strings.Join(types, ",") != want {
		t.Errorf("types = %v, want %s", types, want)
	}
	f.mu.Lock()
	sub := f.requests[1]
	f.mu.Unlock()
	if sub["mode"] != "Depth" || sub["depth"] != float64(5) {
		t.Errorf("depth subscribe = %v", sub)
	}
}

func TestRunOrders(t *testing.T) {
	f := &fakeServer{ticks: 2}
	u := startFake(t, f)
	got, err := collect(t, Options{URL: u, APIKey: testKey, Orders: true, Count: 2})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(got) != 2 || got[0]["type"] != "order_update" {
		t.Errorf("got %v", got)
	}
	want := "authenticate,subscribe_orders,unsubscribe_orders"
	if a := strings.Join(f.actions(), ","); a != want {
		t.Errorf("actions = %s, want %s", a, want)
	}
}

func TestRunAuthFailureIsExit2(t *testing.T) {
	f := &fakeServer{}
	u := startFake(t, f)
	_, err := collect(t, Options{
		URL: u, APIKey: "wrong", Mode: ModeLTP,
		Symbols: []Instrument{{Symbol: "RELIANCE", Exchange: "NSE"}},
	})
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != 401 || apiErr.ExitCode() != 2 {
		t.Errorf("status=%d exit=%d", apiErr.StatusCode, apiErr.ExitCode())
	}
	if !strings.Contains(apiErr.Message, "Invalid API key") {
		t.Errorf("message = %q", apiErr.Message)
	}
	if strings.Contains(err.Error(), "wrong") {
		t.Errorf("error leaks the API key: %v", err)
	}
}

func TestRunBrokerErrorIsExit1(t *testing.T) {
	f := &fakeServer{authReply: map[string]any{"status": "error", "code": "BROKER_ERROR", "message": "No broker configuration found for user"}}
	u := startFake(t, f)
	_, err := collect(t, Options{URL: u, APIKey: testKey, Orders: true})
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.ExitCode() != 1 || apiErr.StatusCode != 502 {
		t.Fatalf("got %v", err)
	}
}

func TestRunSubscribeAllFailed(t *testing.T) {
	f := &fakeServer{subReply: map[string]any{
		"type": "subscribe", "status": "partial",
		"subscriptions": []map[string]any{{"symbol": "NOPE", "exchange": "NSE", "status": "error", "message": "Symbol not found"}},
	}}
	u := startFake(t, f)
	_, err := collect(t, Options{
		URL: u, APIKey: testKey, Mode: ModeQuote,
		Symbols: []Instrument{{Symbol: "NOPE", Exchange: "NSE"}},
	})
	if err == nil || !strings.Contains(err.Error(), "NSE:NOPE: Symbol not found") {
		t.Fatalf("got %v", err)
	}
}

func TestRunPartialSubscribeWarns(t *testing.T) {
	f := &fakeServer{subReply: map[string]any{
		"type": "subscribe", "status": "partial",
		"subscriptions": []map[string]any{
			{"symbol": "RELIANCE", "exchange": "NSE", "status": "success"},
			{"symbol": "NOPE", "exchange": "NSE", "status": "error", "message": "Symbol not found"},
		},
	}}
	u := startFake(t, f)
	var warnings []string
	_, err := collect(t, Options{
		URL: u, APIKey: testKey, Mode: ModeQuote, Duration: 200 * time.Millisecond,
		Symbols: []Instrument{{Symbol: "RELIANCE", Exchange: "NSE"}, {Symbol: "NOPE", Exchange: "NSE"}},
		Warn:    func(s string) { warnings = append(warnings, s) },
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "NSE:NOPE") {
		t.Errorf("warnings = %v", warnings)
	}
}

func TestRunDurationWithNoTicks(t *testing.T) {
	f := &fakeServer{}
	u := startFake(t, f)
	start := time.Now()
	got, err := collect(t, Options{
		URL: u, APIKey: testKey, Mode: ModeLTP, Duration: 300 * time.Millisecond,
		Symbols: []Instrument{{Symbol: "RELIANCE", Exchange: "NSE"}},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no messages, got %v", got)
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Errorf("duration stop took %s", el)
	}
	if a := strings.Join(f.actions(), ","); a != "authenticate,subscribe,unsubscribe" {
		t.Errorf("actions = %s", a)
	}
}

func TestRunAckTimeout(t *testing.T) {
	f := &fakeServer{silentAuth: true}
	u := startFake(t, f)
	_, err := collect(t, Options{URL: u, APIKey: testKey, Orders: true, AckTimeout: 200 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "authentication acknowledgement") {
		t.Fatalf("got %v", err)
	}
}

func TestRunConnectionRefused(t *testing.T) {
	_, err := collect(t, Options{URL: "ws://127.0.0.1:1", APIKey: testKey, Orders: true})
	if err == nil || !strings.Contains(err.Error(), "OPENALGO_WS_URL") {
		t.Fatalf("got %v", err)
	}
}

func TestParseSymbols(t *testing.T) {
	got, err := ParseSymbols("NSE:RELIANCE, nse:SBIN,NSE_INDEX:NIFTY", "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []Instrument{{"RELIANCE", "NSE"}, {"SBIN", "NSE"}, {"NIFTY", "NSE_INDEX"}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	got, err = ParseSymbols("", "RELIANCE", "NSE")
	if err != nil || len(got) != 1 || got[0] != (Instrument{"RELIANCE", "NSE"}) {
		t.Errorf("--symbol/--exchange: %v %v", got, err)
	}
	got, err = ParseSymbols("RELIANCE,SBIN", "", "BSE")
	if err != nil || len(got) != 2 || got[1].Exchange != "BSE" {
		t.Errorf("bare symbols with --exchange: %v %v", got, err)
	}

	for _, bad := range [][3]string{{"", "", ""}, {"RELIANCE", "", ""}, {"", "RELIANCE", ""}, {"NSE:", "", ""}} {
		if _, err := ParseSymbols(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("ParseSymbols(%q) should fail", bad)
		}
	}
}

// TestLive runs against a real OpenAlgo WebSocket server. It is skipped
// unless OPENALGO_LIVE_WS=1 and OPENALGO_API_KEY are set; markets may be
// closed, so it only requires the auth and subscribe acks.
func TestLive(t *testing.T) {
	key := os.Getenv("OPENALGO_API_KEY")
	if os.Getenv("OPENALGO_LIVE_WS") == "" || key == "" {
		t.Skip("set OPENALGO_LIVE_WS=1 and OPENALGO_API_KEY to run")
	}
	u := os.Getenv("OPENALGO_WS_URL")
	if u == "" {
		u = "ws://127.0.0.1:8765"
	}
	for _, opts := range []Options{
		{Mode: ModeLTP, Symbols: []Instrument{{"RELIANCE", "NSE"}, {"SBIN", "NSE"}}},
		{Mode: ModeDepth, Depth: 5, Symbols: []Instrument{{"RELIANCE", "NSE"}}},
		{Orders: true},
	} {
		opts.URL, opts.APIKey, opts.Raw, opts.Duration, opts.Count = u, key, true, 4*time.Second, 3
		opts.Warn = func(s string) { t.Logf("warn: %s", s) }
		got, err := collect(t, opts)
		if err != nil {
			t.Fatalf("live %s: %v", opts.Mode, err)
		}
		types := map[string]bool{}
		for _, m := range got {
			b, _ := json.Marshal(m)
			t.Logf("%s", b)
			if s, ok := m["type"].(string); ok {
				types[s] = true
			}
		}
		if !types["auth"] {
			t.Errorf("no auth ack observed")
		}
		if !types["subscribe"] && !types["subscribe_orders"] {
			t.Errorf("no subscribe ack observed")
		}
	}
}
