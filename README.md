# OpenAlgo CLI

CLI for the [OpenAlgo](https://github.com/marketcalls/openalgo) trading platform API. Place and manage orders, read positions, funds and holdings, pull market data, work with option chains and Greeks, run strategies, and stream live quotes from the command line.

OpenAlgo is a self-hosted, broker-agnostic algorithmic trading platform for Indian markets. One OpenAlgo server connects to your broker and exposes a single REST and WebSocket API with a common symbol format across NSE, BSE, NFO, BFO, CDS, BCD, MCX and more. This CLI talks to your own OpenAlgo server over HTTP. Server documentation lives at [docs.openalgo.in](https://docs.openalgo.in).

> [!WARNING]
> **Alpha Preview** - This CLI is under active development. Commands, flags, and output formats may change or be removed without notice between releases. Do not depend on current behavior in production workflows.

## Major Use Cases

1. **AI agent harnesses.** Give an agent (Claude Code, Codex, a Claude Agent SDK or any tool-calling harness) a shell tool and this CLI, and it can read positions, funds and quotes, reason about them, and place or manage orders. Every command returns JSON, errors are structured JSON with exit codes, `--help-all` and `--schema` let the agent discover the surface at runtime, and the bundled [agent skill](.agents/skills/openalgo-cli/SKILL.md) teaches it the tool.
2. **Human-in-the-loop execution.** An agent or script proposes an order with `--dry-run`, a human or policy layer approves the exact request body, and the same body is sent.
3. **Scheduled automation.** Cron jobs, CI or task schedulers square off positions, snapshot P&L, or alert on rejected orders with a few lines of shell: `openalgo position list --jq '...'`.
4. **Multiple brokers and instances.** One profile per OpenAlgo instance (`-p zerodha`, `-p dhan`) lets a single script or agent report on or act across every account.
5. **Research and data pipelines.** Historical candles, the instrument master, option chains and Greeks go straight into pandas, DuckDB or spreadsheets with `--csv`.
6. **Live monitoring.** `openalgo stream ltp|quote|depth|orders` emits NDJSON for alerting scripts, dashboards or an agent watching order updates.
7. **Strategy Module operations.** Start, stop and audit strategies (runs, orders, risk events) from scripts or chat-ops.
8. **Ops and debugging.** `openalgo doctor` checks connectivity, the broker session and sandbox mode. `--debug` and `--trace` show the exact request with the API key redacted, which is useful for reproducing a TradingView or Amibroker integration issue.
9. **A universal adapter.** Anything that can run a process (Excel VBA, Node, Rust, n8n, Make) can drive OpenAlgo without an SDK.

Where it is not the right tool: tick-by-tick, latency-sensitive strategies (each command is a new process; use the Python SDK or the WebSocket directly), and long-running conversational sessions, where OpenAlgo's MCP server fits better.

## Built For Agents

OpenAlgo CLI is designed for AI agents, scripts, and automation pipelines. It is not an interactive trading terminal: there are no confirmation prompts, "are you sure?" dialogs, or interactive guardrails. Every command executes immediately against whatever your OpenAlgo server is connected to.

Destructive commands are truly destructive:

- `openalgo position close-all` squares off every open position in the account, across all exchanges. `--strategy` is only a tag, not a filter.
- `openalgo order cancel-all` cancels every open order in the account without listing them first.
- `openalgo strategy close-all` exits every open leg of a running strategy.
- `openalgo analyzer toggle --mode=false` switches the server out of analyzer mode, so every subsequent order goes to your broker as a live order.

## Sandbox Mode (Analyzer Mode)

OpenAlgo has a server-side **analyzer mode**, also called **sandbox mode**. While it is on, orders are simulated by the server against live market data instead of being sent to the broker, and positions, funds and the order book reflect the sandbox. The mode belongs to the server, not to the CLI: the same `openalgo order place` is a sandbox order or a live order depending on the server's current state.

```bash
openalgo analyzer status                 # is the server in analyzer mode?
openalgo analyzer toggle --mode=true     # switch to analyzer (sandbox) mode
openalgo analyzer toggle --mode=false    # switch to live trading - real orders
```

Check `openalgo analyzer status` before trading when you are not sure which mode the server is in. In analyzer (sandbox) mode, `MIS` orders are refused after the square-off time (15:15 IST by default) until the next session; use `CNC` for equity and `NRML` for futures and options.

## Prerequisites

- **A running OpenAlgo server** (default `http://127.0.0.1:5000`), logged in to your broker. See the [OpenAlgo installation guide](https://docs.openalgo.in).
- **An OpenAlgo API key**, generated on the server's API Key page.
- **Go is not required** for the one-line installers or Homebrew: they install a prebuilt single binary for macOS, Linux or Windows (amd64 and arm64). Go 1.24 or newer is needed only for `go install` or to build from source.

## Install

**macOS and Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.ps1 | iex
```

**Homebrew (macOS and Linux):**

```bash
brew install marketcalls/tap/openalgo-cli
```

**Go (1.24+):**

```bash
go install github.com/marketcalls/openalgo-cli/cmd/openalgo@latest
```

Make sure `$(go env GOPATH)/bin` (usually `~/go/bin`) is on your `PATH`.

**From source:**

```bash
git clone https://github.com/marketcalls/openalgo-cli.git
cd openalgo-cli
make build            # binary at bin/openalgo
make install          # or install into $GOPATH/bin
```

The installers verify the release checksum and accept `OPENALGO_CLI_VERSION` (pin a release tag) and `OPENALGO_INSTALL_DIR` (install location).

**Verify:**

```bash
openalgo version
openalgo doctor
```

## Quick Start

```bash
# Save the server address and API key in a profile
openalgo profile login

# Check connectivity and the trading mode
openalgo ping
openalgo analyzer status

# Account
openalgo account funds
openalgo position list

# Market data
openalgo data quote --symbol RELIANCE --exchange NSE
openalgo data history --symbol SBIN --exchange NSE --interval 5m --start-date 2026-09-01 --end-date 2026-09-25

# Place a limit order for equity delivery
openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --pricetype LIMIT --price 800 --product CNC

# Open orders and trades
openalgo order list
openalgo order trades
```

## Authentication

OpenAlgo authenticates every request with a single API key issued by your OpenAlgo server. The key is only valid for the server that issued it, so the CLI always stores and resolves the key together with that server's host.

`openalgo profile login` prompts for the host (default `http://127.0.0.1:5000`) and the API key, and stores a profile in `~/.config/openalgo/profiles/<name>.yaml` with restricted file permissions.

```bash
openalgo profile login                     # default profile
openalgo profile login --name vps          # named profile for another server
openalgo profile list                      # list profiles
openalgo profile switch vps                # switch active profile
openalgo profile logout vps                # remove a profile
openalgo --profile vps position list       # one-off profile override
```

### Multiple OpenAlgo instances

OpenAlgo is single-user, but traders often run several instances side by side, one per broker or account. Instance *n* listens on port `5000+n-1`, with its WebSocket on `8765+n-1`. Create one profile per instance and select it with `-p`:

```bash
openalgo profile login --name zerodha --host http://127.0.0.1:5000
openalgo profile login --name dhan    --host http://127.0.0.1:5001
openalgo profile login --name angel   --host https://angel.example.com

openalgo -p zerodha position list
openalgo -p dhan account funds
openalgo -p dhan stream ltp --symbols NSE:SBIN    # ws://127.0.0.1:8766
```

Each profile keeps its own host and API key, and a key is only ever sent to the host it belongs to.

### Environment variables

For scripts, CI, and agents, prefer environment variables so the key does not touch disk:

```bash
export OPENALGO_API_KEY=...
export OPENALGO_HOST=http://127.0.0.1:5000   # optional, this is the default
openalgo account funds --quiet
```

Credential lookup uses the first complete bundle, and never mixes fields across sources:

1. `OPENALGO_API_KEY` (with `OPENALGO_HOST`, default `http://127.0.0.1:5000`)
2. Profile `api_key` (with profile `host`, default `http://127.0.0.1:5000`)

`OPENALGO_HOST` on its own is ignored, so a stray host variable can never send a stored profile key to a different server. The API key is never printed, including in `--dry-run` and `--debug` output.

## Commands

The CLI is generated from a curated OpenAPI spec of the OpenAlgo v1 API (`api/specs/openalgo-api.json`), so the installed binary is the source of truth for commands, flags, enum completions, validation, and response schemas.

```bash
openalgo --help-all              # Full command reference
openalgo order place --help      # Flags for one command
openalgo order place --schema    # Response fields without an API call
```

Every parameter is an explicit `--flag`. Flag names are the API field names in kebab-case: `trigger_price` becomes `--trigger-price`, `position_size` becomes `--position-size`, `expiry_date` becomes `--expiry-date`, `strategy_id` becomes `--strategy-id`.

### Trading

| Command | API endpoint | Description |
|---|---|---|
| `order place` | `/placeorder` | Place an order |
| `order smart` | `/placesmartorder` | Place an order sized against the current position (`--position-size`) |
| `order basket` | `/basketorder` | Place several orders at once (`--orders` JSON array) |
| `order split` | `/splitorder` | Split a large order into chunks (`--splitsize`) |
| `order modify` | `/modifyorder` | Modify an open order |
| `order cancel` | `/cancelorder` | Cancel one order (`--orderid`) |
| `order cancel-all` | `/cancelallorder` | Cancel every open order in the account |
| `order status` | `/orderstatus` | Status of one order |
| `order list` | `/orderbook` | Order book |
| `order trades` | `/tradebook` | Trade book |
| `gtt place` / `modify` / `cancel` / `list` | `/placegttorder` ... | Good-till-triggered orders (CNC or NRML only) |
| `position list` | `/positionbook` | Position book |
| `position get` | `/openposition` | Open quantity for one symbol |
| `position close-all` | `/closeposition` | Square off every open position in the account |
| `option order` | `/optionsorder` | Place an option order by ATM offset (`--offset ATM --option-type CE`) |
| `option multi-order` | `/optionsmultiorder` | Multi-leg option order (`--legs` JSON array) |
| `option symbol` | `/optionsymbol` | Resolve an option symbol from an offset |
| `option chain` | `/optionchain` | Option chain, optionally with Greeks |
| `option synthetic-future` | `/syntheticfuture` | Synthetic future price |
| `option greeks` / `multi-greeks` | `/optiongreeks` ... | Implied volatility and Greeks |
| `strategy list` / `status` | `/strategy/...` | Strategies and their state |
| `strategy start` / `stop` | `/strategy/...` | Start (`--mode sandbox` or `--mode live`) or stop a strategy run |
| `strategy close-all` / `close-leg` | `/strategy/...` | Exit all legs or one leg of a strategy |
| `strategy runs` / `orders` / `events` | `/strategy/...` | Run history, orders and event log |

### Account and Market Data

| Command | API endpoint | Description |
|---|---|---|
| `account funds` | `/funds` | Available cash and margin used |
| `account holdings` | `/holdings` | Demat holdings |
| `account margin` | `/margin` | Margin required for a set of positions (`--positions` JSON array) |
| `data quote` / `quotes` | `/quotes`, `/multiquotes` | Quotes for one symbol or many (`--symbols` JSON array) |
| `data depth` | `/depth` | Market depth |
| `data history` | `/history` | Historical candles |
| `data intervals` | `/intervals` | Intervals the broker supports |
| `data ticker` | `/ticker/{symbol}` | Candles by ticker, JSON or text (`--format`) |
| `symbol get` / `search` | `/symbol`, `/search` | Symbol details and search |
| `symbol expiry` | `/expiry` | Expiry dates for futures or options |
| `symbol instruments` | `/instruments` | Instrument master download (JSON or `--format csv`) |
| `market holidays` / `timings` | `/market/...` | Exchange holidays and trading sessions |
| `analyzer status` / `toggle` | `/analyzer`, `/analyzer/toggle` | Analyzer (sandbox) mode |
| `analyzer pnl` | `/pnl/symbols` | Per-symbol P&L |

### Notifications

| Command | Description |
|---|---|
| `telegram config get` / `set` | Telegram bot configuration |
| `telegram start` / `stop` | Start or stop the Telegram bot |
| `telegram users` / `stats` | Linked users and bot statistics |
| `telegram broadcast` / `notify` | Send a message to all users or one user |
| `telegram preferences get` / `set` | Per-user notification preferences |
| `whatsapp notify` | Send a WhatsApp notification |

### Utilities

| Command | Description |
|---|---|
| `ping` | Check the API key and server reachability |
| `chart get` / `set` | Chart preferences (`set --preferences '{"key":"value"}'`) |
| `stream ltp` / `quote` / `depth` / `orders` | Live WebSocket feeds as NDJSON |
| `profile login` / `logout` / `list` / `switch` | Manage stored profiles |
| `api [METHOD] <path>` | Raw API request escape hatch |
| `doctor` | Check configuration and connectivity |
| `update` | Check for and install CLI updates |
| `version` | Print the CLI version |
| `completion` | Shell completion scripts |

### Strategy name default

Order commands that take a `strategy` field (`order *`, `gtt *`, `position get`, `position close-all`, `option order`, `option multi-order`) default `--strategy` to `openalgo-cli`, so orders placed from the CLI are easy to find in the OpenAlgo order book. Pass `--strategy` to use your own tag.

### JSON-valued flags

Flags that carry arrays or objects (`--orders`, `--legs`, `--positions`, `--symbols`, `--preferences`, `--filters`) take literal JSON, `@path/to/file.json`, or `-` for stdin. Array flags must be JSON arrays, and `--help` lists the item fields each needs. `--help` also marks required flags and lists allowed values for enum flags such as `--exchange` and `--product`; other values are rejected before a request is sent. `--expiry-date` accepts `DDMMMYY` (`27OCT26`) or the `DD-MMM-YY` form `symbol expiry` prints (`27-OCT-26`).

```bash
openalgo data quotes --symbols '[{"symbol":"RELIANCE","exchange":"NSE"},{"symbol":"INFY","exchange":"NSE"}]'
openalgo order basket --orders @basket.json
cat legs.json | openalgo option multi-order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --legs -
```

## Symbols and Order Constants

OpenAlgo uses one symbol format across all brokers:

| Instrument | Format | Example |
|---|---|---|
| Equity | `[Base]` | `RELIANCE`, `SBIN` |
| Future | `[Base][DDMMMYY]FUT` | `BANKNIFTY24APR24FUT` |
| Option | `[Base][DDMMMYY][Strike][CE/PE]` | `NIFTY28MAR2420800CE` |

| Constant | Values |
|---|---|
| Exchange | `NSE` `BSE` `NFO` `BFO` `CDS` `BCD` `MCX` `NCDEX` `NCO` `NSE_INDEX` `BSE_INDEX` `MCX_INDEX` `GLOBAL_INDEX` `CRYPTO` |
| Product | `CNC` (equity delivery), `NRML` (futures and options), `MIS` (intraday) |
| Price type | `MARKET`, `LIMIT`, `SL` (stop-loss limit), `SL-M` (stop-loss market) |
| Action | `BUY`, `SELL` |

Index exchanges (`NSE_INDEX`, `BSE_INDEX`, `MCX_INDEX`, `GLOBAL_INDEX`) are quote-only. Expiry dates for option commands use `DDMMMYY`, for example `27OCT26`. Use `openalgo symbol search --query NIFTY --exchange NFO` to find exact symbols.

## Output

API commands return the OpenAlgo JSON response on stdout by default. They also support CSV output, inline jq filtering, quiet mode, response schemas, and request timeouts.

```bash
openalgo position list
openalgo position list --csv
openalgo order list --jq '.data.orders[] | {orderid, symbol, order_status}'
openalgo account funds --quiet
openalgo data history --symbol NIFTY --exchange NSE_INDEX --interval D --start-date 2025-01-01 --end-date 2026-09-25 --timeout 120
openalgo order list --schema
```

- `--csv` renders the list inside the response (for example the orders in the order book, basket and split results, multi-quote results, or option chain strikes) as CSV. Nested objects become dotted columns (`data.ltp`, `ce.ltp`), arrays are written as JSON text, numbers keep the exact text the server sent, and a list of plain values (expiry dates) becomes one column.
- `--jq` applies a jq expression without requiring an external `jq` install.
- `--schema` prints the response fields without calling the API.
- `--dry-run` on any command that changes state prints the request body without sending it. The API key is never included.

```bash
openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --pricetype LIMIT --price 800 --product CNC --dry-run
```

Non-JSON responses, such as `symbol instruments --format csv` and `data ticker --format txt`, are written to stdout verbatim; `--jq` on them is an error.

Operational commands such as `version`, `doctor`, `profile`, `update`, `completion`, and help emit human-readable text. The exception is `openalgo update --check`, which emits JSON for automation.

Use `openalgo api [METHOD] <path>` for endpoints or fields the generated commands do not cover. Paths are relative to `/api/v1` and the API key is injected for you:

```bash
openalgo api POST /funds
echo '{"symbol":"RELIANCE","exchange":"NSE"}' | openalgo api POST /quotes
openalgo api POST /quotes --body @quote.json --csv
```

The body comes from `--body` (JSON, `@file`, or `-` for stdin) or from piped or redirected stdin; piped stdin that stays silent for 2 seconds counts as no body, so an inherited open pipe never hangs the command. With `--csv`, the response's `data` field is rendered as rows.

## Streaming

`stream` commands connect to the OpenAlgo WebSocket server and print one JSON object per line (NDJSON) until Ctrl-C, `--count N` messages, or `--duration 30s`. `--jq` applies to each message. `--verbose` prints the WebSocket URL and connect time, `--debug` every frame sent and received (API key redacted), and `--schema` the message shape.

```bash
openalgo stream ltp --help
openalgo stream quote --symbols NSE:RELIANCE --count 10
openalgo stream orders --duration 5m
```

The WebSocket URL is derived from the host: `http://host:5000` becomes `ws://host:8765`, additional instances keep the same offset (`http://host:5001` becomes `ws://host:8766`), and `https://domain` becomes `wss://domain/ws`. Override it with a profile `ws_url` field (`openalgo profile login --ws-url ...`), or with `OPENALGO_WS_URL` when the credentials come from `OPENALGO_API_KEY`. With profile credentials `OPENALGO_WS_URL` is ignored, so an environment variable can never redirect a stored key.

## Errors and Exit Codes

Errors are JSON on stderr:

```json
{"error":"Invalid openalgo apikey","status":403,"hint":"Invalid API key, or the operation is blocked by the current mode. Run `openalgo profile login` or check the key under API Key in the OpenAlgo dashboard.","method":"POST","path":"/api/v1/funds"}
```

OpenAlgo can return HTTP 200 with `{"status":"error"}`; the CLI treats that as an error too.

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | API or general error |
| `2` | Authentication error (401 or 403) |

Read-only requests are retried on 429 and 5xx with exponential backoff. Requests that change state (orders, cancels, toggles) are retried only on 429, so a server error never causes a duplicate order. When a state-changing request times out or loses its connection after it was sent, or a gateway answers 502/503/504, the error starts with `outcome unknown`: the order may have been placed, so check `openalgo order list` / `openalgo order trades` before retrying.

## Configuration

| Variable | Description |
|----------|-------------|
| `OPENALGO_API_KEY` | API key from your OpenAlgo server. |
| `OPENALGO_HOST` | Server address used with `OPENALGO_API_KEY`. Defaults to `http://127.0.0.1:5000`. |
| `OPENALGO_WS_URL` | WebSocket URL for `stream` commands. Applies only together with `OPENALGO_API_KEY`; profiles use their `ws_url` field. Defaults to one derived from the host. |
| `OPENALGO_PROFILE` | Profile name to use. |
| `OPENALGO_OUTPUT` | Default output format: `json` or `csv`. |
| `OPENALGO_CONFIG_DIR` | Config directory. Defaults to `~/.config/openalgo`. |
| `OPENALGO_QUIET` | Suppress non-data output such as warnings, hints, and color. |
| `OPENALGO_VERBOSE` | Show HTTP request summaries on stderr. |
| `OPENALGO_DEBUG` | Show HTTP request and response headers and bodies on stderr. |
| `OPENALGO_TRACE` | Show HTTP timing breakdown on stderr. |

Global flags: `--csv`, `--jq`, `--profile`/`-p`, `--verbose`/`-v`, `--debug`, `--trace`, `--quiet`/`-q`, `--schema`, `--timeout` (seconds, default 30).

## Automation Notes

For AI agents, see the [`openalgo-cli` Agent Skill](.agents/skills/openalgo-cli/SKILL.md) for structured installation, authentication, and usage guidance.

- Run `openalgo analyzer status --quiet` before placing orders so you know whether they are sandbox or live.
- Use `--dry-run` to preview any state-changing request.
- Use `--quiet` and check exit codes rather than parsing stderr.
- Keep `--strategy` consistent per workflow so its orders are easy to find in the order book. It is a tag only: `position close-all` and `order cancel-all` act on the whole account.

## Diagnostics

```bash
openalgo doctor                    # Check config, server reachability and the API key
openalgo account funds --verbose   # Request summary on stderr
openalgo account funds --trace     # DNS, TLS, TTFB, and total timing on stderr
openalgo account funds --debug     # Headers and bodies on stderr
```

The API key is always scrubbed from diagnostic output.

## Shell Completions

```bash
openalgo completion bash
openalgo completion zsh
openalgo completion fish
openalgo completion powershell
```

Save the generated completion script where your shell expects it, then open a new shell. Enum-valued flags such as `--exchange`, `--product`, `--pricetype` and `--action` complete with valid values from the spec.

## Self-Update

```bash
openalgo update          # Check for updates and prompt to install
openalgo update --yes    # Check and install without prompting
openalgo update --check  # Machine-readable update check
```

| Install method | Upgrade command |
|---|---|
| Go | `go install github.com/marketcalls/openalgo-cli/cmd/openalgo@latest` |
| Homebrew | `brew upgrade marketcalls/tap/openalgo-cli` |
| Install script (`install.sh` / `install.ps1`) | Re-runs the install script with `OPENALGO_INSTALL_DIR` set to the current binary's directory |

## Development

The CLI is driven by the OpenAPI spec in `api/specs/openalgo-api.json`, curated from the OpenAlgo API docs and the server's request schemas. Do not edit generated files directly. Change the spec or generator, then run `make generate`. See [AGENTS.md](AGENTS.md) for contributor rules.

```bash
make build            # Build binary to bin/openalgo
make test             # Run unit tests
make lint             # Run linter
make check            # Lint, test, and build
make generate         # Regenerate API client and command metadata
make spec-check       # Validate the spec
make test-integration # Run integration tests against an OpenAlgo server
```

Integration tests need an OpenAlgo server in analyzer (sandbox) mode:

```bash
export OPENALGO_TEST_API_KEY=...
export OPENALGO_TEST_HOST=http://127.0.0.1:5000   # optional
make test-integration
```

## Support

- **CLI issues:** Bugs, feature requests, or questions specific to this CLI: [GitHub Issues](https://github.com/marketcalls/openalgo-cli/issues).
- **OpenAlgo server:** Installation, brokers and platform questions: [OpenAlgo repository](https://github.com/marketcalls/openalgo) and [docs.openalgo.in](https://docs.openalgo.in).

## License

Apache 2.0. See [LICENSE](LICENSE).

Parts of this CLI (the command architecture, code generator, output pipeline, and build tooling) are adapted from the [Alpaca CLI](https://github.com/alpacahq/cli), Copyright 2026 Alpaca, also Apache 2.0. See [NOTICE](NOTICE).

The OpenAlgo server itself is licensed under AGPL-3.0. This CLI is a separate program that talks to an OpenAlgo server over its HTTP and WebSocket APIs; it does not include or link any OpenAlgo server code.
