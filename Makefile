SHELL := /bin/bash
.DEFAULT_GOAL := help

# Define variables
PGVECTOR_IMAGE_NAME = custom-pgvector
PGVECTOR_IMAGE_TAG = latest
PGVECTOR_IMAGE_DIR = infra/pgvector-image
NPM ?= npm --prefix frontend
TAILWIND ?= npx --prefix frontend tailwindcss
COMPOSE_FILE ?= infra/docker/docker-compose.yml

# Define a function to check for docker compose command
define find_docker_compose
  if command -v docker-compose >/dev/null 2>&1; then \
    echo docker-compose; \
  elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then \
    echo "docker compose"; \
  else \
    echo ""; \
  fi
endef

# Determine if you have docker-compose or docker compose installed locally
DCO_BIN := $(shell $(find_docker_compose))

.PHONY: help
help: ## Show this help message.
	@echo "Available options:"
	@echo
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z_-]+:.*?## .*$$/ { printf "\033[36m%-30s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo
	@echo "To see the details of each command, run: make <command>"

.PHONY: ensure-compose
ensure-compose:
	@if [ -z "$(DCO_BIN)" ]; then \
		echo "No docker compose command found (docker-compose or docker compose)."; \
		exit 1; \
	fi

.PHONY: hooks hooks-install
hooks: hooks-install ## Install git hooks

.PHONY: hooks-install
hooks-install: ## Install git hooks and commit-msg enforcement
	mkdir -p .githooks
	chmod +x .githooks/pre-commit
	chmod +x .githooks/prepare-commit-msg
	chmod +x .githooks/commit-msg
	git config core.hooksPath .githooks

.PHONY: llm-txt
llm-txt: ## Generate root LLM.txt from README and docs markdown files
	bash tools/scripts/generate-llm-txt.sh

.PHONY: check-compile
check-compile: ## Compile app/packages and route tests without running tests
	bash tools/scripts/check-compile.sh

.PHONY: templ-gen
templ-gen: ## Generate templ code next to .templ files via ship CLI
	go run ./tools/cli/ship/cmd/ship templ generate --path app

.PHONY: ship-install
ship-install: ## Install the latest local ship CLI binary into GOBIN (or GOPATH/bin)
	go install ./tools/cli/ship/cmd/ship

# Core workflow ------------------------------------------------------------------------------

.PHONY: dev
dev: ## Start the canonical app-on dev loop (single-node web loop; distributed uses full mode)
	go run ./tools/cli/ship/cmd/ship dev

.PHONY: run
run: ## Start the app in single-binary mode (no Docker)
	go run ./cmd/web

.PHONY: dev-worker
dev-worker: ## Start local development worker only (infra + worker process)
	bash tools/scripts/up.sh "$(DCO_BIN)"
	overmind start -f Procfile.worker

.PHONY: dev-full
dev-full: ## Start local development including web, worker, and JS/CSS watchers
	bash tools/scripts/up.sh "$(DCO_BIN)"
	echo "Tip: run 'nvm use v18.20.7' if JS tooling fails."
	overmind start -f Procfile

.PHONY: dev-reset
dev-reset: reset deps-js build-js build-css seed watch ## Full reset then start dev (destructive to local DB state)

.PHONY: db
db: ## Connect to the optional compose-backed Postgres helper, if enabled
	docker exec -it goship_db psql postgresql://admin:admin@localhost:5432/goship_db

.PHONY: db-test
db-test: ## Connect to the optional compose-backed Postgres test DB helper
	docker exec -it goship_db psql postgresql://admin:admin@localhost:5432/goship_db_test


.PHONY: build-image 
build-image: ## Build the Docker image for pgvector
	@echo "Building Docker image $(PGVECTOR_IMAGE_NAME):$(PGVECTOR_IMAGE_TAG) from directory $(PGVECTOR_IMAGE_DIR)..."
	docker build -t $(PGVECTOR_IMAGE_NAME):$(PGVECTOR_IMAGE_TAG) $(PGVECTOR_IMAGE_DIR)


