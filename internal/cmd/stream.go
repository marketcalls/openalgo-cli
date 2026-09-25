package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/marketcalls/openalgo-cli/internal/client"
	"github.com/marketcalls/openalgo-cli/internal/cmdutil"
	"github.com/marketcalls/openalgo-cli/internal/output"
	"github.com/marketcalls/openalgo-cli/internal/stream"
	"github.com/marketcalls/openalgo-cli/internal/useragent"
	"github.com/spf13/cobra"
)

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Stream live market data and order updates over WebSocket",
	Long: `Stream live market data (LTP, quote, depth) or account order updates from the
OpenAlgo WebSocket server as NDJSON: one JSON object per line on stdout.

The stream runs until Ctrl-C, --count data messages, or --duration elapses,
then unsubscribes and closes. --jq is applied to each message.

The WebSocket URL is derived from the host (http://h:5000 -> ws://h:8765,
http://h:5001 -> ws://h:8766 for a second instance,
https://domain -> wss://domain/ws). To override it, set ws_url in the profile
yaml, or OPENALGO_WS_URL when using OPENALGO_API_KEY (the env variable is
ignored with profile credentials, so it can never redirect a stored key).`,
	GroupID: "util",
}

var streamLTPCmd = &cobra.Command{
	Use:   "ltp",
	Short: "Stream last traded price ticks",
	Example: `  openalgo stream ltp --symbols NSE:RELIANCE,NSE:SBIN
  openalgo stream ltp --symbol RELIANCE --exchange NSE --count 5
  openalgo stream ltp --symbols NSE_INDEX:NIFTY --duration 30s --jq '{symbol, ltp: .data.ltp}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMarketStream(cmd, stream.ModeLTP)
	},
}

var streamQuoteCmd = &cobra.Command{
	Use:   "quote",
	Short: "Stream quote ticks (OHLC, LTP, volume, average price)",
	Example: `  openalgo stream quote --symbols NSE:RELIANCE,NSE:INFY
  openalgo stream quote --symbols NFO:NIFTY27OCT2626000CE --count 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMarketStream(cmd, stream.ModeQuote)
	},
}

var streamDepthCmd = &cobra.Command{
	Use:   "depth",
	Short: "Stream market depth (order book) ticks",
	Example: `  openalgo stream depth --symbols NSE:RELIANCE
  openalgo stream depth --symbols NSE:SBIN --depth 20 --count 1 --jq '.data.depth.buy[0]'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMarketStream(cmd, stream.ModeDepth)
	},
}

var streamOrdersCmd = &cobra.Command{
	Use:   "orders",
	Short: "Stream order status updates for the account",
	Long: `Stream order status updates (open, complete, rejected, cancelled) for every
order in the account, including sandbox fills when analyzer mode is on.`,
	Example: `  openalgo stream orders
  openalgo stream orders --count 1 --jq '{orderid, order_status}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStream(cmd, stream.Options{Orders: true})
	},
}

func init() {
	for _, c := range []*cobra.Command{streamLTPCmd, streamQuoteCmd, streamDepthCmd} {
		c.Flags().String("symbols", "", "comma-separated EXCHANGE:SYMBOL list, e.g. NSE:RELIANCE,NSE:SBIN")
		c.Flags().String("symbol", "", "single trading symbol (use with --exchange)")
		c.Flags().String("exchange", "", "exchange for --symbol, or for bare symbols in --symbols (NSE, BSE, NFO, MCX, NSE_INDEX, ...)")
		addStreamFlags(c)
		streamCmd.AddCommand(c)
	}
	streamDepthCmd.Flags().Int("depth", 5, "order book levels to request (5, 20, 30 or 50; broker dependent)")
	addStreamFlags(streamOrdersCmd)
	streamCmd.AddCommand(streamOrdersCmd)
	rootCmd.AddCommand(streamCmd)
}

