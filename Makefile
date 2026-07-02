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
.PHONY: install install-copy uninstall preflight doctor check oracle verify-formal test help

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

preflight: ## Check machinery's runtime prerequisites (warns; installs nothing)
	@echo "machinery prerequisites:"
	@if command -v modelith >/dev/null 2>&1; then echo "  ok       modelith $$(modelith --version 2>&1 | awk '{print $$NF}') (pinned $(MODELITH_VERSION))"; \
	 else echo "  MISSING  modelith (Phase 1 domain model lint/render) -- install: go install github.com/stacklok/modelith/cmd/modelith@$(MODELITH_VERSION)"; fi
	@if command -v python3 >/dev/null 2>&1; then echo "  ok       python3 $$(python3 --version 2>&1 | awk '{print $$2}') (the gate tools need 3.10+)"; \
	 else echo "  MISSING  python3 3.10+ (the deterministic gate tools)"; fi
	@if python3 -c 'import yaml' >/dev/null 2>&1; then echo "  ok       PyYAML"; \
	 else echo "  MISSING  PyYAML (the gate tools parse YAML) -- install: python3 -m pip install pyyaml, or run gates via: uv run --with pyyaml -- python3 ..."; fi
	@if command -v java >/dev/null 2>&1; then echo "  ok       java ($$(java -version 2>&1 | head -1))"; \
	 else echo "  MISSING  Java 11+ (the formal layer; tlc.sh runs TLC)"; fi
	@echo "  auto     the TLA+ tools (tla2tools.jar) download on first 'make verify-formal', pinned and checksum-verified"
	@if command -v uv >/dev/null 2>&1; then echo "  ok       uv"; \
	 else echo "  optional uv (runs 'make test' and resolves PyYAML on the fly) -- https://docs.astral.sh/uv/"; fi
	@if command -v structurizr-cli >/dev/null 2>&1 || command -v structurizr >/dev/null 2>&1; then echo "  ok       structurizr-cli (C4 diagram export)"; \
	 else echo "  optional structurizr-cli (C4 diagram EXPORT only; the DSL and every gate need only text) -- https://structurizr.com/cli"; fi

doctor: ## Check prerequisites and install status
	@$(MAKE) --no-print-directory preflight
	@echo "install status:"
	@for home in $(AGENT_HOMES); do \
	  test -e "$$home/skills/machinery" && echo "  ok       skill at $$home/skills/machinery" || echo "  MISSING  skill at $$home/skills/machinery -- run make install"; \
	  test -e "$$home/agents/machinery-fsm-author.md" && echo "  ok       fsm-author role at $$home/agents" || echo "  MISSING  fsm-author role at $$home/agents -- run make install"; \
	  test -e "$$home/agents/machinery-build-writer.md" && echo "  ok       build-writer role at $$home/agents" || echo "  MISSING  build-writer role at $$home/agents -- run make install"; \
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
