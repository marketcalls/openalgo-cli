VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BINARY := openalgo

# The spec is curated by hand from the OpenAlgo docs (docs/api) and the
# request schemas in restx_api - there is no upstream URL to fetch it from.
SPEC := api/specs/openalgo-api.json

.PHONY: build install test test-integration lint check clean generate spec-check release

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/openalgo

install:
	go install $(LDFLAGS) ./cmd/openalgo

test:
	go test -race ./...

check: lint test build

test-integration:
	go test -v -tags integration -count=1 -timeout 5m ./test/integration/...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

generate:
	go run ./cmd/generate

spec-check:
	@python3 -m json.tool $(SPEC) >/dev/null && echo "$(SPEC): valid JSON"
	@python3 -c 'import json,sys; s=json.load(open("$(SPEC)")); ops=[o["operationId"] for p in s["paths"].values() for o in p.values() if isinstance(o,dict) and "operationId" in o]; dup=sorted({x for x in ops if ops.count(x)>1}); sys.exit("duplicate operationIds: "+", ".join(dup)) if dup else print(str(len(ops))+" operations, operationIds unique")'
	@printf 'Run `make generate` to apply spec changes.\n'

release:
	@if [ -n "$$(git status --porcelain)" ]; then echo "error: working tree is dirty" >&2; exit 1; fi
	@git fetch --tags origin
	@LAST=$$(git tag -l 'v0.0.*' --sort=-v:refname | head -1); \
	if [ -z "$$LAST" ]; then NEXT=v0.0.1; \
	else NEXT=v0.0.$$((  $${LAST##*.} + 1  )); fi; \
	echo "$$LAST -> $$NEXT"; \
	read -p "Tag $$NEXT and push? [y/N] " confirm; \
	if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then echo "Aborted."; exit 1; fi; \
	git tag -a "$$NEXT" -m "Release $$NEXT" && git push origin "$$NEXT"; \
	echo "Tagged $$NEXT - release workflow will run at https://github.com/marketcalls/openalgo-cli/actions"