.PHONY: migrate_diff
makemigrations: ## Create a migration through ship CLI
	go run ./tools/cli/ship/cmd/ship db:make "$(name)"

.PHONY: migrate_apply
migrate: ## Apply migrations through ship CLI
	go run ./tools/cli/ship/cmd/ship db:migrate

.PHONY: db-status
db-status: ## Show migration status through ship CLI
	go run ./tools/cli/ship/cmd/ship db:status

.PHONY: db-create
db-create: ## Validate/create DB target through ship CLI
	go run ./tools/cli/ship/cmd/ship db:create

.PHONY: db-drop
db-drop: ## Preview drop plan (use direct ship command with --yes to execute)
	go run ./tools/cli/ship/cmd/ship db:drop --dry-run

.PHONY: db-reset
db-reset: ## Preview reset plan (use direct ship command with --yes to execute)
	go run ./tools/cli/ship/cmd/ship db:reset --dry-run

.PHONY: schemaspy
schema: ## Create DB schema
	@docker run --rm \
		--network="host" \
		-e "DATABASE_TYPE=pgsql" \
		-e "DATABASE_NAME=app" \
		-e "DATABASE_USER=admin" \
		-e "DATABASE_PASSWORD=admin" \
		-e "DATABASE_HOST=localhost" \
		-e "DATABASE_PORT=5432" \
		-v "$(PWD)/schemaspy-output:/output" \
		schemaspy/schemaspy:latest \
		-t pgsql -host localhost:5432 -db app -u admin -p admin

.PHONY: cache
cache: ## Connect to the primary cache
	docker exec -it goship_cache redis-cli

.PHONY: cache-clear
cache-clear: ## Clear the primary cache
	docker exec -it goship_cache redis-cli flushall


.PHONY: cache-test
cache-test: ## Connect to the test cache
	docker exec -it goship_cache redis-cli -n 1

.PHONY: model-new
model-new: ## Create a new model scaffold
	go run ./tools/cli/ship/cmd/ship make:model $(name)
 
.PHONY: generate
generate: templ-gen ## Run code generation
	go generate ./...

.PHONY: up
up: ensure-compose ## Start Docker containers
	bash tools/scripts/up.sh "$(DCO_BIN)"

.PHONY: down
down: ensure-compose ## Stop Docker containers
	$(DCO_BIN) -f $(COMPOSE_FILE) down

.PHONY: down-volume
down-volume: ensure-compose ## Stop Docker containers and delete volumes
	$(DCO_BIN) -f $(COMPOSE_FILE) down --volumes

.PHONY: seed
seed: ## Seed with data (must be clean to begin with or will die)
	go run cmd/seed/main.go

.PHONY: reset
reset: down up ## Rebuild Docker containers to wipe all data

.PHONY: init
init: dev-reset ## Backward-compatible alias for full reset dev startup

.PHONY: build-js
build-js: ## Build JS/Svelte assets
	$(NPM) run build

.PHONY: js-build-vite
js-build-vite: ## Build JS assets with Vite islands output
	$(NPM) run build

.PHONY: deps-js
deps-js: ## Install JS dependencies
	$(NPM) install

.PHONY: watch-js
watch-js: ## Watch and rebuild JS/Svelte assets
	$(NPM) run watch

.PHONY: build-css
build-css: ## Build CSS assets (auto reload changes)
	$(TAILWIND) --config ./frontend/tailwind.config.js -i ./app/styles/styles.css -o ./app/static/styles_bundle.css

.PHONY: watch-css
watch-css: ## Build CSS assets (auto reload changes)
	$(TAILWIND) --config ./frontend/tailwind.config.js -i ./app/styles/styles.css -o ./app/static/styles_bundle.css --watch

.PHONY: watch-go
watch-go: ## Run the application with air (auto reload changes)
	clear
	air

watch: ## Start all dev watchers/processes through overmind
	@echo "Tip: run 'nvm use v18.20.7' if JS tooling fails."
	overmind start -f Procfile

.PHONY: test
test: ## Run Docker-free unit test package set
	bash tools/scripts/test-unit.sh
	bash tools/scripts/check-module-isolation.sh

