// Code generated from api/specs/openalgo-api.json by cmd/generate; DO NOT EDIT.

package api

var BasketOrderOp = Op{
	Name: "BasketOrder", Method: "POST", Path: "/basketorder",
	Summary: "Place a basket of orders",
	Long:    "Place multiple independent orders in one request; BUY orders are processed before SELL orders. Partial success is possible and each order reports its own status",
	Example: `  openalgo order basket --orders '[{"symbol":"SBIN","exchange":"NSE","action":"BUY","quantity":1,"pricetype":"MARKET","product":"CNC"},{"symbol":"INFY","exchange":"NSE","action":"BUY","quantity":1,"pricetype":"MARKET","product":"CNC"}]'
  openalgo order basket --orders @basket.json`,
	Mutating: true,
	RowsPath: "results",
	Flags: []FlagDef{
		{Name: "orders", OASName: "orders", Type: "array", Description: "orders to place (at least one); JSON array of {exchange, symbol, action, quantity} objects; literal, @file, or - for stdin", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "results", Type: "[]object", Description: "per-order results", Fields: []ResponseField{
			{Name: "symbol", Type: "string", Description: "trading symbol"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "product", Type: "string", Description: "product type"},
			{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
			{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
			{Name: "message", Type: "string", Description: "error or informational message"},
			{Name: "batch_order", Type: "boolean", Description: "true when the order was sent as part of a broker batch"},
			{Name: "is_last_order", Type: "boolean", Description: "true for the final order of the batch"},
		}},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var CancelAllOrderOp = Op{
	Name: "CancelAllOrder", Method: "POST", Path: "/cancelallorder",
	Summary: "Cancel all open orders",
	Long:    "Cancels EVERY open and trigger-pending order in the account, not only orders placed by the CLI. There is no confirmation step; use --dry-run to preview the request. Returns success even if some cancellations fail; inspect failed_cancellations.",
	Example: `  openalgo order cancel-all
  openalgo order cancel-all --dry-run`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "summary of cancellation results"},
		{Name: "canceled_orders", Type: "[]string", Description: "order ids successfully cancelled"},
		{Name: "failed_cancellations", Type: "[]object", Description: "orders that could not be cancelled", Fields: []ResponseField{
			{Name: "orderid", Type: "string", Description: "order id that failed to cancel"},
			{Name: "reason", Type: "string", Description: "reason for failure"},
		}},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var CancelGTTOrderOp = Op{
	Name: "CancelGTTOrder", Method: "POST", Path: "/cancelgttorder",
	Summary: "Cancel an active GTT order",
	Long:    "Cancel an active GTT trigger. Trigger ids are strings: numeric for live brokers, GTT-<date>-<hex> in analyzer mode.",
	Example: `  openalgo gtt cancel --trigger-id 123456789
  openalgo gtt cancel --trigger-id GTT-260926-0e55ab27`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "trigger-id", OASName: "trigger_id", Type: "string", Description: "id of the active GTT trigger to cancel", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "trigger_id", Type: "string", Description: "GTT trigger id"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var CancelOrderOp = Op{
	Name: "CancelOrder", Method: "POST", Path: "/cancelorder",
	Summary:  "Cancel an order",
	Long:     "Cancel a specific open or pending order by its order id. Completed orders cannot be cancelled",
	Example:  `  openalgo order cancel --orderid 250408000989443`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "orderid", OASName: "orderid", Type: "string", Description: "broker order id to cancel", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var ClosePositionOp = Op{
	Name: "ClosePosition", Method: "POST", Path: "/closeposition",
	Summary: "Close all open positions",
	Long:    "Squares off ALL open positions across every exchange with MARKET orders, not only positions opened by the CLI. There is no confirmation step; use --dry-run to preview the request.",
	Example: `  openalgo position close-all
  openalgo position close-all --dry-run`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "result message"},
		{Name: "closed_positions", Type: "integer", Description: "number of positions closed; sandbox (analyzer mode) only"},
		{Name: "failed_closures", Type: "integer", Description: "number of positions that could not be closed; sandbox (analyzer mode) only"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetAnalyzerStatusOp = Op{
	Name: "GetAnalyzerStatus", Method: "POST", Path: "/analyzer",
	Summary:  "Get analyzer status",
	Long:     "Return whether analyzer (sandbox) mode is active and how many simulated orders have been logged",
	Example:  `  openalgo analyzer status`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "object", Description: "analyzer status", Fields: []ResponseField{
			{Name: "analyze_mode", Type: "boolean", Description: "true if analyzer mode is active"},
			{Name: "mode", Type: "string", Description: "current mode", EnumValues: []string{"analyze", "live"}},
			{Name: "total_logs", Type: "integer", Description: "number of orders logged in analyzer mode"},
		}},
	},
}

var GetChartPreferencesOp = Op{
	Name: "GetChartPreferences", Method: "GET", Path: "/chart",
	Summary: "Get chart preferences",
	Long:    "Return all stored chart workspace preferences for the API key",
	Example: `  openalgo chart get`,
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "object", Description: "stored preference keys and values"},
	},
}

var GetDepthOp = Op{
	Name: "GetDepth", Method: "POST", Path: "/depth",
	Summary:  "Get market depth for a symbol",
	Long:     "Returns a single snapshot of market depth (top 5 bids and asks) with OHLC and volume for a symbol. Use 'openalgo stream depth' for continuous updates",
	Example:  `  openalgo data depth --symbol SBIN --exchange NSE`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "object", Description: "market depth data", Fields: []ResponseField{
			{Name: "open", Type: "number", Description: "day open price"},
			{Name: "high", Type: "number", Description: "day high price"},
			{Name: "low", Type: "number", Description: "day low price"},
			{Name: "ltp", Type: "number", Description: "last traded price"},
			{Name: "ltq", Type: "number", Description: "last traded quantity"},
			{Name: "prev_close", Type: "number", Description: "previous day close"},
			{Name: "volume", Type: "number", Description: "total traded volume"},
			{Name: "oi", Type: "number", Description: "open interest, for F&O symbols"},
			{Name: "totalbuyqty", Type: "number", Description: "total buy quantity in the order book"},
			{Name: "totalsellqty", Type: "number", Description: "total sell quantity in the order book"},
			{Name: "asks", Type: "[]object", Description: "top ask (sell) levels", Fields: []ResponseField{
				{Name: "price", Type: "number", Description: "price level"},
				{Name: "quantity", Type: "number", Description: "quantity at this price"},
			}},
			{Name: "bids", Type: "[]object", Description: "top bid (buy) levels", Fields: []ResponseField{
				{Name: "price", Type: "number", Description: "price level"},
				{Name: "quantity", Type: "number", Description: "quantity at this price"},
			}},
		}},
	},
}

var GetExpiryOp = Op{
	Name: "GetExpiry", Method: "POST", Path: "/expiry",
	Summary: "Get expiry dates",
	Long:    "Return available expiry dates for a futures or options underlying, sorted nearest first in DD-MMM-YY format",
	Example: `  openalgo symbol expiry --symbol NIFTY --exchange NFO --instrumenttype options
  openalgo symbol expiry --symbol CRUDEOIL --exchange MCX --instrumenttype futures`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "derivatives exchange code", Completions: []string{"NFO", "BFO", "MCX", "CDS", "NCO", "BCD", "NCDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "instrumenttype", OASName: "instrumenttype", Type: "string", Description: "instrument type to list expiries for", Completions: []string{"futures", "options"}, Required: true, Source: "body"},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "underlying symbol, e.g. NIFTY", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "[]string", Description: "expiry dates in DD-MMM-YY format, ascending"},
	},
}

var GetFundsOp = Op{
	Name: "GetFunds", Method: "POST", Path: "/funds",
	Summary:  "Get account funds",
	Long:     "Returns available cash, collateral, realized and unrealized M2M and utilized margin for the connected broker account. Values are strings for some live brokers and numbers in analyzer mode",
	Example:  `  openalgo account funds`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "object", Description: "funds data", Fields: []ResponseField{
			{Name: "availablecash", Type: "string|number", Description: "available cash for new trades"},
			{Name: "collateral", Type: "string|number", Description: "collateral margin from pledged holdings"},
			{Name: "m2mrealized", Type: "string|number", Description: "realized mark-to-market profit or loss"},
			{Name: "m2munrealized", Type: "string|number", Description: "unrealized mark-to-market profit or loss"},
			{Name: "utiliseddebits", Type: "string|number", Description: "margin utilized for existing positions"},
			{Name: "grossexposure", Type: "number", Description: "gross exposure of open positions; sandbox (analyzer mode) only"},
			{Name: "totalpnl", Type: "number", Description: "total profit or loss; sandbox (analyzer mode) only"},
			{Name: "today_realized_pnl", Type: "number", Description: "profit or loss realized today; sandbox (analyzer mode) only"},
			{Name: "total_realized_pnl", Type: "number", Description: "profit or loss realized since the last reset; sandbox (analyzer mode) only"},
			{Name: "reset_count", Type: "integer", Description: "number of sandbox fund resets; sandbox (analyzer mode) only"},
			{Name: "last_reset", Type: "string", Description: "time of the last sandbox fund reset; sandbox (analyzer mode) only"},
		}},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetGTTOrderbookOp = Op{
	Name: "GetGTTOrderbook", Method: "POST", Path: "/gttorderbook",
	Summary: "List GTT triggers",
	Long:    "List the user's GTT triggers, active only by default. Send status all to include triggered, cancelled, expired and rejected history",
	Example: `  openalgo gtt list
  openalgo gtt list --status all --csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "status", OASName: "status", Type: "string", Default: "active", Description: "active returns triggers that can still fire; all adds triggered, cancelled, expired and rejected history", Completions: []string{"active", "all"}, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "data", Type: "[]object", Description: "GTT triggers", Fields: []ResponseField{
			{Name: "trigger_id", Type: "string", Description: "unique trigger id"},
			{Name: "trigger_type", Type: "string", Description: "single or two-leg (OCO)"},
			{Name: "status", Type: "string", Description: "trigger status such as active, triggered, cancelled, expired or rejected"},
			{Name: "symbol", Type: "string", Description: "symbol in OpenAlgo format"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "trigger_prices", Type: "[]number", Description: "trigger prices sorted ascending; OCO lists stoploss then target"},
			{Name: "last_price", Type: "number", Description: "LTP captured at place or last modify time, 0 if not exposed"},
			{Name: "legs", Type: "[]object", Description: "child order details per leg", Fields: []ResponseField{
				{Name: "action", Type: "string", Description: "BUY or SELL"},
				{Name: "quantity", Type: "integer", Description: "order quantity"},
				{Name: "price", Type: "number", Description: "child order limit price, 0 for MARKET-style legs"},
				{Name: "pricetype", Type: "string", Description: "LIMIT or MARKET"},
				{Name: "product", Type: "string", Description: "CNC or NRML"},
				{Name: "triggered_order_id", Type: "string|null", Description: "analyzer mode only: sandbox order placed when this leg fired, null until then"},
			}},
			{Name: "created_at", Type: "string", Description: "creation timestamp from broker"},
			{Name: "updated_at", Type: "string", Description: "last update timestamp, empty if never modified"},
			{Name: "expires_at", Type: "string", Description: "expiry timestamp, empty if not exposed"},
			{Name: "strategy", Type: "string", Description: "analyzer mode only: strategy sent at placement"},
			{Name: "margin_blocked", Type: "number", Description: "analyzer mode only: margin reserved while the trigger is active"},
		}},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetHistoryOp = Op{
	Name: "GetHistory", Method: "POST", Path: "/history",
	Summary: "Get historical candles",
	Long:    "Returns historical OHLCV candles for a symbol between two dates. Data availability and supported intervals depend on the broker; use source db to read locally stored Historify data",
	Example: `  openalgo data history --symbol SBIN --exchange NSE --interval D --start-date 2026-09-01 --end-date 2026-09-25
  openalgo data history --symbol NIFTY --exchange NSE_INDEX --interval 5m --start-date 2026-09-24 --end-date 2026-09-25 --csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "end-date", OASName: "end_date", Type: "string", Description: "end date in YYYY-MM-DD format", Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "interval", OASName: "interval", Type: "string", Description: "candle interval; run 'openalgo data intervals' to see what the broker supports", Completions: []string{"1s", "5s", "10s", "15s", "30s", "45s", "1m", "2m", "3m", "5m", "10m", "15m", "20m", "30m", "1h", "2h", "3h", "4h", "D", "W", "M", "Q", "Y"}, Required: true, Source: "body"},
		{Name: "start-date", OASName: "start_date", Type: "string", Description: "start date in YYYY-MM-DD format", Required: true, Source: "body"},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol", Required: true, Source: "body"},
		{Name: "source", OASName: "source", Type: "string", Default: "api", Description: "data source: api for the broker or db for local Historify data", Completions: []string{"api", "db"}, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "[]object", Description: "OHLCV candles", Fields: []ResponseField{
			{Name: "timestamp", Type: "integer", Description: "candle open time as a Unix timestamp in seconds"},
			{Name: "open", Type: "number", Description: "opening price"},
			{Name: "high", Type: "number", Description: "highest price"},
			{Name: "low", Type: "number", Description: "lowest price"},
			{Name: "close", Type: "number", Description: "closing price"},
			{Name: "volume", Type: "number", Description: "volume traded"},
			{Name: "oi", Type: "number", Description: "open interest, 0 when not applicable"},
		}},
	},
}

