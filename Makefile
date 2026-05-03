include .env

# --- Configuration ---
BINARY_NAME=app
MAIN_PACKAGE_PATH=./cmd/api
BUILD_DIR=bin
MIGRATIONS_DIR=sql/migrations
GOLANGCI_VERSION=v1.64.5

# --- Standard Targets ---

.PHONY: all
all: install-tools setup-hooks generate frontend-install build

# Install tooling (Linter, Security Scanner, SQL Generator)
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)
	@echo "Tools installed successfully in $(shell go env GOPATH)/bin"

# Install git hooks to enforce quality at the developer level
.PHONY: setup-hooks
setup-hooks:
	@echo "Installing git hooks..."
	mkdir -p .git/hooks
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Hooks installed successfully."

# Generate type-safe SQL code (Run this after changing any .sql file)
.PHONY: generate
generate:
	@echo "Generating SQLC code..."
	sqlc generate

# --- Building ---

.PHONY: build
build:
	@echo "Building optimized HFT binary..."
	# -s -w strips debug symbols for a smaller, cleaner binary
	go build -ldflags="-s -w" -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

.PHONY: run
run: build
	@echo "Starting API..."
	${BUILD_DIR}/${BINARY_NAME}

# --- Quality Gates (CI/CD) ---

# Run all non-destructive checks (Used in CI/CD pipelines)
.PHONY: check
check: tidy lint secure test-unit frontend-check

# Ensure go.mod and go.sum are perfectly synced
.PHONY: tidy
tidy:
	@echo "Tidying modules..."
	go mod tidy
	git diff --exit-code go.mod go.sum

# Run the strictly-configured golangci-lint (Shadowing, Allocations, Floats)
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

# Scan for known vulnerabilities in dependencies
.PHONY: secure
secure:
	@echo "Running vulnerability scan..."
	govulncheck ./...

# --- Frontend Quality Gates ---

# Install frontend dependencies
.PHONY: frontend-install
frontend-install:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

# Run frontend linting
.PHONY: frontend-lint
frontend-lint:
	@echo "Running frontend lint..."
	cd frontend && npm run lint

# Check frontend formatting
.PHONY: frontend-format
frontend-format:
	@echo "Checking frontend formatting..."
	cd frontend && npm run format:check

# Run frontend type checking
.PHONY: frontend-type-check
frontend-type-check:
	@echo "Running frontend type check..."
	cd frontend && npm run type-check

# Run all frontend quality checks
.PHONY: frontend-check
frontend-check: frontend-lint frontend-format frontend-type-check

# --- Testing ---

# Run unit tests with the Race Detector enabled (Mandatory for concurrent engines)
.PHONY: test-unit
test-unit:
	@echo "Running unit tests with race detector..."
	# -count=1 disables test caching to ensure fresh results
	go test -v -race -count=1 ./internal/...

# Run integration tests (Requires DB/Kafka/Redis running)
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v -race -count=1 ./cmd/api/...

# Generate and view visual code coverage for core matching logic
.PHONY: test-coverage
test-coverage:
	@echo "Checking coverage..."
	go test -coverprofile=coverage.out ./internal/core/...
	go tool cover -html=coverage.out

# --- Performance & Profiling ---

# Run benchmarks for the matching engine
.PHONY: bench
bench:
	@echo "Running nanosecond benchmarks..."
	go test -bench=. -benchmem ./internal/core/domain/...

# --- Infrastructure & Database ---

.PHONY: migrate-up
migrate-up:
	goose -dir ${MIGRATIONS_DIR} postgres "${DB_URL}" up

.PHONY: migrate-down
migrate-down:
	goose -dir ${MIGRATIONS_DIR} postgres "${DB_URL}" down

.PHONY: docker-up
docker-up:
	@echo "Starting infrastructure..."
	docker-compose up -d

.PHONY: docker-down
docker-down:
	@echo "Stopping infrastructure..."
	docker-compose down

.PHONY: docker-refresh
docker-refresh:
	@echo "Performing clean slate refresh (wiping volumes)..."
	docker-compose down -v
	docker-compose up -d --build

.PHONY: load-test
load-test:
	@echo "Starting load test..."
	python3 load_test/load_test.py

# --- Maintenance ---

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ${BUILD_DIR}
	rm -f coverage.out
	go clean

# Helper to show all available commands
.PHONY: help
help:
	@echo "OBMINNIK Orderbook Makefile"
	@echo "Usage: make [target]"
	@echo ""
	@echo "Setup Targets:"
	@echo "  install-tools    Install linter, security scanner, and sqlc"
	@echo "  setup-hooks      Install pre-commit git hooks"
	@echo "  generate         Generate Go code from SQL"
	@echo ""
	@echo "Quality Targets:"
	@echo "  check            Run tidy, lint, secure, and unit tests"
	@echo "  lint             Run golangci-lint"
	@echo "  secure           Run govulncheck security scan"
	@echo "  bench            Run performance benchmarks"
	@echo ""
	@echo "Testing Targets:"
	@echo "  test-unit        Run unit tests with race detector"
	@echo "  test-integration Run integration tests"
	@echo ""
	@echo "Build Targets:"
	@echo "  build            Build optimized binary"
	@echo "  run              Build and run the API"