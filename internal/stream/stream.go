// Package stream is a small client for OpenAlgo's WebSocket proxy
// (default ws://127.0.0.1:8765). It authenticates with the API key,
// subscribes to market data (LTP / Quote / Depth) or account-level order
// updates, hands every inbound frame to a caller-supplied emit func, and
// unsubscribes before closing when the context ends or the message budget
// is spent.
package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/marketcalls/openalgo-cli/internal/client"
)

// Market data modes, spelled the way the server's subscribe message
// accepts them (it also takes 1/2/3, but the labels read better in logs).
const (
	ModeLTP   = "LTP"
	ModeQuote = "Quote"
	ModeDepth = "Depth"
)

// Message types the server pushes that count as data (everything else is
// a control or acknowledgement frame, only emitted with Options.Raw).
const (
	TypeMarketData  = "market_data"
	TypeOrderUpdate = "order_update"
)

// DefaultAckTimeout bounds how long Run waits for the authenticate and
// subscribe acknowledgements. Authentication can be slow because the server
// connects the broker adapter before it answers.
const DefaultAckTimeout = 30 * time.Second

// Envelope status values sent by the OpenAlgo WebSocket server.
const (
	statusSuccess = "success"
	statusError   = "error"
)

// closeAckTimeout bounds the wait for the unsubscribe acknowledgement on
// shutdown, so Ctrl-C never hangs on a slow or dead server.
const closeAckTimeout = 3 * time.Second

// Instrument is one symbol/exchange pair in a subscribe request.
type Instrument struct {
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
}

// Options configures a streaming session. Either Orders is true (order
// update stream) or Mode and Symbols are set (market data stream).
type Options struct {
	URL       string
	APIKey    string
	UserAgent string

	Mode    string
	Depth   int
	Symbols []Instrument
	Orders  bool

	// Count stops the stream after this many data messages (0 = no limit).
	Count int
	// Duration stops the stream after this long (0 = until ctx ends).
	Duration time.Duration
	// Raw also emits control frames (auth/subscribe/unsubscribe acks, pongs).
	Raw bool
	// AckTimeout overrides DefaultAckTimeout.
	AckTimeout time.Duration
	// Warn receives non-fatal problems, e.g. a symbol the broker refused
	// while others subscribed fine. Nil discards them.
	Warn func(msg string)
	// Verbose receives connection details (URL, connect time) for
	// --verbose. Nil discards them.
	Verbose func(msg string)
	// Frame receives every frame sent (dir ">") and received (dir "<") for
	// --debug. The sent auth frame carries the API key, so the callback
	// must redact it. Nil discards them.
	Frame func(dir string, data []byte)
}

// envelope is the subset of fields Run inspects to route a frame.
type envelope struct {
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	Code          string          `json:"code"`
	Message       string          `json:"message"`
	Subscriptions []subscribeItem `json:"subscriptions"`
}

type subscribeItem struct {
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// ParseSymbols builds the instrument list from --symbols (comma-separated
// EXCHANGE:SYMBOL pairs) and/or --symbol with --exchange. A bare SYMBOL in
// --symbols takes --exchange as its exchange.
func ParseSymbols(list, symbol, exchange string) ([]Instrument, error) {
	exchange = strings.ToUpper(strings.TrimSpace(exchange))
	var out []Instrument
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		ex, sym, ok := strings.Cut(part, ":")
		if !ok {
			ex, sym = exchange, part
		}
		ex, sym = strings.ToUpper(strings.TrimSpace(ex)), strings.TrimSpace(sym)
		if ex == "" || sym == "" {
			return nil, fmt.Errorf("invalid symbol %q: use EXCHANGE:SYMBOL, e.g. NSE:RELIANCE (or pass --exchange)", part)
		}
		out = append(out, Instrument{Symbol: sym, Exchange: ex})
	}
	if s := strings.TrimSpace(symbol); s != "" {
		if exchange == "" {
			return nil, errors.New("--symbol requires --exchange (e.g. --symbol RELIANCE --exchange NSE)")
		}
		out = append(out, Instrument{Symbol: s, Exchange: exchange})
	}
	if len(out) == 0 {
		return nil, errors.New("--symbols or --symbol with --exchange is required (e.g. --symbols NSE:RELIANCE,NSE:SBIN)")
	}
	return out, nil
}

