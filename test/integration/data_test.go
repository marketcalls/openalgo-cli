//go:build integration

package integration

import "testing"

func TestDataQuote(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "data", "quote", "--symbol", "RELIANCE", "--exchange", "NSE"))
	data, _ := resp["data"].(map[string]any)
	requireFields(t, data, "ltp", "open", "high", "low")
}

// TestDataQuoteLowercaseEnum checks that enum values typed in any case are
// sent in the canonical spelling the server validates against.
func TestDataQuoteLowercaseEnum(t *testing.T) {
	t.Parallel()
	requireSuccess(t, openalgo(t, "data", "quote", "--symbol", "RELIANCE", "--exchange", "nse"))
}

func TestDataQuotes(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "data", "quotes",
		"--symbols", `[{"symbol":"RELIANCE","exchange":"NSE"},{"symbol":"SBIN","exchange":"NSE"}]`))
	requireFields(t, resp, "results")
}

func TestDataDepth(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "data", "depth", "--symbol", "SBIN", "--exchange", "NSE"))
	data, _ := resp["data"].(map[string]any)
	requireFields(t, data, "bids", "asks", "ltp")
}

func TestDataHistory(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "data", "history",
		"--symbol", "SBIN", "--exchange", "NSE", "--interval", "D",
		"--start-date", daysAgo(20), "--end-date", daysAgo(1)))
	requireFields(t, resp, "data")
}

func TestDataIntervals(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "data", "intervals"))
	requireFields(t, resp, "data")
}