var GetHoldingsOp = Op{
	Name: "GetHoldings", Method: "POST", Path: "/holdings",
	Summary: "Get portfolio holdings",
	Long:    "Returns delivery holdings with per-holding P&L and portfolio statistics",
	Example: `  openalgo account holdings
  openalgo account holdings --csv`,
	RowsPath: "data.holdings",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "object", Description: "holdings data", Fields: []ResponseField{
			{Name: "holdings", Type: "[]object", Description: "holdings", Fields: []ResponseField{
				{Name: "symbol", Type: "string", Description: "stock symbol"},
				{Name: "exchange", Type: "string", Description: "exchange code"},
				{Name: "product", Type: "string", Description: "product type, typically CNC"},
				{Name: "quantity", Type: "number", Description: "number of shares held"},
				{Name: "pnl", Type: "number", Description: "profit or loss in currency"},
				{Name: "pnlpercent", Type: "number", Description: "profit or loss percentage"},
			}},
			{Name: "statistics", Type: "object", Description: "portfolio statistics", Fields: []ResponseField{
				{Name: "totalholdingvalue", Type: "number", Description: "current market value of holdings"},
				{Name: "totalinvvalue", Type: "number", Description: "total investment value at cost"},
				{Name: "totalprofitandloss", Type: "number", Description: "total profit or loss in currency"},
				{Name: "totalpnlpercentage", Type: "number", Description: "total profit or loss percentage"},
			}},
		}},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetInstrumentsOp = Op{
	Name: "GetInstruments", Method: "GET", Path: "/instruments",
	Summary: "Download instruments",
	Long:    "Download the locally stored instrument master for one exchange. With --format csv the CSV file is written verbatim to stdout. The API documents --exchange as optional (all exchanges), but current OpenAlgo servers reject a request without it, so always pass --exchange.",
	Example: `  openalgo symbol instruments --exchange NSE
  openalgo symbol instruments --exchange NFO --format csv > nfo.csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange to filter by; current OpenAlgo servers reject a request without it", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Source: "query"},
		{Name: "format", OASName: "format", Type: "string", Default: "json", Description: "output format", Completions: []string{"json", "csv"}, Source: "query"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "[]object", Description: "instrument rows", Fields: []ResponseField{
			{Name: "symbol", Type: "string", Description: "openAlgo standard symbol"},
			{Name: "brsymbol", Type: "string", Description: "broker-specific symbol"},
			{Name: "name", Type: "string", Description: "underlying or symbol name"},
			{Name: "exchange", Type: "string", Description: "openAlgo exchange code"},
			{Name: "brexchange", Type: "string", Description: "broker-specific exchange code"},
			{Name: "instrumenttype", Type: "string", Description: "instrument type such as EQ, FUT, CE or PE (may be empty for equity)"},
			{Name: "expiry", Type: "string", Description: "expiry date in DD-MMM-YY format, empty for non-derivatives"},
			{Name: "strike", Type: "number", Description: "strike price (-0.01 or 0 for non-options)"},
			{Name: "lotsize", Type: "integer", Description: "lot size (1 for equity)"},
			{Name: "tick_size", Type: "number", Description: "minimum price movement"},
			{Name: "token", Type: "string", Description: "broker-specific instrument token"},
		}},
	},
}

var GetIntervalsOp = Op{
	Name: "GetIntervals", Method: "POST", Path: "/intervals",
	Summary:  "Get supported history intervals",
	Long:     "Returns the historical data intervals supported by the connected broker, grouped by unit",
	Example:  `  openalgo data intervals`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "object", Description: "available intervals by category", Fields: []ResponseField{
			{Name: "months", Type: "[]string", Description: "months intervals"},
			{Name: "weeks", Type: "[]string", Description: "weeks intervals"},
			{Name: "days", Type: "[]string", Description: "days intervals"},
			{Name: "hours", Type: "[]string", Description: "hours intervals"},
			{Name: "minutes", Type: "[]string", Description: "minutes intervals"},
			{Name: "seconds", Type: "[]string", Description: "seconds intervals"},
		}},
	},
}

var GetMarginOp = Op{
	Name: "GetMargin", Method: "POST", Path: "/margin",
	Summary: "Calculate margin for a basket of positions",
	Long:    "Calculates the margin required for a basket of 1 to 50 positions, including hedging benefits where the broker supports it. Not all brokers support margin calculation",
	Example: `  openalgo account margin --positions '[{"symbol":"NIFTY27OCT26FUT","exchange":"NFO","action":"BUY","quantity":"65","product":"NRML","pricetype":"MARKET"}]'
  openalgo account margin --positions @positions.json`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "positions", OASName: "positions", Type: "array", Description: "positions to calculate margin for (1 to 50 items); JSON array of {symbol, exchange, action, quantity, product, pricetype} objects; literal, @file, or - for stdin", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "object", Description: "margin calculation results", Fields: []ResponseField{
			{Name: "total_margin_required", Type: "number", Description: "total margin required for the basket"},
			{Name: "span_margin", Type: "number", Description: "SPAN margin component"},
			{Name: "exposure_margin", Type: "number", Description: "exposure margin component"},
			{Name: "margin_benefit", Type: "number", Description: "margin benefit from hedged positions, when reported by the broker"},
		}},
	},
}

var GetMarketHolidaysOp = Op{
	Name: "GetMarketHolidays", Method: "POST", Path: "/market/holidays",
	Summary: "Get market holidays",
	Long:    "Returns market holidays for a year, including exchanges closed and exchanges running special sessions. Defaults to the current year",
	Example: `  openalgo market holidays
  openalgo market holidays --year 2026 --csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "year", OASName: "year", Type: "int", Description: "year to get holidays for (2020 to 2050), defaults to the current year", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "year", Type: "integer", Description: "year for which holidays are returned"},
		{Name: "timezone", Type: "string", Description: "timezone of the calendar, Asia/Kolkata"},
		{Name: "data", Type: "[]object", Description: "holidays", Fields: []ResponseField{
			{Name: "date", Type: "string", Description: "holiday date in YYYY-MM-DD format"},
			{Name: "description", Type: "string", Description: "holiday name or reason"},
			{Name: "holiday_type", Type: "string", Description: "type of holiday", EnumValues: []string{"TRADING_HOLIDAY", "SETTLEMENT_HOLIDAY", "SPECIAL_SESSION"}},
			{Name: "closed_exchanges", Type: "[]string", Description: "exchanges fully closed"},
			{Name: "open_exchanges", Type: "[]object", Description: "exchanges with special or partial sessions", Fields: []ResponseField{
				{Name: "exchange", Type: "string", Description: "exchange code"},
				{Name: "start_time", Type: "integer", Description: "session start in epoch milliseconds"},
				{Name: "end_time", Type: "integer", Description: "session end in epoch milliseconds"},
			}},
		}},
	},
}

var GetMarketTimingsOp = Op{
	Name: "GetMarketTimings", Method: "POST", Path: "/market/timings",
	Summary:  "Get market timings for a date",
	Long:     "Returns trading session start and end times per exchange for a date, in epoch milliseconds. Returns an empty array on weekends and full holidays",
	Example:  `  openalgo market timings --date 2026-09-25`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "date", OASName: "date", Type: "string", Description: "date in YYYY-MM-DD format", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "[]object", Description: "exchange timings", Fields: []ResponseField{
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "start_time", Type: "integer", Description: "session start in epoch milliseconds"},
			{Name: "end_time", Type: "integer", Description: "session end in epoch milliseconds"},
		}},
	},
}

var GetMultiOptionGreeksOp = Op{
	Name: "GetMultiOptionGreeks", Method: "POST", Path: "/multioptiongreeks",
	Summary:  "Calculate Greeks for multiple options",
	Long:     "Calculate Black-76 implied volatility and Greeks for 1 to 50 options in one batch. Individual items can fail while the batch succeeds, so inspect each item and the summary",
	Example:  `  openalgo option multi-greeks --symbols '[{"symbol":"NIFTY27OCT2626000CE","exchange":"NFO"},{"symbol":"NIFTY27OCT2626000PE","exchange":"NFO"}]'`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "symbols", OASName: "symbols", Type: "array", Description: "option requests (1 to 50); JSON array of {symbol, exchange} objects; literal, @file, or - for stdin", Required: true, Source: "body"},
		{Name: "expiry-time", OASName: "expiry_time", Type: "string", Description: "common expiry time in HH:MM format", Source: "body"},
		{Name: "interest-rate", OASName: "interest_rate", Type: "number", Description: "common annualized risk-free rate in percent", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success when all items succeed, partial when some fail, error when all fail", EnumValues: []string{"success", "partial", "error"}},
		{Name: "message", Type: "string", Description: "summary of failures, present when any item failed"},
		{Name: "data", Type: "[]object", Description: "per-option results in request order", Fields: []ResponseField{
			{Name: "status", Type: "string", Description: "item status", EnumValues: []string{"success", "error"}},
			{Name: "message", Type: "string", Description: "error message for a failed item"},
			{Name: "symbol", Type: "string", Description: "option symbol"},
			{Name: "exchange", Type: "string", Description: "exchange"},
			{Name: "underlying", Type: "string", Description: "underlying symbol"},
			{Name: "strike", Type: "number", Description: "strike price"},
			{Name: "option_type", Type: "string", Description: "CE or PE"},
			{Name: "expiry_date", Type: "string", Description: "expiry date, e.g. 25-Aug-2026"},
			{Name: "days_to_expiry", Type: "number", Description: "fractional days remaining to expiry"},
			{Name: "spot_price", Type: "number", Description: "forward or underlying price used"},
			{Name: "option_price", Type: "number", Description: "option last traded price"},
			{Name: "interest_rate", Type: "number", Description: "risk-free rate used"},
			{Name: "implied_volatility", Type: "number", Description: "implied volatility in percent"},
			{Name: "greeks", Type: "object", Description: "option Greeks", Fields: []ResponseField{
				{Name: "delta", Type: "number", Description: "price sensitivity to underlying movement"},
				{Name: "gamma", Type: "number", Description: "delta sensitivity to underlying movement"},
				{Name: "theta", Type: "number", Description: "time decay per calendar day"},
				{Name: "vega", Type: "number", Description: "price sensitivity to a 1 percent IV change"},
				{Name: "rho", Type: "number", Description: "price sensitivity to a 1 percent interest rate change"},
			}},
		}},
		{Name: "summary", Type: "object", Description: "batch counts", Fields: []ResponseField{
			{Name: "total", Type: "integer", Description: "number of options requested"},
			{Name: "success", Type: "integer", Description: "number of successful calculations"},
			{Name: "failed", Type: "integer", Description: "number of failed calculations"},
		}},
	},
}

var GetMultiQuotesOp = Op{
	Name: "GetMultiQuotes", Method: "POST", Path: "/multiquotes",
	Summary:  "Get quotes for multiple symbols",
	Long:     "Returns real-time quotes for several symbols in one call. Symbols that fail lookup are returned with an error field instead of data",
	Example:  `  openalgo data quotes --symbols '[{"symbol":"RELIANCE","exchange":"NSE"},{"symbol":"INFY","exchange":"NSE"},{"symbol":"NIFTY","exchange":"NSE_INDEX"}]'`,
	RowsPath: "results",
	Flags: []FlagDef{
		{Name: "symbols", OASName: "symbols", Type: "array", Description: "symbols to quote (at least one); JSON array of {symbol, exchange} objects; literal, @file, or - for stdin", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "results", Type: "[]object", Description: "quote results", Fields: []ResponseField{
			{Name: "symbol", Type: "string", Description: "trading symbol"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "data", Type: "object", Description: "quote data", Fields: []ResponseField{
				{Name: "open", Type: "number", Description: "day open price"},
				{Name: "high", Type: "number", Description: "day high price"},
				{Name: "low", Type: "number", Description: "day low price"},
				{Name: "ltp", Type: "number", Description: "last traded price"},
				{Name: "ask", Type: "number", Description: "best ask price"},
				{Name: "bid", Type: "number", Description: "best bid price"},
				{Name: "ask_qty", Type: "number", Description: "quantity at best ask"},
				{Name: "bid_qty", Type: "number", Description: "quantity at best bid"},
				{Name: "prev_close", Type: "number", Description: "previous day close price"},
				{Name: "volume", Type: "number", Description: "total traded volume"},
				{Name: "oi", Type: "number", Description: "open interest, for F&O symbols"},
			}},
			{Name: "error", Type: "string", Description: "error message if the symbol lookup failed"},
		}},
	},
}

var GetOpenPositionOp = Op{
	Name: "GetOpenPosition", Method: "POST", Path: "/openposition",
	Summary: "Get the open position for a symbol",
	Long:    "Return the net open quantity for a symbol, exchange and product combination, or 0 when flat",
	Example: `  openalgo position get --symbol SBIN --exchange NSE --product CNC
  openalgo position get --symbol NIFTY27OCT26FUT --exchange NFO --product NRML --jq '.quantity'`,
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Description: "product type", Completions: []string{"MIS", "CNC", "NRML"}, Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "quantity", Type: "string|number", Description: "net position quantity; positive long, negative short, 0 flat (a number in analyzer mode)"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetOptionChainOp = Op{
	Name: "GetOptionChain", Method: "POST", Path: "/optionchain",
	Summary: "Get option chain",
	Long:    "Return the option chain for an underlying and expiry with quotes for each strike, optionally with Black-76 implied volatility and Greeks per leg at no extra broker calls",
	Example: `  openalgo option chain --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --strike-count 10
  openalgo option chain --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --strike-count 5 --with-greeks`,
	RowsPath: "chain",
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "underlying exchange, e.g. NSE_INDEX", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "expiry-date", OASName: "expiry_date", Type: "string", Description: "expiry date in DDMMMYY format (DD-MMM-YY also accepted), e.g. 25AUG26", Required: true, Source: "body"},
		{Name: "underlying", OASName: "underlying", Type: "string", Description: "underlying symbol, e.g. NIFTY", Required: true, Source: "body"},
		{Name: "interest-rate", OASName: "interest_rate", Type: "number", Description: "annualized risk-free rate in percent, used for Greeks only (defaults to 0)", Source: "body"},
		{Name: "strike-count", OASName: "strike_count", Type: "int", Description: "number of strikes above and below ATM; entire chain when omitted", Source: "body"},
		{Name: "with-greeks", OASName: "with_greeks", Type: "bool", Default: "false", Description: "attach implied volatility and Greeks to every leg", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "underlying", Type: "string", Description: "underlying symbol"},
		{Name: "underlying_symbol", Type: "string", Description: "underlying symbol used for the quote"},
		{Name: "underlying_exchange", Type: "string", Description: "exchange of the underlying quote, e.g. NSE_INDEX"},
		{Name: "underlying_ltp", Type: "number", Description: "current underlying price"},
		{Name: "underlying_prev_close", Type: "number", Description: "underlying previous close"},
		{Name: "expiry_date", Type: "string", Description: "expiry date in DDMMMYY format"},
		{Name: "expiry_ts", Type: "integer|null", Description: "expiry instant as epoch seconds including exchange cut-off time"},
		{Name: "server_ts", Type: "integer", Description: "server time as epoch seconds"},
		{Name: "atm_strike", Type: "number", Description: "at-the-money strike"},
		{Name: "quotes_included", Type: "boolean", Description: "whether live quotes were fetched"},
		{Name: "greeks_included", Type: "boolean", Description: "whether Greeks were attached"},
		{Name: "forward_price", Type: "number|null", Description: "forward price used for Greeks, null when Greeks were not requested"},
		{Name: "chain", Type: "[]object", Description: "per-strike call and put data", Fields: []ResponseField{
			{Name: "strike", Type: "number", Description: "strike price"},
			{Name: "ce", Type: "object|null", Description: "call leg, null when not in the master contract", Fields: []ResponseField{
				{Name: "symbol", Type: "string", Description: "option symbol"},
				{Name: "label", Type: "string", Description: "moneyness label: ATM, ITMn or OTMn"},
				{Name: "ltp", Type: "number", Description: "last traded price"},
				{Name: "bid", Type: "number", Description: "best bid price"},
				{Name: "ask", Type: "number", Description: "best ask price"},
				{Name: "bid_qty", Type: "number", Description: "quantity at best bid"},
				{Name: "ask_qty", Type: "number", Description: "quantity at best ask"},
				{Name: "open", Type: "number", Description: "day open"},
				{Name: "high", Type: "number", Description: "day high"},
				{Name: "low", Type: "number", Description: "day low"},
				{Name: "prev_close", Type: "number", Description: "previous close"},
				{Name: "volume", Type: "number", Description: "trading volume"},
				{Name: "oi", Type: "number", Description: "open interest"},
				{Name: "lotsize", Type: "number", Description: "lot size"},
				{Name: "tick_size", Type: "number", Description: "tick size"},
				{Name: "implied_volatility", Type: "number", Description: "implied volatility in percent (only with with_greeks)"},
				{Name: "delta", Type: "number", Description: "delta per 1 point underlying move (only with with_greeks)"},
				{Name: "gamma", Type: "number", Description: "gamma per 1 point underlying move (only with with_greeks)"},
				{Name: "theta", Type: "number", Description: "theta per calendar day (only with with_greeks)"},
				{Name: "vega", Type: "number", Description: "vega per 1 percent volatility change (only with with_greeks)"},
			}},
			{Name: "pe", Type: "object|null", Description: "put leg, null when not in the master contract", Fields: []ResponseField{
				{Name: "symbol", Type: "string", Description: "option symbol"},
				{Name: "label", Type: "string", Description: "moneyness label: ATM, ITMn or OTMn"},
				{Name: "ltp", Type: "number", Description: "last traded price"},
				{Name: "bid", Type: "number", Description: "best bid price"},
				{Name: "ask", Type: "number", Description: "best ask price"},
				{Name: "bid_qty", Type: "number", Description: "quantity at best bid"},
				{Name: "ask_qty", Type: "number", Description: "quantity at best ask"},
				{Name: "open", Type: "number", Description: "day open"},
				{Name: "high", Type: "number", Description: "day high"},
				{Name: "low", Type: "number", Description: "day low"},
				{Name: "prev_close", Type: "number", Description: "previous close"},
				{Name: "volume", Type: "number", Description: "trading volume"},
				{Name: "oi", Type: "number", Description: "open interest"},
				{Name: "lotsize", Type: "number", Description: "lot size"},
				{Name: "tick_size", Type: "number", Description: "tick size"},
				{Name: "implied_volatility", Type: "number", Description: "implied volatility in percent (only with with_greeks)"},
				{Name: "delta", Type: "number", Description: "delta per 1 point underlying move (only with with_greeks)"},
				{Name: "gamma", Type: "number", Description: "gamma per 1 point underlying move (only with with_greeks)"},
				{Name: "theta", Type: "number", Description: "theta per calendar day (only with with_greeks)"},
				{Name: "vega", Type: "number", Description: "vega per 1 percent volatility change (only with with_greeks)"},
			}},
		}},
	},
}

