.DEFAULT_GOAL := help
SHELL := /bin/sh
JOBS ?= 4
TOOLS := $(CURDIR)/.bin
export MACHINERY_TOOLS := $(TOOLS)

.PHONY: help tools build check e2e e2e-one crm verify

help: ## List the targets
	@grep -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-10s %s\n", $$1, $$2}'

tools: ## Install the pinned external tools into .bin (modelith, xstate, formal jars)
	GOBIN=$(TOOLS) go install github.com/stacklok/modelith/cmd/modelith@v0.6.0
	GOBIN=$(TOOLS)/old go install github.com/stacklok/modelith/cmd/modelith@v0.5.0
	mkdir -p $(TOOLS)/xstate && cd $(TOOLS)/xstate && { [ -f package.json ] || npm init -y >/dev/null; } && npm install --no-audit --no-fund --save-exact xstate@5.33.4
	go run ./cmd/machinery formal tools

build: ## Build .bin/machinery
	go build -o $(TOOLS)/machinery ./cmd/machinery

check: build ## Formatting, go vet, build, and no unit tests in this repository
	@out="$$(gofmt -l $$(git ls-files '*.go' | grep -v testdata/))"; if [ -n "$$out" ]; then echo "gofmt: $$out"; exit 1; fi
	go vet ./...
	@out="$$(git ls-files '*_test.go' | grep -v -E '^(e2e|examples/[^/]+/e2e)/' || true)"; if [ -n "$$out" ]; then echo "unit test outside e2e: $$out"; exit 1; fi

e2e: build ## The whole end-to-end suite against the real binary
	GOMAXPROCS=$(JOBS) go test ./e2e/... -count=1 -p $(JOBS) -parallel $(JOBS) -timeout 30m

e2e-one: build ## One script: make e2e-one S=<script name without .txtar>
	GOMAXPROCS=$(JOBS) go test ./e2e/... -count=1 -run 'TestScripts/$(S)$$' -timeout 10m

crm: build ## The reference project: its own check and landing (needs Docker)
	$(MAKE) -C examples/crm verify

verify: check e2e crm ## Full verification: check, the e2e suite, the reference project
