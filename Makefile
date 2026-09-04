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
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
SRC := $(CURDIR)
INTERNAL_VERSION := v0.6.7
MODELITH_VERSION := v0.4.0
MACH ?= $(CURDIR)/.bin/machinery
EXAMPLE_INVENTORY := scripts/example-inventory.sh
MODELITH_INVENTORY := scripts/modelith-inventory.sh
MODELITH_RENDER := scripts/modelith-render.sh
# Single source of truth for the linter version, shared with CI (ci.yml reads
# the same file) and the local preflight gate.
GOLANGCI_VERSION := $(shell cat .golangci-version 2>/dev/null)
ACTIONLINT_VERSION := $(shell cat .actionlint-version 2>/dev/null)
# Where dev-link copies the built binary. Override: INSTALL_DIR=/usr/local/bin
INSTALL_DIR ?= $(HOME)/.local/bin

.DEFAULT_GOAL := help
.PHONY: build dev-link uninstall test test-install golden golden-update check verify-formal modelith-inventory modelith-render modelith-render-check preflight hooks lint-install help

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
	@$(EXAMPLE_INVENTORY) rows | while IFS=$$'\t' read -r -a row; do \
		design="$${row[0]}"; impl="$${row[1]}"; complete="$${row[5]}"; \
		args=("$$design" --warnings-as-errors); \
		if [[ "$$impl" != - ]]; then args+=(--impl "$$impl"); fi; \
		if [[ "$$complete" == yes ]]; then args+=(--complete); fi; \
		$(MACH) check "$${args[@]}"; \
	done

verify-formal: build ## Regenerate + TLC-check the whole formal suite across the examples (needs Java)
	@$(EXAMPLE_INVENTORY) formal | while IFS= read -r design; do \
		echo "== $$design =="; \
		$(MACH) verify-formal "$$design"; \
	done

modelith-render: ## Regenerate Modelith renders and mechanically normalize house style
	@$(MODELITH_RENDER) render examples "$(MODELITH_VERSION)"

modelith-inventory: ## Require an exact source/render pair for every authoritative Modelith model
	@$(MODELITH_INVENTORY) check

modelith-render-check: modelith-inventory modelith-render ## Regenerate committed Modelith renders and reject byte drift
	@$(MODELITH_INVENTORY) git-diff

preflight: ## Run every required local CI/formal gate, cheapest first
	@scripts/preflight.sh

hooks: ## Install the git pre-push hook (points core.hooksPath at .githooks)
	@git config core.hooksPath .githooks
	@chmod +x .githooks/pre-push scripts/preflight.sh
	@echo "pre-push hook installed. Bypass once with: SKIP_PREFLIGHT=1 git push"

lint-install: ## Install the pinned static-analysis tools so local matches CI exactly
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	@go install github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)
	@echo "installed golangci-lint $(GOLANGCI_VERSION) to $(shell go env GOPATH)/bin"
	@echo "installed actionlint $(ACTIONLINT_VERSION) to $(shell go env GOPATH)/bin"

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'