.PHONY: test-module-isolation
test-module-isolation: ## Ensure extracted modules do not import goship internals
	bash tools/scripts/check-module-isolation.sh

.PHONY: test-sql-portability
test-sql-portability: ## Verify sql-core-v1 runtime portability contract
	bash tools/scripts/check-sql-portability.sh

.PHONY: test-integration
test-integration: ## Run integration test package set (may require Docker/infra)
	bash tools/scripts/test-integration.sh

.PHONY: testall
testall: ## Run both unit and integration test package sets
	bash tools/scripts/test-unit.sh
	bash tools/scripts/test-integration.sh

.PHONY: cover
cover: ## Create a html coverage report and open it once generated
	@echo "Running tests with coverage..."
	@go test -coverprofile=/tmp/coverage.out -count=1 -p 1  ./...
	@echo "Generating HTML coverage report..."
	@go tool cover -html=/tmp/coverage.out

.PHONY: cover-treemap
cover-treemap: ## Create a treemap view of the coverage report
	@echo "Running tests with coverage..."
	@go test -coverprofile=/tmp/coverage.out -count=1 -p 1  ./...
	@echo "Generating SVG coverage treemap..."
	@go-cover-treemap -coverprofile /tmp/coverage.out > /tmp/coverage-treemap.svg
	@echo "Opening SVG coverage treemap..."
	@open /tmp/coverage-treemap.svg

.PHONY: worker
worker: ## Run the worker
	clear
	go run ./cmd/worker

.PHONY: workerui
workerui: ## Run the worker asynq dash
	asynq -u "127.0.0.1:6376" dash

.PHONY: check-updates
check-updates: ## Check for direct dependency updates
	go list -u -m -f '{{if not .Indirect}}{{.}}{{end}}' all | grep "\["


# See https://tailwindcss.com/blog/standalone-cli
.PHONY: tailwind-watch
tailwind-watch: ## Start a Tailwind watcher
	$(TAILWIND) --config ./frontend/tailwind.config.js -o app/static/output.css --watch

# See https://tailwindcss.com/blog/standalone-cli
.PHONY: tailwind-compile
tailwind-compile: ## Compile and minify your CSS for production
	$(TAILWIND) --config ./frontend/tailwind.config.js -i app/styles/styles.css -o app/static/output.css --minify

.PHONY: deploy-cherie
deploy-goship: ## Deploy new Goship version
	kamal deploy -c infra/deploy/kamal/deploy.yml

# TODO: below is not working, only interactive mode is
.PHONY: test-e2e
e2e: ## Run Playwright tests
	@echo "Running end-to-end tests..."
	@cd tests/e2e && npm install && npx playwright test

.PHONY: e2e-smoke
e2e-smoke: ## Run Playwright smoke test with managed app startup
	@echo "Running Playwright smoke test..."
	@cd tests/e2e && npm install && npx playwright test tests/smoke.spec.ts

.PHONY: test-e2e
e2eui: ## Run Playwright tests
	@echo "Running end-to-end tests..."
	@cd tests/e2e && npm install && npx playwright test --ui

# To run for mobile: `make codegen mobile=true`
.PHONY: codegen
codegen: ## Generate Playwright tests interactively
ifeq ($(mobile),true)
	@echo "Running Playwright codegen for mobile on predefined device (iPhone 12) at URL http://localhost:8002..."
	@cd tests/e2e && npx playwright codegen --device="iPhone 12" http://localhost:8002
else
	@echo "Running Playwright codegen for desktop at URL http://localhost:8002..."
	@cd tests/e2e && npx playwright codegen http://localhost:8002
endif


.PHONY: js-reinstall
js-reinstall: ## Reinstall all JS dependencies
	rm -rf node_modules package-lock.json
	npm install

.PHONY: doc
pkgsite: ## Create pkgsite docs
	pkgsite -open .

.PHONY: golds
# Documentation: https://go101.org/apps-and-libs/golds.html
golds: ## Create golds docs
	golds ./...

stripe-webhook: ## Forward events from test mode to local webhooks endpoint 
	stripe listen --forward-to localhost:8002/Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks --latest