var GetOptionGreeksOp = Op{
	Name: "GetOptionGreeks", Method: "POST", Path: "/optiongreeks",
	Summary: "Calculate option Greeks",
	Long:    "Calculate Black-76 implied volatility and Greeks for one option. The forward defaults to a per-expiry synthetic future, falling back to the underlying quote",
	Example: `  openalgo option greeks --symbol NIFTY27OCT2626000CE --exchange NFO
  openalgo option greeks --symbol NIFTY27OCT2626000PE --exchange NFO --interest-rate 6.5`,
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "derivatives exchange code", Completions: []string{"NFO", "BFO", "MCX", "CDS", "NCO", "BCD", "NCDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "option symbol, e.g. NIFTY25AUG2626000CE", Required: true, Source: "body"},
		{Name: "expiry-time", OASName: "expiry_time", Type: "string", Description: "custom expiry time in HH:MM format, e.g. 15:30", Source: "body"},
		{Name: "forward-price", OASName: "forward_price", Type: "number", Description: "custom forward price that bypasses automatic forward resolution", Source: "body"},
		{Name: "interest-rate", OASName: "interest_rate", Type: "number", Description: "annualized risk-free rate in percent; exchange default when omitted", Source: "body"},
		{Name: "underlying-exchange", OASName: "underlying_exchange", Type: "string", Description: "override for the underlying exchange, e.g. NSE_INDEX", Source: "body"},
		{Name: "underlying-symbol", OASName: "underlying_symbol", Type: "string", Description: "override for the underlying symbol, e.g. NIFTY", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "symbol", Type: "string", Description: "option symbol"},
		{Name: "exchange", Type: "string", Description: "exchange"},
		{Name: "underlying", Type: "string", Description: "underlying symbol"},
		{Name: "strike", Type: "number", Description: "strike price"},
		{Name: "option_type", Type: "string", Description: "CE or PE"},
		{Name: "expiry_date", Type: "string", Description: "expiry date, e.g. 25-Aug-2026"},
		{Name: "days_to_expiry", Type: "number", Description: "fractional days remaining to expiry"},
		{Name: "spot_price", Type: "number", Description: "forward or underlying price used"},
		{Name: "option_price", Type: "number", Description: "option last traded price"},
		{Name: "interest_rate", Type: "number", Description: "risk-free rate used"},
		{Name: "implied_volatility", Type: "number", Description: "implied volatility in percent"},
		{Name: "greeks", Type: "object", Description: "option Greeks", Fields: []ResponseField{
			{Name: "delta", Type: "number", Description: "price sensitivity to underlying movement"},
			{Name: "gamma", Type: "number", Description: "delta sensitivity to underlying movement"},
			{Name: "theta", Type: "number", Description: "time decay per calendar day"},
			{Name: "vega", Type: "number", Description: "price sensitivity to a 1 percent IV change"},
			{Name: "rho", Type: "number", Description: "price sensitivity to a 1 percent interest rate change"},
		}},
	},
}

var GetOptionSymbolOp = Op{
	Name: "GetOptionSymbol", Method: "POST", Path: "/optionsymbol",
	Summary: "Resolve an option symbol",
	Long:    "Resolve the option symbol for an underlying, expiry, strike offset (ATM, ITMn, OTMn) and option type using actual strikes from the master contract",
	Example: `  openalgo option symbol --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ATM --option-type CE
  openalgo option symbol --underlying BANKNIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ITM3 --option-type PE`,
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "underlying exchange, e.g. NSE_INDEX or BSE_INDEX", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "offset", OASName: "offset", Type: "string", Description: "strike offset: ATM, ITM1-ITM50 or OTM1-OTM50", Required: true, Source: "body"},
		{Name: "option-type", OASName: "option_type", Type: "string", Description: "CE for call or PE for put", Completions: []string{"CE", "PE"}, Required: true, Source: "body"},
		{Name: "underlying", OASName: "underlying", Type: "string", Description: "underlying symbol, e.g. NIFTY, or a futures symbol that embeds the expiry", Required: true, Source: "body"},
		{Name: "expiry-date", OASName: "expiry_date", Type: "string", Description: "expiry date in DDMMMYY format (DD-MMM-YY also accepted), e.g. 25AUG26; optional if the underlying includes the expiry", Source: "body"},
		{Name: "strike-int", OASName: "strike_int", Type: "int", Description: "optional strike interval; actual database strikes are used when omitted", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "symbol", Type: "string", Description: "resolved option symbol"},
		{Name: "exchange", Type: "string", Description: "options exchange, e.g. NFO or BFO"},
		{Name: "lotsize", Type: "integer", Description: "lot size of the option"},
		{Name: "tick_size", Type: "number", Description: "minimum price movement"},
		{Name: "freeze_qty", Type: "number", Description: "maximum quantity per order"},
		{Name: "underlying_ltp", Type: "number", Description: "underlying price used for ATM calculation"},
	},
}

var GetOrderStatusOp = Op{
	Name: "GetOrderStatus", Method: "POST", Path: "/orderstatus",
	Summary: "Get the status of an order",
	Long:    "Return the current details and status of a specific order by its order id",
	Example: `  openalgo order status --orderid 250408000989443
  openalgo order status --orderid 250408000989443 --jq '.data.order_status'`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "orderid", OASName: "orderid", Type: "string", Description: "broker order id to query", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "data", Type: "object", Description: "the order's current details", Fields: []ResponseField{
			{Name: "orderid", Type: "string", Description: "order id"},
			{Name: "symbol", Type: "string", Description: "trading symbol"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "action", Type: "string", Description: "BUY or SELL"},
			{Name: "quantity", Type: "string|number", Description: "order quantity"},
			{Name: "price", Type: "number", Description: "order price, 0 for MARKET orders"},
			{Name: "trigger_price", Type: "number", Description: "trigger price for SL orders"},
			{Name: "pricetype", Type: "string", Description: "MARKET, LIMIT, SL or SL-M"},
			{Name: "price_type", Type: "string", Description: "MARKET, LIMIT, SL or SL-M; sandbox (analyzer mode) only, where it replaces pricetype"},
			{Name: "product", Type: "string", Description: "MIS, CNC or NRML"},
			{Name: "order_status", Type: "string", Description: "current order status such as complete, open, pending, rejected or cancelled"},
			{Name: "average_price", Type: "number", Description: "average execution price"},
			{Name: "timestamp", Type: "string", Description: "order timestamp in IST"},
			{Name: "filled_quantity", Type: "string|number", Description: "quantity filled so far; sandbox (analyzer mode) only"},
			{Name: "pending_quantity", Type: "string|number", Description: "quantity still open; sandbox (analyzer mode) only"},
			{Name: "strategy", Type: "string", Description: "strategy tag the order was placed with; sandbox (analyzer mode) only"},
		}},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetOrderbookOp = Op{
	Name: "GetOrderbook", Method: "POST", Path: "/orderbook",
	Summary: "Get the order book",
	Long:    "Returns all orders placed during the current trading day, including completed, cancelled and rejected orders, along with summary statistics",
	Example: `  openalgo order list
  openalgo order list --csv
  openalgo order list --jq '.data.orders[] | select(.order_status == "open")'`,
	RowsPath: "data.orders",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "object", Description: "order book data", Fields: []ResponseField{
			{Name: "orders", Type: "[]object", Description: "orders for the day", Fields: []ResponseField{
				{Name: "orderid", Type: "string", Description: "unique order id"},
				{Name: "symbol", Type: "string", Description: "trading symbol"},
				{Name: "exchange", Type: "string", Description: "exchange code"},
				{Name: "action", Type: "string", Description: "BUY or SELL"},
				{Name: "quantity", Type: "string|number", Description: "order quantity"},
				{Name: "filled_quantity", Type: "string|number", Description: "quantity filled so far"},
				{Name: "pending_quantity", Type: "string|number", Description: "quantity still open"},
				{Name: "price", Type: "number", Description: "order price"},
				{Name: "average_price", Type: "number", Description: "average execution price"},
				{Name: "trigger_price", Type: "number", Description: "trigger price for stop-loss orders"},
				{Name: "pricetype", Type: "string", Description: "MARKET, LIMIT, SL or SL-M"},
				{Name: "product", Type: "string", Description: "MIS, CNC or NRML"},
				{Name: "order_status", Type: "string", Description: "current order status such as complete, open, pending, rejected or cancelled"},
				{Name: "rejection_reason", Type: "string", Description: "broker rejection reason, empty unless rejected"},
				{Name: "timestamp", Type: "string", Description: "order placement time"},
				{Name: "strategy", Type: "string", Description: "strategy tag the order was placed with"},
			}},
			{Name: "statistics", Type: "object", Description: "order statistics summary", Fields: []ResponseField{
				{Name: "total_buy_orders", Type: "number", Description: "total buy orders placed"},
				{Name: "total_sell_orders", Type: "number", Description: "total sell orders placed"},
				{Name: "total_completed_orders", Type: "number", Description: "orders fully executed"},
				{Name: "total_open_orders", Type: "number", Description: "pending or open orders"},
				{Name: "total_rejected_orders", Type: "number", Description: "rejected orders"},
				{Name: "total_trigger_pending_orders", Type: "number", Description: "trigger-pending stop orders"},
			}},
		}},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetPnlSymbolsOp = Op{
	Name: "GetPnlSymbols", Method: "POST", Path: "/pnl/symbols",
	Summary: "Get sandbox P&L by symbol",
	Long:    "Return today's realized and unrealized P&L per open sandbox position. Available only while analyzer mode is enabled; live mode returns HTTP 400",
	Example: `  openalgo analyzer pnl
  openalgo analyzer pnl --csv`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "[]object", Description: "per-position P&L", Fields: []ResponseField{
			{Name: "symbol", Type: "string", Description: "trading symbol"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "product", Type: "string", Description: "product type"},
			{Name: "quantity", Type: "number", Description: "net quantity"},
			{Name: "pnl", Type: "number", Description: "position P&L"},
			{Name: "unrealized_pnl", Type: "number", Description: "unrealized P&L"},
			{Name: "today_realized_pnl", Type: "number", Description: "realized P&L today"},
			{Name: "total_pnl_today", Type: "number", Description: "total P&L today"},
		}},
		{Name: "total_pnl", Type: "number", Description: "aggregate P&L"},
		{Name: "total_unrealized_pnl", Type: "number", Description: "aggregate unrealized P&L"},
		{Name: "total_today_realized_pnl", Type: "number", Description: "aggregate realized P&L today"},
		{Name: "total_pnl_today", Type: "number", Description: "aggregate total P&L today"},
		{Name: "mode", Type: "string", Description: "always analyze for this endpoint"},
	},
}

var GetPositionbookOp = Op{
	Name: "GetPositionbook", Method: "POST", Path: "/positionbook",
	Summary: "Get the position book",
	Long:    "Returns all positions for the trading day, including closed positions with zero quantity. Positive quantity is long, negative is short",
	Example: `  openalgo position list
  openalgo position list --csv
  openalgo position list --jq '.data[] | select((.quantity | tonumber) != 0)'`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "[]object", Description: "positions", Fields: []ResponseField{
			{Name: "symbol", Type: "string", Description: "trading symbol"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "product", Type: "string", Description: "MIS, CNC or NRML"},
			{Name: "quantity", Type: "string|number", Description: "net position quantity, negative for short; a number in analyzer mode"},
			{Name: "average_price", Type: "string|number", Description: "average entry price"},
			{Name: "ltp", Type: "string|number", Description: "last traded price"},
			{Name: "pnl", Type: "string|number", Description: "profit or loss"},
			{Name: "pnlpercent", Type: "number", Description: "profit or loss percentage; sandbox (analyzer mode) only"},
			{Name: "unrealized_pnl", Type: "number", Description: "unrealized profit or loss; sandbox (analyzer mode) only"},
			{Name: "today_realized_pnl", Type: "number", Description: "profit or loss realized today; sandbox (analyzer mode) only"},
			{Name: "total_pnl_today", Type: "number", Description: "realized plus unrealized profit or loss today; sandbox (analyzer mode) only"},
			{Name: "lot_size", Type: "number", Description: "contract value multiplier; sandbox (analyzer mode) only"},
			{Name: "average_price_basis", Type: "string", Description: "present only when average_price and pnl are not measured from entry cost", EnumValues: []string{"carry_forward_valuation"}},
		}},
		{Name: "total_pnl", Type: "number", Description: "total profit or loss today; sandbox (analyzer mode) only"},
		{Name: "total_pnl_today", Type: "number", Description: "total profit or loss today; sandbox (analyzer mode) only"},
		{Name: "total_today_realized_pnl", Type: "number", Description: "total profit or loss realized today; sandbox (analyzer mode) only"},
		{Name: "total_unrealized_pnl", Type: "number", Description: "total unrealized profit or loss; sandbox (analyzer mode) only"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var GetQuotesOp = Op{
	Name: "GetQuotes", Method: "POST", Path: "/quotes",
	Summary: "Get a quote for a symbol",
	Long:    "Returns a real-time quote for a single symbol including OHLC, LTP, bid, ask and volume",
	Example: `  openalgo data quote --symbol RELIANCE --exchange NSE
  openalgo data quote --symbol NIFTY --exchange NSE_INDEX --jq '.data.ltp'`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "object", Description: "quote data", Fields: []ResponseField{
			{Name: "open", Type: "number", Description: "day open price"},
			{Name: "high", Type: "number", Description: "day high price"},
			{Name: "low", Type: "number", Description: "day low price"},
			{Name: "ltp", Type: "number", Description: "last traded price"},
			{Name: "ask", Type: "number", Description: "best ask price"},
			{Name: "bid", Type: "number", Description: "best bid price"},
			{Name: "ask_qty", Type: "number", Description: "quantity at best ask"},
			{Name: "bid_qty", Type: "number", Description: "quantity at best bid"},
			{Name: "prev_close", Type: "number", Description: "previous day close price"},
			{Name: "volume", Type: "number", Description: "total traded volume"},
			{Name: "oi", Type: "number", Description: "open interest, for F&O symbols"},
		}},
	},
}

