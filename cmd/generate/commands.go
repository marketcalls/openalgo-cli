package main

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
)

type cmdDef struct {
	parent      string
	use         string
	self        bool // the parent group itself runs this op (attachCmd)
	examples    string
	long        string
	defaults    map[string]string // OAS body field -> CLI default, emitted with CLIDefault: true
	flagAliases map[string]string // kebab flag name -> CLI alias (resolves collisions with global flags)
	objectFlag  string            // flag name for a free-form object body merged at the body root
	// rowsPath overrides the derived --csv rows location for responses
	// whose records are not under "data" (e.g. "results", "chain").
	rowsPath string
}

type parentDef struct {
	use    string
	short  string
	long   string
	parent string
}

// strategyDefault tags orders placed or queried by the CLI so they are
// distinguishable from other clients in the OpenAlgo order log.
var strategyDefault = map[string]string{"strategy": "openalgo-cli"}

// orderNotes is appended to the help of order-placing commands: product and
// pricetype defaults come from the server, and F&O quantities must be lot
// multiples, which change over time.
const orderNotes = " When --product or --pricetype is omitted the server uses MIS and MARKET; in analyzer (sandbox) mode MIS is refused after the 15:15 IST square-off, so prefer CNC for equity and NRML for F&O. F&O quantities must be a multiple of the lot size shown by 'openalgo symbol get --symbol <symbol> --exchange NFO --jq .data.lotsize'."

// reservedFlags are persistent/global flag names (or flags fetchCmd adds)
// that a generated flag must not reuse. Collisions need a flagAliases entry.
var reservedFlags = map[string]bool{
	"csv": true, "jq": true, "profile": true, "verbose": true, "debug": true,
	"trace": true, "quiet": true, "timeout": true, "schema": true, "help": true,
	"dry-run": true, "version": true, "output": true, "help-all": true,
}

var cmdParents = map[string]parentDef{
	"order": {
		use: "order", short: "Place, modify, cancel, and list orders",
		long: "Place regular, smart (position-sized), basket, and split orders; modify or cancel open orders; view the order book, trade book, and order status. Orders carry --strategy openalgo-cli unless overridden.",
	},
	"gtt": {
		use: "gtt", short: "Manage GTT (Good Till Triggered) orders",
		long: "Place, modify, cancel, and list GTT triggers. SINGLE fires one child order when LTP crosses the trigger; OCO pairs a stoploss leg with a target leg.",
	},
	"position": {
		use: "position", short: "View and close positions",
		long: "List the day's positions, look up the net open quantity for one symbol, or square off every open position.",
	},
	"option": {
		use: "option", short: "Options orders, symbols, chains, and Greeks",
		long: "Place options orders by strike offset (ATM, ITMn, OTMn), resolve option symbols, fetch option chains and synthetic futures, and calculate Black-76 Greeks.",
	},
	"strategy": {
		use: "strategy", short: "Manage OpenAlgo strategies and their runs",
		long: "List strategies, inspect configuration and runs, start or stop runs, close legs, and read the order and risk-event audit trail.",
	},
	"account": {
		use: "account", short: "Account funds, holdings, and margin",
		long: "View available funds and collateral, delivery holdings with P&L, and calculate margin for a basket of positions.",
	},
	"data": {
		use: "data", short: "Market data: quotes, depth, and history",
		long: "Real-time quotes and market depth, historical candles, and the intervals supported by the connected broker. For continuous updates use 'openalgo stream'.",
	},
	"symbol": {
		use: "symbol", short: "Look up symbols, expiries, and instruments",
		long: "Resolve symbol details from the master contract, search symbols, list expiry dates, and download the instrument master.",
	},
	"market": {
		use: "market", short: "Market holidays and trading sessions",
		long: "List exchange holidays for a year and trading session times for a date.",
	},
	"analyzer": {
		use: "analyzer", short: "Analyzer (sandbox) mode",
		long: "Check or toggle analyzer mode, in which orders are routed to the sandbox instead of the live broker, and view sandbox P&L by symbol.",
	},
	"chart": {
		use: "chart", short: "Chart workspace preferences",
		long: "Read and update the chart workspace preferences stored for the API key.",
	},
	"telegram": {
		use: "telegram", short: "Telegram bot and notifications",
		long: "Configure, start, and stop the Telegram bot; list linked users; send notifications and broadcasts; manage per-user notification preferences.",
	},
	"telegramConfig":      {use: "config", short: "Telegram bot configuration", parent: "telegram"},
	"telegramPreferences": {use: "preferences", short: "Telegram user notification preferences", parent: "telegram"},
	"whatsapp": {
		use: "whatsapp", short: "WhatsApp notifications",
		long: "Send text, images, or documents through the paired WhatsApp device.",
	},
	"ping": {
		use: "ping", short: "Check that the API key resolves to an active broker session",
	},
}

