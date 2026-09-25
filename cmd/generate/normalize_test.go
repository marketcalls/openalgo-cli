package main

import "testing"

func TestNormalizeOASDescription(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "whitespace only", raw: "   \n\t  ", want: ""},
		{
			name: "simple one-liner",
			raw:  "Returns your account details.",
			want: "Returns your account details",
		},
		{
			name: "first paragraph only",
			raw:  "First paragraph here.\n\nSecond paragraph ignored.",
			want: "First paragraph here",
		},
		{
			name: "multi-line first paragraph joined",
			raw:  "Line one of the description.\nLine two continues.\n\nSecond paragraph.",
			want: "Line one of the description. Line two continues",
		},
		{
			name: "blockquote markers stripped",
			raw:  "> Warning\n>\n> This is a warning block.",
			want: "Warning This is a warning block",
		},
		{
			name: "markdown links converted to text",
			raw:  "Use [the docs](https://example.com) for details.",
			want: "Use the docs for details",
		},
		{
			name: "multiple markdown links",
			raw:  "See [foo](https://a.com) and [bar](https://b.com).",
			want: "See foo and bar",
		},
		{
			name: "unclosed bracket not mangled",
			raw:  "Array items like [0] are fine.",
			want: "Array items like [0] are fine",
		},
		{
			name: "trailing periods stripped",
			raw:  "Multiple trailing periods...",
			want: "Multiple trailing periods",
		},
		{
			name: "long text truncated at sentence boundary",
			raw:  "First sentence here. " + string(make([]byte, 300)),
			want: "First sentence here",
		},
		{
			name: "long text truncated at word boundary when no sentence break",
			raw:  "Word " + repeatWord("word", 80),
			want: "Word " + repeatWord("word", 59),
		},
		{
			name: "exactly 300 chars not truncated",
			raw:  repeat('A', 300),
			want: repeat('A', 300),
		},
		{
			name: "301 chars truncated",
			raw:  repeat('A', 301),
			want: repeat('A', 300),
		},
		{
			name: "backticks stripped",
			raw:  "Use `source` db for Historify data.",
			want: "Use source db for Historify data",
		},
		{
			name: "real OAS: close position",
			raw:  "Destructive operation that squares off every open position across all exchanges using MARKET orders with the same product type. There is no confirmation step.",
			want: "Destructive operation that squares off every open position across all exchanges using MARKET orders with the same product type. There is no confirmation step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeOASDescription(tt.raw)
			if got != tt.want {
				t.Errorf("normalizeOASDescription(%q)\n  got:  %q\n  want: %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestStripMarkdownLinks(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"no links here", "no links here"},
		{"[text](url)", "text"},
		{"before [text](url) after", "before text after"},
		{"[a](x) and [b](y)", "a and b"},
		{"unclosed [bracket", "unclosed [bracket"},
		{"[text](", "[text]("},
		{"[text](url) [no-url]", "text [no-url]"},
	}
	for _, tt := range tests {
		got := stripMarkdownLinks(tt.in)
		if got != tt.want {
			t.Errorf("stripMarkdownLinks(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func repeat(ch byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func repeatWord(w string, n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += " "
		}
		s += w
	}
	return s
}

func TestNormalizeDesc(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"Exchange code.", "exchange code"},
		{"The order id", "order id"},
		{"NSE:RELIANCE style symbol", "NSE:RELIANCE style symbol"},
		{"use `json` or txt", "use json or txt"},
		{repeatWord("word", 20) + ". Second sentence " + repeatWord("x", 60), repeatWord("word", 20)},
		{"trigger below LTP; " + repeatWord("word", 30), "trigger below LTP"},
	}
	for _, tt := range tests {
		if got := normalizeDesc(tt.in); got != tt.want {
			t.Errorf("normalizeDesc(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeSummary(t *testing.T) {
	tests := []struct{ method, in, want string }{
		{"POST", "Place an order", "Place an order"},
		{"POST", "Send a WhatsApp message", "Send a WhatsApp message"},
		{"GET", "Get Telegram Bot Configuration", "Get Telegram bot configuration"},
		{"POST", "Get sandbox P&L by symbol.", "Get sandbox P&L by symbol"},
		{"POST", "Modify An Active GTT Order", "Modify an active GTT order"},
		{"GET", "Market holidays", "Get market holidays"},
		{"POST", "Option chain", "Get option chain"},
		{"POST", "", ""},
	}
	for _, tt := range tests {
		if got := normalizeSummary(tt.method, tt.in); got != tt.want {
			t.Errorf("normalizeSummary(%q, %q) = %q, want %q", tt.method, tt.in, got, tt.want)
		}
	}
}

func TestToGoName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"PlaceGTTOrder", "PlaceGTTOrder"},
		{"symbol", "Symbol"},
		{"telegram_id", "TelegramID"},
		{"webhook_url", "WebhookURL"},
	}
	for _, tt := range tests {
		if got := toGoName(tt.in); got != tt.want {
			t.Errorf("toGoName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := goIdent("telegram_id"); got != "telegramID" {
		t.Errorf("goIdent(telegram_id) = %q", got)
	}
}
