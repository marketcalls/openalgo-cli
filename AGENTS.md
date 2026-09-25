# OpenAlgo CLI

A single-binary CLI for the OpenAlgo trading platform API. Think `gh` for GitHub, `stripe` for Stripe - but for a self-hosted OpenAlgo server.

## Core Principle: Generate Everything

The CLI is driven by an OpenAPI spec. Maximize what's generated, minimize what's hand-written. **Do not edit generated files directly** (`internal/api/*.gen.go`, `internal/cmd/commands.gen.go`). Change the spec or the generator (`cmd/generate/`), then `make generate`.

Every `operationId` in the spec must appear in either `cmdRegistry` or `cmdSkip` in the generator, and every registry entry must carry examples. The generator fails loudly otherwise.

## The spec

`api/specs/openalgo-api.json` is an OpenAPI 3.1 document **curated by hand** from the OpenAlgo server: the API docs under `docs/api/` and the request schemas under `restx_api/` in [marketcalls/openalgo](https://github.com/marketcalls/openalgo). There is no upstream spec to download.

- The spec must match what the server actually accepts. When a field, enum, or endpoint changes upstream, update the spec from the server's `restx_api` schema (the docs can lag), then `make spec-check && make generate`.
- `apikey` is omitted from every operation - the CLI injects it.
- Operations with side effects carry `"x-mutating": true`. This drives `--dry-run` and the retry policy (mutating requests are never retried on 5xx). Get it right.
- Shared enums live in `components/schemas` (`Exchange`, `Product`, `PriceType`, `Action`); reference them instead of repeating values.

## Design Philosophy

- **Agent-first**: the primary consumer is an AI agent. All parameters are explicit `--flag value` - no positional arguments. Exceptions: `openalgo api [METHOD] <path>` (raw escape hatch) and the `profile` helpers.
- **Flag names are the OAS property names in kebab-case** (`trigger_price` -> `--trigger-price`). Names that collide with global flags must be aliased in the registry.
- **No backward compatibility**: pre-1.0. No aliases, shims, or deprecation wrappers. Just make the change.

## After every change

```
make check     # lint + test + build
```

Fix any failures you introduce before moving on.

When a refactor changes command names, flags, or output shape, review `test/integration/` and update any affected tests so they stay in sync.

## Output contract

Commands fall into two categories with different output rules:

**API commands** (order, position, account, data, option, strategy, etc.) - call OpenAlgo endpoints and return the OpenAlgo JSON response on stdout. These are the agent pipeline. They support `--csv`, `--jq`, `--quiet`, `--schema`, and `--timeout`; mutating ones also support `--dry-run`. Errors go to stderr as JSON: `{"error","status","hint","method","path"}`.

**Operational commands** (`version`, `doctor`, `profile *`, `update`, `completion`, `--help`, `--help-all`, `--schema`) - manage the CLI itself. These emit human-readable text on stdout. The machine-readable signal is the **exit code** (0 = success, non-zero = failure), not the output format. Do not convert these to JSON - an agent that needs to verify connectivity runs `openalgo ping --quiet`, not `openalgo doctor`.

The one exception is `update --check`, which emits JSON because agents need to programmatically decide whether to upgrade.

**Streaming commands** (`stream *`) emit NDJSON: one JSON object per line on stdout.

**Rule of thumb:** if a command hits the OpenAlgo API and returns API data, it emits JSON. If it manages the CLI's own state or helps a human troubleshoot, it emits text.

Exit codes: 0 ok, 1 API or general error, 2 authentication error. OpenAlgo sometimes returns HTTP 200 with `{"status":"error"}` - that is an error, not a success.

## Vocabulary and style

- Say **sandbox mode** or **analyzer mode**. Never "paper trading" or "virtual trading".
- No emojis or icon glyphs anywhere - output, docs, comments, commit messages. Use plain words like `ok` and `FAIL`.
- Symbols use the OpenAlgo format: `RELIANCE`, `BANKNIFTY24APR24FUT`, `NIFTY28MAR2420800CE`.
- Never print, log, or persist the API key outside a profile file. It must be scrubbed from `--debug`, `--verbose`, `--dry-run`, and error output.

## Design Notes

- **`FlagDef.OASName` must stay**: flag names are kebab-case (`trigger-price`), OAS names are snake_case (`trigger_price`). The mapping `_ -> -` is lossy, so keep both fields.
- **Credentials are an atomic bundle**: the API key and host always come from the same source (env or one profile). Never mix a profile key with an env host - that would leak the key to another server.
- **`--strategy` defaults to `openalgo-cli`** on order ops through the registry `defaults`, so CLI orders are identifiable in the order book.

## Integration tests (`test/integration/`)

Gated by `//go:build integration` and require `OPENALGO_TEST_API_KEY` (plus optional `OPENALGO_TEST_HOST`, default `http://127.0.0.1:5000`, and `OPENALGO_TEST_WS_URL`). Without the key the suite prints a notice and exits 0. They run against a real OpenAlgo server that **must be in analyzer (sandbox) mode**. CI skips the job when the secret is absent, because OpenAlgo is self-hosted.

Rules:
- **Always run `make test-integration` after editing** when a server is available - `make check` only runs unit tests.
- **Sandbox mode only** - check `analyzer status` at the start and skip if the server is live. Never place orders against a live server.
- **Never toggle analyzer mode off** - never call `analyzer toggle --mode=false`, in tests or anywhere else.
- **Never use `MIS` in tests** - in analyzer (sandbox) mode `MIS` orders are refused after the square-off time (15:15 IST by default) until the next session. Use `CNC` for equity and `NRML` for futures and options.
- **Write tests must clean up** - use `t.Cleanup()` to cancel orders placed by the test. Never leave open orders behind. Prefer far-from-market `LIMIT` orders so nothing fills.
- **Never call `position close-all` or `order cancel-all` in tests** - they act on the whole account regardless of `--strategy`. Cancel test orders one by one by `--orderid`. Never call `strategy close-all` on a strategy the test did not create.
- **Read-only tests must call `t.Parallel()`**.
- **Flat tests for independent calls, sub-tests for sequential chains** - use `t.Run` only when steps depend on prior state (place -> status -> modify -> cancel).
- **Never hardcode the API key** in tests, fixtures, or docs.
- **Some endpoints depend on the broker or server setup** (Telegram, WhatsApp, strategies, Greeks). Tests for these must accept either a valid response or a structured JSON error.
- **One file per feature area** - `order_test.go`, `data_test.go`, `option_test.go`, etc. Cross-cutting flows go in `e2e_test.go`.

## Keep docs in sync

When a change affects CLI behavior, update any stale docs:

- `README.md` - user-facing documentation
- `.agents/skills/openalgo-cli/SKILL.md` - agent-facing skill
- `.agents/skills/openalgo-cli-regenerate/SKILL.md` - spec and generator workflow

Code is the source of truth. If a doc contradicts the code, update the doc.