var GetSymbolOp = Op{
	Name: "GetSymbol", Method: "POST", Path: "/symbol",
	Summary: "Get symbol details",
	Long:    "Return master-contract details for a single symbol, including the broker-specific symbol, token, lot size and tick size",
	Example: `  openalgo symbol get --symbol RELIANCE --exchange NSE
  openalgo symbol get --symbol NIFTY27OCT26FUT --exchange NFO`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol in OpenAlgo format", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "object", Description: "symbol details", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "internal symbol id"},
			{Name: "symbol", Type: "string", Description: "openAlgo standard symbol"},
			{Name: "brsymbol", Type: "string", Description: "broker-specific symbol"},
			{Name: "name", Type: "string", Description: "underlying or symbol name"},
			{Name: "exchange", Type: "string", Description: "openAlgo exchange code"},
			{Name: "brexchange", Type: "string", Description: "broker-specific exchange code"},
			{Name: "instrumenttype", Type: "string", Description: "instrument type such as EQ, FUT, CE or PE (may be empty for equity)"},
			{Name: "expiry", Type: "string", Description: "expiry date in DD-MMM-YY format, empty for non-derivatives"},
			{Name: "strike", Type: "number", Description: "strike price (-0.01 or 0 for non-options)"},
			{Name: "lotsize", Type: "integer", Description: "lot size (1 for equity)"},
			{Name: "tick_size", Type: "number", Description: "minimum price movement"},
			{Name: "freeze_qty", Type: "number", Description: "maximum quantity allowed per order"},
			{Name: "token", Type: "string", Description: "broker-specific instrument token"},
		}},
	},
}

var GetSyntheticFutureOp = Op{
	Name: "GetSyntheticFuture", Method: "POST", Path: "/syntheticfuture",
	Summary: "Calculate synthetic future price",
	Long:    "Calculate the synthetic futures price from ATM call and put premiums via put-call parity. Does not place any orders",
	Example: `  openalgo option synthetic-future --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26`,
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "underlying exchange, e.g. NSE_INDEX or BSE_INDEX", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "expiry-date", OASName: "expiry_date", Type: "string", Description: "expiry date in DDMMMYY format (DD-MMM-YY also accepted), e.g. 25AUG26", Required: true, Source: "body"},
		{Name: "underlying", OASName: "underlying", Type: "string", Description: "underlying symbol, e.g. NIFTY", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "underlying", Type: "string", Description: "underlying symbol"},
		{Name: "underlying_ltp", Type: "number", Description: "current spot price"},
		{Name: "expiry", Type: "string", Description: "expiry date"},
		{Name: "atm_strike", Type: "number", Description: "ATM strike used for the calculation"},
		{Name: "synthetic_future_price", Type: "number", Description: "strike plus call premium minus put premium"},
	},
}

var GetTickerOp = Op{
	Name: "GetTicker", Method: "GET", Path: "/ticker/{symbol}",
	Summary: "Get ticker-compatible historical candles",
	Long:    "Returns broker historical candles for an EXCHANGE:SYMBOL pair in JSON or, with --format txt, comma-separated plain text written verbatim to stdout. The range is capped to 30 days for intraday intervals and 10 years for D, W and M.",
	Example: `  openalgo data ticker --symbol NSE:RELIANCE --from 2026-09-01 --to 2026-09-25
  openalgo data ticker --symbol NSE:SBIN --interval 5m --from 2026-09-24 --to 2026-09-25 --format txt`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "from", OASName: "from", Type: "string", Description: "start date in YYYY-MM-DD format", Required: true, Source: "query"},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "exchange and symbol separated by one colon, e.g. NSE:RELIANCE", Required: true, Source: "path"},
		{Name: "to", OASName: "to", Type: "string", Description: "end date in YYYY-MM-DD format", Required: true, Source: "query"},
		{Name: "format", OASName: "format", Type: "string", Default: "json", Description: "response format: json or txt (comma-separated plain text rows)", Completions: []string{"json", "txt"}, Source: "query"},
		{Name: "interval", OASName: "interval", Type: "string", Default: "D", Description: "candle interval", Completions: []string{"1s", "5s", "10s", "15s", "30s", "45s", "1m", "2m", "3m", "5m", "10m", "15m", "20m", "30m", "1h", "2h", "3h", "4h", "D", "W", "M", "Q", "Y"}, Source: "query"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "[]object", Description: "OHLCV candles", Fields: []ResponseField{
			{Name: "timestamp", Type: "integer", Description: "candle open time as a Unix timestamp in seconds"},
			{Name: "open", Type: "number", Description: "opening price"},
			{Name: "high", Type: "number", Description: "highest price"},
			{Name: "low", Type: "number", Description: "lowest price"},
			{Name: "close", Type: "number", Description: "closing price"},
			{Name: "volume", Type: "number", Description: "volume traded"},
			{Name: "oi", Type: "number", Description: "open interest, 0 when not applicable"},
		}},
	},
}

