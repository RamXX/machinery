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
.PHONY: install install-copy uninstall build install-binary install-cli preflight doctor check oracle verify-formal test golden golden-update help

install: ## Symlink machinery skill+agents into agent homes, install the CLI binary on PATH
	@for home in $(AGENT_HOMES); do \
	  mkdir -p "$$home/skills" "$$home/agents"; \
	  rm -rf "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/skills/machinery" "$$home/skills/machinery"; \
	  ln -sfn "$(SRC)/agents/machinery-fsm-author.md" "$$home/agents/machinery-fsm-author.md"; \
	  ln -sfn "$(SRC)/agents/machinery-build-writer.md" "$$home/agents/machinery-build-writer.md"; \
	  echo "linked machinery -> $$home"; \
	done
	@$(MAKE) --no-print-directory install-cli
	@$(MACH) preflight

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
MACHINERY_VERSION ?= latest
INTERNAL_VERSION := v0.1.0
MACH ?= $(CURDIR)/.bin/machinery
# Where to install the binary on PATH. Default: ~/.local/bin (no sudo, on PATH
# on most systems). Override: INSTALL_DIR=/usr/local/bin make install-cli
INSTALL_DIR ?= $(HOME)/.local/bin

# Detect OS and arch for binary downloads (matching the release matrix).
MACH_OS := $(shell uname -s | tr A-Z a-z)
MACH_ARCH := $(shell uname -m)
ifeq ($(MACH_ARCH),x86_64)
  MACH_ARCH := amd64
endif
ifeq ($(MACH_ARCH),aarch64)
  MACH_ARCH := arm64
endif

.PHONY: build install-binary install-cli
build: ## Build the machinery binary from source (needs Go)
	@mkdir -p .bin && go build -ldflags "-s -w -X main.version=$(INTERNAL_VERSION)" -o .bin/machinery ./cmd/machinery

install-cli: ## Install the machinery CLI binary onto PATH ($(INSTALL_DIR))
	@mkdir -p "$(INSTALL_DIR)"
	@$(MAKE) --no-print-directory $(MACH) >/dev/null
	@cp "$(MACH)" "$(INSTALL_DIR)/machinery"
	@echo "installed machinery -> $(INSTALL_DIR)/machinery"
	@command -v machinery >/dev/null 2>&1 && machinery version || \
	  { echo ""; echo "$(INSTALL_DIR) is not on your PATH. Add it:"; \
	    echo "  echo 'export PATH=\"$(INSTALL_DIR):\$$PATH\"' >> ~/.zshrc"; }

install-binary: ## Download a prebuilt binary, or build from source if no release exists yet
	@mkdir -p .bin
	@if command -v curl >/dev/null 2>&1 && \
	  curl -fsSL -o /dev/null -w "%{http_code}" \
	    "https://api.github.com/repos/ramirosalas/machinery/releases" 2>/dev/null | \
	  grep -q "200"; then \
	  echo "Downloading machinery $(MACHINERY_VERSION) for $(MACH_OS)/$(MACH_ARCH)..."; \
	  ext=""; [ "$(MACH_OS)" = "windows" ] && ext=".exe"; \
	  if [ "$(MACHINERY_VERSION)" = "latest" ]; then \
	    url=$$(curl -fsSL "https://api.github.com/repos/ramirosalas/machinery/releases/latest" | \
	      grep -o "https://[^[:space:]\"']*machinery-$(MACH_OS)-$(MACH_ARCH)$$ext[^[:space:]\"']*" | head -1); \
	  else \
	    url="https://github.com/ramirosalas/machinery/releases/download/$(MACHINERY_VERSION)/machinery-$(MACH_OS)-$(MACH_ARCH)$$ext"; \
	  fi; \
	  [ -z "$$url" ] && { echo "No matching binary found."; exit 1; }; \
	  curl -fsSL -o .bin/machinery "$$url"; \
	  sums_url=$$(dirname "$$url")/checksums-sha256.txt; \
	  if curl -fsSL -o .bin/checksums-sha256.txt "$$sums_url" 2>/dev/null; then \
	    want=$$(grep "machinery-$(MACH_OS)-$(MACH_ARCH)$$ext$$" .bin/checksums-sha256.txt | awk '{print $$1}'); \
	    got=$$(shasum -a 256 .bin/machinery 2>/dev/null | awk '{print $$1}' || sha256sum .bin/machinery | awk '{print $$1}'); \
	    if [ -z "$$want" ] || [ "$$want" != "$$got" ]; then \
	      rm -f .bin/machinery; \
	      echo "checksum mismatch for machinery-$(MACH_OS)-$(MACH_ARCH)$$ext (want $$want, got $$got); refusing to install"; \
	      exit 1; \
	    fi; \
	    echo "checksum verified"; \
	  else \
	    rm -f .bin/machinery; \
	    echo "release has no checksums-sha256.txt; refusing to install an unverified binary"; \
	    exit 1; \
	  fi; \
	  chmod +x .bin/machinery; \
	  echo "Installed: $$(.bin/machinery version)"; \
	else \
	  echo "No GitHub remote yet (or no connectivity). Building from source..."; \
	  $(MAKE) --no-print-directory build; \
	fi

# If the binary doesn't exist, try downloading it first; fall back to building.
$(MACH):
	@mkdir -p .bin
	@if command -v go >/dev/null 2>&1; then \
	  echo "Building from source (Go detected)..."; \
	  go build -ldflags "-s -w -X main.version=$(INTERNAL_VERSION)" -o .bin/machinery ./cmd/machinery; \
	else \
	  echo "No binary and no Go toolchain; downloading prebuilt..."; \
	  $(MAKE) --no-print-directory install-binary; \
	fi

preflight: $(MACH) ## Check machinery's runtime prerequisites (warns; installs nothing)
	@$(MACH) preflight

doctor: $(MACH) ## Check prerequisites and install status
	@$(MACH) doctor

test: ## Run the Go toolchain test suite (needs Go)
	@go test ./...

golden: ## Run the golden-corpus byte-for-byte regression net
	@go test -count=1 -run TestGolden ./cmd/machinery

golden-update: ## Re-capture the golden corpus from the current binary (review the diff!)
	@go test -count=1 -run TestGolden ./cmd/machinery -update

check: $(MACH) ## Run the deterministic gate suite on the examples
	@$(MACH) check examples/go-crm/design --impl examples/go-crm/impl
	@$(MACH) check examples/fulfillment/design
	@$(MACH) check examples/portfolio-engine/design
	@$(MACH) check examples/checkout-split/parent/design
	@$(MACH) check examples/checkout-split/orders/design
	@$(MACH) check examples/checkout-split/payments/design

oracle: $(MACH) ## Regenerate the transition oracles from the machine JSON (go-crm)
	@$(MACH) oracle examples/go-crm/design/machines

verify-formal: $(MACH) ## Regenerate + TLC-check the whole formal suite for the examples (from source)
	@echo "== go-crm =="; $(MACH) verify-formal examples/go-crm/design
	@echo "== fulfillment =="; $(MACH) verify-formal examples/fulfillment/design
	@echo "== portfolio-engine =="; $(MACH) verify-formal examples/portfolio-engine/design
	@echo "== checkout-split/orders =="; $(MACH) verify-formal examples/checkout-split/orders/design
	@echo "== checkout-split/payments =="; $(MACH) verify-formal examples/checkout-split/payments/design

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'
