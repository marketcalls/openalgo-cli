# Changelog

All notable changes to the OpenAlgo CLI. The CLI is an alpha preview: commands, flags and output may still change between releases.

## v0.0.2 - 2026-09-26

Windows hardening. If you use the CLI on macOS or Linux nothing changes, but upgrading is still recommended.

### Fixed

- **`openalgo update` on Windows.** The update replaced `openalgo.exe` while it was still running, which Windows refuses, so the upgrade failed with a "file in use" error. `install.ps1` now moves the running binary aside (`openalgo.exe.old`) before copying the new one, and removes the leftover on the next install. The fix lives in the install script on `main`, so updating from v0.0.1 also works.
- **Install method detection on Windows.** A binary installed with `go install` (for example `C:\Users\you\go\bin\openalgo.exe`) was treated as a script install because paths were compared with forward slashes only. `openalgo update` now suggests `go install ...@latest` for those binaries, as it already did on macOS and Linux.
- **JSON files written by Windows tools.** `--orders @file.json` (and every other JSON-valued flag, stdin with `-`, and `openalgo api --body`) now accepts files with a UTF-8 byte order mark or UTF-16 encoding, which Windows PowerShell 5.1 produces with `Out-File`, `>` and `Set-Content -Encoding utf8`. Before, these failed with "invalid JSON".

### Added

- **Windows shells section in the README** with tested quoting for PowerShell 7.3+, Windows PowerShell 5.1 and Command Prompt, and guidance to prefer `@file` or stdin for JSON flags.
- **CI on Linux, macOS and Windows.** Unit tests (with the race detector) and a binary smoke test run on all three on every commit, `go vet` covers all six release targets, and a `windows-shells` job runs every documented Windows quoting form in PowerShell 7, Windows PowerShell 5.1, Command Prompt and Git Bash.
- **Install smoke test workflow.** On every release, `install.sh` (Linux amd64, Linux arm64, macOS) and `install.ps1` (PowerShell 7 and 5.1) install the new version, then install v0.0.1 and run `openalgo update --yes` to prove an in-place upgrade works.

### Not changed

- Profiles are written with mode 0600 on macOS and Linux. Windows has no equivalent permission bits; the profile file inherits your user folder's access rules.
- Homebrew is still not available (`marketcalls/tap` is not published yet). Use the install scripts or `go install`.

## v0.0.1 - 2026-09-26

First public release.

### Added

- **65 OpenAlgo REST operations** generated from a curated OpenAPI spec: orders (regular, smart, basket, split, options, multi-leg options, GTT), positions, funds, holdings, margin, quotes, depth, history, ticker, symbol search and expiry, instruments, option chain, Greeks and synthetic futures, market holidays and timings, analyzer (sandbox) mode, the Strategy Module, Telegram and WhatsApp.
- **WebSocket streaming:** `openalgo stream ltp|quote|depth|orders`, newline-delimited JSON, with `--count` and `--duration`.
- **Profiles** per OpenAlgo instance (`openalgo profile login --name ... --host ...`, `-p name`). The API key and host always resolve together, so a key is only ever sent to the server it belongs to. The WebSocket URL follows the instance's port offset (5001 maps to 8766).
- **Agent-first output:** JSON on stdout, `--csv`, built-in `--jq`, `--schema` (response shape without an API call), `--dry-run` on every order-changing command, and structured JSON errors on stderr with exit codes 0, 1 and 2.
- **Safe retries:** rate-limited requests (429) are retried; order-changing requests are never replayed after a server error, because OpenAlgo has no idempotency key.
- **Diagnostics:** `openalgo doctor`, `--verbose`, `--debug` and `--trace`, with the API key always redacted.
- **Install scripts** for macOS and Linux (`install.sh`) and Windows (`install.ps1`) that verify release checksums, plus `openalgo update` for self-update.
- **Agent skill** at `.agents/skills/openalgo-cli/SKILL.md`, installable with `npx skills add marketcalls/openalgo-cli --skill openalgo-cli`.

### Known limitations

- `openalgo update` fails on Windows (fixed in v0.0.2).
- Windows was cross-compiled but not tested in this release.
