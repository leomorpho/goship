# Define variables
PGVECTOR_IMAGE_NAME = custom-pgvector-for-atlas
PGVECTOR_IMAGE_TAG = latest
PGVECTOR_IMAGE_DIR = pgvector-image

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

# TODO: https://github.com/casey/just?tab=readme-ov-file seems like a nice alternative/improvement to make
# Determine if you have docker-compose or docker compose installed locally
# If this does not work on your system, just set the name of the executable you have installed
DCO_BIN := $(shell $(find_docker_compose))
define Comment
	- Run `make help` to see all the available options.
endef

.PHONY: help
help: ## Show this help message.
	@echo "Available options:"
	@echo
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z_-]+:.*?## .*$$/ { printf "\033[36m%-30s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo
	@echo "To see the details of each command, run: make <command>"

.PHONY: db
db: ## Connect to the primary database
	docker exec -it goship_db psql postgresql://admin:admin@localhost:5432/goship_db

.PHONY: db-test
db-test: ## Connect to the test database (you must run tests first before running this)
	docker exec -it goship_db psql postgresql://admin:admin@localhost:5432/goship_db_test


.PHONY: build-image 
build-image: ## Build the Docker image for pgvector
	@echo "Building Docker image $(PGVECTOR_IMAGE_NAME):$(PGVECTOR_IMAGE_TAG) from directory $(PGVECTOR_IMAGE_DIR)..."
	docker build -t $(PGVECTOR_IMAGE_NAME):$(PGVECTOR_IMAGE_TAG) $(PGVECTOR_IMAGE_DIR)


.PHONY: updateatlas
updateatlas: ## Update the Atlas migration tool we use
	curl -sSf https://atlasgo.sh | sh

.PHONY: migrate_diff
makemigrations: build-image ## Create a migration
	@echo "Running Atlas migrate diff..."
	atlas migrate diff "$(name)" \
	  --dir "file://ent/migrate/migrations" \
	  --to "ent://ent/schema" \
	  --dev-url "docker+postgres://_/$(PGVECTOR_IMAGE_NAME):$(PGVECTOR_IMAGE_TAG)/dev?search_path=public&sslmode=disable"

.PHONY: migrate_apply
migrate: ## Apply migrations
	@atlas migrate apply \
	  --dir "file://ent/migrate/migrations" \
	  --url "postgres://admin:admin@localhost:5432/app?search_path=public&sslmode=disable"

.PHONY: inspectschema
inspectsql: ## Inspect the SQL DB schema
	@atlas schema inspect \
		-u "ent://ent/schema" \
		--dev-url "sqlite://file?mode=memory&_fk=1" \
		--format '{{ sql . "  " }}'

.PHONY: inspecterd
inspecterd: ## Inspect the ERD DB schema
	atlas schema inspect \
		-u "ent://ent/schema" \
		--dev-url "sqlite://file?mode=memory&_fk=1" \
		-w

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

.PHONY: ent-install
ent-install: ## Install Ent code-generation module
	go get -d entgo.io/ent/cmd/ent

.PHONY: ent-gen
ent-gen: ## Generate Ent code
	go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/upsert,sql/execquery ./ent/schema

.PHONY: ent-new
ent-new: ## Create a new Ent entity
	go run entgo.io/ent/cmd/ent new $(name)
 
 .PHONY: generate
generate: ## Run code generation
	go generate ./...

.PHONY: up
up: ## Start the Docker containers
	$(DCO_BIN) up -d --remove-orphans
	sleep 3

.PHONY: down
down: ## Stop the Docker containers
	$(DCO_BIN) down

.PHONY: down
down-volume: ## Stop the Docker containers
	$(DCO_BIN) down --volumes

.PHONY: seed
seed: ## Seed with data (must be clean to begin with or will die)
	go run cmd/seed/main.go

.PHONY: reset
reset: ## Rebuild Docker containers to wipe all data
	$(DCO_BIN) down
	make up

.PHONY: init 
init: ## Set up the repo and run a fully working version of GoShip
	make reset
	make build-js 
	make build-css 
	make seed 
	make watch

.PHONY: build-js
build-js: ## Build JS/Svelte assets
	npm run build


.PHONY: build-js
watch-js: ## Build JS/Svelte assets (auto reload changes)
	npm install
	npm run watch 

build-css: ## Build CSS assets (auto reload changes)
	npx tailwindcss -i ./styles/styles.css -o ./static/styles_bundle.css

watch-css: ## Build CSS assets (auto reload changes)
	npx tailwindcss -i ./styles/styles.css -o ./static/styles_bundle.css --watch


.PHONY: run
watch-go: ## Run the application with air (auto reload changes)
	clear
	air

watch: 
	overmind start

.PHONY: test
test: ## Run all tests
	go test -p 1 ./...

.PHONY: testall
testall: ## Run all tests with no caching
	go test -count=1 -p 1 -count=1 ./...

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
	go run cmd/worker/main.go

.PHONY: workerui
workerui: ## Run the worker asynq dash
	asynq -u "127.0.0.1:6376" dash

.PHONY: check-updates
check-updates: ## Check for direct dependency updates
	go list -u -m -f '{{if not .Indirect}}{{.}}{{end}}' all | grep "\["


# See https://tailwindcss.com/blog/standalone-cli
.PHONY: tailwind-watch
tailwind-watch: ## Start a Tailwind watcher
	./tailwindcss -o static/output.css --watch

# See https://tailwindcss.com/blog/standalone-cli
.PHONY: tailwind-compile
tailwind-compile: ## Compile and minify your CSS for production
	./tailwindcss -i input.css -o static/output.css --minify

.PHONY: deploy-cherie
deploy-goship: ## Deploy new Goship version
	kamal deploy -c deploy.yml

# TODO: below is not working, only interactive mode is
.PHONY: test-e2e
e2e: ## Run Playwright tests
	@echo "Running end-to-end tests..."
	@cd e2e_tests && npm install && npx playwright test

.PHONY: test-e2e
e2eui: ## Run Playwright tests
	@echo "Running end-to-end tests..."
	@cd e2e_tests && npm install && npx playwright test --ui

# To run for mobile: `make codegen mobile=true`
.PHONY: codegen
codegen: ## Generate Playwright tests interactively
ifeq ($(mobile),true)
	@echo "Running Playwright codegen for mobile on predefined device (iPhone 12) at URL http://localhost:8002..."
	@cd e2e_tests && npx playwright codegen --device="iPhone 12" http://localhost:8002
else
	@echo "Running Playwright codegen for desktop at URL http://localhost:8002..."
	@cd e2e_tests && npx playwright codegen http://localhost:8002
endif


.PHONY: js-reinstall
js-reinstall: ## Reinstall all JS dependencies
	rm -rf node_modules package-lock.json
	npm install

stripe-webhook: ## Forward events from test mode to local webhooks endpoint 
	stripe listen --forward-to localhost:8002/Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks --latest
