//go:build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"
)

func TestOutputCSV(t *testing.T) {
	t.Parallel()
	out := string(openalgo(t, "data", "history",
		"--symbol", "SBIN", "--exchange", "NSE", "--interval", "D",
		"--start-date", daysAgo(20), "--end-date", daysAgo(1), "--csv"))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 || !strings.Contains(lines[0], "close") {
		t.Errorf("csv output = %q, want a header with close and data rows", out)
	}
}

func TestOutputJQ(t *testing.T) {
	t.Parallel()
	out := strings.TrimSpace(string(openalgo(t, "ping", "--jq", ".data.message")))
	if out != `"pong"` {
		t.Errorf("jq output = %q, want \"pong\"", out)
	}
}

func TestOutputSchemaNoCall(t *testing.T) {
	t.Parallel()
	out := string(openalgo(t, "order", "place", "--schema"))
	if !strings.Contains(out, "orderid") {
		t.Errorf("schema output missing orderid:\n%s", out)
	}
}

func TestInvalidKeyIsAuthError(t *testing.T) {
	t.Parallel()
	cmd := makeCmd("account", "funds")
	for i, kv := range cmd.Env {
		if strings.HasPrefix(kv, "OPENALGO_API_KEY=") {
			cmd.Env[i] = "OPENALGO_API_KEY=invalid-key-for-integration-test"
		}
	}
	stdout, err := cmd.Output()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected an exit error, got %v (stdout %s)", err, stdout)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("exit = %d, want 2 (auth error); stderr %s", exitErr.ExitCode(), exitErr.Stderr)
	}
	if len(stdout) != 0 {
		t.Errorf("stdout on error: %s", stdout)
	}
	data := parseJSONMap(t, exitErr.Stderr)
	requireFields(t, data, "error", "status", "hint")
	if strings.Contains(string(exitErr.Stderr), "invalid-key-for-integration-test") {
		t.Error("API key leaked into the error output")
	}
}
