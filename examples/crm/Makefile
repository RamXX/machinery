.DEFAULT_GOAL := help
SHELL := /bin/sh

# machinery and the pinned tools. The caller may set MACHINERY_TOOLS.
ROOT := $(abspath ../..)
MACHINERY_TOOLS ?= $(ROOT)/.bin
export MACHINERY_TOOLS
export PATH := $(ROOT)/.bin:$(MACHINERY_TOOLS):$(PATH)
MACHINERY ?= $(ROOT)/.bin/machinery

# The one Postgres container of this project, published on 127.0.0.1 only.
PG_CONTAINER := machinery-crm-pg
PG_IMAGE := postgres:17-alpine
PG_PORT := 15432
PG_ADMIN := crm_admin
export CRM_ADDR ?= 127.0.0.1:18080
export CRM_DATABASE_URL ?= postgres://crm_app:crm_app@127.0.0.1:$(PG_PORT)/crm?sslmode=disable
export CRM_ADMIN_DATABASE_URL ?= postgres://$(PG_ADMIN):$(PG_ADMIN)@127.0.0.1:$(PG_PORT)/crm?sslmode=disable

# The end-to-end command; keep it equal to commands.e2e in machinery.yaml.
E2E := env GOMAXPROCS=2 go test ./e2e/... -count=1 -p 1
# What the receipts say the suite ran against.
STACK := postgres 17 in docker, crm over http
BASE ?= experimental/v2
MAX ?= 6

.PHONY: help up down build serve psql check e2e evidence verify probe

help: ## List the targets
	@grep -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-8s %s\n", $$1, $$2}'

up: ## Start Postgres (empty, on 127.0.0.1:15432) and wait until it is healthy
	@docker inspect $(PG_CONTAINER) >/dev/null 2>&1 || docker run -d --name $(PG_CONTAINER) \
		-p 127.0.0.1:$(PG_PORT):5432 --tmpfs /var/lib/postgresql/data \
		-e POSTGRES_DB=crm -e POSTGRES_USER=$(PG_ADMIN) -e POSTGRES_PASSWORD=$(PG_ADMIN) \
		--health-cmd "pg_isready -h 127.0.0.1 -U $(PG_ADMIN) -d crm" \
		--health-interval 1s --health-timeout 2s --health-retries 60 \
		$(PG_IMAGE) >/dev/null
	@i=0; while [ "$$(docker inspect -f '{{.State.Health.Status}}' $(PG_CONTAINER))" != healthy ]; do \
		i=$$((i+1)); if [ $$i -gt 60 ]; then echo "up: $(PG_CONTAINER) not healthy after 60s"; exit 1; fi; sleep 1; done
	@echo "up: $(PG_CONTAINER) healthy on 127.0.0.1:$(PG_PORT)"

down: ## Stop Postgres and remove its container and data
	@docker rm -f -v $(PG_CONTAINER) >/dev/null 2>&1 || true
	@echo "down: $(PG_CONTAINER) removed"

build: ## Build the service into .bin/crm
	go build -o .bin/crm ./cmd/crm

serve: build ## Build and run the service (127.0.0.1:18080) against the local Postgres
	./.bin/crm

psql: ## Open a psql shell as the admin role
	docker exec -it $(PG_CONTAINER) psql -U $(PG_ADMIN) -d crm

check: ## gofmt, go vet and machinery check
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt: $$out"; exit 1; fi
	go vet ./...
	$(MACHINERY) check

e2e: ## The end-to-end suite: the real service over HTTP against the real Postgres (needs make up)
	$(E2E)

evidence: ## Run the e2e suite through machinery run: receipts for this commit (needs make up)
	$(MACHINERY) run --stack "$(STACK)" -- $(E2E)

verify: ## The end gate: machinery landing (check --base BASE, the whole suite through run, verify)
	$(MACHINERY) landing --base $(BASE) --stack "$(STACK)" -- $(E2E)

probe: ## Mutate the tables and expect the suite to notice (the first MAX mutations, default 6)
	$(MACHINERY) probe --max $(MAX) -- $(E2E)
