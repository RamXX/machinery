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
.PHONY: install install-copy uninstall doctor check oracle verify-formal test help

install: ## Symlink machinery into every agent home (live edits from this repo)
	@for home in $(AGENT_HOMES); do \
	  mkdir -p "$$home/skills" "$$home/agents"; \
	  rm -rf "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/skills/machinery" "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/agents/machinery-fsm-author.md" "$$home/agents/machinery-fsm-author.md"; \
	  ln -sfn "$(SRC)/agents/machinery-build-writer.md" "$$home/agents/machinery-build-writer.md"; \
	  echo "linked machinery -> $$home"; \
	done

install-copy: ## Copy machinery into every agent home (no live edits)
	@for home in $(AGENT_HOMES); do \
	  mkdir -p "$$home/skills" "$$home/agents"; \
	  rm -rf "$$home/skills/machinery"; \
	  cp -R "$(SRC)/skills/machinery" "$$home/skills/machinery"; \
	  cp "$(SRC)/agents/machinery-fsm-author.md" "$$home/agents/"; \
	  cp "$(SRC)/agents/machinery-build-writer.md" "$$home/agents/"; \
	  echo "copied machinery -> $$home"; \
	done

uninstall: ## Remove machinery from every agent home
	@for home in $(AGENT_HOMES); do \
	  rm -rf "$$home/skills/machinery"; \
	  rm -f "$$home/agents/machinery-fsm-author.md" "$$home/agents/machinery-build-writer.md"; \
	  echo "removed machinery from $$home"; \
	done

MODELITH_VERSION ?= v0.4.0

doctor: ## Check dependencies and install status
	@command -v modelith >/dev/null 2>&1 && echo "ok: modelith $$(modelith --version) (pinned: $(MODELITH_VERSION))" || echo "MISSING: modelith (go install github.com/stacklok/modelith/cmd/modelith@$(MODELITH_VERSION))"
	@for home in $(AGENT_HOMES); do \
	  test -e "$$home/skills/machinery" && echo "ok: skill at $$home/skills/machinery" || echo "not installed at $$home: run make install"; \
	  test -e "$$home/agents/machinery-fsm-author.md" && echo "ok: fsm-author agent at $$home/agents" || echo "fsm-author not installed at $$home: run make install"; \
	  test -e "$$home/agents/machinery-build-writer.md" && echo "ok: build-writer agent at $$home/agents" || echo "build-writer not installed at $$home: run make install"; \
	done

test: ## Run the toolchain test suite (pytest via uv)
	@uv run -q -- pytest -q

check: ## Run the deterministic gate suite on the examples
	@python3 skills/machinery/tools/machinery_check.py examples/go-crm/design --impl examples/go-crm/impl
	@python3 skills/machinery/tools/machinery_check.py examples/fulfillment/design
	@python3 skills/machinery/tools/machinery_check.py examples/portfolio-engine/design

oracle: ## Regenerate the transition oracles from the machine JSON (go-crm)
	@python3 skills/machinery/tools/oracle_gen.py examples/go-crm/design/machines

verify-formal: ## Regenerate + TLC-check the whole formal suite for the examples (from source)
	@echo "== go-crm =="; bash skills/machinery/tools/verify_formal.sh examples/go-crm/design
	@echo "== fulfillment =="; bash skills/machinery/tools/verify_formal.sh examples/fulfillment/design
	@echo "== portfolio-engine =="; bash skills/machinery/tools/verify_formal.sh examples/portfolio-engine/design

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'
