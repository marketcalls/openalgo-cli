package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/marketcalls/openalgo-cli/internal/config"
	"github.com/marketcalls/openalgo-cli/internal/useragent"
)

const (
	ExitAPIError  = 1
	ExitAuthError = 2
)

const maxRetries = 3

type Client struct {
	HTTP      *http.Client
	BaseURL   string
	APIKey    string
	UserAgent string
	Verbose   bool
	Debug     bool
	Quiet     bool
	Trace     bool
	Timeout   time.Duration
}

type APIError struct {
	StatusCode int    `json:"-"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	hint       string
	retryAfter time.Duration
	// envelope is true when the body was a well-formed OpenAlgo
	// {"status":"error","message":...} reply: the handler ran and gave a
	// definite answer, so repeating the request would not change it.
	envelope bool
	// transport is true when no HTTP response arrived (timeout, reset,
	// refused, ...). unsent narrows that to failures before the request
	// could have left the machine (DNS lookup or dial), where a retry is
	// always safe.
	transport bool
	unsent    bool
}

// NewError builds an APIError that did not come from an HTTP response, such
// as a local validation failure, with an optional hint.
func NewError(message, hint string) *APIError {
	return &APIError{Message: message, hint: hint}
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("API error (HTTP %d)", e.StatusCode)
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s (HTTP %d)", e.Message, e.StatusCode)
	}
	return e.Message
}

// ExitCode maps the error to a process exit code. OpenAlgo answers an
// invalid or missing API key with 401 or 403, so both are auth errors.
func (e *APIError) ExitCode() int {
	if e.StatusCode == 401 || e.StatusCode == 403 {
		return ExitAuthError
	}
	return ExitAPIError
}

func (e *APIError) Hint() string {
	if e.hint != "" {
		return e.hint
	}
	switch e.StatusCode {
	case 400:
		return "Validation error. Check parameter values. Common issues: unknown symbol, wrong exchange, bad price type, or missing required fields."
	case 401, 403:
		return "Invalid API key, or the operation is blocked by the current mode. Run `openalgo profile login` or check the key under API Key in the OpenAlgo dashboard."
	case 404:
		return "Not found. Check the symbol, order ID, or strategy ID, and that the broker session is logged in."
	case 429:
		return "Rate limited. Reduce request frequency or add delays between calls."
	case 500, 502, 503:
		return "Server or broker error. Check that OpenAlgo is running and the broker session is active (`openalgo doctor`)."
	}
	return ""
}

var Version = "dev"

func New(cfg *config.Resolved) *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		BaseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		APIKey:    cfg.APIKey,
		UserAgent: useragent.Build(Version),
		Timeout:   30 * time.Second,
	}
}

func (c *Client) SetTimeout(d time.Duration) {
	c.Timeout = d
	c.HTTP.Timeout = d
}

// Do sends a request for an idempotent (read-only) operation. 429 and 5xx
// responses are retried with exponential backoff.
func (c *Client) Do(method, path string, params url.Values, body any) (json.RawMessage, error) {
	return c.request(method, c.BaseURL+path, params, body, true)
}

// DoWrite sends a request for an operation with side effects (placing,
// modifying or canceling orders, starting strategies, ...). OpenAlgo has no
// idempotency key, so a 5xx after the broker accepted an order must not be
// replayed. Only 429 (rejected by the rate limiter before the handler runs)
// is retried.
func (c *Client) DoWrite(method, path string, params url.Values, body any) (json.RawMessage, error) {
	return c.request(method, c.BaseURL+path, params, body, false)
}

func (c *Client) Get(path string, params url.Values) (json.RawMessage, error) {
	return c.Do("GET", path, params, nil)
}

func (c *Client) Post(path string, body any) (json.RawMessage, error) {
	return c.Do("POST", path, nil, body)
}

// RawRequest sends a request to a full URL. It is used by `openalgo api`,
// where the CLI cannot know whether the endpoint is safe to replay, so it is
// treated as a write.
func (c *Client) RawRequest(method, fullURL string, body any) (json.RawMessage, error) {
	return c.request(method, fullURL, nil, body, false)
}

func (c *Client) request(method, reqURL string, params url.Values, body any, idempotent bool) (json.RawMessage, error) {
	reqURL, payload, err := c.authenticate(method, reqURL, params, body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := range maxRetries {
		result, err := c.do(method, reqURL, payload)
		if err == nil {
			return result, nil
		}
		lastErr = err

		apiErr, ok := err.(*APIError)
		if !ok || !isRetryable(apiErr, idempotent) || attempt == maxRetries-1 {
			if ok && !idempotent {
				markOutcomeUnknown(apiErr)
			}
			return nil, err
		}

		delay := retryDelay(apiErr, attempt)
		if c.Verbose {
			fmt.Fprintf(os.Stderr, "  retrying in %s (attempt %d/%d)\n", delay, attempt+1, maxRetries)
		} else if !c.Quiet && apiErr.StatusCode == 429 {
			fmt.Fprintf(os.Stderr, "Rate limited, retrying in %s...\n", delay)
		}
		time.Sleep(delay)
	}
	return nil, lastErr
}

// authenticate injects the API key. OpenAlgo reads it from the JSON body for
// POST requests and from the `apikey` query parameter for GET requests.
func (c *Client) authenticate(method, reqURL string, params url.Values, body any) (string, []byte, error) {
	q := url.Values{}
	for k, v := range params {
		q[k] = v
	}

	var payload []byte
	if method == "GET" || method == "DELETE" {
		if c.APIKey != "" && q.Get("apikey") == "" && !strings.Contains(reqURL, "apikey=") {
			q.Set("apikey", c.APIKey)
		}
	} else {
		obj, err := toObject(body)
		if err != nil {
			return "", nil, err
		}
		if _, ok := obj["apikey"]; !ok && c.APIKey != "" {
			obj["apikey"] = c.APIKey
		}
		payload, err = json.Marshal(obj)
		if err != nil {
			return "", nil, fmt.Errorf("encoding request body: %w", err)
		}
	}

	if len(q) > 0 {
		sep := "?"
		if strings.Contains(reqURL, "?") {
			sep = "&"
		}
		reqURL += sep + q.Encode()
	}
	return reqURL, payload, nil
}

// toObject converts a request body into a JSON object so the API key can be
// added alongside the caller's fields.
func toObject(body any) (map[string]any, error) {
	switch v := body.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		out := make(map[string]any, len(v)+1)
		for k, val := range v {
			out[k] = val
		}
		return out, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		out := map[string]any{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("request body must be a JSON object: %w", err)
		}
		return out, nil
	}
}

// isRetryable reports whether a failed request is worth repeating. A 500
// that carries an OpenAlgo {"status":"error"} body is a definite answer from
// the handler (OpenAlgo uses 500 for some input errors, such as a bad option
// symbol), so only 500s without one are retried, along with 502/503/504.
func isRetryable(apiErr *APIError, idempotent bool) bool {
	status := apiErr.StatusCode
	if status == 429 {
		return true
	}
	if !idempotent {
		return false
	}
	switch status {
	case 502, 503, 504:
		return true
	case 500:
		return !apiErr.envelope
	}
	return false
}

// outcomeUnknownHint is shown when a write may have been applied even though
// the CLI saw an error. OpenAlgo has no idempotency key, so blindly repeating
// the command can place a duplicate order.
const outcomeUnknownHint = "The request may have reached the server. Check `openalgo order list` / `openalgo order trades` before retrying."

// markOutcomeUnknown rewrites the error for a write whose outcome cannot be
// known: the connection failed after the request may have been sent (timeout,
// reset, EOF), or a gateway answered 502/503/504 in place of OpenAlgo. Dial
// and DNS failures keep their "could not reach" wording since nothing was
// sent.
func markOutcomeUnknown(e *APIError) {
	switch {
	case e.transport && !e.unsent:
		e.Message = "outcome unknown: " + strings.TrimPrefix(e.Message, "could not reach ")
	case !e.transport && (e.StatusCode == 502 || e.StatusCode == 503 || e.StatusCode == 504):
		e.Message = "outcome unknown: " + e.Message
	default:
		return
	}
	e.hint = outcomeUnknownHint
}

// notSent reports whether a transport error happened before the request
// could have been written: a DNS failure or a failed dial (e.g. connection
// refused).
func notSent(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}

func retryDelay(apiErr *APIError, attempt int) time.Duration {
	if apiErr.StatusCode == 429 && apiErr.retryAfter > 0 {
		return apiErr.retryAfter
	}
	base := time.Duration(math.Pow(2, float64(attempt))) * 500 * time.Millisecond
	jitter := time.Duration(rand.Int64N(int64(base/2) + 1))
	return base + jitter
}

func (c *Client) do(method, reqURL string, payload []byte) (json.RawMessage, error) {
	start := time.Now()

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	if c.Debug {
		fmt.Fprintf(os.Stderr, "> %s %s\n", method, c.scrub(reqURL))
		c.debugHeaders(">", req.Header)
		if len(payload) > 0 {
			fmt.Fprintf(os.Stderr, "> %s\n", c.scrub(string(payload)))
		}
	}

	var tt *traceTimings
	if c.Trace {
		tt = newTrace(os.Stderr, method, c.scrub(reqURL))
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), tt.clientTrace()))
	}

	path := c.scrub(urlPath(reqURL))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &APIError{
			Message:   fmt.Sprintf("could not reach %s: %v", c.scrub(reqURL), c.scrubErr(err)),
			Method:    method,
			Path:      path,
			hint:      "check that OpenAlgo is running and the host is correct. Run `openalgo doctor` to verify configuration",
			transport: true,
			unsent:    notSent(err),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	elapsed := time.Since(start)
	if tt != nil {
		tt.finish(os.Stderr, resp.StatusCode, elapsed)
	} else if c.Verbose {
		fmt.Fprintf(os.Stderr, "%s %s -> %d (%dms)\n", method, c.scrub(reqURL), resp.StatusCode, elapsed.Milliseconds())
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &APIError{
			Message:   fmt.Sprintf("reading response from %s: %v", path, c.scrubErr(err)),
			Method:    method,
			Path:      path,
			transport: true,
		}
	}

	if c.Debug {
		fmt.Fprintf(os.Stderr, "< %s %s\n", resp.Proto, resp.Status)
		c.debugHeaders("<", resp.Header)
		if len(respBody) > 0 {
			fmt.Fprintf(os.Stderr, "< %s\n", c.scrub(string(respBody)))
		}
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Method: method, Path: path, RequestID: resp.Header.Get("X-Request-Id")}
		if env, ok := parseErrorBody(respBody); ok {
			apiErr.Status, apiErr.Message, apiErr.envelope = env.status, env.message, env.status == "error"
		}
		if apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(respBody))
			if apiErr.Message == "" || looksLikeHTML(apiErr.Message) {
				apiErr.Message = http.StatusText(resp.StatusCode)
				if looksLikeHTML(string(respBody)) {
					apiErr.hint = "Received a non-API (HTML) response. Verify the host with `openalgo doctor`: it should point at the OpenAlgo server, e.g. http://127.0.0.1:5000."
				}
			}
		}
		if resp.StatusCode == 429 {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					apiErr.retryAfter = time.Duration(secs) * time.Second
				}
			}
		}
		return nil, apiErr
	}

	// OpenAlgo occasionally reports failures in a 200 response with
	// {"status": "error"}. Surface those as errors so the exit code is honest.
	if env, ok := parseErrorBody(respBody); ok && env.status == "error" {
		msg := env.message
		if msg == "" {
			msg = "request failed"
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Status: "error", Message: msg, Method: method, Path: path, envelope: true}
	}

	if len(respBody) == 0 {
		return nil, nil
	}

	return json.RawMessage(respBody), nil
}

type errorBody struct {
	status  string
	message string
}

// parseErrorBody reads an OpenAlgo JSON reply's status and message. The
// message may be a string or, for request validation failures, an object of
// field errors ({"interval": ["Must be one of: ..."]}); objects are flattened
// to "interval: Must be one of: ...". Field errors in a separate "errors"
// object are appended the same way. ok is false when the body is not a JSON
// object.
func parseErrorBody(body []byte) (errorBody, bool) {
	var raw struct {
		Status  string          `json:"status"`
		Message json.RawMessage `json:"message"`
		Errors  json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(body, &raw) != nil {
		return errorBody{}, false
	}
	msg := flattenMessage(raw.Message)
	if details := flattenMessage(raw.Errors); details != "" && details != msg {
		if msg == "" {
			msg = details
		} else {
			msg += ": " + details
		}
	}
	return errorBody{status: raw.Status, message: msg}, true
}

// flattenMessage renders a JSON message value as one line of text: strings
// as-is, arrays joined with "; ", objects as "field: msg; field2: msg" with
// keys sorted so the text is stable.
func flattenMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	return flattenValue(v)
}

func flattenValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			if s := flattenValue(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "; ")
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			if s := flattenValue(val[k]); s != "" {
				parts = append(parts, k+": "+s)
			}
		}
		return strings.Join(parts, "; ")
	default:
		return fmt.Sprint(val)
	}
}

func stripQuery(u string) string {
	if i := strings.IndexByte(u, '?'); i >= 0 {
		return u[:i]
	}
	return u
}

// urlPath returns just the path of a request URL (e.g. /api/v1/placeorder)
// for error reports; the host is already known and the query may hold the key.
func urlPath(u string) string {
	if parsed, err := url.Parse(u); err == nil && parsed.Path != "" {
		return parsed.Path
	}
	return stripQuery(u)
}

func looksLikeHTML(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(s, "<!doctype") || strings.HasPrefix(s, "<html")
}

// debugHeaders prints headers sorted by name so runs can be diffed.
func (c *Client) debugHeaders(prefix string, h http.Header) {
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for _, v := range h[name] {
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", prefix, name, c.scrub(v))
		}
	}
}

func (c *Client) scrub(s string) string {
	if c.APIKey != "" {
		s = strings.ReplaceAll(s, c.APIKey, "[REDACTED]")
		s = strings.ReplaceAll(s, url.QueryEscape(c.APIKey), "[REDACTED]")
	}
	return s
}

func (c *Client) scrubErr(err error) error {
	if err == nil {
		return nil
	}
	msg := c.scrub(err.Error())
	if msg == err.Error() {
		return err
	}
	return fmt.Errorf("%s", msg)
}

// traceTimings prints each HTTP phase to stderr as it completes.
type traceTimings struct {
	w                      io.Writer
	dnsStart, connectStart time.Time
	tlsStart, wroteRequest time.Time
	gotFirstByte           time.Time
}

func newTrace(w io.Writer, method, url string) *traceTimings {
	fmt.Fprintf(w, "trace: %s %s\n", method, url)
	return &traceTimings{w: w}
}

func (t *traceTimings) clientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { t.dnsStart = time.Now() },
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			fmt.Fprintf(t.w, "  dns:     %dms\n", time.Since(t.dnsStart).Milliseconds())
		},
		ConnectStart: func(_, _ string) { t.connectStart = time.Now() },
		ConnectDone: func(_, addr string, _ error) {
			fmt.Fprintf(t.w, "  tcp:     %dms  (%s)\n", time.Since(t.connectStart).Milliseconds(), addr)
		},
		TLSHandshakeStart: func() { t.tlsStart = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			fmt.Fprintf(t.w, "  tls:     %dms\n", time.Since(t.tlsStart).Milliseconds())
		},
		WroteRequest:         func(_ httptrace.WroteRequestInfo) { t.wroteRequest = time.Now() },
		GotFirstResponseByte: func() { t.gotFirstByte = time.Now() },
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				fmt.Fprintf(t.w, "  conn:    reused (%s)\n", info.Conn.RemoteAddr())
			}
		},
	}
}

func (t *traceTimings) finish(w io.Writer, status int, elapsed time.Duration) {
	if !t.wroteRequest.IsZero() && !t.gotFirstByte.IsZero() {
		fmt.Fprintf(w, "  ttfb:    %dms\n", t.gotFirstByte.Sub(t.wroteRequest).Milliseconds())
	}
	fmt.Fprintf(w, "  total:   %dms -> %d\n", elapsed.Milliseconds(), status)
}
