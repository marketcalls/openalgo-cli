//go:build integration

package integration

import (
	"strings"
	"testing"
)

// TestStreamOrders connects, authenticates and subscribes to order updates
// for a few seconds. With no orders placed it prints nothing; the check is a
// clean exit, or a structured error when the WebSocket server is not
// reachable from the test host.
func TestStreamOrders(t *testing.T) {
	t.Parallel()
	stdout, stderr, code := openalgoWithStderr(t, "stream", "orders", "--duration", "3s")
	if code == 0 {
		for _, line := range strings.Split(strings.TrimSpace(string(stdout)), "\n") {
			if line != "" {
				parseJSONMap(t, []byte(line))
			}
		}
		return
	}
	if !strings.Contains(string(stderr), "WebSocket") {
		t.Fatalf("stream orders exit %d: %s", code, stderr)
	}
	t.Skipf("WebSocket server not reachable: %s", stderr)
}

func TestStreamRequiresSymbols(t *testing.T) {
	t.Parallel()
	_, stderr, code := openalgoFail(t, "stream", "quote", "--count", "1")
	if code != 1 || !strings.Contains(string(stderr), "--symbols") {
		t.Errorf("exit %d, stderr %s", code, stderr)
	}
}
