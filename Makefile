# machinery installer. Symlinks (or copies) the skill and agents into ~/.claude.

CLAUDE_DIR ?= $(HOME)/.claude
SKILLS_DIR := $(CLAUDE_DIR)/skills
AGENTS_DIR := $(CLAUDE_DIR)/agents
SRC := $(CURDIR)

.DEFAULT_GOAL := help
.PHONY: install install-copy uninstall doctor check oracle verify-formal test help

install: ## Symlink machinery into ~/.claude (live edits from this repo)
	@mkdir -p $(SKILLS_DIR) $(AGENTS_DIR)
	@rm -rf $(SKILLS_DIR)/machinery
	@ln -sfn $(SRC)/skills/machinery $(SKILLS_DIR)/machinery
	@ln -sfn $(SRC)/agents/machinery-fsm-author.md $(AGENTS_DIR)/machinery-fsm-author.md
	@ln -sfn $(SRC)/agents/machinery-build-writer.md $(AGENTS_DIR)/machinery-build-writer.md
	@echo "linked machinery -> $(CLAUDE_DIR)"

install-copy: ## Copy machinery into ~/.claude (no live edits)
	@mkdir -p $(SKILLS_DIR) $(AGENTS_DIR)
	@rm -rf $(SKILLS_DIR)/machinery
	@cp -R $(SRC)/skills/machinery $(SKILLS_DIR)/machinery
	@cp $(SRC)/agents/machinery-fsm-author.md $(AGENTS_DIR)/
	@cp $(SRC)/agents/machinery-build-writer.md $(AGENTS_DIR)/
	@echo "copied machinery -> $(CLAUDE_DIR)"

uninstall: ## Remove machinery from ~/.claude
	@rm -rf $(SKILLS_DIR)/machinery
	@rm -f $(AGENTS_DIR)/machinery-fsm-author.md $(AGENTS_DIR)/machinery-build-writer.md
	@echo "removed machinery from $(CLAUDE_DIR)"

MODELITH_VERSION ?= v0.4.0

doctor: ## Check dependencies and install status
	@command -v modelith >/dev/null 2>&1 && echo "ok: modelith $$(modelith --version) (pinned: $(MODELITH_VERSION))" || echo "MISSING: modelith (go install github.com/stacklok/modelith/cmd/modelith@$(MODELITH_VERSION))"
	@test -e $(SKILLS_DIR)/machinery && echo "ok: skill at $(SKILLS_DIR)/machinery" || echo "not installed: run make install"
	@test -e $(AGENTS_DIR)/machinery-fsm-author.md && echo "ok: fsm-author agent installed" || echo "not installed: run make install"
	@test -e $(AGENTS_DIR)/machinery-build-writer.md && echo "ok: build-writer agent installed" || echo "not installed: run make install"

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
