//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	cliBinary     string
	testConfigDir string
	// sandbox is true when the server reported analyzer (sandbox) mode at
	// startup. Tests that place orders skip unless it is set.
	sandbox bool
)

// testStrategy tags every order a test places so it is easy to find (and
// clean up by hand) in the order book.
const testStrategy = "openalgo-cli-itest"

func TestMain(m *testing.M) {
	if os.Getenv("OPENALGO_TEST_API_KEY") == "" {
		// OpenAlgo is self-hosted, so there is no shared server to test
		// against. Without credentials there is nothing to run.
		fmt.Fprintln(os.Stderr, "Skipping integration tests: set OPENALGO_TEST_API_KEY (and optionally OPENALGO_TEST_HOST) to run them against an OpenAlgo server in analyzer mode")
		os.Exit(0)
	}

	dir, err := os.MkdirTemp("", "openalgo-cli-test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	binary := filepath.Join(dir, "openalgo")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	fmt.Println("Building CLI binary...")
	build := exec.Command("go", "build", "-o", binary, "./cmd/openalgo")
	build.Dir = projectRoot()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	cliBinary = binary

	// An empty config dir keeps a developer's own profiles out of the run;
	// credentials come only from the env bundle in cliEnv.
	testConfigDir = filepath.Join(dir, "config")
	if err := os.MkdirAll(testConfigDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create config dir: %v\n", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	sandbox = detectSandbox()
	if !sandbox {
		fmt.Fprintln(os.Stderr, "Server is not in analyzer (sandbox) mode: order tests will be skipped")
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// detectSandbox asks the server whether analyzer mode is on. Any error
// counts as "not sandbox" so write tests stay off.
func detectSandbox() bool {
	cmd := makeCmd("analyzer", "status")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	var resp struct {
		Data struct {
			AnalyzeMode bool   `json:"analyze_mode"`
			Mode        string `json:"mode"`
		} `json:"data"`
	}
	if json.Unmarshal(out, &resp) != nil {
		return false
	}
	return resp.Data.AnalyzeMode && resp.Data.Mode == "analyze"
}

// requireSandbox skips a test that places orders unless the server is in
// analyzer mode. Never place orders against a live server.
func requireSandbox(t *testing.T) {
	t.Helper()
	if !sandbox {
		t.Skip("server is not in analyzer (sandbox) mode")
	}
}

// openalgo runs the CLI and returns stdout. Fatals on non-zero exit.
func openalgo(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := makeCmd(args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("openalgo %s failed (exit %d):\nstdout: %s\nstderr: %s",
				strings.Join(args, " "), exitErr.ExitCode(), string(out), string(exitErr.Stderr))
		}
		t.Fatalf("openalgo %s failed: %v", strings.Join(args, " "), err)
	}
	return out
}

// openalgoWithStderr runs the CLI and returns stdout, stderr and the exit
// code. Does NOT fatal on non-zero exit - caller must check.
func openalgoWithStderr(t *testing.T, args ...string) (stdout, stderr []byte, exitCode int) {
	t.Helper()
	cmd := makeCmd(args...)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	stdout, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stdout, stderrBuf.Bytes(), exitErr.ExitCode()
		}
		t.Fatalf("openalgo %s failed unexpectedly: %v", strings.Join(args, " "), err)
	}
	return stdout, stderrBuf.Bytes(), 0
}

// openalgoJSONOrStructuredError is for endpoints that depend on the broker
// or server setup (Telegram, strategies, Greeks): it accepts either a JSON
// response on stdout or a structured JSON error on stderr with nothing on
// stdout. ok reports which one it got.
func openalgoJSONOrStructuredError(t *testing.T, args ...string) (map[string]any, bool) {
	t.Helper()
	stdout, stderr, exitCode := openalgoWithStderr(t, args...)
	if exitCode == 0 {
		return parseJSONMap(t, stdout), true
	}
	if len(stdout) != 0 {
		t.Fatalf("openalgo %s wrote stdout on error: %s", strings.Join(args, " "), string(stdout))
	}
	data := parseJSONMap(t, stderr)
	requireFields(t, data, "error", "status")
	return data, false
}

// openalgoFail runs the CLI and expects a non-zero exit. Fatals if it
// succeeds.
func openalgoFail(t *testing.T, args ...string) (stdout, stderr []byte, exitCode int) {
	t.Helper()
	stdout, stderr, exitCode = openalgoWithStderr(t, args...)
	if exitCode == 0 {
		t.Fatalf("openalgo %s succeeded but expected failure", strings.Join(args, " "))
	}
	return stdout, stderr, exitCode
}

func parseJSON[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("Failed to parse JSON:\n%s\nError: %v", string(data), err)
	}
	return v
}

