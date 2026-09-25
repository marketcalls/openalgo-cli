---
name: openalgo-cli-regenerate
description: >-
  Keep the CLI in sync with the OpenAlgo server API. Updates the curated
  OpenAPI spec from the OpenAlgo docs and restx_api schemas, runs the code
  generator, registers new operations, updates golden files, and runs all
  checks. Use when an OpenAlgo endpoint was added or changed, generation is
  stale, or the user says "regenerate", "generate", "sync spec", "update
  spec", or "new endpoint".
---

# Generate - Keep the CLI Up to Date

## Pipeline

Run each phase in order. Stop and fix before moving to the next.

### Phase 1: Update the spec

`api/specs/openalgo-api.json` is curated by hand - there is no upstream
spec to download. The sources of truth are in
[marketcalls/openalgo](https://github.com/marketcalls/openalgo):

- `restx_api/*.py` and the marshmallow schemas they use - what the server
  actually accepts (field names, required fields, enums, types)
- `docs/api/**` - descriptions, examples, and response shapes
- `docs/prompt/order-constants.md`, `docs/prompt/symbol-format.md` -
  shared constants

When the docs and the schemas disagree, the schema wins.

Rules for editing the spec:

- One operation per endpoint, with a unique PascalCase `operationId`.
- Omit `apikey` everywhere - the CLI injects it.
- Mark every operation with side effects `"x-mutating": true`.
- Reference `components/schemas` `Exchange`, `Product`, `PriceType`,
  `Action` instead of repeating enums.
- Describe the response (including the list to render for `--csv`) so
  `--schema` and CSV headers stay accurate.

Validate:

```bash
make spec-check
git diff --stat api/specs/
```

If nothing changed, skip to Phase 4 to verify generated code is still fresh.

### Phase 2: Generate code

```bash
make generate
```

This runs `go run ./cmd/generate`, which reads the spec and writes:

| Generated file | Contents |
|---|---|
| `internal/api/openalgo_client.gen.go` | One client method per operation |
| `internal/api/descriptions.gen.go` | Op metadata, flags, response schemas |
| `internal/cmd/commands.gen.go` | Cobra command tree |

**Never edit `*.gen.go` files directly.** Change the spec or generator,
then re-run `make generate`.

### Phase 3: Fix generator failures

The generator enforces exhaustiveness. Common failures and fixes:

#### New operation not registered

Every `operationId` must appear in either `cmdRegistry` or `cmdSkip` in
`cmd/generate/`. Register it with a parent group, a kebab-case `use` name,
and examples:

```go
"FooBar": {
    parent:   "<parent-group>",
    use:      "<subcommand-name>",
    examples: "  openalgo <parent> <subcommand> --flag value",
},
```

Rules for registry entries:
- `parent` must be an existing parent group (or add a new one)
- `examples` is **required** - the generator fails without it
- Examples use OpenAlgo symbol format and prefer `CNC` (equity) / `NRML`
  (F&O) - analyzer (sandbox) mode refuses `MIS` after the
  square-off time (15:15 IST by default)
- Add `defaults` for CLI defaults (order ops with a `strategy` field
  default it to `openalgo-cli`)
- Add `flagAliases` when a property name collides with a global flag
  (`csv`, `jq`, `profile`, `verbose`, `debug`, `trace`, `quiet`, `timeout`,
  `schema`, `help`, `dry-run`, `version`, `output`)
- Set `rowsPath` when the records for `--csv` are not under `data` (for
  example `results` for basket/split/multi-quotes, `chain` for the option
  chain)
- F&O quantities in examples must be lot multiples (check with
  `openalgo symbol get --jq .data.lotsize`); fields whose type differs
  between live and analyzer mode use a type array such as
  `["string", "number"]` in the spec

To skip an operation, add it to `cmdSkip` with a reason.

#### New parent group needed

Add the parent group in the generator. Top-level groups also need root
wiring in `internal/cmd/root.go` under the right cobra group ("Trading",
"Account & Market Data", "Notifications", "Utilities"), or the command will
not appear in `--help` or `--help-all`.

After fixing, re-run `make generate` until it succeeds.

### Phase 4: Update golden files

If the command tree or op metadata changed, golden files will be stale:

```bash
go test ./internal/cmd -run TestCommandTreeGolden -update
go test ./internal/cmd -run TestOpsGolden -update
```

### Phase 5: Run all checks

```bash
make check
```

This runs `golangci-lint`, `go test -race ./...`, and `go build`. Fix any
failures before proceeding.

### Phase 6: Verify against a server

With an OpenAlgo server in analyzer (sandbox) mode:

```bash
export OPENALGO_TEST_API_KEY=...
make test-integration
```

Try the new command by hand with `--dry-run` first, then for real. Never
turn analyzer mode off to test, and never use `MIS` in tests.

### Phase 7: Verify completeness

```bash
git diff --stat
```

- [ ] `api/specs/openalgo-api.json` - spec updated and `make spec-check` passes
- [ ] `internal/api/*.gen.go` - client and descriptions regenerated
- [ ] `internal/cmd/commands.gen.go` - command tree regenerated
- [ ] `internal/cmd/testdata/*.golden` - golden files updated if needed
- [ ] `cmd/generate/` - new operations registered with examples
- [ ] `internal/cmd/root.go` - new top-level groups wired
- [ ] `test/integration/` - tests added for new commands (see `AGENTS.md`)
- [ ] `README.md` command tables and `.agents/skills/openalgo-cli/SKILL.md` updated

## Quick reference

| Task | Command |
|---|---|
| Validate spec | `make spec-check` |
| Generate code | `make generate` |
| Update golden files | `go test ./internal/cmd -run TestCommandTreeGolden -update && go test ./internal/cmd -run TestOpsGolden -update` |
| Lint + test + build | `make check` |
| Integration tests | `make test-integration` (needs a sandbox-mode server) |

## Key files

| File | Role | Editable? |
|---|---|---|
| `api/specs/openalgo-api.json` | Curated OAS spec | Yes - from OpenAlgo docs and restx_api |
| `cmd/generate/` | Generator and command registry | Yes |
| `internal/api/types.go` | Op/FlagDef contract | Yes, with care |
| `internal/api/*.gen.go` | Generated client and metadata | No - regenerate |
| `internal/cmd/commands.gen.go` | Generated Cobra commands | No - regenerate |
| `internal/cmd/root.go` | Root command wiring | Yes |
| `internal/cmd/testdata/*.golden` | Golden test snapshots | Update via `-update` flag |

## Anti-Patterns

- **NEVER** edit `*.gen.go` files directly.
- **NEVER** add a field to the spec that the server's `restx_api` schema does not accept.
- **NEVER** forget `"x-mutating": true` on an operation with side effects - it controls `--dry-run` and retries.
- **NEVER** put an API key in the spec, examples, tests, or golden files.
- **NEVER** skip `make check` after generating.
- **NEVER** update golden files without reviewing the diff.
- **NEVER** add a `cmdRegistry` entry without `examples`, or a `cmdSkip` entry without a reason.