// streamSchemas describes one NDJSON message per stream command for
// --schema. Market data fields vary by broker, so the data members list the
// common ones.
var streamSchemas = map[string][]api.ResponseField{
	"ltp":   marketDataSchema(api.ResponseField{Name: "ltp", Type: "number", Description: "last traded price"}),
	"quote": marketDataSchema(quoteFields()...),
	"depth": marketDataSchema(
		api.ResponseField{Name: "ltp", Type: "number", Description: "last traded price"},
		api.ResponseField{Name: "depth", Type: "object", Description: "order book levels", Fields: []api.ResponseField{
			{Name: "buy", Type: "[]object", Description: "bid levels, best first", Fields: depthLevel()},
			{Name: "sell", Type: "[]object", Description: "ask levels, best first", Fields: depthLevel()},
		}},
	),
	"orders": {
		{Name: "type", Type: "string", Description: "always order_update", EnumValues: []string{"order_update"}},
		{Name: "user_id", Type: "string", Description: "OpenAlgo user the order belongs to"},
		{Name: "mode", Type: "string", Description: "live (broker) or analyze (sandbox)"},
		{Name: "broker", Type: "string", Description: "broker name; \"sandbox\" in analyzer mode"},
		{Name: "orderid", Type: "string", Description: "order id"},
		{Name: "symbol", Type: "string", Description: "trading symbol in OpenAlgo format"},
		{Name: "exchange", Type: "string", Description: "exchange code"},
		{Name: "action", Type: "string", Description: "BUY or SELL"},
		{Name: "quantity", Type: "number", Description: "order quantity"},
		{Name: "price", Type: "number", Description: "order price"},
		{Name: "trigger_price", Type: "number", Description: "trigger price"},
		{Name: "pricetype", Type: "string", Description: "MARKET, LIMIT, SL or SL-M"},
		{Name: "product", Type: "string", Description: "CNC, NRML or MIS"},
		{Name: "order_status", Type: "string", Description: "open, trigger pending, complete, rejected or cancelled"},
		{Name: "filled_quantity", Type: "number", Description: "quantity filled so far"},
		{Name: "pending_quantity", Type: "number", Description: "quantity still open"},
		{Name: "average_price", Type: "number", Description: "average fill price"},
		{Name: "rejection_reason", Type: "string", Description: "broker rejection text, empty unless rejected"},
	},
}

func marketDataSchema(data ...api.ResponseField) []api.ResponseField {
	data = append(data, api.ResponseField{Name: "timestamp", Type: "number", Description: "tick time in epoch milliseconds"})
	return []api.ResponseField{
		{Name: "type", Type: "string", Description: "always market_data", EnumValues: []string{"market_data"}},
		{Name: "symbol", Type: "string", Description: "trading symbol"},
		{Name: "exchange", Type: "string", Description: "exchange code"},
		{Name: "mode", Type: "integer", Description: "1 LTP, 2 quote, 3 depth"},
		{Name: "broker", Type: "string", Description: "broker name"},
		{Name: "data", Type: "object", Description: "tick fields; the exact set depends on the broker", Fields: data},
	}
}

func quoteFields() []api.ResponseField {
	var out []api.ResponseField
	for _, f := range [][2]string{
		{"ltp", "last traded price"},
		{"open", "day open"},
		{"high", "day high"},
		{"low", "day low"},
		{"close", "previous close"},
		{"volume", "traded volume"},
		{"average_price", "average traded price"},
		{"last_quantity", "last traded quantity"},
		{"total_buy_quantity", "total bid quantity"},
		{"total_sell_quantity", "total ask quantity"},
	} {
		out = append(out, api.ResponseField{Name: f[0], Type: "number", Description: f[1]})
	}
	return out
}

func depthLevel() []api.ResponseField {
	return []api.ResponseField{
		{Name: "price", Type: "number", Description: "price level"},
		{Name: "quantity", Type: "number", Description: "quantity at this level"},
		{Name: "orders", Type: "number", Description: "order count at this level"},
	}
}