// Run connects, authenticates, subscribes and streams until ctx is done,
// Duration elapses, Count data messages have been emitted, emit returns an
// error, or the server drops the connection. A clean stop (ctx, Duration,
// Count) unsubscribes and returns nil.
func Run(ctx context.Context, opts Options, emit func(json.RawMessage) error) error {
	if !opts.Orders && len(opts.Symbols) == 0 {
		return errors.New("no symbols to subscribe")
	}
	ackTimeout := opts.AckTimeout
	if ackTimeout <= 0 {
		ackTimeout = DefaultAckTimeout
	}
	warn := opts.Warn
	if warn == nil {
		warn = func(string) {}
	}
	if opts.Duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Duration)
		defer cancel()
	}

	start := time.Now()
	conn, err := dial(ctx, opts)
	if err != nil {
		return err
	}
	defer conn.Close()
	if opts.Verbose != nil {
		opts.Verbose(fmt.Sprintf("connected to %s (%dms)", opts.URL, time.Since(start).Milliseconds()))
	}

	// A single reader goroutine feeds frames to the loop below, so the loop
	// can select on ctx while blocked on the network. gorilla's default ping
	// handler answers the server's control pings with pongs from this reader.
	frames := make(chan json.RawMessage, 64)
	readErr := make(chan error, 1)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			if opts.Frame != nil {
				opts.Frame("<", data)
			}
			select {
			case frames <- json.RawMessage(data):
			case <-done:
				return
			}
		}
	}()

	if err := send(conn, opts, map[string]any{"action": "authenticate", "api_key": opts.APIKey}); err != nil {
		return connError(opts.URL, err)
	}

	const (
		stateAuth = iota
		stateSubscribe
		stateStreaming
	)
	state := stateAuth
	deadline := time.NewTimer(ackTimeout)
	defer deadline.Stop()
	received := 0

	for {
		select {
		case <-ctx.Done():
			if state == stateStreaming {
				unsubscribe(conn, opts, frames, readErr, emit)
			}
			closeConn(conn)
			return nil

		case <-deadline.C:
			what := "authentication"
			if state == stateSubscribe {
				what = "subscription"
			}
			closeConn(conn)
			return &client.APIError{
				StatusCode: http.StatusGatewayTimeout,
				Status:     statusError,
				Message:    fmt.Sprintf("timed out after %s waiting for %s acknowledgement", ackTimeout, what),
				Method:     "WS",
				Path:       opts.URL,
			}

		case err := <-readErr:
			if state == stateAuth {
				var ce *websocket.CloseError
				if errors.As(err, &ce) && ce.Code == 4401 {
					return authError(opts.URL, "server closed the connection before authentication completed")
				}
			}
			return connError(opts.URL, err)

		case data := <-frames:
			var env envelope
			if err := json.Unmarshal(data, &env); err != nil {
				warn(fmt.Sprintf("ignoring non-JSON frame: %.200s", string(data)))
				continue
			}

			if env.Type == TypeMarketData || env.Type == TypeOrderUpdate {
				if err := emit(data); err != nil {
					unsubscribe(conn, opts, frames, readErr, nil)
					closeConn(conn)
					return err
				}
				received++
				if opts.Count > 0 && received >= opts.Count {
					unsubscribe(conn, opts, frames, readErr, emit)
					closeConn(conn)
					return nil
				}
				continue
			}

			if opts.Raw {
				if err := emit(data); err != nil {
					closeConn(conn)
					return err
				}
			}

			switch state {
			case stateAuth:
				if env.Status == statusError {
					closeConn(conn)
					if env.Code == "AUTHENTICATION_ERROR" {
						return authError(opts.URL, env.Message)
					}
					return serverError(opts.URL, env)
				}
				if env.Type == "auth" && env.Status == statusSuccess {
					if err := send(conn, opts, subscribeMessage(opts)); err != nil {
						return connError(opts.URL, err)
					}
					state = stateSubscribe
					resetTimer(deadline, ackTimeout)
				}

			case stateSubscribe:
				if env.Status == statusError {
					closeConn(conn)
					if env.Code == "NOT_AUTHENTICATED" {
						return authError(opts.URL, env.Message)
					}
					return serverError(opts.URL, env)
				}
				if env.Type == "subscribe" || env.Type == "subscribe_orders" {
					if err := checkSubscribeAck(env, warn); err != nil {
						closeConn(conn)
						return serverError(opts.URL, envelope{Message: err.Error()})
					}
					state = stateStreaming
					deadline.Stop()
				}

			case stateStreaming:
				// Errors after subscribing (e.g. a broker hiccup) are not
				// fatal: the server keeps the session, so surface and go on.
				if env.Status == statusError && !opts.Raw {
					warn(fmt.Sprintf("server error: %s %s", env.Code, env.Message))
				}
			}
		}
	}
}