func parseJSONMap(t *testing.T, data []byte) map[string]any {
	t.Helper()
	return parseJSON[map[string]any](t, data)
}

// requireFields asserts that the JSON map has all listed fields.
func requireFields(t *testing.T, m map[string]any, fields ...string) {
	t.Helper()
	for _, f := range fields {
		if _, ok := m[f]; !ok {
			t.Errorf("missing required field %q in %v", f, m)
		}
	}
}

// requireSuccess parses an OpenAlgo envelope and asserts status=success.
func requireSuccess(t *testing.T, out []byte) map[string]any {
	t.Helper()
	m := parseJSONMap(t, out)
	if m["status"] != "success" {
		t.Fatalf("status = %v, want success: %s", m["status"], string(out))
	}
	return m
}

// cliEnv builds the child environment: the test key and host as the env
// credential bundle, an isolated config dir, and every other OPENALGO_*
// setting cleared so the developer's shell cannot change the results.
func cliEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "OPENALGO_") {
			env = append(env, kv)
		}
	}
	host := os.Getenv("OPENALGO_TEST_HOST")
	if host == "" {
		host = "http://127.0.0.1:5000"
	}
	env = append(env,
		"OPENALGO_CONFIG_DIR="+testConfigDir,
		"OPENALGO_API_KEY="+os.Getenv("OPENALGO_TEST_API_KEY"),
		"OPENALGO_HOST="+host,
	)
	if ws := os.Getenv("OPENALGO_TEST_WS_URL"); ws != "" {
		env = append(env, "OPENALGO_WS_URL="+ws)
	}
	return env
}

// makeCmd creates an exec.Command for the CLI with the test environment.
func makeCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(cliBinary, args...)
	cmd.Env = cliEnv()
	return cmd
}

func projectRoot() string {
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "."
}

func daysAgo(n int) string {
	return time.Now().AddDate(0, 0, -n).Format("2006-01-02")
}

func pollFor(t *testing.T, timeout time.Duration, desc string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if fn() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %v waiting for %s", timeout, desc)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// placeTestOrder places a CNC LIMIT buy of one share at half the last price,
// far enough from the market that it never fills, and returns the order id.
// It registers a t.Cleanup that cancels the order by id; tests must never
// use order cancel-all or position close-all.
func placeTestOrder(t *testing.T, symbol string) (orderID string, price string) {
	t.Helper()
	requireSandbox(t)
	ltp := parseJSON[float64](t, openalgo(t, "data", "quote", "--symbol", symbol, "--exchange", "NSE", "--jq", ".data.ltp"))
	if ltp <= 0 {
		t.Skipf("no last price for %s", symbol)
	}
	price = fmt.Sprintf("%.2f", float64(int(ltp/2*20))/20) // half the LTP on a 0.05 tick
	out := openalgo(t, "order", "place",
		"--symbol", symbol,
		"--exchange", "NSE",
		"--action", "BUY",
		"--quantity", "1",
		"--product", "CNC",
		"--pricetype", "LIMIT",
		"--price", price,
		"--strategy", testStrategy,
	)
	resp := requireSuccess(t, out)
	id, _ := resp["orderid"].(string)
	if id == "" {
		t.Fatalf("order place returned no orderid: %s", string(out))
	}
	t.Cleanup(func() {
		_ = makeCmd("order", "cancel", "--orderid", id, "--strategy", testStrategy).Run()
	})
	return id, price
}