func addStreamFlags(c *cobra.Command) {
	c.Flags().Int("count", 0, "exit after N data messages (0 = no limit)")
	c.Flags().Duration("duration", 0, "exit after this long, e.g. 30s or 5m (0 = until Ctrl-C)")
	c.Flags().Bool("raw", false, "also print control messages (auth, subscribe and unsubscribe acks)")
}

func runMarketStream(cmd *cobra.Command, mode string) error {
	syms, err := stream.ParseSymbols(cmdutil.Str(cmd, "symbols"), cmdutil.Str(cmd, "symbol"), cmdutil.Str(cmd, "exchange"))
	if err != nil {
		return err
	}
	opts := stream.Options{Mode: mode, Symbols: syms}
	if mode == stream.ModeDepth {
		opts.Depth = cmdutil.Int(cmd, "depth")
		if opts.Depth <= 0 {
			return errors.New("--depth must be a positive number of levels (e.g. 5)")
		}
	}
	return runStream(cmd, opts)
}

// runStream fills in the connection settings shared by every stream
// subcommand and pumps messages to stdout until the session ends.
func runStream(cmd *cobra.Command, opts stream.Options) error {
	if cmd.Flags().Changed("csv") {
		return errors.New("--csv is not supported for stream commands; output is NDJSON (use --jq to reshape)")
	}
	opts.Count = cmdutil.Int(cmd, "count")
	if opts.Count < 0 {
		return errors.New("--count must be zero or positive")
	}
	opts.Duration, _ = cmd.Flags().GetDuration("duration")
	if opts.Duration < 0 {
		return errors.New("--duration must be zero or positive")
	}
	opts.Raw = cmdutil.Bool(cmd, "raw")
	if t, err := cmd.Flags().GetInt("timeout"); err == nil && t > 0 {
		opts.AckTimeout = time.Duration(t) * time.Second
	}

	// Fail on a malformed --jq before connecting, not on the first tick.
	if jqFlag != "" {
		if _, err := output.ApplyJQ(map[string]any{}, jqFlag); err != nil {
			return err
		}
	}

	// cfg.WSURL already applies OPENALGO_WS_URL or the profile's ws_url,
	// falling back to the URL derived from the host of the same bundle.
	if cfg != nil {
		opts.APIKey = cfg.APIKey
		opts.URL = cfg.WSURL
	}
	if opts.APIKey == "" {
		return &client.APIError{StatusCode: 401, Status: "error", Message: "no API key configured"}
	}
	opts.UserAgent = useragent.Build(client.Version)

	stderr := cmd.ErrOrStderr()
	opts.Warn = func(msg string) {
		if !quietFlag {
			fmt.Fprintln(stderr, "warning: "+msg)
		}
	}
	if verboseFlag || debugFlag {
		opts.Verbose = func(msg string) { fmt.Fprintln(stderr, msg) }
	}
	if debugFlag {
		key := opts.APIKey
		opts.Frame = func(dir string, data []byte) {
			fmt.Fprintf(stderr, "%s %s\n", dir, strings.ReplaceAll(string(data), key, "[REDACTED]"))
		}
	}

	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	out := cmd.OutOrStdout()
	return stream.Run(ctx, opts, func(msg json.RawMessage) error {
		return writeStreamMessage(out, msg, jqFlag)
	})
}

// writeStreamMessage prints one message as a single compact JSON line,
// after applying the --jq expression when one is set. A jq filter that
// yields nothing (e.g. select) prints nothing for that message.
func writeStreamMessage(w io.Writer, msg json.RawMessage, jq string) error {
	var line []byte
	if jq == "" {
		var buf bytes.Buffer
		if err := json.Compact(&buf, msg); err != nil {
			return fmt.Errorf("invalid JSON from server: %w", err)
		}
		line = buf.Bytes()
	} else {
		v, err := output.ApplyJQ(msg, jq)
		if err != nil {
			return err
		}
		if s, ok := v.([]any); ok && s == nil {
			return nil
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return err
		}
		line = bytes.TrimRight(buf.Bytes(), "\n")
	}
	_, err := fmt.Fprintf(w, "%s\n", line)
	return err
}
