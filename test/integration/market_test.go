//go:build integration

package integration

import "testing"

func TestMarketHolidays(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "market", "holidays"))
	requireFields(t, resp, "data")
}

func TestMarketTimings(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "market", "timings", "--date", daysAgo(1)))
	requireFields(t, resp, "data")
}
