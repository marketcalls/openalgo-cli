package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteStreamMessage(t *testing.T) {
	msg := json.RawMessage(`{
  "type": "market_data", "symbol": "RELIANCE", "exchange": "NSE", "mode": 1,
  "data": {"ltp": 1424.5, "timestamp": 1756376445123}
}`)
	cases := []struct {
		name, jq, want string
	}{
		{"raw is compacted to one line", "", `{"type":"market_data","symbol":"RELIANCE","exchange":"NSE","mode":1,"data":{"ltp":1424.5,"timestamp":1756376445123}}` + "\n"},
		{"jq object", "{symbol, ltp: .data.ltp}", `{"ltp":1424.5,"symbol":"RELIANCE"}` + "\n"},
		{"jq scalar", ".data.ltp", "1424.5\n"},
		{"jq select no match prints nothing", "select(.data.ltp > 5000)", ""},
		{"jq select match", "select(.data.ltp > 1000) | .symbol", `"RELIANCE"` + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeStreamMessage(&buf, msg, tc.jq); err != nil {
				t.Fatalf("writeStreamMessage: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWriteStreamMessageBadJQ(t *testing.T) {
	var buf bytes.Buffer
	if err := writeStreamMessage(&buf, json.RawMessage(`{}`), ".["); err == nil {
		t.Error("expected jq parse error")
	}
}

func TestStreamCommandTree(t *testing.T) {
	for _, name := range []string{"ltp", "quote", "depth", "orders"} {
		c, _, err := rootCmd.Find([]string{"stream", name})
		if err != nil || c.Name() != name {
			t.Fatalf("stream %s not registered: %v", name, err)
		}
		for _, f := range []string{"count", "duration", "raw"} {
			if c.Flags().Lookup(f) == nil {
				t.Errorf("stream %s missing --%s", name, f)
			}
		}
	}
	if streamDepthCmd.Flags().Lookup("depth").DefValue != "5" {
		t.Error("stream depth --depth should default to 5")
	}
	if streamCmd.GroupID != "util" {
		t.Errorf("stream group = %q", streamCmd.GroupID)
	}
}