var cmdRegistry = map[string]cmdDef{
	// --- order ---
	"PlaceOrder": {
		parent:   "order",
		use:      "place",
		defaults: strategyDefault,
		long:     "Place a new order with the broker. In analyzer mode the order is routed to the sandbox and the response carries a mode field." + orderNotes,
		examples: `  openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --product CNC --pricetype MARKET
  openalgo order place --symbol RELIANCE --exchange NSE --action BUY --quantity 5 --product CNC --pricetype LIMIT --price 2800
  openalgo order place --symbol NIFTY27OCT26FUT --exchange NFO --action SELL --quantity 65 --product NRML --pricetype SL --price 24990 --trigger-price 25000
  openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --product CNC --dry-run`,
	},
	"PlaceSmartOrder": {
		parent:   "order",
		use:      "smart",
		defaults: strategyDefault,
		long:     "Place an order sized to move the current open position for the symbol to --position-size (positive long, negative short, 0 flat). No order is placed if the position already matches." + orderNotes,
		examples: `  openalgo order smart --symbol SBIN --exchange NSE --action BUY --quantity 1 --position-size 5 --product CNC
  openalgo order smart --symbol SBIN --exchange NSE --action SELL --quantity 1 --position-size 0 --product CNC`,
	},
	"BasketOrder": {
		parent:   "order",
		use:      "basket",
		defaults: strategyDefault,
		rowsPath: "results",
		examples: `  openalgo order basket --orders '[{"symbol":"SBIN","exchange":"NSE","action":"BUY","quantity":1,"pricetype":"MARKET","product":"CNC"},{"symbol":"INFY","exchange":"NSE","action":"BUY","quantity":1,"pricetype":"MARKET","product":"CNC"}]'
  openalgo order basket --orders @basket.json`,
	},
	"SplitOrder": {
		parent:   "order",
		use:      "split",
		defaults: strategyDefault,
		rowsPath: "results",
		long:     "Place a large quantity as sequential child orders of --splitsize each, with the remainder in the last order. At most 100 child orders are allowed per request." + orderNotes,
		examples: `  openalgo order split --symbol YESBANK --exchange NSE --action BUY --quantity 105 --splitsize 20 --product CNC
  openalgo order split --symbol NIFTY27OCT26FUT --exchange NFO --action BUY --quantity 650 --splitsize 130 --product NRML`,
	},
	"ModifyOrder": {
		parent: "order",
		use:    "modify",
		// The server requires disclosed_quantity and trigger_price even for
		// a plain LIMIT change; send 0 unless the user sets them.
		defaults: map[string]string{"strategy": "openalgo-cli", "disclosed_quantity": "0", "trigger_price": "0"},
		examples: `  openalgo order modify --orderid 250408000989443 --symbol SBIN --exchange NSE --action BUY --product CNC --pricetype LIMIT --price 780 --quantity 1
  openalgo order modify --orderid 250408000989443 --symbol SBIN --exchange NSE --action BUY --product CNC --pricetype SL --price 781 --trigger-price 780 --quantity 1`,
	},
	"CancelOrder": {
		parent:   "order",
		use:      "cancel",
		defaults: strategyDefault,
		examples: `  openalgo order cancel --orderid 250408000989443`,
	},
	"CancelAllOrder": {
		parent:   "order",
		use:      "cancel-all",
		defaults: strategyDefault,
		long:     "Cancels EVERY open and trigger-pending order in the account, not only orders placed by the CLI. There is no confirmation step; use --dry-run to preview the request. Returns success even if some cancellations fail; inspect failed_cancellations.",
		examples: `  openalgo order cancel-all
  openalgo order cancel-all --dry-run`,
	},
	"GetOrderStatus": {
		parent:   "order",
		use:      "status",
		defaults: strategyDefault,
		examples: `  openalgo order status --orderid 250408000989443
  openalgo order status --orderid 250408000989443 --jq '.data.order_status'`,
	},
	"GetOrderbook": {
		parent: "order",
		use:    "list",
		examples: `  openalgo order list
  openalgo order list --csv
  openalgo order list --jq '.data.orders[] | select(.order_status == "open")'`,
	},
	"GetTradebook": {
		parent: "order",
		use:    "trades",
		examples: `  openalgo order trades
  openalgo order trades --csv`,
	},

	// --- gtt ---
	"PlaceGTTOrder": {
		parent:   "gtt",
		use:      "place",
		defaults: strategyDefault,
		examples: `  openalgo gtt place --trigger-type SINGLE --symbol SBIN --exchange NSE --action BUY --product CNC --quantity 1 --pricetype LIMIT --price 756 --triggerprice-sl 755
  openalgo gtt place --trigger-type OCO --symbol SBIN --exchange NSE --action SELL --product CNC --quantity 1 --price 0 --triggerprice-sl 740 --stoploss 739 --triggerprice-tg 860 --target 861`,
	},
	"ModifyGTTOrder": {
		parent:   "gtt",
		use:      "modify",
		defaults: strategyDefault,
		long:     "Modify an active GTT trigger. Trigger ids are strings: numeric for live brokers, GTT-<date>-<hex> in analyzer mode.",
		examples: `  openalgo gtt modify --trigger-id 123456789 --trigger-type SINGLE --symbol SBIN --exchange NSE --action BUY --product CNC --quantity 2 --price 751 --triggerprice-sl 750
  openalgo gtt modify --trigger-id GTT-260926-0e55ab27 --trigger-type SINGLE --symbol SBIN --exchange NSE --action BUY --product CNC --quantity 1 --price 752 --triggerprice-sl 751`,
	},
	"CancelGTTOrder": {
		parent:   "gtt",
		use:      "cancel",
		defaults: strategyDefault,
		long:     "Cancel an active GTT trigger. Trigger ids are strings: numeric for live brokers, GTT-<date>-<hex> in analyzer mode.",
		examples: `  openalgo gtt cancel --trigger-id 123456789
  openalgo gtt cancel --trigger-id GTT-260926-0e55ab27`,
	},
	"GetGTTOrderbook": {
		parent: "gtt",
		use:    "list",
		examples: `  openalgo gtt list
  openalgo gtt list --status all --csv`,
	},

	// --- position ---
	"GetPositionbook": {
		parent: "position",
		use:    "list",
		examples: `  openalgo position list
  openalgo position list --csv
  openalgo position list --jq '.data[] | select((.quantity | tonumber) != 0)'`,
	},
	"GetOpenPosition": {
		parent:   "position",
		use:      "get",
		defaults: strategyDefault,
		examples: `  openalgo position get --symbol SBIN --exchange NSE --product CNC
  openalgo position get --symbol NIFTY27OCT26FUT --exchange NFO --product NRML --jq '.quantity'`,
	},
	"ClosePosition": {
		parent:   "position",
		use:      "close-all",
		defaults: strategyDefault,
		long:     "Squares off ALL open positions across every exchange with MARKET orders, not only positions opened by the CLI. There is no confirmation step; use --dry-run to preview the request.",
		examples: `  openalgo position close-all
  openalgo position close-all --dry-run`,
	},

	// --- account ---
	"GetFunds": {
		parent:   "account",
		use:      "funds",
		examples: `  openalgo account funds`,
	},
	"GetHoldings": {
		parent: "account",
		use:    "holdings",
		examples: `  openalgo account holdings
  openalgo account holdings --csv`,
	},
	"GetMargin": {
		parent: "account",
		use:    "margin",
		examples: `  openalgo account margin --positions '[{"symbol":"NIFTY27OCT26FUT","exchange":"NFO","action":"BUY","quantity":"65","product":"NRML","pricetype":"MARKET"}]'
  openalgo account margin --positions @positions.json`,
	},

	// --- data ---
	"GetQuotes": {
		parent: "data",
		use:    "quote",
		examples: `  openalgo data quote --symbol RELIANCE --exchange NSE
  openalgo data quote --symbol NIFTY --exchange NSE_INDEX --jq '.data.ltp'`,
	},
	"GetMultiQuotes": {
		parent:   "data",
		use:      "quotes",
		rowsPath: "results",
		examples: `  openalgo data quotes --symbols '[{"symbol":"RELIANCE","exchange":"NSE"},{"symbol":"INFY","exchange":"NSE"},{"symbol":"NIFTY","exchange":"NSE_INDEX"}]'`,
	},
	"GetDepth": {
		parent:   "data",
		use:      "depth",
		rowsPath: "data",
		examples: `  openalgo data depth --symbol SBIN --exchange NSE`,
	},
	"GetHistory": {
		parent: "data",
		use:    "history",
		examples: `  openalgo data history --symbol SBIN --exchange NSE --interval D --start-date 2026-09-01 --end-date 2026-09-25
  openalgo data history --symbol NIFTY --exchange NSE_INDEX --interval 5m --start-date 2026-09-24 --end-date 2026-09-25 --csv`,
	},
	"GetIntervals": {
		parent:   "data",
		use:      "intervals",
		examples: `  openalgo data intervals`,
	},
	"GetTicker": {
		parent: "data",
		use:    "ticker",
		long:   "Returns broker historical candles for an EXCHANGE:SYMBOL pair in JSON or, with --format txt, comma-separated plain text written verbatim to stdout. The range is capped to 30 days for intraday intervals and 10 years for D, W and M.",
		examples: `  openalgo data ticker --symbol NSE:RELIANCE --from 2026-09-01 --to 2026-09-25
  openalgo data ticker --symbol NSE:SBIN --interval 5m --from 2026-09-24 --to 2026-09-25 --format txt`,
	},

	// --- symbol ---
	"GetSymbol": {
		parent: "symbol",
		use:    "get",
		examples: `  openalgo symbol get --symbol RELIANCE --exchange NSE
  openalgo symbol get --symbol NIFTY27OCT26FUT --exchange NFO`,
	},
	"SearchSymbols": {
		parent: "symbol",
		use:    "search",
		examples: `  openalgo symbol search --query RELIANCE
  openalgo symbol search --query "NIFTY 26000 OCT CE" --exchange NFO`,
	},
	"GetExpiry": {
		parent: "symbol",
		use:    "expiry",
		examples: `  openalgo symbol expiry --symbol NIFTY --exchange NFO --instrumenttype options
  openalgo symbol expiry --symbol CRUDEOIL --exchange MCX --instrumenttype futures`,
	},
	"GetInstruments": {
		parent: "symbol",
		use:    "instruments",
		long:   "Download the locally stored instrument master for one exchange. With --format csv the CSV file is written verbatim to stdout. The API documents --exchange as optional (all exchanges), but current OpenAlgo servers reject a request without it, so always pass --exchange.",
		examples: `  openalgo symbol instruments --exchange NSE
  openalgo symbol instruments --exchange NFO --format csv > nfo.csv`,
	},

	// --- option ---
	"OptionsOrder": {
		parent:   "option",
		use:      "order",
		defaults: strategyDefault,
		long:     "Resolve an option symbol from the underlying, expiry and ATM/ITM/OTM offset, then place the order. Index underlyings are placed on NFO or BFO; --splitsize greater than 0 returns per-child results. --quantity must be a multiple of the lot size shown by 'openalgo option symbol'. --expiry-date takes DDMMMYY (27OCT26); DD-MMM-YY from 'openalgo symbol expiry' is converted.",
		examples: `  openalgo option order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ATM --option-type CE --action BUY --quantity 65 --product NRML
  openalgo option order --underlying BANKNIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset OTM2 --option-type PE --action SELL --quantity 30 --product NRML --pricetype LIMIT --price 120`,
	},
	"OptionsMultiOrder": {
		parent:   "option",
		use:      "multi-order",
		defaults: strategyDefault,
		long:     "Place 1 to 20 option legs sharing a common underlying, each resolved from its strike offset. BUY legs execute before SELL legs and a failed leg does not stop later legs. Leg quantities must be a multiple of the lot size shown by 'openalgo option symbol'.",
		examples: `  openalgo option multi-order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --legs '[{"offset":"OTM2","option_type":"CE","action":"BUY","quantity":65,"product":"NRML"},{"offset":"OTM2","option_type":"PE","action":"BUY","quantity":65,"product":"NRML"},{"offset":"ATM","option_type":"CE","action":"SELL","quantity":65,"product":"NRML"},{"offset":"ATM","option_type":"PE","action":"SELL","quantity":65,"product":"NRML"}]'
  openalgo option multi-order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --legs @legs.json`,
	},
	"GetOptionSymbol": {
		parent: "option",
		use:    "symbol",
		examples: `  openalgo option symbol --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ATM --option-type CE
  openalgo option symbol --underlying BANKNIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ITM3 --option-type PE`,
	},
	"GetOptionChain": {
		parent:   "option",
		use:      "chain",
		rowsPath: "chain",
		examples: `  openalgo option chain --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --strike-count 10
  openalgo option chain --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --strike-count 5 --with-greeks`,
	},
	"GetSyntheticFuture": {
		parent:   "option",
		use:      "synthetic-future",
		examples: `  openalgo option synthetic-future --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26`,
	},
	"GetOptionGreeks": {
		parent: "option",
		use:    "greeks",
		examples: `  openalgo option greeks --symbol NIFTY27OCT2626000CE --exchange NFO
  openalgo option greeks --symbol NIFTY27OCT2626000PE --exchange NFO --interest-rate 6.5`,
	},
	"GetMultiOptionGreeks": {
		parent:   "option",
		use:      "multi-greeks",
		examples: `  openalgo option multi-greeks --symbols '[{"symbol":"NIFTY27OCT2626000CE","exchange":"NFO"},{"symbol":"NIFTY27OCT2626000PE","exchange":"NFO"}]'`,
	},

	// --- market ---
	"GetMarketHolidays": {
		parent: "market",
		use:    "holidays",
		examples: `  openalgo market holidays
  openalgo market holidays --year 2026 --csv`,
	},
	"GetMarketTimings": {
		parent:   "market",
		use:      "timings",
		examples: `  openalgo market timings --date 2026-09-25`,
	},

	// --- analyzer ---
	"GetAnalyzerStatus": {
		parent:   "analyzer",
		use:      "status",
		examples: `  openalgo analyzer status`,
	},
	"ToggleAnalyzer": {
		parent:   "analyzer",
		use:      "toggle",
		long:     "Switch analyzer (sandbox) mode on or off application-wide. --mode=false switches to live trading: subsequent orders from every client go to the live broker with real money.",
		examples: `  openalgo analyzer toggle --mode=true`,
	},
	"GetPnlSymbols": {
		parent: "analyzer",
		use:    "pnl",
		examples: `  openalgo analyzer pnl
  openalgo analyzer pnl --csv`,
	},

	// --- chart ---
	"GetChartPreferences": {
		parent:   "chart",
		use:      "get",
		examples: `  openalgo chart get`,
	},
	"UpdateChartPreferences": {
		parent:     "chart",
		use:        "set",
		objectFlag: "preferences",
		long:       "Upsert chart preference keys for the API key. --preferences is a JSON object (literal, @file, or - for stdin) of 1 to 50 keys; each key is up to 50 characters and each value up to 1 MiB.",
		examples: `  openalgo chart set --preferences '{"theme":"dark"}'
  openalgo chart set --preferences @chart-prefs.json`,
	},

	// --- ping ---
	"Ping": {
		parent:   "ping",
		self:     true,
		examples: `  openalgo ping`,
	},

	// --- strategy ---
	"StrategyList": {
		parent:      "strategy",
		use:         "list",
		flagAliases: map[string]string{"q": "query"},
		examples: `  openalgo strategy list
  openalgo strategy list --status running
  openalgo strategy list --query straddle --csv`,
	},
	"StrategyStatus": {
		parent:   "strategy",
		use:      "status",
		examples: `  openalgo strategy status --strategy-id 12`,
	},
	"StrategyStart": {
		parent:   "strategy",
		use:      "start",
		long:     "Start a run of a batch strategy and place every leg's entry order. --mode is required: sandbox routes orders to the sandbox, live places real orders with the broker and is refused unless live trading is enabled on the strategy.",
		examples: `  openalgo strategy start --strategy-id 12 --mode sandbox`,
	},
	"StrategyStop": {
		parent:   "strategy",
		use:      "stop",
		long:     "Stop the current run: exits every position the run owns at market and finalizes once fills confirm it is flat.",
		examples: `  openalgo strategy stop --strategy-id 12`,
	},
	"StrategyCloseAll": {
		parent:   "strategy",
		use:      "close-all",
		long:     "Exits ALL positions owned by the strategy's current run at market and stops the run. Positions outside the strategy are not touched.",
		examples: `  openalgo strategy close-all --strategy-id 12`,
	},
	"StrategyCloseLeg": {
		parent:   "strategy",
		use:      "close-leg",
		examples: `  openalgo strategy close-leg --strategy-id 12 --leg-id 2`,
	},
	"StrategyRuns": {
		parent: "strategy",
		use:    "runs",
		examples: `  openalgo strategy runs --strategy-id 12
  openalgo strategy runs --strategy-id 12 --limit 10 --csv`,
	},
	"StrategyOrders": {
		parent: "strategy",
		use:    "orders",
		examples: `  openalgo strategy orders --strategy-id 12
  openalgo strategy orders --strategy-id 12 --run-id 40 --csv`,
	},
	"StrategyEvents": {
		parent: "strategy",
		use:    "events",
		examples: `  openalgo strategy events --strategy-id 12
  openalgo strategy events --strategy-id 12 --severity critical --limit 50`,
	},

	// --- telegram ---
	"TelegramGetConfig": {
		parent:   "telegramConfig",
		use:      "get",
		examples: `  openalgo telegram config get`,
	},
	"TelegramSetConfig": {
		parent: "telegramConfig",
		use:    "set",
		examples: `  openalgo telegram config set --token <bot-token> --polling-mode=true
  openalgo telegram config set --broadcast-enabled=false --rate-limit-per-minute 30`,
	},
	"TelegramStart": {
		parent:   "telegram",
		use:      "start",
		examples: `  openalgo telegram start`,
	},
	"TelegramStop": {
		parent:   "telegram",
		use:      "stop",
		examples: `  openalgo telegram stop`,
	},
	"TelegramUsers": {
		parent: "telegram",
		use:    "users",
		examples: `  openalgo telegram users
  openalgo telegram users --broker zerodha --notifications-enabled true --csv`,
	},
	"TelegramBroadcast": {
		parent:   "telegram",
		use:      "broadcast",
		examples: `  openalgo telegram broadcast --message "Markets open in 15 minutes"`,
	},
	"TelegramNotify": {
		parent: "telegram",
		use:    "notify",
		examples: `  openalgo telegram notify --username admin --message "SBIN order filled"
  openalgo telegram notify --username admin --message "Stoploss hit" --priority 9 --wait-for-delivery=true`,
	},
	"TelegramStats": {
		parent: "telegram",
		use:    "stats",
		examples: `  openalgo telegram stats
  openalgo telegram stats --days 30`,
	},
	"TelegramGetPreferences": {
		parent:   "telegramPreferences",
		use:      "get",
		examples: `  openalgo telegram preferences get --telegram-id 123456789`,
	},
	"TelegramSetPreferences": {
		parent: "telegramPreferences",
		use:    "set",
		examples: `  openalgo telegram preferences set --telegram-id 123456789 --daily-summary=true --summary-time 15:45
  openalgo telegram preferences set --telegram-id 123456789 --trade-notifications=false`,
	},

	// --- whatsapp ---
	"WhatsAppNotify": {
		parent: "whatsapp",
		use:    "notify",
		long:   "Send text, an image or a document to exactly one recipient form: --self, --username, --phone, or --phones (up to 5). Returns 409 when WhatsApp is not paired or connected.",
		examples: `  openalgo whatsapp notify --self=true --message "SBIN order placed"
  openalgo whatsapp notify --phone 919876543210 --message "Daily P&L report ready"
  openalgo whatsapp notify --phones '["919876543210","919812345678"]' --message "Markets open in 15 minutes"`,
	},
}

