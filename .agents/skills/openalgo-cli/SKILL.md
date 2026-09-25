---
name: openalgo-cli
description: >
  Install, configure, and use the OpenAlgo CLI - a command-line tool for the
  OpenAlgo trading platform API (self-hosted, broker-agnostic, Indian
  markets). Covers installation (install script, Go), API key authentication,
  profile management, sandbox (analyzer) mode, and agent/automation
  integration. Use when the user asks to install the OpenAlgo CLI, set up
  OpenAlgo credentials, place or cancel orders on NSE/BSE/NFO/MCX from the
  command line, read positions, funds or holdings, get quotes, history,
  option chains or Greeks, stream live prices, or integrate the OpenAlgo CLI
  into scripts, CI pipelines, or AI agent workflows. Keywords: openalgo,
  algo trading, NSE, BSE, NFO, F&O, options, Zerodha, broker, market data,
  analyzer mode, sandbox mode, command line, CLI tool, API key setup.
compatibility: Requires a running OpenAlgo server and its API key. Install script (no Go needed) or Go 1.24+ (go install). macOS, Linux, and Windows supported.
---

# OpenAlgo CLI

The CLI talks to the user's own OpenAlgo server (default `http://127.0.0.1:5000`). OpenAlgo connects to the user's broker; the CLI never talks to the broker directly.

## Install

Check if already installed:

```bash
openalgo version
```

If not installed:

```bash
# macOS / Linux (prebuilt binary, no Go needed)
curl -fsSL https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.sh | sh
# Windows PowerShell
irm https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.ps1 | iex
# or with Go 1.24+
go install github.com/marketcalls/openalgo-cli/cmd/openalgo@latest
```

## Authentication

OpenAlgo uses one API key, generated on the server's API Key page. The key only works against the server that issued it, so the CLI always keeps the key and host together.

### Environment variables (for scripts, CI, agents)

No secrets touch disk. Preferred for automation:

```bash
export OPENALGO_API_KEY=...
export OPENALGO_HOST=http://127.0.0.1:5000   # optional, this is the default
```

When `OPENALGO_API_KEY` is set, it and `OPENALGO_HOST` are the authoritative credentials - any profile on disk is ignored. `OPENALGO_HOST` without `OPENALGO_API_KEY` is ignored.

### Interactive login (stores credentials on disk)

```bash
openalgo profile login
```

Prompts for the host and API key and stores them in `~/.config/openalgo/profiles/<name>.yaml` with 0600 permissions.

### Multiple profiles

```bash
openalgo profile login --name vps      # a second server
openalgo profile switch vps            # switch active profile
openalgo profile list
openalgo doctor                        # show active profile + connectivity
```

## Sandbox mode (analyzer mode)

The OpenAlgo server decides whether orders are simulated or live. In **analyzer mode** (also called **sandbox mode**) orders are simulated by the server; otherwise they go to the broker as live orders. The CLI has no separate live/sandbox switch of its own.

```bash
openalgo analyzer status --quiet
```

Always check this before placing orders. Do not call `openalgo analyzer toggle --mode=false` unless the user explicitly asks to go live.

In analyzer (sandbox) mode, `MIS` orders are refused after the square-off time (15:15 IST by default) until the next session; use `CNC` for equity and `NRML` for futures and options.

## Verify installation

```bash
openalgo version
openalgo ping --quiet
```

Exit code `0` = success, `2` = auth error.

## Agent and automation usage

API commands are non-interactive and have **no confirmations**. These act immediately on everything in scope:

- `openalgo position close-all` - squares off every open position in the account
- `openalgo order cancel-all` - cancels every open order in the account
- `openalgo strategy close-all --strategy-id N` - exits every leg of a strategy
- `openalgo analyzer toggle` - switches the server between sandbox and live

`openalgo profile login` prompts on a TTY. For unattended workflows, prefer environment variables.

### Always use `--quiet`

JSON is the default output format. `--quiet` suppresses all non-data output (warnings, hints):

```bash
openalgo position list --quiet
openalgo data quote --symbol RELIANCE --exchange NSE --quiet
```

Or once for the session:

```bash
export OPENALGO_QUIET=1
```

