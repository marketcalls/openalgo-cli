// Code generated from api/specs/openalgo-api.json by cmd/generate; DO NOT EDIT.

package api

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/marketcalls/openalgo-cli/internal/client"
)

// Client provides one method per OpenAlgo API operation. Request bodies
// are plain maps and responses are returned as raw JSON.
type Client struct {
	Raw *client.Client
}

// NewClient wraps a configured HTTP client.
func NewClient(raw *client.Client) *Client {
	return &Client{Raw: raw}
}

// GetAnalyzerStatus calls POST /analyzer: Get analyzer status.
func (c *Client) GetAnalyzerStatus() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/analyzer", nil, nil)
}

// ToggleAnalyzer calls POST /analyzer/toggle: Toggle analyzer mode.
func (c *Client) ToggleAnalyzer(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/analyzer/toggle", nil, body)
}

// BasketOrder calls POST /basketorder: Place a basket of orders.
func (c *Client) BasketOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/basketorder", nil, body)
}

// CancelAllOrder calls POST /cancelallorder: Cancel all open orders.
func (c *Client) CancelAllOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/cancelallorder", nil, body)
}

// CancelGTTOrder calls POST /cancelgttorder: Cancel an active GTT order.
func (c *Client) CancelGTTOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/cancelgttorder", nil, body)
}

// CancelOrder calls POST /cancelorder: Cancel an order.
func (c *Client) CancelOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/cancelorder", nil, body)
}

// GetChartPreferences calls GET /chart: Get chart preferences.
func (c *Client) GetChartPreferences() (json.RawMessage, error) {
	return c.Raw.Do("GET", "/chart", nil, nil)
}

// UpdateChartPreferences calls POST /chart: Update chart preferences.
func (c *Client) UpdateChartPreferences(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/chart", nil, body)
}

// ClosePosition calls POST /closeposition: Close all open positions.
func (c *Client) ClosePosition(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/closeposition", nil, body)
}

// GetDepth calls POST /depth: Get market depth for a symbol.
func (c *Client) GetDepth(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/depth", nil, body)
}

// GetExpiry calls POST /expiry: Get expiry dates.
func (c *Client) GetExpiry(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/expiry", nil, body)
}

// GetFunds calls POST /funds: Get account funds.
func (c *Client) GetFunds() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/funds", nil, nil)
}

// GetGTTOrderbook calls POST /gttorderbook: List GTT triggers.
func (c *Client) GetGTTOrderbook(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/gttorderbook", nil, body)
}

// GetHistory calls POST /history: Get historical candles.
func (c *Client) GetHistory(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/history", nil, body)
}

// GetHoldings calls POST /holdings: Get portfolio holdings.
func (c *Client) GetHoldings() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/holdings", nil, nil)
}

// GetInstruments calls GET /instruments: Download instruments.
func (c *Client) GetInstruments(params url.Values) (json.RawMessage, error) {
	return c.Raw.Do("GET", "/instruments", params, nil)
}

// GetIntervals calls POST /intervals: Get supported history intervals.
func (c *Client) GetIntervals() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/intervals", nil, nil)
}

// GetMargin calls POST /margin: Calculate margin for a basket of positions.
func (c *Client) GetMargin(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/margin", nil, body)
}

// GetMarketHolidays calls POST /market/holidays: Get market holidays.
func (c *Client) GetMarketHolidays(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/market/holidays", nil, body)
}

// GetMarketTimings calls POST /market/timings: Get market timings for a date.
func (c *Client) GetMarketTimings(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/market/timings", nil, body)
}

// ModifyGTTOrder calls POST /modifygttorder: Modify an active GTT order.
func (c *Client) ModifyGTTOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/modifygttorder", nil, body)
}

// ModifyOrder calls POST /modifyorder: Modify an open order.
func (c *Client) ModifyOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/modifyorder", nil, body)
}

// GetMultiOptionGreeks calls POST /multioptiongreeks: Calculate Greeks for multiple options.
func (c *Client) GetMultiOptionGreeks(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/multioptiongreeks", nil, body)
}

// GetMultiQuotes calls POST /multiquotes: Get quotes for multiple symbols.
func (c *Client) GetMultiQuotes(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/multiquotes", nil, body)
}

// GetOpenPosition calls POST /openposition: Get the open position for a symbol.
func (c *Client) GetOpenPosition(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/openposition", nil, body)
}

// GetOptionChain calls POST /optionchain: Get option chain.
func (c *Client) GetOptionChain(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/optionchain", nil, body)
}

// GetOptionGreeks calls POST /optiongreeks: Calculate option Greeks.
func (c *Client) GetOptionGreeks(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/optiongreeks", nil, body)
}

// OptionsMultiOrder calls POST /optionsmultiorder: Place a multi-leg options order.
func (c *Client) OptionsMultiOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/optionsmultiorder", nil, body)
}

// OptionsOrder calls POST /optionsorder: Place an options order by strike offset.
func (c *Client) OptionsOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/optionsorder", nil, body)
}

// GetOptionSymbol calls POST /optionsymbol: Resolve an option symbol.
func (c *Client) GetOptionSymbol(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/optionsymbol", nil, body)
}

// GetOrderbook calls POST /orderbook: Get the order book.
func (c *Client) GetOrderbook() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/orderbook", nil, nil)
}

// GetOrderStatus calls POST /orderstatus: Get the status of an order.
func (c *Client) GetOrderStatus(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/orderstatus", nil, body)
}

// Ping calls POST /ping: Ping the API.
func (c *Client) Ping() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/ping", nil, nil)
}

// PlaceGTTOrder calls POST /placegttorder: Place a GTT order.
func (c *Client) PlaceGTTOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/placegttorder", nil, body)
}

