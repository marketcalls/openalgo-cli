//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestPing(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "ping"))
	data, _ := resp["data"].(map[string]any)
	if data["message"] != "pong" {
		t.Errorf("ping data = %v, want message pong", data)
	}
	if b, _ := data["broker"].(string); b == "" {
		t.Error("ping data has no broker")
	}
}

func TestPingQuiet(t *testing.T) {
	t.Parallel()
	_, stderr, code := openalgoWithStderr(t, "ping", "--quiet")
	if code != 0 {
		t.Fatalf("ping --quiet exit %d: %s", code, stderr)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()
	out := strings.TrimSpace(string(openalgo(t, "version")))
	if out == "" {
		t.Error("version printed nothing")
	}
}

func TestAnalyzerStatus(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "analyzer", "status"))
	data, _ := resp["data"].(map[string]any)
	requireFields(t, data, "analyze_mode", "mode")
}