### Placing orders

Every parameter is a `--flag` named after the API field in kebab-case:

```bash
# Equity delivery, limit
openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 \
  --pricetype LIMIT --price 800 --product CNC

# Stop-loss limit on a future
openalgo order place --symbol BANKNIFTY27OCT26FUT --exchange NFO --action SELL --quantity 30 \
  --pricetype SL --price 51000 --trigger-price 51050 --product NRML

# Target a net position size (buys or sells the difference)
openalgo order smart --symbol SBIN --exchange NSE --action BUY --quantity 1 --position-size 5 --product CNC

# ATM call on NIFTY by offset
openalgo option order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 \
  --offset ATM --option-type CE --action BUY --quantity 65 --product NRML

# Modify / cancel / status
openalgo order modify --orderid 250926000012345 --symbol SBIN --exchange NSE --action BUY \
  --product CNC --pricetype LIMIT --price 795 --quantity 1
openalgo order cancel --orderid 250926000012345
openalgo order status --orderid 250926000012345
```

`order modify` sends `--disclosed-quantity 0 --trigger-price 0` unless you set them (the server requires both). For `--pricetype SL` or `SL-M`, every order command requires an explicit `--trigger-price` above 0. When `--product` or `--pricetype` is omitted the server applies MIS and MARKET; in analyzer (sandbox) mode MIS is refused after the square-off time (15:15 IST by default), so pass CNC (equity) or NRML (F&O). F&O quantities must be a multiple of the lot size: check `openalgo symbol get --symbol <symbol> --exchange NFO --jq .data.lotsize` (NIFTY 65, BANKNIFTY 30 at the time of writing).

`--strategy` defaults to `openalgo-cli`. It is a tag for the order book only - it does not limit what `position close-all` or `order cancel-all` touch; those act on the whole account.

### Dry run

Preview any state-changing request without sending it (the API key is never printed):

```bash
openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --pricetype MARKET --product CNC --dry-run
```

### Retries and duplicates

Read requests retry on 429 and 5xx. Order requests retry only on 429, so a server error never places a duplicate. After a timeout or ambiguous failure on an order, check `openalgo order list --quiet` before resubmitting.

### JSON-valued flags

`--orders`, `--legs`, `--positions`, `--symbols`, `--preferences`, `--filters` take literal JSON, `@file.json`, or `-` for stdin. Array flags must be JSON arrays (`--phones '["919876543210"]'`, not a bare number); `--help` shows the item fields each one needs:

```bash
openalgo data quotes --symbols '[{"symbol":"RELIANCE","exchange":"NSE"},{"symbol":"INFY","exchange":"NSE"}]'
openalgo order basket --orders @basket.json
```

### Symbols and constants

- Equity `RELIANCE`; future `BANKNIFTY24APR24FUT`; option `NIFTY28MAR2420800CE`.
- Exchanges: `NSE BSE NFO BFO CDS BCD MCX NCDEX NCO NSE_INDEX BSE_INDEX MCX_INDEX GLOBAL_INDEX CRYPTO` (index exchanges are quote-only).
- Product: `CNC NRML MIS`. Price type: `MARKET LIMIT SL SL-M`. Action: `BUY SELL`.
- Option expiry: `DDMMMYY`, e.g. `27OCT26`. Find expiries with `openalgo symbol expiry --symbol NIFTY --exchange NFO --instrumenttype options`; its `DD-MMM-YY` output (`27-OCT-26`) is also accepted by `--expiry-date` and converted.
- `--help` marks required flags and lists allowed values for enum flags; values outside the list are rejected before any request is sent.
- Never guess a symbol: `openalgo symbol search --query "NIFTY 27OCT26" --exchange NFO`.

### Structured errors on stderr

```json
{"error":"Invalid openalgo apikey","status":403,"hint":"...","method":"POST","path":"/api/v1/funds"}
```

HTTP 200 with `{"status":"error"}` is also reported as an error.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | API or general error |
| `2` | Authentication error (401/403) |

### Filter and shape output

```bash
openalgo order list --jq '.data.orders[] | {orderid, symbol, order_status}'
openalgo position list --csv
openalgo order list --schema     # response fields, no API call
```