var GetTradebookOp = Op{
	Name: "GetTradebook", Method: "POST", Path: "/tradebook",
	Summary: "Get the trade book",
	Long:    "Returns all executed trades for the current trading day. A single order may appear multiple times when it was partially filled",
	Example: `  openalgo order trades
  openalgo order trades --csv`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "error or informational message, present on failure"},
		{Name: "data", Type: "[]object", Description: "executed trades", Fields: []ResponseField{
			{Name: "orderid", Type: "string", Description: "order id that generated this trade"},
			{Name: "symbol", Type: "string", Description: "trading symbol"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "action", Type: "string", Description: "BUY or SELL"},
			{Name: "quantity", Type: "number", Description: "traded quantity"},
			{Name: "average_price", Type: "number", Description: "execution price"},
			{Name: "product", Type: "string", Description: "MIS, CNC or NRML"},
			{Name: "timestamp", Type: "string", Description: "trade execution time"},
			{Name: "trade_value", Type: "number", Description: "total trade value"},
		}},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var ModifyGTTOrderOp = Op{
	Name: "ModifyGTTOrder", Method: "POST", Path: "/modifygttorder",
	Summary: "Modify an active GTT order",
	Long:    "Modify an active GTT trigger. Trigger ids are strings: numeric for live brokers, GTT-<date>-<hex> in analyzer mode.",
	Example: `  openalgo gtt modify --trigger-id 123456789 --trigger-type SINGLE --symbol SBIN --exchange NSE --action BUY --product CNC --quantity 2 --price 751 --triggerprice-sl 750
  openalgo gtt modify --trigger-id GTT-260926-0e55ab27 --trigger-type SINGLE --symbol SBIN --exchange NSE --action BUY --product CNC --quantity 1 --price 752 --triggerprice-sl 751`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "action", OASName: "action", Type: "string", Description: "order action; for OCO applies to both legs", Completions: []string{"BUY", "SELL"}, Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Description: "SINGLE child order limit price; send 0 for MARKET, ignored for OCO", Required: true, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Description: "product type; GTT supports only CNC or NRML", Completions: []string{"CNC", "NRML"}, Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "number", Description: "order quantity; fractional values allowed only for CRYPTO", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol in OpenAlgo format", Required: true, Source: "body"},
		{Name: "trigger-id", OASName: "trigger_id", Type: "string", Description: "id of the active GTT trigger to modify", Required: true, Source: "body"},
		{Name: "trigger-type", OASName: "trigger_type", Type: "string", Description: "SINGLE for one trigger or OCO for a stoploss plus target pair; must match the original", Completions: []string{"SINGLE", "OCO"}, Required: true, Source: "body"},
		{Name: "pricetype", OASName: "pricetype", Type: "string", Default: "LIMIT", Description: "child order price type", Completions: []string{"LIMIT", "MARKET"}, Source: "body"},
		{Name: "stoploss", OASName: "stoploss", Type: "number", Description: "OCO only: limit price of the stoploss leg child order", Source: "body"},
		{Name: "target", OASName: "target", Type: "number", Description: "OCO only: limit price of the target leg child order", Source: "body"},
		{Name: "triggerprice-sl", OASName: "triggerprice_sl", Type: "number", Default: "0", Description: "trigger below LTP; SINGLE uses this or triggerprice_tg, OCO requires it as the stoploss-leg trigger", Source: "body"},
		{Name: "triggerprice-tg", OASName: "triggerprice_tg", Type: "number", Default: "0", Description: "trigger above LTP; SINGLE uses this or triggerprice_sl, OCO requires it as the target-leg trigger", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "trigger_id", Type: "string", Description: "GTT trigger id"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var ModifyOrderOp = Op{
	Name: "ModifyOrder", Method: "POST", Path: "/modifyorder",
	Summary: "Modify an open order",
	Long:    "Modify price, quantity, trigger price or price type of an open or pending order. All fields are required even when unchanged",
	Example: `  openalgo order modify --orderid 250408000989443 --symbol SBIN --exchange NSE --action BUY --product CNC --pricetype LIMIT --price 780 --quantity 1
  openalgo order modify --orderid 250408000989443 --symbol SBIN --exchange NSE --action BUY --product CNC --pricetype SL --price 781 --trigger-price 780 --quantity 1`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "action", OASName: "action", Type: "string", Description: "order action; cannot be changed", Completions: []string{"BUY", "SELL"}, Required: true, Source: "body"},
		{Name: "disclosed-quantity", OASName: "disclosed_quantity", Type: "int", Default: "0", Description: "new disclosed quantity; send 0 when unused", Required: true, Source: "body", CLIDefault: true},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "orderid", OASName: "orderid", Type: "string", Description: "broker order id to modify", Required: true, Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Description: "new order price", Required: true, Source: "body"},
		{Name: "pricetype", OASName: "pricetype", Type: "string", Description: "price type", Completions: []string{"MARKET", "LIMIT", "SL", "SL-M"}, Required: true, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Description: "product type; cannot be changed", Completions: []string{"MIS", "CNC", "NRML"}, Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "number", Description: "new order quantity; fractional values allowed only for CRYPTO", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol of the order", Required: true, Source: "body"},
		{Name: "trigger-price", OASName: "trigger_price", Type: "number", Default: "0", Description: "new trigger price; send 0 when unused", Required: true, Source: "body", CLIDefault: true},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var OptionsMultiOrderOp = Op{
	Name: "OptionsMultiOrder", Method: "POST", Path: "/optionsmultiorder",
	Summary: "Place a multi-leg options order",
	Long:    "Place 1 to 20 option legs sharing a common underlying, each resolved from its strike offset. BUY legs execute before SELL legs and a failed leg does not stop later legs. Leg quantities must be a multiple of the lot size shown by 'openalgo option symbol'.",
	Example: `  openalgo option multi-order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --legs '[{"offset":"OTM2","option_type":"CE","action":"BUY","quantity":65,"product":"NRML"},{"offset":"OTM2","option_type":"PE","action":"BUY","quantity":65,"product":"NRML"},{"offset":"ATM","option_type":"CE","action":"SELL","quantity":65,"product":"NRML"},{"offset":"ATM","option_type":"PE","action":"SELL","quantity":65,"product":"NRML"}]'
  openalgo option multi-order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --legs @legs.json`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange of the underlying, typically NSE_INDEX, BSE_INDEX, NSE or BSE", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "legs", OASName: "legs", Type: "array", Description: "option legs (1 to 20); JSON array of {offset, option_type, action, quantity} objects; literal, @file, or - for stdin", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "underlying", OASName: "underlying", Type: "string", Description: "underlying symbol, e.g. NIFTY, BANKNIFTY, RELIANCE", Required: true, Source: "body"},
		{Name: "expiry-date", OASName: "expiry_date", Type: "string", Description: "common expiry in DDMMMYY format (DD-MMM-YY also accepted); can be overridden per leg", Source: "body"},
		{Name: "strike-int", OASName: "strike_int", Type: "int", Description: "optional strike interval", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "underlying", Type: "string", Description: "underlying symbol"},
		{Name: "underlying_ltp", Type: "number", Description: "last traded price of the underlying used for all legs"},
		{Name: "results", Type: "[]object", Description: "per-leg results", Fields: []ResponseField{
			{Name: "leg", Type: "integer", Description: "leg number starting at 1"},
			{Name: "symbol", Type: "string", Description: "resolved option symbol"},
			{Name: "exchange", Type: "string", Description: "exchange where the leg was placed"},
			{Name: "product", Type: "string", Description: "product type of the leg"},
			{Name: "offset", Type: "string", Description: "offset used"},
			{Name: "option_type", Type: "string", Description: "CE or PE"},
			{Name: "action", Type: "string", Description: "BUY or SELL"},
			{Name: "orderid", Type: "string", Description: "broker order id (non-split legs, on success)"},
			{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
			{Name: "message", Type: "string", Description: "error or informational message"},
			{Name: "total_quantity", Type: "integer", Description: "total leg quantity (split legs only)"},
			{Name: "split_size", Type: "integer", Description: "split size used (split legs only)"},
			{Name: "split_results", Type: "[]object", Description: "child order results (split legs only)", Fields: []ResponseField{
				{Name: "order_num", Type: "integer", Description: "order sequence number"},
				{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
				{Name: "quantity", Type: "integer", Description: "quantity of this child order"},
				{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
				{Name: "message", Type: "string", Description: "error or informational message"},
			}},
			{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
		}},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var OptionsOrderOp = Op{
	Name: "OptionsOrder", Method: "POST", Path: "/optionsorder",
	Summary: "Place an options order by strike offset",
	Long:    "Resolve an option symbol from the underlying, expiry and ATM/ITM/OTM offset, then place the order. Index underlyings are placed on NFO or BFO; --splitsize greater than 0 returns per-child results. --quantity must be a multiple of the lot size shown by 'openalgo option symbol'. --expiry-date takes DDMMMYY (27OCT26); DD-MMM-YY from 'openalgo symbol expiry' is converted.",
	Example: `  openalgo option order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ATM --option-type CE --action BUY --quantity 65 --product NRML
  openalgo option order --underlying BANKNIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset OTM2 --option-type PE --action SELL --quantity 30 --product NRML --pricetype LIMIT --price 120`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "action", OASName: "action", Type: "string", Description: "order action", Completions: []string{"BUY", "SELL"}, Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange of the underlying, typically NSE_INDEX, BSE_INDEX, NSE, BSE, NFO or BFO", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "offset", OASName: "offset", Type: "string", Description: "strike offset relative to ATM: ATM, ITM1 to ITM50 or OTM1 to OTM50", Required: true, Source: "body"},
		{Name: "option-type", OASName: "option_type", Type: "string", Description: "option type, CE for call or PE for put", Completions: []string{"CE", "PE"}, Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "int", Description: "order quantity", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "underlying", OASName: "underlying", Type: "string", Description: "underlying symbol, e.g. NIFTY, BANKNIFTY, RELIANCE or NIFTY28NOV24FUT", Required: true, Source: "body"},
		{Name: "disclosed-quantity", OASName: "disclosed_quantity", Type: "int", Default: "0", Description: "disclosed quantity", Source: "body"},
		{Name: "expiry-date", OASName: "expiry_date", Type: "string", Description: "expiry date in DDMMMYY format (DD-MMM-YY also accepted), e.g. 25AUG26; optional if underlying includes expiry", Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Default: "0", Description: "limit price, for LIMIT and SL orders", Source: "body"},
		{Name: "pricetype", OASName: "pricetype", Type: "string", Default: "MARKET", Description: "price type", Completions: []string{"MARKET", "LIMIT", "SL", "SL-M"}, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Default: "MIS", Description: "product type; options support only MIS or NRML", Completions: []string{"MIS", "NRML"}, Source: "body"},
		{Name: "splitsize", OASName: "splitsize", Type: "int", Default: "0", Description: "if greater than 0, split the order into child orders of this size", Source: "body"},
		{Name: "strike-int", OASName: "strike_int", Type: "int", Description: "optional strike interval; when omitted actual strikes from the database are used", Source: "body"},
		{Name: "trigger-price", OASName: "trigger_price", Type: "number", Default: "0", Description: "trigger price, for SL and SL-M orders", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "orderid", Type: "string", Description: "broker order id (non-split orders)"},
		{Name: "symbol", Type: "string", Description: "resolved option symbol"},
		{Name: "exchange", Type: "string", Description: "exchange where the order was placed, e.g. NFO or BFO"},
		{Name: "offset", Type: "string", Description: "offset used for resolution"},
		{Name: "option_type", Type: "string", Description: "CE or PE"},
		{Name: "underlying", Type: "string", Description: "underlying symbol used for price reference"},
		{Name: "underlying_ltp", Type: "number", Description: "last traded price of the underlying"},
		{Name: "total_quantity", Type: "integer", Description: "total quantity (split orders only)"},
		{Name: "split_size", Type: "integer", Description: "split size used (split orders only)"},
		{Name: "results", Type: "[]object", Description: "child order results (split orders only)", Fields: []ResponseField{
			{Name: "order_num", Type: "integer", Description: "order sequence number"},
			{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
			{Name: "quantity", Type: "integer", Description: "quantity of this child order"},
			{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
			{Name: "message", Type: "string", Description: "error or informational message"},
		}},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var PingOp = Op{
	Name: "Ping", Method: "POST", Path: "/ping",
	Summary:  "Ping the API",
	Long:     "Verify that the API key resolves to an active broker session. Not an anonymous health check; an invalid key or missing broker session returns HTTP 403",
	Example:  `  openalgo ping`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "object", Description: "ping result", Fields: []ResponseField{
			{Name: "message", Type: "string", Description: "always pong"},
			{Name: "broker", Type: "string", Description: "active broker name"},
		}},
	},
}

var PlaceGTTOrderOp = Op{
	Name: "PlaceGTTOrder", Method: "POST", Path: "/placegttorder",
	Summary: "Place a GTT order",
	Long:    "Place a Good Till Triggered order that fires a child order when LTP crosses the trigger. OCO requires triggerprice_sl, stoploss, triggerprice_tg and target with triggerprice_sl below triggerprice_tg; on Upstox an OCO opens a position at market first",
	Example: `  openalgo gtt place --trigger-type SINGLE --symbol SBIN --exchange NSE --action BUY --product CNC --quantity 1 --pricetype LIMIT --price 756 --triggerprice-sl 755
  openalgo gtt place --trigger-type OCO --symbol SBIN --exchange NSE --action SELL --product CNC --quantity 1 --price 0 --triggerprice-sl 740 --stoploss 739 --triggerprice-tg 860 --target 861`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "action", OASName: "action", Type: "string", Description: "order action; for OCO applies to both legs", Completions: []string{"BUY", "SELL"}, Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Description: "SINGLE child order limit price; send 0 for MARKET, ignored for OCO", Required: true, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Description: "product type; GTT supports only CNC or NRML", Completions: []string{"CNC", "NRML"}, Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "number", Description: "order quantity; fractional values allowed only for CRYPTO", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol in OpenAlgo format", Required: true, Source: "body"},
		{Name: "trigger-type", OASName: "trigger_type", Type: "string", Description: "SINGLE for one trigger or OCO for a stoploss plus target pair", Completions: []string{"SINGLE", "OCO"}, Required: true, Source: "body"},
		{Name: "expires-at", OASName: "expires_at", Type: "string", Description: "optional expiry timestamp for the trigger where the broker supports it", Source: "body"},
		{Name: "pricetype", OASName: "pricetype", Type: "string", Default: "LIMIT", Description: "child order price type", Completions: []string{"LIMIT", "MARKET"}, Source: "body"},
		{Name: "stoploss", OASName: "stoploss", Type: "number", Description: "OCO only: limit price of the stoploss leg child order", Source: "body"},
		{Name: "target", OASName: "target", Type: "number", Description: "OCO only: limit price of the target leg child order", Source: "body"},
		{Name: "triggerprice-sl", OASName: "triggerprice_sl", Type: "number", Default: "0", Description: "trigger below LTP; SINGLE uses this or triggerprice_tg, OCO requires it as the stoploss-leg trigger", Source: "body"},
		{Name: "triggerprice-tg", OASName: "triggerprice_tg", Type: "number", Default: "0", Description: "trigger above LTP; SINGLE uses this or triggerprice_sl, OCO requires it as the target-leg trigger", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "trigger_id", Type: "string", Description: "GTT trigger id"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var PlaceOrderOp = Op{
	Name: "PlaceOrder", Method: "POST", Path: "/placeorder",
	Summary: "Place an order",
	Long:    "Place a new order with the broker. In analyzer mode the order is routed to the sandbox and the response carries a mode field. When --product or --pricetype is omitted the server uses MIS and MARKET; in analyzer (sandbox) mode MIS is refused after the 15:15 IST square-off, so prefer CNC for equity and NRML for F&O. F&O quantities must be a multiple of the lot size shown by 'openalgo symbol get --symbol <symbol> --exchange NFO --jq .data.lotsize'.",
	Example: `  openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --product CNC --pricetype MARKET
  openalgo order place --symbol RELIANCE --exchange NSE --action BUY --quantity 5 --product CNC --pricetype LIMIT --price 2800
  openalgo order place --symbol NIFTY27OCT26FUT --exchange NFO --action SELL --quantity 65 --product NRML --pricetype SL --price 24990 --trigger-price 25000
  openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --product CNC --dry-run`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "action", OASName: "action", Type: "string", Description: "order action", Completions: []string{"BUY", "SELL"}, Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "number", Description: "positive order quantity; fractional values allowed only for CRYPTO", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol in OpenAlgo format, e.g. RELIANCE or NIFTY25AUG26FUT", Required: true, Source: "body"},
		{Name: "disclosed-quantity", OASName: "disclosed_quantity", Type: "int", Default: "0", Description: "disclosed quantity for iceberg orders", Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Default: "0", Description: "order price, required for LIMIT and SL orders", Source: "body"},
		{Name: "pricetype", OASName: "pricetype", Type: "string", Default: "MARKET", Description: "price type", Completions: []string{"MARKET", "LIMIT", "SL", "SL-M"}, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Default: "MIS", Description: "product type", Completions: []string{"MIS", "CNC", "NRML"}, Source: "body"},
		{Name: "trigger-price", OASName: "trigger_price", Type: "number", Default: "0", Description: "trigger price, required for SL and SL-M orders", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var PlaceSmartOrderOp = Op{
	Name: "PlaceSmartOrder", Method: "POST", Path: "/placesmartorder",
	Summary: "Place a position-aware smart order",
	Long:    "Place an order sized to move the current open position for the symbol to --position-size (positive long, negative short, 0 flat). No order is placed if the position already matches. When --product or --pricetype is omitted the server uses MIS and MARKET; in analyzer (sandbox) mode MIS is refused after the 15:15 IST square-off, so prefer CNC for equity and NRML for F&O. F&O quantities must be a multiple of the lot size shown by 'openalgo symbol get --symbol <symbol> --exchange NFO --jq .data.lotsize'.",
	Example: `  openalgo order smart --symbol SBIN --exchange NSE --action BUY --quantity 1 --position-size 5 --product CNC
  openalgo order smart --symbol SBIN --exchange NSE --action SELL --quantity 1 --position-size 0 --product CNC`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "action", OASName: "action", Type: "string", Description: "order action", Completions: []string{"BUY", "SELL"}, Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "position-size", OASName: "position_size", Type: "number", Description: "target net position size; positive for long, negative for short, zero for flat", Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "number", Description: "order quantity; fractional values allowed only for CRYPTO", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol in OpenAlgo format", Required: true, Source: "body"},
		{Name: "disclosed-quantity", OASName: "disclosed_quantity", Type: "int", Default: "0", Description: "disclosed quantity for iceberg orders", Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Default: "0", Description: "order price, required for LIMIT and SL orders", Source: "body"},
		{Name: "pricetype", OASName: "pricetype", Type: "string", Default: "MARKET", Description: "price type", Completions: []string{"MARKET", "LIMIT", "SL", "SL-M"}, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Default: "MIS", Description: "product type", Completions: []string{"MIS", "CNC", "NRML"}, Source: "body"},
		{Name: "trigger-price", OASName: "trigger_price", Type: "number", Default: "0", Description: "trigger price, required for SL and SL-M orders", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var SearchSymbolsOp = Op{
	Name: "SearchSymbols", Method: "POST", Path: "/search",
	Summary: "Search symbols",
	Long:    "Search the master contract for symbols matching a free-text query such as name, strike, month and option type. Search is case-insensitive and results are limited",
	Example: `  openalgo symbol search --query RELIANCE
  openalgo symbol search --query "NIFTY 26000 OCT CE" --exchange NFO`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "query", OASName: "query", Type: "string", Description: "search query string, e.g. NIFTY 26000 DEC CE", Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "optional exchange filter", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "[]object", Description: "matching symbols", Fields: []ResponseField{
			{Name: "symbol", Type: "string", Description: "openAlgo standard symbol"},
			{Name: "brsymbol", Type: "string", Description: "broker-specific symbol"},
			{Name: "name", Type: "string", Description: "underlying or symbol name"},
			{Name: "exchange", Type: "string", Description: "openAlgo exchange code"},
			{Name: "brexchange", Type: "string", Description: "broker-specific exchange code"},
			{Name: "instrumenttype", Type: "string", Description: "instrument type such as EQ, FUT, CE or PE (may be empty for equity)"},
			{Name: "expiry", Type: "string", Description: "expiry date in DD-MMM-YY format, empty for non-derivatives"},
			{Name: "strike", Type: "number", Description: "strike price (-0.01 or 0 for non-options)"},
			{Name: "lotsize", Type: "integer", Description: "lot size (1 for equity)"},
			{Name: "tick_size", Type: "number", Description: "minimum price movement"},
			{Name: "freeze_qty", Type: "number", Description: "maximum quantity allowed per order"},
			{Name: "token", Type: "string", Description: "broker-specific instrument token"},
		}},
	},
}

var SplitOrderOp = Op{
	Name: "SplitOrder", Method: "POST", Path: "/splitorder",
	Summary: "Split a large order into smaller orders",
	Long:    "Place a large quantity as sequential child orders of --splitsize each, with the remainder in the last order. At most 100 child orders are allowed per request. When --product or --pricetype is omitted the server uses MIS and MARKET; in analyzer (sandbox) mode MIS is refused after the 15:15 IST square-off, so prefer CNC for equity and NRML for F&O. F&O quantities must be a multiple of the lot size shown by 'openalgo symbol get --symbol <symbol> --exchange NFO --jq .data.lotsize'.",
	Example: `  openalgo order split --symbol YESBANK --exchange NSE --action BUY --quantity 105 --splitsize 20 --product CNC
  openalgo order split --symbol NIFTY27OCT26FUT --exchange NFO --action BUY --quantity 650 --splitsize 130 --product NRML`,
	Mutating: true,
	RowsPath: "results",
	Flags: []FlagDef{
		{Name: "action", OASName: "action", Type: "string", Description: "order action", Completions: []string{"BUY", "SELL"}, Required: true, Source: "body"},
		{Name: "exchange", OASName: "exchange", Type: "string", Description: "exchange code", Completions: []string{"NSE", "BSE", "NFO", "BFO", "CDS", "BCD", "MCX", "NCDEX", "NCO", "NSE_INDEX", "BSE_INDEX", "MCX_INDEX", "GLOBAL_INDEX", "CRYPTO"}, Required: true, Source: "body"},
		{Name: "quantity", OASName: "quantity", Type: "number", Description: "total quantity to split; fractional values allowed only for CRYPTO", Required: true, Source: "body"},
		{Name: "splitsize", OASName: "splitsize", Type: "int", Description: "quantity of each child order", Required: true, Source: "body"},
		{Name: "strategy", OASName: "strategy", Type: "string", Default: "openalgo-cli", Description: "strategy identifier used for tracking", Required: true, Source: "body", CLIDefault: true},
		{Name: "symbol", OASName: "symbol", Type: "string", Description: "trading symbol in OpenAlgo format", Required: true, Source: "body"},
		{Name: "disclosed-quantity", OASName: "disclosed_quantity", Type: "int", Default: "0", Description: "disclosed quantity for iceberg orders", Source: "body"},
		{Name: "price", OASName: "price", Type: "number", Default: "0", Description: "order price, for LIMIT orders", Source: "body"},
		{Name: "pricetype", OASName: "pricetype", Type: "string", Default: "MARKET", Description: "price type", Completions: []string{"MARKET", "LIMIT", "SL", "SL-M"}, Source: "body"},
		{Name: "product", OASName: "product", Type: "string", Default: "MIS", Description: "product type", Completions: []string{"MIS", "CNC", "NRML"}, Source: "body"},
		{Name: "trigger-price", OASName: "trigger_price", Type: "number", Default: "0", Description: "trigger price, required for SL and SL-M orders", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
		{Name: "split_size", Type: "number", Description: "split size used"},
		{Name: "total_quantity", Type: "number", Description: "total quantity processed"},
		{Name: "results", Type: "[]object", Description: "child order results", Fields: []ResponseField{
			{Name: "order_num", Type: "integer", Description: "order sequence number"},
			{Name: "orderid", Type: "string", Description: "broker order id (on success)"},
			{Name: "quantity", Type: "integer", Description: "quantity of this child order"},
			{Name: "status", Type: "string", Description: "request outcome", EnumValues: []string{"success", "error"}},
			{Name: "message", Type: "string", Description: "error or informational message"},
		}},
		{Name: "message", Type: "string", Description: "error or informational message"},
		{Name: "mode", Type: "string", Description: "execution mode, live or analyze (analyzer mode)"},
	},
}

var StrategyCloseAllOp = Op{
	Name: "StrategyCloseAll", Method: "POST", Path: "/strategy/close_all",
	Summary:  "Close all legs of a strategy run",
	Long:     "Exits ALL positions owned by the strategy's current run at market and stops the run. Positions outside the strategy are not touched.",
	Example:  `  openalgo strategy close-all --strategy-id 12`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "run_id", Type: "integer", Description: "the run the stop applies to; it stays current while stop_pending is true"},
		{Name: "stop_pending", Type: "boolean", Description: "true while owned exposure still needs a fill, retry or reconciliation; false only once flatness was confirmed"},
		{Name: "exits", Type: "[]object", Description: "per-leg exit outcome", Fields: []ResponseField{
			{Name: "leg_id", Type: "integer", Description: "the leg's id within the strategy"},
			{Name: "ok", Type: "boolean", Description: "whether the exit order was accepted"},
			{Name: "position_ref", Type: "string|null", Description: "exact durable owner the exit targets"},
			{Name: "exit_owner", Type: "string", Description: "owner side being exited", EnumValues: []string{"live", "superseded"}},
			{Name: "symbol", Type: "string", Description: "held symbol, included when an unfilled entry is refused before dispatch"},
			{Name: "broker_order_id", Type: "string|null", Description: "broker order id when available"},
			{Name: "error", Type: "string|null", Description: "rejection reason when ok is false"},
		}},
	},
}

var StrategyCloseLegOp = Op{
	Name: "StrategyCloseLeg", Method: "POST", Path: "/strategy/close_leg",
	Summary:  "Close one leg of a strategy run",
	Long:     "Exit one leg of the current run at market while the run continues with the rest. A leg that is not open returns 409",
	Example:  `  openalgo strategy close-leg --strategy-id 12 --leg-id 2`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "leg-id", OASName: "leg_id", Type: "int", Description: "the leg's 1-based id within the strategy", Required: true, Source: "body"},
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "run_id", Type: "integer", Description: "the run the leg belongs to"},
		{Name: "leg_id", Type: "integer", Description: "the leg that was closed, echoed back"},
		{Name: "run_stopped", Type: "boolean", Description: "true only when this call observed the last owner fill and finalised the run"},
		{Name: "exits", Type: "[]object", Description: "exit outcome for the leg", Fields: []ResponseField{
			{Name: "leg_id", Type: "integer", Description: "the leg's id within the strategy"},
			{Name: "ok", Type: "boolean", Description: "whether the exit order was accepted"},
			{Name: "position_ref", Type: "string|null", Description: "exact durable owner the exit targets"},
			{Name: "exit_owner", Type: "string", Description: "owner side being exited", EnumValues: []string{"live", "superseded"}},
			{Name: "symbol", Type: "string", Description: "held symbol, included when an unfilled entry is refused before dispatch"},
			{Name: "broker_order_id", Type: "string|null", Description: "broker order id when available"},
			{Name: "error", Type: "string|null", Description: "rejection reason when ok is false"},
		}},
	},
}

var StrategyEventsOp = Op{
	Name: "StrategyEvents", Method: "POST", Path: "/strategy/events",
	Summary: "List a strategy's risk events",
	Long:    "Return the append-only risk-event audit trail for a strategy, newest first. Out-of-vocabulary kind or severity, or limit outside 1 to 1000, is a 400",
	Example: `  openalgo strategy events --strategy-id 12
  openalgo strategy events --strategy-id 12 --severity critical --limit 50`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
		{Name: "kind", OASName: "kind", Type: "string", Description: "filter by event kind; omit for no filter", Completions: []string{"strategy_created", "strategy_updated", "webhook_token_rotated", "live_enabled", "live_disabled", "webhook_locked", "webhook_unlocked", "run_started", "run_paused", "run_resumed", "run_stop_requested", "run_stopped", "run_stop_failed", "flip_outgoing_exit_rejected", "close_all_manual", "leg_entry_placed", "leg_entry_filled", "leg_entry_rejected", "leg_exit_placed", "leg_exit_filled", "leg_exit_rejected", "leg_close_manual", "leg_expiry_fallback", "order_ack_unrecorded", "leg_sl_hit", "leg_target_hit", "leg_trail_armed", "leg_trail_advanced", "overall_sl_hit", "overall_target_hit", "lock_profit_armed", "lock_profit_floor_advanced", "lock_profit_triggered", "trail_to_entry_activated", "eod_squareoff", "expiry_squareoff", "tick_source_switched_to_polling", "tick_source_switched_to_ws", "tick_source_stale", "recovery_succeeded", "recovery_failed"}, Source: "body"},
		{Name: "limit", OASName: "limit", Type: "int", Default: "500", Description: "how many events to return, 1 to 1000", Source: "body"},
		{Name: "run-id", OASName: "run_id", Type: "int", Description: "narrow the result to one run; omit for every run", Source: "body"},
		{Name: "severity", OASName: "severity", Type: "string", Description: "filter by severity; omit for no filter", Completions: []string{"info", "warn", "critical"}, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "[]object", Description: "event objects, newest first by timestamp", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "event row id"},
			{Name: "run_id", Type: "integer|null", Description: "run the event belongs to, null for configuration-layer events"},
			{Name: "strategy_id", Type: "integer", Description: "owning strategy"},
			{Name: "ts", Type: "string", Description: "ISO 8601 UTC timestamp"},
			{Name: "kind", Type: "string", Description: "what happened", EnumValues: []string{"strategy_created", "strategy_updated", "webhook_token_rotated", "live_enabled", "live_disabled", "webhook_locked", "webhook_unlocked", "run_started", "run_paused", "run_resumed", "run_stop_requested", "run_stopped", "run_stop_failed", "flip_outgoing_exit_rejected", "close_all_manual", "leg_entry_placed", "leg_entry_filled", "leg_entry_rejected", "leg_exit_placed", "leg_exit_filled", "leg_exit_rejected", "leg_close_manual", "leg_expiry_fallback", "order_ack_unrecorded", "leg_sl_hit", "leg_target_hit", "leg_trail_armed", "leg_trail_advanced", "overall_sl_hit", "overall_target_hit", "lock_profit_armed", "lock_profit_floor_advanced", "lock_profit_triggered", "trail_to_entry_activated", "eod_squareoff", "expiry_squareoff", "tick_source_switched_to_polling", "tick_source_switched_to_ws", "tick_source_stale", "recovery_succeeded", "recovery_failed"}},
			{Name: "severity", Type: "string", Description: "event severity", EnumValues: []string{"info", "warn", "critical"}},
			{Name: "leg_id", Type: "integer|null", Description: "leg the event concerns, when it concerns one"},
			{Name: "message", Type: "string", Description: "human-readable description"},
			{Name: "payload", Type: "object|null", Description: "structured detail, null on most events; shape varies by kind"},
		}},
	},
}

var StrategyListOp = Op{
	Name: "StrategyList", Method: "POST", Path: "/strategy/list",
	Summary: "List strategies",
	Long:    "List the strategies owned by this API key, newest first. The list form omits legs; an out-of-vocabulary status filter is a 400",
	Example: `  openalgo strategy list
  openalgo strategy list --status running
  openalgo strategy list --query straddle --csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "query", OASName: "q", Type: "string", Description: "case-insensitive substring match on the strategy name, at most 100 characters", Source: "body"},
		{Name: "status", OASName: "status", Type: "string", Description: "filter by strategy status; omit for no filter", Completions: []string{"stopped", "running", "paused", "errored"}, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "[]object", Description: "strategy objects, newest first", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "strategy id, used as strategy_id on every other strategy endpoint"},
			{Name: "name", Type: "string", Description: "strategy name, unique per user"},
			{Name: "strategy_kind", Type: "string", Description: "strategy kind", EnumValues: []string{"batch", "signal"}},
			{Name: "direction", Type: "string", Description: "allowed direction, signal mode only", EnumValues: []string{"both", "long_only", "short_only"}},
			{Name: "universe_tab", Type: "string", Description: "instrument universe the strategy was built from", EnumValues: []string{"weekly_monthly", "monthly_only", "stocks_fno", "mcx"}},
			{Name: "underlying", Type: "string", Description: "underlying symbol"},
			{Name: "underlying_exchange", Type: "string", Description: "exchange the underlying is quoted on"},
			{Name: "strategy_type", Type: "string", Description: "strategy type", EnumValues: []string{"intraday", "positional"}},
			{Name: "entry_time", Type: "string|null", Description: "IST entry time as HH:MM"},
			{Name: "exit_time", Type: "string|null", Description: "IST square-off time as HH:MM"},
			{Name: "product", Type: "string", Description: "configured product intent (MIS intraday, anything else carry)", EnumValues: []string{"CNC", "NRML", "MIS"}},
			{Name: "pricetype", Type: "string", Description: "order price type, always MARKET"},
			{Name: "overall_sl_mtm", Type: "number|null", Description: "strategy-level stop loss in rupees of MTM"},
			{Name: "overall_target_mtm", Type: "number|null", Description: "strategy-level target in rupees of MTM"},
			{Name: "lock_profit", Type: "object|null", Description: "lock profit configuration, or null", Fields: []ResponseField{
				{Name: "mode", Type: "string", Description: "lock profit mode, e.g. lock_and_trail"},
				{Name: "if_profit_reaches", Type: "number", Description: "profit level that arms the lock"},
				{Name: "lock_profit", Type: "number", Description: "profit amount locked once armed"},
				{Name: "trail_step", Type: "number", Description: "step by which the locked floor trails"},
			}},
			{Name: "trail_sl_to_entry", Type: "boolean", Description: "whether a stop on one leg trails the others to entry"},
			{Name: "scheduler", Type: "object|null", Description: "scheduler configuration, or null", Fields: []ResponseField{
				{Name: "enabled", Type: "boolean", Description: "whether the built-in scheduler is enabled"},
				{Name: "days", Type: "[]string", Description: "weekdays the scheduler runs on, e.g. MON"},
				{Name: "start_time", Type: "string", Description: "IST start time as HH:MM"},
				{Name: "auto_stop_time", Type: "string", Description: "IST auto stop time as HH:MM"},
				{Name: "default_mode", Type: "string", Description: "mode used for scheduled starts", EnumValues: []string{"live", "sandbox"}},
			}},
			{Name: "live_enabled", Type: "boolean", Description: "whether this strategy may run in live mode"},
			{Name: "webhook_locked", Type: "boolean", Description: "whether the webhook kill switch is engaged"},
			{Name: "webhook_ip_allowlist", Type: "[]string|null", Description: "CIDR ranges allowed to trigger the webhook"},
			{Name: "daily_loss_limit_inr", Type: "number|null", Description: "daily loss ceiling in rupees"},
			{Name: "status", Type: "string", Description: "strategy status", EnumValues: []string{"stopped", "running", "paused", "errored"}},
			{Name: "current_run_id", Type: "integer|null", Description: "the run this strategy is executing, if any"},
			{Name: "created_at", Type: "string", Description: "ISO 8601 UTC creation time"},
			{Name: "updated_at", Type: "string", Description: "ISO 8601 UTC last update time"},
			{Name: "last_finalized_run", Type: "object|null", Description: "most recently finalised run, or null", Fields: []ResponseField{
				{Name: "id", Type: "integer", Description: "run id"},
				{Name: "pnl_realized", Type: "number", Description: "durable final realized P&L of that run"},
				{Name: "stopped_at", Type: "string", Description: "ISO 8601 UTC stop time"},
			}},
		}},
	},
}

var StrategyOrdersOp = Op{
	Name: "StrategyOrders", Method: "POST", Path: "/strategy/orders",
	Summary: "List a strategy's orders",
	Long:    "Return every order the engine placed across a strategy's runs, oldest first, optionally narrowed to one run. A run_id from another strategy matches nothing",
	Example: `  openalgo strategy orders --strategy-id 12
  openalgo strategy orders --strategy-id 12 --run-id 40 --csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
		{Name: "run-id", OASName: "run_id", Type: "int", Description: "narrow the result to one run; omit for every run", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "[]object", Description: "order objects, oldest first by placement time", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "strategy order row id"},
			{Name: "run_id", Type: "integer", Description: "run the order belongs to"},
			{Name: "leg_id", Type: "integer", Description: "leg within the strategy"},
			{Name: "kind", Type: "string", Description: "why the order was placed", EnumValues: []string{"entry", "exit_sl", "exit_target", "exit_trail", "exit_overall_sl", "exit_overall_target", "exit_lock_profit", "exit_eod", "exit_expiry", "exit_daily_loss_limit", "exit_close_all", "exit_leg_manual", "exit_recovery", "exit_signal"}},
			{Name: "position_ref", Type: "string|null", Description: "durable owner identity"},
			{Name: "broker_order_id", Type: "string|null", Description: "broker or sandbox order id, null before the order path answered"},
			{Name: "symbol", Type: "string", Description: "openAlgo symbol"},
			{Name: "exchange", Type: "string", Description: "exchange code"},
			{Name: "action", Type: "string", Description: "order side", EnumValues: []string{"BUY", "SELL"}},
			{Name: "qty", Type: "integer", Description: "quantity sent"},
			{Name: "product", Type: "string|null", Description: "product actually sent to the venue"},
			{Name: "pricetype", Type: "string", Description: "order price type, MARKET"},
			{Name: "price", Type: "number", Description: "order price, 0 for a market order"},
			{Name: "trigger_price", Type: "number", Description: "trigger price, 0 when unused"},
			{Name: "status", Type: "string", Description: "order status", EnumValues: []string{"pending", "open", "complete", "cancelled", "rejected"}},
			{Name: "placed_at", Type: "string", Description: "ISO 8601 UTC time, written before the broker answered"},
			{Name: "filled_at", Type: "string|null", Description: "ISO 8601 UTC time the order first became complete"},
			{Name: "avg_fill_price", Type: "number|null", Description: "average fill price reported by the broker"},
			{Name: "filled_qty", Type: "integer|null", Description: "quantity actually filled"},
			{Name: "reject_reason", Type: "string|null", Description: "broker or engine rejection text"},
		}},
	},
}

var StrategyRunsOp = Op{
	Name: "StrategyRuns", Method: "POST", Path: "/strategy/runs",
	Summary: "List a strategy's runs",
	Long:    "Return every activation of a strategy, newest first. limit outside 1 to 500 is a 400 rather than being clamped",
	Example: `  openalgo strategy runs --strategy-id 12
  openalgo strategy runs --strategy-id 12 --limit 10 --csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
		{Name: "limit", OASName: "limit", Type: "int", Default: "100", Description: "how many runs to return, 1 to 500", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "[]object", Description: "run objects, newest first by start time", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "run id, usable as run_id on orders and events"},
			{Name: "strategy_id", Type: "integer", Description: "owning strategy"},
			{Name: "mode", Type: "string", Description: "run mode, fixed for the life of the run", EnumValues: []string{"live", "sandbox"}},
			{Name: "broker", Type: "string", Description: "broker the run is bound to, sandbox for a sandbox run"},
			{Name: "started_at", Type: "string", Description: "ISO 8601 UTC start time"},
			{Name: "stopped_at", Type: "string|null", Description: "ISO 8601 UTC stop time, null while the run is open"},
			{Name: "stop_reason", Type: "string|null", Description: "why the run ended, null while open", EnumValues: []string{"manual", "scheduler", "overall_sl", "overall_target", "lock_profit", "eod", "expiry", "daily_loss_limit", "tick_stale", "recovery_failed", "error"}},
			{Name: "stop_requested_at", Type: "string|null", Description: "ISO 8601 UTC time a durable stop was requested, null when none is pending"},
			{Name: "stop_requested_reason", Type: "string|null", Description: "reason for a pending stop; while set new signal entries are gated"},
			{Name: "pnl_realized", Type: "number", Description: "realized P&L written at confirmed-flat finalization"},
			{Name: "pnl_peak", Type: "number", Description: "highest P&L the run reached"},
			{Name: "pnl_trough", Type: "number", Description: "lowest P&L the run reached"},
			{Name: "trigger_source", Type: "string", Description: "what started the run", EnumValues: []string{"manual", "webhook", "scheduler"}},
			{Name: "webhook_event_id", Type: "integer|null", Description: "inbound webhook event that started the run, when one did"},
			{Name: "resolved_expiries", Type: "object|null", Description: "leg id (as string) to the expiry resolved at run start"},
		}},
	},
}

var StrategyStartOp = Op{
	Name: "StrategyStart", Method: "POST", Path: "/strategy/start",
	Summary:  "Start a strategy run",
	Long:     "Start a run of a batch strategy and place every leg's entry order. --mode is required: sandbox routes orders to the sandbox, live places real orders with the broker and is refused unless live trading is enabled on the strategy.",
	Example:  `  openalgo strategy start --strategy-id 12 --mode sandbox`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "mode", OASName: "mode", Type: "string", Description: "run mode, exact and case-sensitive; live places real orders", Completions: []string{"live", "sandbox"}, Required: true, Source: "body"},
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "run_id", Type: "integer", Description: "the run that was opened"},
		{Name: "mode", Type: "string", Description: "the mode the run was started in, echoed back", EnumValues: []string{"live", "sandbox"}},
		{Name: "legs", Type: "[]object", Description: "per-leg placement outcome", Fields: []ResponseField{
			{Name: "leg_id", Type: "integer", Description: "the leg's id within the strategy"},
			{Name: "ok", Type: "boolean", Description: "whether the entry order was accepted"},
			{Name: "acknowledged", Type: "boolean", Description: "whether the broker id and status were durably written back; false with ok true is still a real order"},
			{Name: "symbol", Type: "string", Description: "the contract the leg resolved to"},
			{Name: "broker_order_id", Type: "string|null", Description: "broker or sandbox order id, null when not accepted"},
			{Name: "error", Type: "string|null", Description: "rejection reason when ok is false"},
		}},
	},
}

var StrategyStatusOp = Op{
	Name: "StrategyStatus", Method: "POST", Path: "/strategy/status",
	Summary:  "Get one strategy's configuration and current run",
	Long:     "Return one strategy's full configuration including legs, plus its current run or null. A strategy that is not yours returns 404 Strategy not found",
	Example:  `  openalgo strategy status --strategy-id 12`,
	RowsPath: "data.legs",
	Flags: []FlagDef{
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "object", Description: "strategy configuration including legs", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "strategy id, used as strategy_id on every other strategy endpoint"},
			{Name: "name", Type: "string", Description: "strategy name, unique per user"},
			{Name: "strategy_kind", Type: "string", Description: "strategy kind", EnumValues: []string{"batch", "signal"}},
			{Name: "direction", Type: "string", Description: "allowed direction, signal mode only", EnumValues: []string{"both", "long_only", "short_only"}},
			{Name: "universe_tab", Type: "string", Description: "instrument universe the strategy was built from", EnumValues: []string{"weekly_monthly", "monthly_only", "stocks_fno", "mcx"}},
			{Name: "underlying", Type: "string", Description: "underlying symbol"},
			{Name: "underlying_exchange", Type: "string", Description: "exchange the underlying is quoted on"},
			{Name: "strategy_type", Type: "string", Description: "strategy type", EnumValues: []string{"intraday", "positional"}},
			{Name: "entry_time", Type: "string|null", Description: "IST entry time as HH:MM"},
			{Name: "exit_time", Type: "string|null", Description: "IST square-off time as HH:MM"},
			{Name: "product", Type: "string", Description: "configured product intent (MIS intraday, anything else carry)", EnumValues: []string{"CNC", "NRML", "MIS"}},
			{Name: "pricetype", Type: "string", Description: "order price type, always MARKET"},
			{Name: "overall_sl_mtm", Type: "number|null", Description: "strategy-level stop loss in rupees of MTM"},
			{Name: "overall_target_mtm", Type: "number|null", Description: "strategy-level target in rupees of MTM"},
			{Name: "lock_profit", Type: "object|null", Description: "lock profit configuration, or null", Fields: []ResponseField{
				{Name: "mode", Type: "string", Description: "lock profit mode, e.g. lock_and_trail"},
				{Name: "if_profit_reaches", Type: "number", Description: "profit level that arms the lock"},
				{Name: "lock_profit", Type: "number", Description: "profit amount locked once armed"},
				{Name: "trail_step", Type: "number", Description: "step by which the locked floor trails"},
			}},
			{Name: "trail_sl_to_entry", Type: "boolean", Description: "whether a stop on one leg trails the others to entry"},
			{Name: "scheduler", Type: "object|null", Description: "scheduler configuration, or null", Fields: []ResponseField{
				{Name: "enabled", Type: "boolean", Description: "whether the built-in scheduler is enabled"},
				{Name: "days", Type: "[]string", Description: "weekdays the scheduler runs on, e.g. MON"},
				{Name: "start_time", Type: "string", Description: "IST start time as HH:MM"},
				{Name: "auto_stop_time", Type: "string", Description: "IST auto stop time as HH:MM"},
				{Name: "default_mode", Type: "string", Description: "mode used for scheduled starts", EnumValues: []string{"live", "sandbox"}},
			}},
			{Name: "live_enabled", Type: "boolean", Description: "whether this strategy may run in live mode"},
			{Name: "webhook_locked", Type: "boolean", Description: "whether the webhook kill switch is engaged"},
			{Name: "webhook_ip_allowlist", Type: "[]string|null", Description: "CIDR ranges allowed to trigger the webhook"},
			{Name: "daily_loss_limit_inr", Type: "number|null", Description: "daily loss ceiling in rupees"},
			{Name: "status", Type: "string", Description: "strategy status", EnumValues: []string{"stopped", "running", "paused", "errored"}},
			{Name: "current_run_id", Type: "integer|null", Description: "the run this strategy is executing, if any"},
			{Name: "created_at", Type: "string", Description: "ISO 8601 UTC creation time"},
			{Name: "updated_at", Type: "string", Description: "ISO 8601 UTC last update time"},
			{Name: "legs", Type: "[]object", Description: "leg configuration", Fields: []ResponseField{
				{Name: "id", Type: "integer", Description: "1-based leg id within the strategy"},
				{Name: "segment", Type: "string", Description: "leg segment, e.g. options"},
				{Name: "position", Type: "string", Description: "B for buy or S for sell"},
				{Name: "lots", Type: "integer", Description: "number of lots"},
				{Name: "option_type", Type: "string", Description: "CE or PE for option legs"},
				{Name: "strike_mode", Type: "string", Description: "strike selection mode, e.g. atm"},
				{Name: "atm_offset", Type: "string", Description: "strike offset from ATM, e.g. ATM"},
				{Name: "expiry", Type: "string", Description: "expiry rank, e.g. weekly"},
				{Name: "risk_unit", Type: "string", Description: "unit for sl_pts, target_pts and trail", EnumValues: []string{"points", "percent"}},
				{Name: "sl_pts", Type: "number", Description: "leg stop loss in risk_unit"},
				{Name: "target_pts", Type: "number", Description: "leg target in risk_unit"},
				{Name: "trail", Type: "object|null", Description: "trailing stop configuration, or null", Fields: []ResponseField{
					{Name: "x", Type: "number", Description: "move required to advance the trail"},
					{Name: "y", Type: "number", Description: "amount the stop advances"},
				}},
			}},
		}},
		{Name: "run", Type: "object|null", Description: "current run, or null when not running", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "run id, usable as run_id on orders and events"},
			{Name: "strategy_id", Type: "integer", Description: "owning strategy"},
			{Name: "mode", Type: "string", Description: "run mode, fixed for the life of the run", EnumValues: []string{"live", "sandbox"}},
			{Name: "broker", Type: "string", Description: "broker the run is bound to, sandbox for a sandbox run"},
			{Name: "started_at", Type: "string", Description: "ISO 8601 UTC start time"},
			{Name: "stopped_at", Type: "string|null", Description: "ISO 8601 UTC stop time, null while the run is open"},
			{Name: "stop_reason", Type: "string|null", Description: "why the run ended, null while open", EnumValues: []string{"manual", "scheduler", "overall_sl", "overall_target", "lock_profit", "eod", "expiry", "daily_loss_limit", "tick_stale", "recovery_failed", "error"}},
			{Name: "stop_requested_at", Type: "string|null", Description: "ISO 8601 UTC time a durable stop was requested, null when none is pending"},
			{Name: "stop_requested_reason", Type: "string|null", Description: "reason for a pending stop; while set new signal entries are gated"},
			{Name: "pnl_realized", Type: "number", Description: "realized P&L written at confirmed-flat finalization"},
			{Name: "pnl_peak", Type: "number", Description: "highest P&L the run reached"},
			{Name: "pnl_trough", Type: "number", Description: "lowest P&L the run reached"},
			{Name: "trigger_source", Type: "string", Description: "what started the run", EnumValues: []string{"manual", "webhook", "scheduler"}},
			{Name: "webhook_event_id", Type: "integer|null", Description: "inbound webhook event that started the run, when one did"},
			{Name: "resolved_expiries", Type: "object|null", Description: "leg id (as string) to the expiry resolved at run start"},
		}},
	},
}

var StrategyStopOp = Op{
	Name: "StrategyStop", Method: "POST", Path: "/strategy/stop",
	Summary:  "Stop a strategy run",
	Long:     "Stop the current run: exits every position the run owns at market and finalizes once fills confirm it is flat.",
	Example:  `  openalgo strategy stop --strategy-id 12`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "strategy-id", OASName: "strategy_id", Type: "int", Description: "strategy id, a positive integer", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "run_id", Type: "integer", Description: "the run the stop applies to; it stays current while stop_pending is true"},
		{Name: "stop_pending", Type: "boolean", Description: "true while owned exposure still needs a fill, retry or reconciliation; false only once flatness was confirmed"},
		{Name: "exits", Type: "[]object", Description: "per-leg exit outcome", Fields: []ResponseField{
			{Name: "leg_id", Type: "integer", Description: "the leg's id within the strategy"},
			{Name: "ok", Type: "boolean", Description: "whether the exit order was accepted"},
			{Name: "position_ref", Type: "string|null", Description: "exact durable owner the exit targets"},
			{Name: "exit_owner", Type: "string", Description: "owner side being exited", EnumValues: []string{"live", "superseded"}},
			{Name: "symbol", Type: "string", Description: "held symbol, included when an unfilled entry is refused before dispatch"},
			{Name: "broker_order_id", Type: "string|null", Description: "broker order id when available"},
			{Name: "error", Type: "string|null", Description: "rejection reason when ok is false"},
		}},
	},
}

var TelegramBroadcastOp = Op{
	Name: "TelegramBroadcast", Method: "POST", Path: "/telegram/broadcast",
	Summary:  "Broadcast a Telegram message",
	Long:     "Validate and broadcast a message to linked users. Delivery is not yet implemented server-side, so success_count and fail_count are currently always 0. Limited to 5 requests per minute; returns 403 when broadcast is disabled",
	Example:  `  openalgo telegram broadcast --message "Markets open in 15 minutes"`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "message", OASName: "message", Type: "string", Description: "message to broadcast, at most 4096 characters", Required: true, Source: "body"},
		{Name: "filters", OASName: "filters", Type: "json", Default: "{}", Description: "optional user filters; JSON object; literal, @file, or - for stdin", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "success_count", Type: "integer", Description: "number of users the broadcast reached"},
		{Name: "fail_count", Type: "integer", Description: "number of users the broadcast failed for"},
	},
}

var TelegramGetConfigOp = Op{
	Name: "TelegramGetConfig", Method: "GET", Path: "/telegram/config",
	Summary:  "Get Telegram bot configuration",
	Long:     "Return the Telegram bot configuration with bot_token truncated to its first 10 characters",
	Example:  `  openalgo telegram config get`,
	RowsPath: "data",
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "object", Description: "bot configuration", Fields: []ResponseField{
			{Name: "bot_token", Type: "string|null", Description: "bot token, truncated to the first 10 characters followed by an ellipsis"},
			{Name: "token", Type: "string|null", Description: "backward-compatible alias of the bot token"},
			{Name: "is_active", Type: "boolean", Description: "whether the bot is active; stopping it suppresses automatic alerts"},
			{Name: "bot_username", Type: "string|null", Description: "telegram username of the bot"},
			{Name: "max_message_length", Type: "integer", Description: "maximum message length"},
			{Name: "rate_limit_per_minute", Type: "integer", Description: "bot message rate limit per minute"},
			{Name: "broadcast_enabled", Type: "boolean", Description: "whether broadcast is enabled"},
			{Name: "created_at", Type: "string|null", Description: "configuration creation time"},
			{Name: "updated_at", Type: "string|null", Description: "configuration last update time"},
		}},
	},
}

var TelegramGetPreferencesOp = Op{
	Name: "TelegramGetPreferences", Method: "GET", Path: "/telegram/preferences",
	Summary:  "Get Telegram user preferences",
	Long:     "Return notification preferences for a Telegram user; defaults are returned when none are stored",
	Example:  `  openalgo telegram preferences get --telegram-id 123456789`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "telegram-id", OASName: "telegram_id", Type: "int", Description: "telegram user id", Required: true, Source: "query"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "object", Description: "user preferences", Fields: []ResponseField{
			{Name: "order_notifications", Type: "boolean", Description: "enable order notifications"},
			{Name: "trade_notifications", Type: "boolean", Description: "enable trade notifications"},
			{Name: "pnl_notifications", Type: "boolean", Description: "enable P&L notifications"},
			{Name: "daily_summary", Type: "boolean", Description: "enable daily summary"},
			{Name: "summary_time", Type: "string", Description: "daily summary time as HH:MM"},
			{Name: "language", Type: "string", Description: "preferred language, e.g. en"},
			{Name: "timezone", Type: "string", Description: "user timezone, e.g. Asia/Kolkata"},
		}},
	},
}

var TelegramNotifyOp = Op{
	Name: "TelegramNotify", Method: "POST", Path: "/telegram/notify",
	Summary: "Send a Telegram notification to one user",
	Long:    "Send a message to one OpenAlgo user linked to Telegram. By default it is queued and HTTP 200 means queued, not delivered. Returns 409 when the bot is stopped and 404 when the user is not linked",
	Example: `  openalgo telegram notify --username admin --message "SBIN order filled"
  openalgo telegram notify --username admin --message "Stoploss hit" --priority 9 --wait-for-delivery=true`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "message", OASName: "message", Type: "string", Description: "notification message", Required: true, Source: "body"},
		{Name: "username", OASName: "username", Type: "string", Description: "openAlgo username linked to a Telegram id", Required: true, Source: "body"},
		{Name: "priority", OASName: "priority", Type: "int", Default: "5", Description: "priority from 1 to 10; out-of-range values fall back to 5", Source: "body"},
		{Name: "wait-for-delivery", OASName: "wait_for_delivery", Type: "bool", Default: "false", Description: "wait for an immediate delivery attempt instead of queueing", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
	},
}

var TelegramSetConfigOp = Op{
	Name: "TelegramSetConfig", Method: "POST", Path: "/telegram/config",
	Summary: "Update Telegram bot configuration",
	Long:    "Update the supplied Telegram bot configuration fields; omitted fields are left unchanged",
	Example: `  openalgo telegram config set --token <bot-token> --polling-mode=true
  openalgo telegram config set --broadcast-enabled=false --rate-limit-per-minute 30`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "broadcast-enabled", OASName: "broadcast_enabled", Type: "bool", Description: "enable broadcast messages", Source: "body"},
		{Name: "polling-mode", OASName: "polling_mode", Type: "bool", Description: "use polling mode instead of webhook mode", Source: "body"},
		{Name: "rate-limit-per-minute", OASName: "rate_limit_per_minute", Type: "int", Description: "rate limit per minute, 1 to 120", Source: "body"},
		{Name: "token", OASName: "token", Type: "string", Description: "telegram bot token", Source: "body"},
		{Name: "webhook-url", OASName: "webhook_url", Type: "string", Description: "webhook URL for the bot", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
	},
}

var TelegramSetPreferencesOp = Op{
	Name: "TelegramSetPreferences", Method: "POST", Path: "/telegram/preferences",
	Summary: "Update Telegram user preferences",
	Long:    "Update the supplied notification preferences for a Telegram user; omitted fields are left unchanged",
	Example: `  openalgo telegram preferences set --telegram-id 123456789 --daily-summary=true --summary-time 15:45
  openalgo telegram preferences set --telegram-id 123456789 --trade-notifications=false`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "telegram-id", OASName: "telegram_id", Type: "int", Description: "telegram user id", Required: true, Source: "body"},
		{Name: "daily-summary", OASName: "daily_summary", Type: "bool", Description: "enable daily summary", Source: "body"},
		{Name: "language", OASName: "language", Type: "string", Description: "preferred language, e.g. en", Source: "body"},
		{Name: "order-notifications", OASName: "order_notifications", Type: "bool", Description: "enable order notifications", Source: "body"},
		{Name: "pnl-notifications", OASName: "pnl_notifications", Type: "bool", Description: "enable P&L notifications", Source: "body"},
		{Name: "summary-time", OASName: "summary_time", Type: "string", Description: "daily summary time as HH:MM", Source: "body"},
		{Name: "timezone", OASName: "timezone", Type: "string", Description: "user timezone, e.g. Asia/Kolkata", Source: "body"},
		{Name: "trade-notifications", OASName: "trade_notifications", Type: "bool", Description: "enable trade notifications", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
	},
}

var TelegramStartOp = Op{
	Name: "TelegramStart", Method: "POST", Path: "/telegram/start",
	Summary:  "Start the Telegram bot",
	Long:     "Initialize and start the Telegram bot from the stored configuration. Returns 400 when no bot token is configured",
	Example:  `  openalgo telegram start`,
	Mutating: true,
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
	},
}

var TelegramStatsOp = Op{
	Name: "TelegramStats", Method: "GET", Path: "/telegram/stats",
	Summary: "Get Telegram bot usage statistics",
	Long:    "Return Telegram bot command statistics for the last N days",
	Example: `  openalgo telegram stats
  openalgo telegram stats --days 30`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "days", OASName: "days", Type: "int", Default: "7", Description: "look-back window in days, clamped to 1 to 365", Source: "query"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "data", Type: "object", Description: "command statistics", Fields: []ResponseField{
			{Name: "total_commands", Type: "integer", Description: "total commands executed in the period"},
			{Name: "commands_by_type", Type: "object", Description: "command name to execution count"},
			{Name: "active_users", Type: "integer", Description: "distinct users who issued commands"},
			{Name: "top_users", Type: "[][]any", Description: "up to 10 most active users as [telegram_username, command_count] pairs"},
			{Name: "period_days", Type: "integer", Description: "look-back window applied, in days"},
		}},
	},
}

var TelegramStopOp = Op{
	Name: "TelegramStop", Method: "POST", Path: "/telegram/stop",
	Summary:  "Stop the Telegram bot",
	Long:     "Stop the Telegram bot service. Automatic order and Flow alerts are suppressed while the bot is stopped",
	Example:  `  openalgo telegram stop`,
	Mutating: true,
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
	},
}

var TelegramUsersOp = Op{
	Name: "TelegramUsers", Method: "GET", Path: "/telegram/users",
	Summary: "List linked Telegram users",
	Long:    "List active Telegram users linked to OpenAlgo, optionally filtered by broker or notification setting",
	Example: `  openalgo telegram users
  openalgo telegram users --broker zerodha --notifications-enabled true --csv`,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "broker", OASName: "broker", Type: "string", Description: "filter by broker name", Source: "query"},
		{Name: "notifications-enabled", OASName: "notifications_enabled", Type: "string", Description: "filter by notification setting; the string true matches enabled, anything else disabled", Completions: []string{"true", "false"}, Source: "query"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "count", Type: "integer", Description: "number of users returned"},
		{Name: "data", Type: "[]object", Description: "linked users", Fields: []ResponseField{
			{Name: "id", Type: "integer", Description: "row id"},
			{Name: "telegram_id", Type: "integer", Description: "telegram user id"},
			{Name: "openalgo_username", Type: "string", Description: "linked OpenAlgo username"},
			{Name: "first_name", Type: "string|null", Description: "telegram first name"},
			{Name: "last_name", Type: "string|null", Description: "telegram last name"},
			{Name: "telegram_username", Type: "string|null", Description: "telegram username"},
			{Name: "broker", Type: "string|null", Description: "broker of the linked user"},
			{Name: "notifications_enabled", Type: "boolean", Description: "whether notifications are enabled"},
			{Name: "created_at", Type: "string|null", Description: "link creation time"},
			{Name: "last_command_at", Type: "string|null", Description: "time of the user's last bot command"},
		}},
	},
}

var ToggleAnalyzerOp = Op{
	Name: "ToggleAnalyzer", Method: "POST", Path: "/analyzer/toggle",
	Summary:  "Toggle analyzer mode",
	Long:     "Switch analyzer (sandbox) mode on or off application-wide. --mode=false switches to live trading: subsequent orders from every client go to the live broker with real money.",
	Example:  `  openalgo analyzer toggle --mode=true`,
	Mutating: true,
	RowsPath: "data",
	Flags: []FlagDef{
		{Name: "mode", OASName: "mode", Type: "bool", Description: "true to enable analyzer mode, false to switch to live", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
		{Name: "data", Type: "object", Description: "toggle result", Fields: []ResponseField{
			{Name: "analyze_mode", Type: "boolean", Description: "true if analyzer mode is active"},
			{Name: "mode", Type: "string", Description: "current mode", EnumValues: []string{"analyze", "live"}},
			{Name: "total_logs", Type: "integer", Description: "number of orders logged in analyzer mode"},
			{Name: "message", Type: "string", Description: "confirmation message"},
		}},
	},
}

var UpdateChartPreferencesOp = Op{
	Name: "UpdateChartPreferences", Method: "POST", Path: "/chart",
	Summary: "Update chart preferences",
	Long:    "Upsert chart preference keys for the API key. --preferences is a JSON object (literal, @file, or - for stdin) of 1 to 50 keys; each key is up to 50 characters and each value up to 1 MiB.",
	Example: `  openalgo chart set --preferences '{"theme":"dark"}'
  openalgo chart set --preferences @chart-prefs.json`,
	Mutating: true,
	Flags: []FlagDef{
		{Name: "preferences", OASName: "preferences", Type: "object", Description: "JSON object (literal, @file, or - for stdin) whose keys are merged into the request body", Required: true, Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "request status", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable message"},
	},
}

var WhatsAppNotifyOp = Op{
	Name: "WhatsAppNotify", Method: "POST", Path: "/whatsapp/notify",
	Summary: "Send a WhatsApp message",
	Long:    "Send text, an image or a document to exactly one recipient form: --self, --username, --phone, or --phones (up to 5). Returns 409 when WhatsApp is not paired or connected.",
	Example: `  openalgo whatsapp notify --self=true --message "SBIN order placed"
  openalgo whatsapp notify --phone 919876543210 --message "Daily P&L report ready"
  openalgo whatsapp notify --phones '["919876543210","919812345678"]' --message "Markets open in 15 minutes"`,
	Mutating: true,
	RowsPath: "data.failed",
	Flags: []FlagDef{
		{Name: "caption", OASName: "caption", Type: "string", Description: "caption for the image, or follow-up text for a document", Source: "body"},
		{Name: "document-path", OASName: "document_path", Type: "string", Description: "server-local path to a document file inside the attachment allowlist", Source: "body"},
		{Name: "filename", OASName: "filename", Type: "string", Description: "override the document display name", Source: "body"},
		{Name: "image-path", OASName: "image_path", Type: "string", Description: "server-local path to an image file inside the attachment allowlist", Source: "body"},
		{Name: "message", OASName: "message", Type: "string", Description: "text body, at most 4096 characters; optional if image_path or document_path is set", Source: "body"},
		{Name: "phone", OASName: "phone", Type: "string", Description: "single E.164 digit string, e.g. 919876543210", Source: "body"},
		{Name: "phones", OASName: "phones", Type: "array", Description: "up to 5 E.164 digit strings; entries beyond 5 are dropped; JSON array of strings; literal, @file, or - for stdin", Source: "body"},
		{Name: "self", OASName: "self", Type: "bool", Default: "false", Description: "send to the paired device's own number", Source: "body"},
		{Name: "username", OASName: "username", Type: "string", Description: "openAlgo username resolved through the linked-users table or the paired owner", Source: "body"},
		{Name: "wait-for-delivery", OASName: "wait_for_delivery", Type: "bool", Default: "true", Description: "block until the send completes and return a per-recipient report; false queues and returns immediately", Source: "body"},
	},
	Response: []ResponseField{
		{Name: "status", Type: "string", Description: "success or error", EnumValues: []string{"success", "error"}},
		{Name: "message", Type: "string", Description: "human-readable result or error message"},
		{Name: "queued", Type: "integer", Description: "number of recipients dispatched to the alert pool, present when wait_for_delivery is false"},
		{Name: "data", Type: "object", Description: "per-recipient delivery report, present when wait_for_delivery is true", Fields: []ResponseField{
			{Name: "sent", Type: "[]string", Description: "JIDs the send was accepted for"},
			{Name: "failed", Type: "[]object", Description: "recipients that failed", Fields: []ResponseField{
				{Name: "to", Type: "string", Description: "recipient JID"},
				{Name: "error", Type: "string", Description: "failure message"},
			}},
			{Name: "skipped", Type: "integer", Description: "recipients trimmed by the 5-recipient cap"},
		}},
	},
}

// AllOps lists every generated Op for iteration in tests and tooling.
var AllOps = []Op{
	BasketOrderOp,
	CancelAllOrderOp,
	CancelGTTOrderOp,
	CancelOrderOp,
	ClosePositionOp,
	GetAnalyzerStatusOp,
	GetChartPreferencesOp,
	GetDepthOp,
	GetExpiryOp,
	GetFundsOp,
	GetGTTOrderbookOp,
	GetHistoryOp,
	GetHoldingsOp,
	GetInstrumentsOp,
	GetIntervalsOp,
	GetMarginOp,
	GetMarketHolidaysOp,
	GetMarketTimingsOp,
	GetMultiOptionGreeksOp,
	GetMultiQuotesOp,
	GetOpenPositionOp,
	GetOptionChainOp,
	GetOptionGreeksOp,
	GetOptionSymbolOp,
	GetOrderStatusOp,
	GetOrderbookOp,
	GetPnlSymbolsOp,
	GetPositionbookOp,
	GetQuotesOp,
	GetSymbolOp,
	GetSyntheticFutureOp,
	GetTickerOp,
	GetTradebookOp,
	ModifyGTTOrderOp,
	ModifyOrderOp,
	OptionsMultiOrderOp,
	OptionsOrderOp,
	PingOp,
	PlaceGTTOrderOp,
	PlaceOrderOp,
	PlaceSmartOrderOp,
	SearchSymbolsOp,
	SplitOrderOp,
	StrategyCloseAllOp,
	StrategyCloseLegOp,
	StrategyEventsOp,
	StrategyListOp,
	StrategyOrdersOp,
	StrategyRunsOp,
	StrategyStartOp,
	StrategyStatusOp,
	StrategyStopOp,
	TelegramBroadcastOp,
	TelegramGetConfigOp,
	TelegramGetPreferencesOp,
	TelegramNotifyOp,
	TelegramSetConfigOp,
	TelegramSetPreferencesOp,
	TelegramStartOp,
	TelegramStatsOp,
	TelegramStopOp,
	TelegramUsersOp,
	ToggleAnalyzerOp,
	UpdateChartPreferencesOp,
	WhatsAppNotifyOp,
}

var opByName map[string]Op

func init() {
	opByName = make(map[string]Op, len(AllOps))
	for _, op := range AllOps {
		opByName[op.Name] = op
	}
}

// OpByName returns the Op with the given name, if any.
func OpByName(name string) (Op, bool) {
	op, ok := opByName[name]
	return op, ok
}

// ResponseSchema returns the response field tree for an operation.
func ResponseSchema(name string) ([]ResponseField, bool) {
	op, ok := opByName[name]
	if !ok || len(op.Response) == 0 {
		return nil, false
	}
	return op.Response, true
}
