//go:build integration

package integration

import "testing"

func TestAccountFunds(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "account", "funds"))
	data, _ := resp["data"].(map[string]any)
	requireFields(t, data, "availablecash", "utiliseddebits")
}

func TestAccountHoldings(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "account", "holdings"))
	requireFields(t, resp, "data")
}

func TestPositionList(t *testing.T) {
	t.Parallel()
	requireSuccess(t, openalgo(t, "position", "list"))
}

func TestOrderList(t *testing.T) {
	t.Parallel()
	resp := requireSuccess(t, openalgo(t, "order", "list"))
	data, _ := resp["data"].(map[string]any)
	requireFields(t, data, "orders", "statistics")
}

func TestOrderTrades(t *testing.T) {
	t.Parallel()
	requireSuccess(t, openalgo(t, "order", "trades"))
}

func TestAccountMargin(t *testing.T) {
	t.Parallel()
	// Margin needs broker support; accept a structured error.
	openalgoJSONOrStructuredError(t, "account", "margin",
		"--positions", `[{"symbol":"SBIN","exchange":"NSE","action":"BUY","quantity":"1","product":"CNC","pricetype":"MARKET"}]`)
}
