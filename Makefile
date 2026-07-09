# machinery: contributor tasks only.
#
# End users do NOT need this file, Make, or a clone. Install the binary and let
# it install the skill:
#   curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
#   machinery install            # place the skill + role docs into your agent homes
#   machinery uninstall          # remove them
# Every design command is a machinery subcommand run on your own path, no clone:
#   machinery check|verify-formal|oracle|lint|... <your-design>
#
# The targets below build and test machinery itself and need the Go source tree.

AGENT_HOMES ?= $(HOME)/.agents $(HOME)/.claude
SRC := $(CURDIR)
INTERNAL_VERSION := v0.1.8-dev
MACH ?= $(CURDIR)/.bin/machinery
# Single source of truth for the linter version, shared with CI (ci.yml reads
# the same file) and the local preflight gate.
GOLANGCI_VERSION := $(shell cat .golangci-version 2>/dev/null)
# Where dev-link copies the built binary. Override: INSTALL_DIR=/usr/local/bin
INSTALL_DIR ?= $(HOME)/.local/bin

.DEFAULT_GOAL := help
.PHONY: build dev-link uninstall test test-install golden golden-update check verify-formal preflight hooks lint-install help

build: ## Build the machinery binary from source into .bin/machinery (needs Go)
	@mkdir -p .bin && go build -ldflags "-s -w -X main.version=$(INTERNAL_VERSION)" -o .bin/machinery ./cmd/machinery

dev-link: build ## DEVELOPER: live-symlink skill+agents from this checkout into agent homes + binary on PATH
	@for home in $(AGENT_HOMES); do \
	  mkdir -p "$$home/skills" "$$home/agents"; \
	  rm -rf "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/skills/machinery" "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/agents/machinery-fsm-author.md" "$$home/agents/machinery-fsm-author.md"; \
	  ln -sfn "$(SRC)/agents/machinery-build-writer.md" "$$home/agents/machinery-build-writer.md"; \
	  echo "linked machinery -> $$home"; \
	done
	@mkdir -p "$(INSTALL_DIR)" && cp "$(MACH)" "$(INSTALL_DIR)/machinery"
	@echo "installed $(MACH) -> $(INSTALL_DIR)/machinery"

uninstall: ## Remove machinery from every agent home
	@for home in $(AGENT_HOMES); do \
	  rm -rf "$$home/skills/machinery"; \
	  rm -f "$$home/agents/machinery-fsm-author.md" "$$home/agents/machinery-build-writer.md"; \
	  echo "removed machinery from $$home"; \
	done

test: ## Run the full Go test suite (needs Go)
	@go test ./...

test-install: ## Verify the install path lays down the canonical-copy + symlink topology (offline)
	@go test -count=1 -run '[Ii]nstall' ./cmd/machinery ./internal/install

golden: ## Run the golden-corpus byte-for-byte regression net
	@go test -count=1 -run TestGolden ./cmd/machinery

golden-update: ## Re-capture the golden corpus from the current binary (review the diff!)
	@go test -count=1 -run TestGolden ./cmd/machinery -update

check: build ## Run the deterministic gate suite across the bundled examples
	@$(MACH) check examples/go-crm/design --impl examples/go-crm/impl
	@$(MACH) check examples/fulfillment/design
	@$(MACH) check examples/portfolio-engine/design
	@$(MACH) check examples/checkout-split/parent/design
	@$(MACH) check examples/checkout-split/orders/design
	@$(MACH) check examples/checkout-split/payments/design

verify-formal: build ## Regenerate + TLC-check the whole formal suite across the examples (needs Java)
	@echo "== go-crm =="; $(MACH) verify-formal examples/go-crm/design
	@echo "== fulfillment =="; $(MACH) verify-formal examples/fulfillment/design
	@echo "== portfolio-engine =="; $(MACH) verify-formal examples/portfolio-engine/design
	@echo "== checkout-split/orders =="; $(MACH) verify-formal examples/checkout-split/orders/design
	@echo "== checkout-split/payments =="; $(MACH) verify-formal examples/checkout-split/payments/design

preflight: ## Run the local CI mirror (ci.yml, cheapest gate first); green here means green in CI
	@scripts/preflight.sh

hooks: ## Install the git pre-push hook (points core.hooksPath at .githooks)
	@git config core.hooksPath .githooks
	@chmod +x .githooks/pre-push scripts/preflight.sh
	@echo "pre-push hook installed. Bypass once with: SKIP_PREFLIGHT=1 git push"

lint-install: ## Install the pinned golangci-lint (.golangci-version) so local matches CI exactly
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
	  | sh -s -- -b "$(shell go env GOPATH)/bin" "$(GOLANGCI_VERSION)"
	@echo "installed golangci-lint $(GOLANGCI_VERSION) to $(shell go env GOPATH)/bin"

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'
