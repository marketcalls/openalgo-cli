//go:build integration

package integration

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOrderLifecycle(t *testing.T) {
	orderID, price := placeTestOrder(t, "SBIN")

	t.Run("status", func(t *testing.T) {
		resp := requireSuccess(t, openalgo(t, "order", "status", "--orderid", orderID, "--strategy", testStrategy))
		data, _ := resp["data"].(map[string]any)
		if data["orderid"] != orderID {
			t.Errorf("status orderid = %v, want %s", data["orderid"], orderID)
		}
		if data["symbol"] != "SBIN" || data["action"] != "BUY" {
			t.Errorf("status data = %v", data)
		}
	})

	t.Run("list", func(t *testing.T) {
		pollFor(t, 5*time.Second, "order to appear in the order book", func() bool {
			out := openalgo(t, "order", "list", "--jq", fmt.Sprintf(`[.data.orders[] | select(.orderid == %q)] | length`, orderID))
			return strings.TrimSpace(string(out)) == "1"
		})
	})

	t.Run("modify", func(t *testing.T) {
		p, _ := strconv.ParseFloat(price, 64)
		newPrice := fmt.Sprintf("%.2f", p+0.05)
		requireSuccess(t, openalgo(t, "order", "modify",
			"--orderid", orderID, "--symbol", "SBIN", "--exchange", "NSE", "--action", "BUY",
			"--product", "CNC", "--pricetype", "LIMIT", "--price", newPrice, "--quantity", "1",
			"--strategy", testStrategy))
		pollFor(t, 5*time.Second, "modified price", func() bool {
			out := openalgo(t, "order", "status", "--orderid", orderID, "--strategy", testStrategy, "--jq", ".data.price")
			got, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
			return fmt.Sprintf("%.2f", got) == newPrice
		})
	})

	t.Run("cancel", func(t *testing.T) {
		requireSuccess(t, openalgo(t, "order", "cancel", "--orderid", orderID, "--strategy", testStrategy))
		pollFor(t, 5*time.Second, "order to be cancelled", func() bool {
			out := openalgo(t, "order", "status", "--orderid", orderID, "--strategy", testStrategy, "--jq", ".data.order_status")
			return strings.Contains(strings.ToLower(string(out)), "cancel")
		})
	})
}

func TestOrderDryRunSendsNothing(t *testing.T) {
	t.Parallel()
	body := parseJSONMap(t, openalgo(t, "order", "place",
		"--symbol", "SBIN", "--exchange", "NSE", "--action", "BUY", "--quantity", "1",
		"--product", "CNC", "--pricetype", "LIMIT", "--price", "1", "--dry-run"))
	if _, ok := body["apikey"]; ok {
		t.Error("dry-run body contains apikey")
	}
	if body["strategy"] != "openalgo-cli" {
		t.Errorf("strategy = %v, want the openalgo-cli default", body["strategy"])
	}
}

func TestOrderStopLossRequiresTriggerPrice(t *testing.T) {
	t.Parallel()
	_, stderr, code := openalgoFail(t, "order", "modify",
		"--orderid", "1", "--symbol", "SBIN", "--exchange", "NSE", "--action", "BUY",
		"--product", "CNC", "--pricetype", "SL", "--price", "781", "--quantity", "1", "--dry-run")
	if code != 1 || !strings.Contains(string(stderr), "trigger-price") {
		t.Errorf("exit %d, stderr %s", code, stderr)
	}
}

func TestOrderInvalidEnumRejectedLocally(t *testing.T) {
	t.Parallel()
	_, stderr, code := openalgoFail(t, "order", "place",
		"--symbol", "SBIN", "--exchange", "NSE", "--action", "HOLD", "--quantity", "1", "--product", "CNC")
	if code != 1 || !strings.Contains(string(stderr), "--action") {
		t.Errorf("exit %d, stderr %s", code, stderr)
	}
}
