# machinery installer. Symlinks (or copies) the skill and agents into ~/.claude.

# machinery is agent-agnostic: it installs the skill under <home>/skills and the
# two role docs under <home>/agents for every agent home listed here. Defaults
# cover Claude Code (~/.claude) and the cross-agent convention (~/.agents).
# Override to add or restrict targets, for example:
#   make install AGENT_HOMES="$(HOME)/.agents"
#   make install AGENT_HOMES="$(HOME)/.claude $(HOME)/.agents /opt/team/.agents"
AGENT_HOMES ?= $(HOME)/.claude $(HOME)/.agents
SRC := $(CURDIR)

.DEFAULT_GOAL := help
.PHONY: install install-copy uninstall build preflight doctor check oracle verify-formal test help

install: ## Symlink machinery into every agent home (live edits from this repo)
	@for home in $(AGENT_HOMES); do \
	  mkdir -p "$$home/skills" "$$home/agents"; \
	  rm -rf "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/skills/machinery" "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/agents/machinery-fsm-author.md" "$$home/agents/machinery-fsm-author.md"; \
	  ln -sfn "$(SRC)/agents/machinery-build-writer.md" "$$home/agents/machinery-build-writer.md"; \
	  echo "linked machinery -> $$home"; \
	done
	@$(MAKE) --no-print-directory preflight

install-copy: ## Copy machinery into every agent home (no live edits)
	@for home in $(AGENT_HOMES); do \
	  mkdir -p "$$home/skills" "$$home/agents"; \
	  rm -rf "$$home/skills/machinery"; \
	  cp -R "$(SRC)/skills/machinery" "$$home/skills/machinery"; \
	  cp "$(SRC)/agents/machinery-fsm-author.md" "$$home/agents/"; \
	  cp "$(SRC)/agents/machinery-build-writer.md" "$$home/agents/"; \
	  echo "copied machinery -> $$home"; \
	done
	@$(MAKE) --no-print-directory preflight

uninstall: ## Remove machinery from every agent home
	@for home in $(AGENT_HOMES); do \
	  rm -rf "$$home/skills/machinery"; \
	  rm -f "$$home/agents/machinery-fsm-author.md" "$$home/agents/machinery-build-writer.md"; \
	  echo "removed machinery from $$home"; \
	done

MODELITH_VERSION ?= v0.4.0
MACH ?= $(CURDIR)/.bin/machinery

.PHONY: build
build: ## Build the machinery Go binary into .bin/
	@mkdir -p .bin && go build -o .bin/machinery ./cmd/machinery

# ensure the binary exists before any tool target uses it
$(MACH):
	@mkdir -p .bin && go build -o .bin/machinery ./cmd/machinery

preflight: ## Check machinery's runtime prerequisites (warns; installs nothing)
	@$(MAKE) --no-print-directory build >/dev/null && $(MACH) preflight

doctor: ## Check prerequisites and install status
	@$(MAKE) --no-print-directory build >/dev/null && $(MACH) doctor

test: ## Run the Go toolchain test suite
	@go test ./...

check: ## Run the deterministic gate suite on the examples (Go binary)
	@$(MACH) check examples/go-crm/design --impl examples/go-crm/impl
	@$(MACH) check examples/fulfillment/design
	@$(MACH) check examples/portfolio-engine/design

oracle: ## Regenerate the transition oracles from the machine JSON (go-crm)
	@$(MACH) oracle examples/go-crm/design/machines

verify-formal: ## Regenerate + TLC-check the whole formal suite for the examples (from source, Go binary)
	@echo "== go-crm =="; $(MACH) verify-formal examples/go-crm/design
	@echo "== fulfillment =="; $(MACH) verify-formal examples/fulfillment/design
	@echo "== portfolio-engine =="; $(MACH) verify-formal examples/portfolio-engine/design

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'