// cmdSkip lists operations intentionally not exposed as generated commands,
// with the reason. Every spec operation must be in cmdRegistry or cmdSkip.
var cmdSkip = map[string]string{}

// checkExhaustive validates the registry against the spec and the derived
// flags, failing loudly so drift is caught at generation time.
func checkExhaustive(epByOp map[string]*endpointInfo, ops []*opDesc) {
	if errs := registryErrors(epByOp, ops); len(errs) > 0 {
		log.Fatalf("%d error(s):\n  - %s", len(errs), strings.Join(errs, "\n  - "))
	}
}

// registryErrors returns every registry/spec inconsistency found.
func registryErrors(epByOp map[string]*endpointInfo, ops []*opDesc) []string {
	var errs []string

	for _, goName := range sortedKeys(epByOp) {
		ep := epByOp[goName]
		_, inRegistry := cmdRegistry[goName]
		_, inSkip := cmdSkip[goName]
		if !inRegistry && !inSkip {
			errs = append(errs, fmt.Sprintf("unregistered operation %q - add to cmdRegistry or cmdSkip in cmd/generate/commands.go", ep.operationID))
		}
		if inRegistry && inSkip {
			errs = append(errs, fmt.Sprintf("operation %q is in both cmdRegistry and cmdSkip", ep.operationID))
		}
	}
	for _, name := range sortedKeys(cmdSkip) {
		if epByOp[name] == nil {
			errs = append(errs, fmt.Sprintf("cmdSkip[%q] references an unknown operation", name))
		}
	}

	for _, key := range sortedKeys(cmdParents) {
		if p := cmdParents[key].parent; p != "" {
			if _, ok := cmdParents[p]; !ok {
				errs = append(errs, fmt.Sprintf("cmdParents[%q] has unknown parent %q", key, p))
			}
		}
	}

	opByName := make(map[string]*opDesc, len(ops))
	for _, op := range ops {
		opByName[op.goName] = op
	}

	usesByParent := map[string]string{}
	for _, opID := range sortedKeys(cmdRegistry) {
		def := cmdRegistry[opID]
		ep := epByOp[opID]
		if ep == nil {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q] references an unknown operation", opID))
			continue
		}
		if _, ok := cmdParents[def.parent]; !ok {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q] has unknown parent %q", opID, def.parent))
			continue
		}
		if def.self == (def.use != "") {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q] must set exactly one of use or self", opID))
		}
		slot := def.parent + " " + def.use
		if prev, dup := usesByParent[slot]; dup {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q] and [%q] both register %q", prev, opID, commandPath(def)))
		}
		usesByParent[slot] = opID

		op := opByName[opID]
		errs = append(errs, checkFlags(opID, def, ep, op)...)

		if def.examples == "" {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q] has empty examples - every generated command must have examples", opID))
		} else {
			errs = append(errs, checkExamples(opID, def, op)...)
		}
	}

	return errs
}

