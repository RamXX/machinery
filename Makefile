.DEFAULT_GOAL := help
SHELL := /bin/sh
JOBS ?= 4
# The release version, stamped by make build. Only a release commit changes it.
VERSION ?= 2.0.0
BUILD_DATE ?= $(shell date -u +%Y-%m-%d)
VERSION_PKG := github.com/RamXX/machinery/v2/internal/version
TOOLS := $(CURDIR)/.bin
export MACHINERY_TOOLS := $(TOOLS)

.PHONY: help tools build check e2e e2e-one formal crm verify

help: ## List the targets
	@grep -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-10s %s\n", $$1, $$2}'

tools: ## Install the pinned external tools into .bin (modelith, xstate, formal jars)
	GOBIN=$(TOOLS) go install github.com/stacklok/modelith/cmd/modelith@v0.6.0
	GOBIN=$(TOOLS)/old go install github.com/stacklok/modelith/cmd/modelith@v0.5.0
	mkdir -p $(TOOLS)/xstate && cd $(TOOLS)/xstate && { [ -f package.json ] || npm init -y >/dev/null; } && npm install --no-audit --no-fund --save-exact xstate@5.33.4
	go run ./cmd/machinery formal tools

build: ## Build .bin/machinery
	go build -ldflags "-X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)" -o $(TOOLS)/machinery ./cmd/machinery

check: build ## Formatting, go vet, build, and machinery checking itself (layers, end-to-end tests only)
	@out="$$(gofmt -l $$(git ls-files '*.go' | grep -v testdata/))"; if [ -n "$$out" ]; then echo "gofmt: $$out"; exit 1; fi
	go vet ./...
	$(TOOLS)/machinery check

e2e: build ## The whole end-to-end suite against the real binary
	GOMAXPROCS=$(JOBS) go test ./e2e/... -count=1 -p $(JOBS) -parallel $(JOBS) -timeout 30m

e2e-one: build ## One script: make e2e-one S=<script name without .txtar>
	GOMAXPROCS=$(JOBS) go test ./e2e/... -count=1 -run 'TestScripts/$(S)$$' -timeout 10m

formal: build ## The formal_ scripts: TLC and Alloy run for real (needs make tools and Java)
	GOMAXPROCS=$(JOBS) go test ./e2e/... -count=1 -run 'TestScripts/^formal_' -p $(JOBS) -parallel $(JOBS) -timeout 20m

crm: build ## The reference project on a real Postgres: check, then its landing with the probe (needs Docker)
	$(MAKE) -C examples/crm up
	$(MAKE) -C examples/crm check verify; status=$$?; $(MAKE) -C examples/crm down; exit $$status

verify: check e2e crm ## Full verification: check, the e2e suite, the reference project