// PlaceOrder calls POST /placeorder: Place an order.
func (c *Client) PlaceOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/placeorder", nil, body)
}

// PlaceSmartOrder calls POST /placesmartorder: Place a position-aware smart order.
func (c *Client) PlaceSmartOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/placesmartorder", nil, body)
}

// GetPnlSymbols calls POST /pnl/symbols: Get sandbox P&L by symbol.
func (c *Client) GetPnlSymbols() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/pnl/symbols", nil, nil)
}

// GetPositionbook calls POST /positionbook: Get the position book.
func (c *Client) GetPositionbook() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/positionbook", nil, nil)
}

// GetQuotes calls POST /quotes: Get a quote for a symbol.
func (c *Client) GetQuotes(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/quotes", nil, body)
}

// SearchSymbols calls POST /search: Search symbols.
func (c *Client) SearchSymbols(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/search", nil, body)
}

// SplitOrder calls POST /splitorder: Split a large order into smaller orders.
func (c *Client) SplitOrder(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/splitorder", nil, body)
}

// StrategyCloseAll calls POST /strategy/close_all: Close all legs of a strategy run.
func (c *Client) StrategyCloseAll(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/strategy/close_all", nil, body)
}

// StrategyCloseLeg calls POST /strategy/close_leg: Close one leg of a strategy run.
func (c *Client) StrategyCloseLeg(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/strategy/close_leg", nil, body)
}

// StrategyEvents calls POST /strategy/events: List a strategy's risk events.
func (c *Client) StrategyEvents(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/strategy/events", nil, body)
}

// StrategyList calls POST /strategy/list: List strategies.
func (c *Client) StrategyList(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/strategy/list", nil, body)
}

// StrategyOrders calls POST /strategy/orders: List a strategy's orders.
func (c *Client) StrategyOrders(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/strategy/orders", nil, body)
}

// StrategyRuns calls POST /strategy/runs: List a strategy's runs.
func (c *Client) StrategyRuns(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/strategy/runs", nil, body)
}

// StrategyStart calls POST /strategy/start: Start a strategy run.
func (c *Client) StrategyStart(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/strategy/start", nil, body)
}

// StrategyStatus calls POST /strategy/status: Get one strategy's configuration and current run.
func (c *Client) StrategyStatus(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/strategy/status", nil, body)
}

// StrategyStop calls POST /strategy/stop: Stop a strategy run.
func (c *Client) StrategyStop(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/strategy/stop", nil, body)
}

// GetSymbol calls POST /symbol: Get symbol details.
func (c *Client) GetSymbol(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/symbol", nil, body)
}

// GetSyntheticFuture calls POST /syntheticfuture: Calculate synthetic future price.
func (c *Client) GetSyntheticFuture(body map[string]any) (json.RawMessage, error) {
	return c.Raw.Do("POST", "/syntheticfuture", nil, body)
}

// TelegramBroadcast calls POST /telegram/broadcast: Broadcast a Telegram message.
func (c *Client) TelegramBroadcast(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/telegram/broadcast", nil, body)
}

// TelegramGetConfig calls GET /telegram/config: Get Telegram bot configuration.
func (c *Client) TelegramGetConfig() (json.RawMessage, error) {
	return c.Raw.Do("GET", "/telegram/config", nil, nil)
}

// TelegramSetConfig calls POST /telegram/config: Update Telegram bot configuration.
func (c *Client) TelegramSetConfig(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/telegram/config", nil, body)
}

// TelegramNotify calls POST /telegram/notify: Send a Telegram notification to one user.
func (c *Client) TelegramNotify(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/telegram/notify", nil, body)
}

// TelegramGetPreferences calls GET /telegram/preferences: Get Telegram user preferences.
func (c *Client) TelegramGetPreferences(params url.Values) (json.RawMessage, error) {
	return c.Raw.Do("GET", "/telegram/preferences", params, nil)
}

// TelegramSetPreferences calls POST /telegram/preferences: Update Telegram user preferences.
func (c *Client) TelegramSetPreferences(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/telegram/preferences", nil, body)
}

// TelegramStart calls POST /telegram/start: Start the Telegram bot.
func (c *Client) TelegramStart() (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/telegram/start", nil, nil)
}

// TelegramStats calls GET /telegram/stats: Get Telegram bot usage statistics.
func (c *Client) TelegramStats(params url.Values) (json.RawMessage, error) {
	return c.Raw.Do("GET", "/telegram/stats", params, nil)
}

// TelegramStop calls POST /telegram/stop: Stop the Telegram bot.
func (c *Client) TelegramStop() (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/telegram/stop", nil, nil)
}

// TelegramUsers calls GET /telegram/users: List linked Telegram users.
func (c *Client) TelegramUsers(params url.Values) (json.RawMessage, error) {
	return c.Raw.Do("GET", "/telegram/users", params, nil)
}

// GetTicker calls GET /ticker/{symbol}: Get ticker-compatible historical candles.
func (c *Client) GetTicker(symbol string, params url.Values) (json.RawMessage, error) {
	return c.Raw.Do("GET", fmt.Sprintf("/ticker/%s", url.PathEscape(symbol)), params, nil)
}

// GetTradebook calls POST /tradebook: Get the trade book.
func (c *Client) GetTradebook() (json.RawMessage, error) {
	return c.Raw.Do("POST", "/tradebook", nil, nil)
}

// WhatsAppNotify calls POST /whatsapp/notify: Send a WhatsApp message.
func (c *Client) WhatsAppNotify(body map[string]any) (json.RawMessage, error) {
	return c.Raw.DoWrite("POST", "/whatsapp/notify", nil, body)
}
