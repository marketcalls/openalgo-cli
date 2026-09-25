# OpenAlgo CLI

The OpenAlgo CLI brings the [OpenAlgo](https://github.com/marketcalls/openalgo) trading platform to your terminal. Place and manage orders, pull quotes, depth and historical candles, work with option chains and Greeks, run strategies, and stream live market data, with any broker OpenAlgo supports, from a single binary.

It is built for AI agents, scripts and automation first: every parameter is an explicit `--flag`, every API command returns JSON, and errors are structured JSON with meaningful exit codes.

## What You Can Do

- Place, modify and cancel regular, smart, basket, split, options and GTT orders
- Read the order book, trade book, position book, holdings and funds
- Get quotes, multi-quotes, market depth, historical candles and ticker data
- Search symbols, resolve expiries and option symbols, download the instrument master
- Build option chains, compute Greeks and synthetic futures
- Start, stop and inspect strategies from the Strategy Module
- Stream LTP, quotes, depth and order updates over WebSocket
- Check sandbox (analyzer) mode, market holidays and timings
- Send Telegram and WhatsApp notifications

## Installation

**macOS and Linux (one line):**

```bash
curl -fsSL https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.ps1 | iex
```

**Homebrew:** coming soon (`brew install marketcalls/tap/openalgo-cli` once the tap is published).

**Go:**

```bash
go install github.com/marketcalls/openalgo-cli/cmd/openalgo@latest
```

Ensure `$GOPATH/bin` (usually `~/go/bin`) is on your `PATH`.

**From source:**

```bash
git clone https://github.com/marketcalls/openalgo-cli
cd openalgo-cli
make build        # binary at bin/openalgo
```

**Verify the installation:**

```bash
openalgo version
openalgo doctor
```

## Prerequisites

The CLI talks to your own OpenAlgo server. You need:

1. A running OpenAlgo instance (default `http://127.0.0.1:5000`), logged in to your broker.
2. An OpenAlgo API key, generated in the OpenAlgo dashboard under **API Key**.

Your broker credentials never touch the CLI. The OpenAlgo API key resolves the broker session on the server.

## Authentication

**Interactive login** (prompts for host and API key, validates it, stores a profile):

```bash
openalgo profile login
```

**Non-interactive:**

```bash
openalgo profile login --host http://127.0.0.1:5000 --key <your_api_key>
openalgo profile login --host https://algo.example.com --key <your_api_key> --name prod
```

Profiles are stored in `~/.config/openalgo/profiles/` with `0600` permissions.

**Multiple profiles:**

```bash
openalgo profile list
openalgo profile switch <name>
openalgo profile logout <name>
openalgo --profile prod account funds
```

**Multiple OpenAlgo instances.** OpenAlgo is single-user, but many traders run several instances side by side (one per broker or account). Instance *n* listens on port `5000+n-1`, with its WebSocket on `8765+n-1`. Create one profile per instance and pick it with `--profile` (`-p`):

```bash
openalgo profile login --name zerodha --host http://127.0.0.1:5000 --key <zerodha_instance_key>
openalgo profile login --name dhan    --host http://127.0.0.1:5001 --key <dhan_instance_key>
openalgo profile login --name angel   --host https://angel.example.com --key <angel_instance_key>

openalgo -p zerodha position list
openalgo -p dhan account funds
openalgo -p angel stream ltp --symbols NSE:SBIN
```

Each profile keeps its own host and API key, and a key is only ever sent to the host it belongs to. The WebSocket URL is derived per profile: `http://127.0.0.1:5001` maps to `ws://127.0.0.1:8766` (the same offset as the REST port), and `https://domain` maps to `wss://domain/ws`. You can override it with `--ws-url` at login (stored as `ws_url` in the profile). `OPENALGO_WS_URL` applies only when the credentials come from `OPENALGO_API_KEY`; it is ignored with profile credentials.

**Environment variables** (CI, scripts and agents; nothing is written to disk):

```bash
export OPENALGO_API_KEY=<your_api_key>
export OPENALGO_HOST=http://127.0.0.1:5000
openalgo account funds
```

The API key and host always resolve together as one bundle. `OPENALGO_HOST` is only used with `OPENALGO_API_KEY`, so a stray host variable can never send a stored profile key to a different server.

## Configuration Variables

| Variable | Purpose |
|----------|---------|
| `OPENALGO_API_KEY` | OpenAlgo API key |
| `OPENALGO_HOST` | OpenAlgo server URL (default `http://127.0.0.1:5000`) |
| `OPENALGO_WS_URL` | WebSocket URL override, only with `OPENALGO_API_KEY` (default derived from host) |
| `OPENALGO_PROFILE` | Active profile |
| `OPENALGO_OUTPUT` | Default format (`json` or `csv`) |
| `OPENALGO_CONFIG_DIR` | Config directory override (default `~/.config/openalgo`) |
| `OPENALGO_QUIET` | Suppress non-data output |
| `OPENALGO_VERBOSE` | Show HTTP request summaries on stderr |
| `OPENALGO_DEBUG` | Show HTTP headers and bodies on stderr (API key redacted) |
| `OPENALGO_TRACE` | Show HTTP timing breakdown on stderr |

## Output Formats

```bash
openalgo position list                                   # JSON (default)
openalgo position list --csv                             # CSV
openalgo order list --jq '.data.orders[] | {symbol, action, order_status}'
openalgo account funds --quiet                           # no warnings, hints or color
openalgo data history --symbol SBIN --exchange NSE --interval D \
  --start-date 2026-01-01 --end-date 2026-03-31 --timeout 120
```

## Command Categories

**Account:**

```bash
openalgo account funds
openalgo account holdings
openalgo account margin --positions '[{"symbol":"SBIN","exchange":"NSE","action":"BUY","quantity":"10","product":"CNC","pricetype":"MARKET"}]'
```

**Orders:**

```bash
openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --product CNC --pricetype MARKET
openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --product CNC --pricetype LIMIT --price 700
openalgo order smart --symbol SBIN --exchange NSE --action BUY --quantity 1 --position-size 5 --product CNC
openalgo order basket --orders @basket.json
openalgo order split --symbol YESBANK --exchange NSE --action BUY --quantity 100 --splitsize 20 --product CNC
openalgo order modify --orderid <id> --symbol SBIN --exchange NSE --action BUY --product CNC \
  --pricetype LIMIT --price 705 --quantity 1
openalgo order cancel --orderid <id>
openalgo order cancel-all
openalgo order status --orderid <id>
openalgo order list
openalgo order trades
openalgo order place --symbol SBIN --exchange NSE --action BUY --quantity 1 --product CNC --dry-run
```

**GTT orders:**

```bash
openalgo gtt place --trigger-type SINGLE --symbol SBIN --exchange NSE --action BUY --product CNC \
  --quantity 1 --price 700 --triggerprice-sl 700
openalgo gtt list
openalgo gtt modify --trigger-id <id> ...
openalgo gtt cancel --trigger-id <id>
```

**Positions:**

```bash
openalgo position list
openalgo position get --symbol SBIN --exchange NSE --product CNC
openalgo position close-all
```

**Market data:**

```bash
openalgo data quote --symbol RELIANCE --exchange NSE
openalgo data quotes --symbols '[{"symbol":"RELIANCE","exchange":"NSE"},{"symbol":"NIFTY","exchange":"NSE_INDEX"}]'
openalgo data depth --symbol SBIN --exchange NSE
openalgo data history --symbol SBIN --exchange NSE --interval 5m --start-date 2026-09-01 --end-date 2026-09-25
openalgo data intervals
openalgo data ticker --symbol NSE:RELIANCE --interval D --from 2026-01-01 --to 2026-09-25
openalgo data ticker --symbol NSE:RELIANCE --interval D --from 2026-01-01 --to 2026-09-25 --format txt
```

**Symbols:**

```bash
openalgo symbol get --symbol NIFTY27OCT2626000CE --exchange NFO
openalgo symbol search --query RELIANCE --exchange NSE
openalgo symbol expiry --symbol NIFTY --exchange NFO --instrumenttype options
openalgo symbol instruments --exchange NSE_INDEX
openalgo symbol instruments --exchange NSE --format csv > nse.csv
```

**Options:**

```bash
openalgo option chain --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --strike-count 10
openalgo option symbol --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ATM --option-type CE
openalgo option greeks --symbol NIFTY27OCT2626000CE --exchange NFO
openalgo option synthetic-future --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26
openalgo option order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --offset ATM \
  --option-type CE --action BUY --quantity 65 --product NRML
openalgo option multi-order --underlying NIFTY --exchange NSE_INDEX --expiry-date 27OCT26 --legs @iron_condor.json
```

**Strategies:**

```bash
openalgo strategy list
openalgo strategy status --strategy-id 7
openalgo strategy start --strategy-id 7 --mode sandbox
openalgo strategy stop --strategy-id 7
openalgo strategy runs --strategy-id 7
openalgo strategy events --strategy-id 7 --severity critical
```

**Streaming (WebSocket):**

```bash
openalgo stream ltp --symbols NSE:RELIANCE,NSE:SBIN
openalgo stream quote --symbols NSE:RELIANCE --count 10
openalgo stream depth --symbols NSE:SBIN --depth 5 --duration 30s
openalgo stream orders
```

Streams print one JSON object per line (NDJSON) and stop on Ctrl-C, after `--count` messages or after `--duration`.

**Market calendar and analyzer:**

```bash
openalgo market holidays --year 2026
openalgo market timings --date 2026-09-25
openalgo analyzer status
openalgo analyzer pnl
openalgo ping
```

**Notifications:**

```bash
openalgo telegram notify --username trader1 --message "Order filled"
openalgo whatsapp notify --self --message "Stop loss hit"
```

**Raw API access:**

```bash
openalgo api POST /funds
openalgo api POST /quotes --body '{"symbol":"SBIN","exchange":"NSE"}'
echo '{"symbol":"SBIN","exchange":"NSE"}' | openalgo api POST /depth
openalgo api GET /instruments --query "exchange=NSE_INDEX"
```

Paths are relative to `/api/v1`, and the API key is added for you.

## Windows Shells

The examples on this page use POSIX shell quoting, which works as-is in Git Bash and WSL. PowerShell and Command Prompt pass quotes differently. Every form below is exercised on each commit by the `windows-shells` CI job.

The most portable option in any shell is a file or stdin, which needs no quote escaping:

```powershell
Set-Content orders.json '[{"symbol":"SBIN","exchange":"NSE","action":"BUY","quantity":1,"product":"CNC"}]'
openalgo order basket --orders "@orders.json" --dry-run
'[{"symbol":"SBIN","exchange":"NSE","action":"BUY","quantity":1,"product":"CNC"}]' | openalgo order basket --orders - --dry-run
```

In PowerShell, quote `"@orders.json"`: an unquoted `@name` is PowerShell's splatting syntax. Files written by Windows tools with a byte order mark (UTF-8 BOM, or UTF-16 from `Out-File` and `>` in Windows PowerShell 5.1) are read correctly.

**PowerShell 7.3+** passes single-quoted strings through unchanged, so the POSIX examples work:

```powershell
openalgo data quote --symbol SBIN --exchange NSE --jq '.data.ltp'
openalgo order basket --orders '[{"symbol":"SBIN","exchange":"NSE","action":"BUY","quantity":1,"product":"CNC"}]' --dry-run
```

**Windows PowerShell 5.1** (the built-in `powershell.exe`) strips double quotes inside arguments passed to programs. Escape each inner double quote with a backslash, or use a file or stdin:

```powershell
openalgo order basket --orders '[{\"symbol\":\"SBIN\",\"exchange\":\"NSE\",\"action\":\"BUY\",\"quantity\":1,\"product\":\"CNC\"}]' --dry-run
openalgo order list --jq '.data.orders[] | select(.action==\"BUY\")'
```

**Command Prompt (`cmd.exe`)** has no single quotes. Wrap arguments in double quotes and escape inner double quotes with a backslash:

```bat
openalgo data quote --symbol SBIN --exchange NSE --jq ".data.ltp"
openalgo order basket --orders "[{\"symbol\":\"SBIN\",\"exchange\":\"NSE\",\"action\":\"BUY\",\"quantity\":1,\"product\":\"CNC\"}]" --dry-run
openalgo order basket --orders @orders.json --dry-run
```

## Discoverability

```bash
openalgo --help               # command groups
openalgo --help-all           # full tree with every flag
openalgo order place --help   # one command
openalgo order list --schema  # response schema, no API call
```

## Sandbox Mode (Analyzer Mode)

OpenAlgo's analyzer mode routes every order to the sandbox engine instead of the broker, so you can test strategies with no real money. The CLI does not change modes on its own:

```bash
openalgo analyzer status             # is sandbox mode on?
openalgo analyzer toggle --mode=true # switch to sandbox mode
```

In analyzer (sandbox) mode, `MIS` orders are refused after the square-off time (15:15 IST by default) until the next session; use `CNC` for equity and `NRML` for futures and options.

## Symbol Format and Order Constants

| Instrument | Format | Example |
|---|---|---|
| Equity | base symbol | `RELIANCE`, `SBIN` |
| Futures | `[Base][Expiry]FUT` | `BANKNIFTY24APR24FUT` |
| Options | `[Base][Expiry][Strike][CE/PE]` | `NIFTY28MAR2420800CE` |

| Constant | Values |
|---|---|
| Exchange | `NSE` `BSE` `NFO` `BFO` `CDS` `BCD` `MCX` `NCDEX` `NCO` `NSE_INDEX` `BSE_INDEX` `MCX_INDEX` `GLOBAL_INDEX` `CRYPTO` |
| Product | `CNC` (delivery), `NRML` (F&O carry), `MIS` (intraday) |
| Price type | `MARKET` `LIMIT` `SL` `SL-M` |
| Action | `BUY` `SELL` |

Enum flags tab-complete in your shell.

## AI Agent Integration

- No confirmation prompts: every command runs immediately
- Structured JSON errors on stderr with exit codes (`0` success, `1` API error, `2` auth error)
- `--dry-run` on every order-changing command prints the request body without sending it
- `--schema` shows the response shape without calling the API
- `--quiet` suppresses secondary output
- Rate-limited requests (HTTP 429) are retried automatically. Order-changing requests are never retried after a server error, because OpenAlgo has no idempotency key and a retry could place a duplicate order.

Agents can load the bundled skill at [`.agents/skills/openalgo-cli/SKILL.md`](../.agents/skills/openalgo-cli/SKILL.md).

## Self-Update

```bash
openalgo update           # check and prompt to install
openalgo update --yes     # install without prompting
openalgo update --check   # machine-readable JSON
```

## Shell Completions

```bash
openalgo completion bash
openalgo completion zsh
openalgo completion fish
openalgo completion powershell
```

## Important Considerations

- The CLI never switches between live and sandbox mode on its own. Check `openalgo analyzer status` before trading.
- Commands run immediately, with no confirmation. `position close-all`, `order cancel-all` and `strategy close-all` act on every position, order or leg.
- Treat your OpenAlgo API key like a password. Prefer environment variables in automation. The CLI redacts the key in `--debug` output.
- Orders go straight to your broker through OpenAlgo. Review them before you send.
- OpenAlgo rate limits apply (defaults: 10 order requests and 50 other requests per second).

## Resources

- CLI source: [github.com/marketcalls/openalgo-cli](https://github.com/marketcalls/openalgo-cli) (Apache 2.0)
- OpenAlgo platform: [github.com/marketcalls/openalgo](https://github.com/marketcalls/openalgo)
- OpenAlgo documentation: [docs.openalgo.in](https://docs.openalgo.in)

## Disclosure

The OpenAlgo CLI is a tool for talking to your own OpenAlgo server. Nothing it outputs is investment advice. Trading involves risk; test in sandbox mode first.