// checkFlags verifies aliases, CLI defaults, and flag-name uniqueness.
func checkFlags(opID string, def cmdDef, ep *endpointInfo, op *opDesc) []string {
	var errs []string
	byOAS := map[string]*flagDesc{}
	seen := map[string]bool{}
	for _, f := range op.flags {
		byOAS[f.oasName] = f
		if seen[f.flagName] {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: duplicate flag --%s - add a flagAliases entry", opID, f.flagName))
		}
		seen[f.flagName] = true
		if reservedFlags[f.flagName] {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: flag --%s collides with a global flag - add a flagAliases entry", opID, f.flagName))
		}
	}
	for _, from := range sortedKeys(def.flagAliases) {
		found := false
		for _, f := range op.flags {
			if kebab(f.oasName) == from {
				found = true
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: flagAliases key %q matches no flag", opID, from))
		}
	}
	for _, name := range sortedKeys(def.defaults) {
		if f := byOAS[name]; f == nil || f.source != "body" {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: defaults key %q is not a body field", opID, name))
		}
	}
	if def.objectFlag != "" && !ep.isFreeFormBody() {
		errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: objectFlag set but the body is not a free-form object", opID))
	}
	if ep.isFreeFormBody() && def.objectFlag == "" {
		errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: free-form object body needs an objectFlag name", opID))
	}

	// Order-management and order-information ops tag their requests with
	// the CLI's strategy name unless the user overrides it.
	if f := byOAS["strategy"]; f != nil && f.source == "body" &&
		(ep.hasTag("order-management") || ep.hasTag("order-information")) {
		if def.defaults["strategy"] != strategyDefault["strategy"] {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: strategy body field needs defaults: strategyDefault", opID))
		}
	}
	return errs
}

// globalFlags are flags valid on every generated command (persistent flags
// plus those added by fetchCmd), accepted when validating examples.
var globalFlags = map[string]bool{
	"csv": true, "jq": true, "profile": true, "verbose": true, "debug": true,
	"trace": true, "quiet": true, "timeout": true, "schema": true, "help": true,
	"dry-run": true,
}

var exampleFlagRe = regexp.MustCompile(`(?:^|\s)--([a-z0-9][a-z0-9-]*)`)

// checkExamples verifies each example invokes this command and only uses
// flags it defines. Quoted JSON is stripped first so keys inside it are not
// mistaken for flags.
func checkExamples(opID string, def cmdDef, op *opDesc) []string {
	var errs []string
	prefix := "openalgo " + commandPath(def)
	flags := map[string]bool{}
	for _, f := range op.flags {
		flags[f.flagName] = true
	}
	for _, line := range strings.Split(def.examples, "\n") {
		if !strings.HasPrefix(line, "  ") {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: example line must be indented two spaces: %q", opID, line))
		}
		line = strings.TrimSpace(line)
		if line != prefix && !strings.HasPrefix(line, prefix+" ") {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: example %q does not start with %q", opID, line, prefix))
			continue
		}
		for _, m := range exampleFlagRe.FindAllStringSubmatch(stripQuoted(line), -1) {
			if name := m[1]; !flags[name] && !globalFlags[name] {
				errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: example uses unknown flag --%s", opID, name))
			}
		}
		if !op.mutating && strings.Contains(line, "--dry-run") {
			errs = append(errs, fmt.Sprintf("cmdRegistry[%q]: --dry-run is only available on mutating commands", opID))
		}
	}
	return errs
}

// stripQuoted removes single- and double-quoted segments from a shell line.
func stripQuoted(s string) string {
	var b strings.Builder
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// commandPath returns the space-separated command path below the root.
func commandPath(def cmdDef) string {
	parts := parentPath(def.parent)
	if !def.self {
		parts = append(parts, def.use)
	}
	return strings.Join(parts, " ")
}

func parentPath(key string) []string {
	p, ok := cmdParents[key]
	if !ok {
		return []string{key}
	}
	if p.parent == "" {
		return []string{p.use}
	}
	return append(parentPath(p.parent), p.use)
}

func genCommands(epByOp map[string]*endpointInfo) string {
	var body bytes.Buffer

	for _, key := range sortedKeys(cmdParents) {
		pdef := cmdParents[key]
		fmt.Fprintf(&body, "var %sCmd = &cobra.Command{\n", key)
		fmt.Fprintf(&body, "\tUse:   %q,\n", pdef.use)
		fmt.Fprintf(&body, "\tShort: %q,\n", pdef.short)
		if pdef.long != "" {
			fmt.Fprintf(&body, "\tLong: %q,\n", pdef.long)
		}
		fmt.Fprintf(&body, "}\n\n")
	}

	opIDs := sortedKeys(cmdRegistry)

	var attachBuf bytes.Buffer
	for _, opID := range opIDs {
		emitCommand(&body, &attachBuf, opID, cmdRegistry[opID], epByOp[opID])
	}

	// Emit single init() for all wiring
	emitInit(&body, &attachBuf, opIDs)

	// Assemble final output with header
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by cmd/generate; DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package cmd\n\n")
	fmt.Fprintf(&buf, "import (\n")
	fmt.Fprintf(&buf, "\t\"github.com/marketcalls/openalgo-cli/internal/api\"\n")
	if strings.Contains(body.String(), "cmdutil.") {
		fmt.Fprintf(&buf, "\t\"github.com/marketcalls/openalgo-cli/internal/cmdutil\"\n")
	}
	fmt.Fprintf(&buf, "\t\"github.com/spf13/cobra\"\n")
	fmt.Fprintf(&buf, ")\n\n")
	buf.Write(body.Bytes())

	return buf.String()
}

func emitCommand(buf, attachBuf *bytes.Buffer, opID string, def cmdDef, ep *endpointInfo) {
	opVar := opID + "Op"
	fetchBody := buildFetchBody(ep)

	if def.self {
		fmt.Fprintf(attachBuf, "\tattachCmd(%sCmd, api.%s, func(cmd *cobra.Command, args []string) (any, error) {\n", def.parent, opVar)
		fmt.Fprintf(attachBuf, "%s", indent(fetchBody, "\t\t"))
		fmt.Fprintf(attachBuf, "\t})\n")
		return
	}
	fmt.Fprintf(buf, "var %s = fetchCmd(%q, api.%s, func(cmd *cobra.Command, args []string) (any, error) {\n", cmdVarName(opID, def), def.use, opVar)
	fmt.Fprintf(buf, "%s", indent(fetchBody, "\t"))
	fmt.Fprintf(buf, "})\n\n")
}

// buildFetchBody returns the statements of the fetch callback (unindented,
// one per line).
func buildFetchBody(ep *endpointInfo) string {
	opVar := "api." + ep.goName + "Op"
	var b strings.Builder
	var args []string
	for _, pp := range ep.pathParams {
		args = append(args, fmt.Sprintf("cmdutil.Str(cmd, %q)", kebab(pp.name)))
	}
	if len(ep.queryParams) > 0 {
		args = append(args, fmt.Sprintf("queryFromFlags(cmd, %s)", opVar))
	}
	if ep.hasBody {
		fmt.Fprintf(&b, "body, err := bodyFromFlags(cmd, %s)\n", opVar)
		b.WriteString("if err != nil {\n\treturn nil, err\n}\n")
		if ep.mutating {
			b.WriteString("if cmdutil.Bool(cmd, \"dry-run\") {\n\treturn body, nil\n}\n")
		}
		args = append(args, "body")
	} else if ep.mutating {
		b.WriteString("if cmdutil.Bool(cmd, \"dry-run\") {\n\treturn map[string]any{}, nil\n}\n")
	}
	fmt.Fprintf(&b, "return openalgoClient.%s(%s)\n", ep.goName, strings.Join(args, ", "))
	return b.String()
}

func indent(s, prefix string) string {
	lines := strings.SplitAfter(s, "\n")
	var b strings.Builder
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			b.WriteString(prefix)
		}
		b.WriteString(l)
	}
	return b.String()
}

