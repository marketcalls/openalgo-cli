package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeOpenAlgo serves /api/v1/ping and /api/v1/analyzer.
func fakeOpenAlgo(t *testing.T, analyzeMode bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ping":
			_, _ = w.Write([]byte(`{"status":"success","data":{"broker":"zerodha","message":"pong"}}`))
		case "/api/v1/analyzer":
			if analyzeMode {
				_, _ = w.Write([]byte(`{"status":"success","data":{"analyze_mode":true,"mode":"analyze","total_logs":7}}`))
			} else {
				_, _ = w.Write([]byte(`{"status":"success","data":{"analyze_mode":false,"mode":"live","total_logs":0}}`))
			}
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDoctor_AllOK(t *testing.T) {
	isolateConfig(t)
	captureColorOutput(t)
	srv := fakeOpenAlgo(t, true)
	t.Setenv("OPENALGO_API_KEY", "doctor-key")
	t.Setenv("OPENALGO_HOST", srv.URL)

	var buf bytes.Buffer
	if err := runDoctor(&buf, "", false); err != nil {
		t.Fatalf("doctor failed: %v\n%s", err, buf.String())
	}
	out := buf.String()
	for _, want := range []string{
		"[ok]   ping: connected (broker: zerodha)",
		"analyzer (sandbox) mode: on (7 orders logged)",
		"WebSocket: ws://127.0.0.1:8765",
		"All checks passed.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "doctor-key") {
		t.Error("doctor output leaks the API key")
	}
}

func TestDoctor_AnalyzerOff(t *testing.T) {
	isolateConfig(t)
	captureColorOutput(t)
	srv := fakeOpenAlgo(t, false)
	t.Setenv("OPENALGO_API_KEY", "k")
	t.Setenv("OPENALGO_HOST", srv.URL)

	var buf bytes.Buffer
	_ = runDoctor(&buf, "", false)
	if !strings.Contains(buf.String(), "analyzer (sandbox) mode: off") {
		t.Errorf("output:\n%s", buf.String())
	}
}

func TestDoctor_NoCredentials(t *testing.T) {
	isolateConfig(t)
	captureColorOutput(t)

	var buf bytes.Buffer
	err := runDoctor(&buf, "", false)
	if err == nil {
		t.Fatal("expected failure without credentials")
	}
	if !strings.Contains(buf.String(), "[FAIL] no API key") {
		t.Errorf("output:\n%s", buf.String())
	}
}

func TestDoctor_PingFails(t *testing.T) {
	isolateConfig(t)
	captureColorOutput(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"status":"error","message":"Invalid openalgo apikey"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OPENALGO_API_KEY", "k")
	t.Setenv("OPENALGO_HOST", srv.URL)

	var buf bytes.Buffer
	if err := runDoctor(&buf, "", false); err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(buf.String(), "[FAIL] ping: Invalid openalgo apikey") {
		t.Errorf("output:\n%s", buf.String())
	}
}

func TestAnalyzerDescription(t *testing.T) {
	if got := analyzerDescription([]byte(`{"data":{"analyze_mode":true,"total_logs":3}}`)); got != "analyzer (sandbox) mode: on (3 orders logged)" {
		t.Errorf("on = %q", got)
	}
	if got := analyzerDescription([]byte(`{"data":{"analyze_mode":false}}`)); !strings.HasPrefix(got, "analyzer (sandbox) mode: off") {
		t.Errorf("off = %q", got)
	}
}

func TestJoinMax(t *testing.T) {
	if got := joinMax([]string{"a", "b"}, 5); got != "[a b]" {
		t.Errorf("got %q", got)
	}
	if got := joinMax([]string{"a", "b", "c"}, 2); got != "[a b] (+1 more)" {
		t.Errorf("got %q", got)
	}
}
