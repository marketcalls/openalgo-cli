//go:build integration

package integration

import "testing"

// nearestExpiry returns the first NIFTY option expiry in the DD-MMM-YY form
// `symbol expiry` prints; option commands accept it as-is.
func nearestExpiry(t *testing.T) string {
	t.Helper()
	expiries := parseJSON[[]string](t, openalgo(t, "symbol", "expiry",
		"--symbol", "NIFTY", "--exchange", "NFO", "--instrumenttype", "options", "--jq", ".data"))
	if len(expiries) == 0 {
		t.Skip("no NIFTY option expiries")
	}
	return expiries[0]
}

func TestOptionChain(t *testing.T) {
	t.Parallel()
	expiry := nearestExpiry(t)
	resp := requireSuccess(t, openalgo(t, "option", "chain",
		"--underlying", "NIFTY", "--exchange", "NSE_INDEX", "--expiry-date", expiry, "--strike-count", "2"))
	requireFields(t, resp, "chain")
}

func TestOptionSymbol(t *testing.T) {
	t.Parallel()
	expiry := nearestExpiry(t)
	resp := requireSuccess(t, openalgo(t, "option", "symbol",
		"--underlying", "NIFTY", "--exchange", "NSE_INDEX", "--expiry-date", expiry,
		"--offset", "ATM", "--option-type", "CE"))
	if s, _ := resp["symbol"].(string); s == "" {
		t.Errorf("option symbol returned no symbol: %v", resp)
	}
}

func TestOptionGreeks(t *testing.T) {
	t.Parallel()
	expiry := nearestExpiry(t)
	resp := requireSuccess(t, openalgo(t, "option", "symbol",
		"--underlying", "NIFTY", "--exchange", "NSE_INDEX", "--expiry-date", expiry,
		"--offset", "ATM", "--option-type", "CE"))
	symbol, _ := resp["symbol"].(string)
	if symbol == "" {
		t.Skip("no ATM symbol")
	}
	// Greeks need live prices; accept a structured error when markets are closed.
	openalgoJSONOrStructuredError(t, "option", "greeks", "--symbol", symbol, "--exchange", "NFO")
}