// send writes one JSON frame, reporting it to opts.Frame first.
func send(conn *websocket.Conn, opts Options, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if opts.Frame != nil {
		opts.Frame(">", data)
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// dial opens the WebSocket, mapping a refused connection or bad handshake
// to an actionable error.
func dial(ctx context.Context, opts Options) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 15 * time.Second,
	}
	header := http.Header{}
	if opts.UserAgent != "" {
		header.Set("User-Agent", opts.UserAgent)
	}
	conn, resp, err := dialer.DialContext(ctx, opts.URL, header)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		apiErr := connError(opts.URL, err)
		if resp != nil {
			apiErr.StatusCode = resp.StatusCode
		}
		return nil, apiErr
	}
	return conn, nil
}

func subscribeMessage(opts Options) map[string]any {
	if opts.Orders {
		return map[string]any{"action": "subscribe_orders", "request_id": "openalgo-cli-sub"}
	}
	msg := map[string]any{
		"action":     "subscribe",
		"symbols":    opts.Symbols,
		"mode":       opts.Mode,
		"request_id": "openalgo-cli-sub",
	}
	if opts.Mode == ModeDepth && opts.Depth > 0 {
		msg["depth"] = opts.Depth
	}
	return msg
}

func unsubscribeMessage(opts Options) map[string]any {
	if opts.Orders {
		return map[string]any{"action": "unsubscribe_orders", "request_id": "openalgo-cli-unsub"}
	}
	return map[string]any{
		"action":     "unsubscribe",
		"symbols":    opts.Symbols,
		"mode":       opts.Mode,
		"request_id": "openalgo-cli-unsub",
	}
}

// checkSubscribeAck fails when no symbol subscribed and warns about each
// symbol the broker refused on a partial success.
func checkSubscribeAck(env envelope, warn func(string)) error {
	if env.Status == statusSuccess || env.Type == "subscribe_orders" {
		return nil
	}
	var failed []string
	ok := 0
	for _, s := range env.Subscriptions {
		if s.Status == statusSuccess {
			ok++
			continue
		}
		failed = append(failed, fmt.Sprintf("%s:%s: %s", s.Exchange, s.Symbol, s.Message))
	}
	if ok == 0 {
		if len(failed) == 0 {
			return fmt.Errorf("subscription failed: %s", env.Message)
		}
		return fmt.Errorf("subscription failed: %s", strings.Join(failed, "; "))
	}
	for _, f := range failed {
		warn("subscription failed for " + f)
	}
	return nil
}

// unsubscribe sends the unsubscribe request and waits briefly for its ack.
// Data frames that race in meanwhile are dropped (the budget is spent);
// the ack itself is emitted when raw output was requested and emit is set.
func unsubscribe(conn *websocket.Conn, opts Options, frames <-chan json.RawMessage, readErr <-chan error, emit func(json.RawMessage) error) {
	_ = conn.SetWriteDeadline(time.Now().Add(closeAckTimeout))
	if err := send(conn, opts, unsubscribeMessage(opts)); err != nil {
		return
	}
	want := "unsubscribe"
	if opts.Orders {
		want = "unsubscribe_orders"
	}
	timer := time.NewTimer(closeAckTimeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return
		case <-readErr:
			return
		case data := <-frames:
			var env envelope
			if json.Unmarshal(data, &env) != nil {
				continue
			}
			if env.Type == want || (env.Type == "" && env.Status == statusError) {
				if opts.Raw && emit != nil {
					_ = emit(data)
				}
				return
			}
		}
	}
}

// closeConn sends a normal close frame; the deferred Close drops the socket.
func closeConn(conn *websocket.Conn) {
	_ = conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
}

func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

func authError(wsURL, msg string) *client.APIError {
	if msg == "" {
		msg = "WebSocket authentication failed"
	}
	return &client.APIError{
		StatusCode: http.StatusUnauthorized,
		Status:     statusError,
		Message:    msg,
		Method:     "WS",
		Path:       wsURL,
	}
}

// serverError maps a non-auth error frame to an APIError. BROKER_ERROR
// means OpenAlgo is up but the broker session is not, which is a 502.
func serverError(wsURL string, env envelope) *client.APIError {
	status := http.StatusBadRequest
	if env.Code == "BROKER_ERROR" {
		status = http.StatusBadGateway
	}
	msg := env.Message
	if env.Code != "" {
		msg = env.Code + ": " + msg
	}
	return &client.APIError{
		StatusCode: status,
		Status:     statusError,
		Message:    msg,
		Method:     "WS",
		Path:       wsURL,
	}
}

func connError(wsURL string, err error) *client.APIError {
	return &client.APIError{
		Status: statusError,
		Message: fmt.Sprintf("WebSocket %s: %v (is the OpenAlgo WebSocket server running? set ws_url in the profile, or OPENALGO_WS_URL alongside OPENALGO_API_KEY, to override the URL)",
			wsURL, err),
		Method: "WS",
		Path:   wsURL,
	}
}