`--jq` works without an external `jq` install.

### Streaming

```bash
openalgo stream ltp --help
openalgo stream quote --symbols NSE:RELIANCE,NSE:SBIN --count 5
openalgo stream orders --duration 10m
```

Output is NDJSON (one JSON object per line). Always bound streams with `--count` or `--duration` in automation.

### Raw API

```bash
openalgo api POST /funds
echo '{"symbol":"RELIANCE","exchange":"NSE"}' | openalgo api POST /quotes
```

Paths are relative to `/api/v1`; the API key is injected.

### Debug API calls

```bash
openalgo account funds --verbose   # one-line request/response summary on stderr
openalgo account funds --trace     # timing breakdown on stderr
openalgo account funds --debug     # full headers and bodies on stderr
```

The API key is always scrubbed from debug output.

## Environment variables

| Variable | Description |
|----------|-------------|
| `OPENALGO_API_KEY` | API key. Env credentials beat any profile. |
| `OPENALGO_HOST` | Server address used with `OPENALGO_API_KEY` (default `http://127.0.0.1:5000`) |
| `OPENALGO_WS_URL` | WebSocket URL for `stream`, only with `OPENALGO_API_KEY`; profiles use `ws_url` (default derived from host) |
| `OPENALGO_PROFILE` | Profile name to use |
| `OPENALGO_OUTPUT` | Default output format (`json`, `csv`) |
| `OPENALGO_CONFIG_DIR` | Config directory (default: `~/.config/openalgo`) |
| `OPENALGO_QUIET` | Suppress non-data output - warnings, hints, color |
| `OPENALGO_VERBOSE` | Show HTTP request summaries on stderr |
| `OPENALGO_DEBUG` | Show HTTP request/response headers and bodies on stderr |
| `OPENALGO_TRACE` | Show HTTP timing breakdown on stderr |

## Self-update

```bash
openalgo update --check --quiet
```

Returns JSON with `update_available` and `update_command`. Run `update_command`, or `openalgo update --yes`.

## Discovering commands

```bash
openalgo --help-all                  # all commands, subcommands, and flags
openalgo option --help               # help for a command group
openalgo option chain --help         # help for a specific command
openalgo option chain --schema       # response fields for a command
openalgo doctor                      # config and server connectivity
```

The installed binary is always current - prefer it over any stale documentation.

## Troubleshooting

**`command not found: openalgo`** - Ensure `$GOPATH/bin` (usually `~/go/bin`) is in your `PATH`.

**Connection refused** - The OpenAlgo server is not running or `OPENALGO_HOST` is wrong. Run `openalgo doctor`.

**Exit code 2 on every command** - The API key is missing, invalid, or belongs to another server. Regenerate it on the server's API Key page and re-run `openalgo profile login`, or fix `OPENALGO_API_KEY`.

**5xx on data or order commands** - Usually the broker session on the server has expired. Log in to the broker again from the OpenAlgo dashboard.

**Order rejected in sandbox mode** - Switch `MIS` to `CNC` (equity) or `NRML` (F&O).

**Error starts with `outcome unknown`** - A write timed out or lost its connection after it was sent. The order may exist: check `openalgo order list` / `openalgo order trades` before retrying, or you may place a duplicate.

## Anti-patterns

- **NEVER** turn analyzer mode off (`analyzer toggle --mode=false`) without explicit user intent - every later order becomes a live order.
- **NEVER** place orders without first checking `openalgo analyzer status` when the mode is not known.
- **NEVER** run `position close-all`, `order cancel-all`, or `strategy close-all` without explicit user intent - they act on the whole account (or the whole strategy) with no confirmation.
- **NEVER** put the API key on the command line or in committed files - use `OPENALGO_API_KEY` or `openalgo profile login`.
- **NEVER** ignore exit code `2` - authentication failed. Do not retry; fix credentials first.
- **NEVER** omit `--quiet` in automation.
- **NEVER** guess symbols or expiries - look them up with `symbol search` / `symbol expiry`.
- **NEVER** call it "paper trading" - the OpenAlgo terms are sandbox mode and analyzer mode.