func cmdVarName(opID string, def cmdDef) string {
	candidate := lcFirst(opID) + "Cmd"
	// Avoid collisions with parent vars (e.g. a Ping op -> pingCmd would
	// collide with the "ping" parent command var).
	if _, isParent := cmdParents[strings.TrimSuffix(candidate, "Cmd")]; isParent {
		candidate = lcFirst(opID) + ucFirst(def.use) + "Cmd"
	}
	return candidate
}

func ucFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func emitInit(buf, attachBuf *bytes.Buffer, opIDs []string) {
	fmt.Fprintf(buf, "func init() {\n")

	// Attach runnable parents (self: true commands)
	if attachBuf.Len() > 0 {
		buf.Write(attachBuf.Bytes())
		fmt.Fprintf(buf, "\n")
	}

	// Wire parent->parent (sub-groups)
	wired := false
	for _, key := range sortedKeys(cmdParents) {
		if p := cmdParents[key].parent; p != "" {
			fmt.Fprintf(buf, "\t%sCmd.AddCommand(%sCmd)\n", p, key)
			wired = true
		}
	}
	if wired {
		fmt.Fprintf(buf, "\n")
	}

	// Wire commands to parents
	for _, opID := range opIDs {
		def := cmdRegistry[opID]
		if def.self {
			continue
		}
		fmt.Fprintf(buf, "\t%sCmd.AddCommand(%s)\n", def.parent, cmdVarName(opID, def))
	}

	fmt.Fprintf(buf, "}\n")
}

func backtickQuote(s string) string {
	if strings.Contains(s, "`") {
		// Fall back to double-quoted string with escaping
		return fmt.Sprintf("%q", s)
	}
	return "`" + s + "`"
}
