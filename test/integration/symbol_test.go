//go:build integration

package integration

import "testing"

func TestSymbolGet(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "symbol", "get", "--symbol", "SBIN", "--exchange", "NSE"))
	data, _ := resp["data"].(map[string]any)
	if data["symbol"] != "SBIN" {
		t.Errorf("symbol = %v, want SBIN", data["symbol"])
	}
}

func TestSymbolSearch(t *testing.T) {
	t.Parallel()
	n := parseJSON[int](t, openalgo(t, "symbol", "search", "--query", "RELIANCE", "--exchange", "NSE", "--jq", ".data | length"))
	if n == 0 {
		t.Error("symbol search RELIANCE returned no results")
	}
}

func TestSymbolExpiry(t *testing.T) {
	t.Parallel()
	expiries := parseJSON[[]string](t, openalgo(t, "symbol", "expiry",
		"--symbol", "NIFTY", "--exchange", "NFO", "--instrumenttype", "options", "--jq", ".data"))
	if len(expiries) == 0 {
		t.Fatal("no NIFTY option expiries")
	}
}

func TestSymbolUnknownIsStructuredError(t *testing.T) {
	t.Parallel()
	stdout, stderr, code := openalgoFail(t, "symbol", "get", "--symbol", "NOSUCHSYMBOLXYZ", "--exchange", "NSE")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if len(stdout) != 0 {
		t.Errorf("stdout on error: %s", stdout)
	}
	requireFields(t, parseJSONMap(t, stderr), "error", "status", "hint")
}
