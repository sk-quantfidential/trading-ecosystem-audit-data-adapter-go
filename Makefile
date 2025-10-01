# Audit Data Adapter Go - Makefile

# Load .env file if it exists (for local development)
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help test test-behavior test-unit test-integration test-performance test-coverage clean setup-test-db teardown-test-db docker-test

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Test targets
test: test-behavior ## Run all tests

test-behavior: ## Run behavior tests
	@echo "Running behavior tests..."
	go test -v ./tests -timeout=10m

test-unit: ## Run unit tests (placeholder for future unit tests)
	@echo "Running unit tests..."
	go test -v ./pkg/... ./internal/... -short

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	go test -v ./tests -run TestIntegrationBehaviorSuite -timeout=10m

test-performance: ## Run performance tests only
	@echo "Running performance tests..."
	TEST_PERFORMANCE_ONLY=true go test -v ./tests -run "Performance|Throughput|Latency|Scalability" -timeout=15m

test-comprehensive: ## Run comprehensive test suite
	@echo "Running comprehensive test suite..."
	go test -v ./tests -run TestComprehensiveBehaviorSuite -timeout=15m

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v ./tests -coverprofile=coverage.out -timeout=10m
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test environment setup
setup-test-db: ## Start test databases using Docker
	@echo "Starting test databases..."
	@docker run -d --name audit-test-postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=audit_test \
		-p 5432:5432 \
		postgres:17-alpine || true
	@docker run -d --name audit-test-redis \
		-p 6379:6379 \
		redis:8-alpine || true
	@echo "Waiting for databases to be ready..."
	@sleep 10
	@echo "Test databases are ready!"

teardown-test-db: ## Stop and remove test databases
	@echo "Stopping test databases..."
	@docker rm -f audit-test-postgres audit-test-redis || true
	@echo "Test databases stopped and removed."

restart-test-db: teardown-test-db setup-test-db ## Restart test databases

# Docker testing
docker-test: ## Run tests in Docker environment
	@echo "Running tests in Docker..."
	@docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	@docker-compose -f docker-compose.test.yml down

# Development targets
build: ## Build the project
	@echo "Building..."
	go build ./pkg/...
	go build ./internal/...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

clean: ## Clean up generated files
	@echo "Cleaning up..."
	rm -f coverage.out coverage.html
	go clean -testcache

# Quick test targets for development
test-quick: ## Run quick behavior tests (skip performance)
	@echo "Running quick behavior tests..."
	SKIP_PERFORMANCE_TESTS=true go test -v ./tests -timeout=5m

test-audit: ## Run audit event tests only
	@echo "Running audit event tests..."
	go test -v ./tests -run TestAuditEventBehaviorSuite -timeout=5m

test-service: ## Run service discovery tests only
	@echo "Running service discovery tests..."
	go test -v ./tests -run TestServiceDiscoveryBehaviorSuite -timeout=5m

test-cache: ## Run cache tests only
	@echo "Running cache tests..."
	go test -v ./tests -run TestCacheBehaviorSuite -timeout=5m

# Debug targets
test-debug: ## Run tests with debug logging
	@echo "Running tests with debug logging..."
	TEST_LOG_LEVEL=debug go test -v ./tests -timeout=10m

test-verbose: ## Run tests with maximum verbosity
	@echo "Running tests with maximum verbosity..."
	TEST_LOG_LEVEL=debug go test -v ./tests -timeout=10m -args -test.v

# Environment setup validation
check-env: ## Check test environment prerequisites
	@echo "Checking test environment..."
	@echo "PostgreSQL connection: $(shell echo "SELECT 1" | psql "${TEST_POSTGRES_URL:-postgres://postgres:postgres@localhost:5432/audit_test?sslmode=disable}" 2>/dev/null && echo "✓ OK" || echo "✗ Failed")"
	@echo "Redis connection: $(shell redis-cli -u "${TEST_REDIS_URL:-redis://localhost:6379/15}" ping 2>/dev/null && echo "✓ OK" || echo "✗ Failed")"

# Benchmarking
benchmark: ## Run benchmark tests
	@echo "Running benchmarks..."
	go test -v ./tests -bench=. -benchmem -timeout=15m

# CI targets
ci-test: ## Run tests suitable for CI environment
	@echo "Running CI tests..."
	CI=true SKIP_PERFORMANCE_TESTS=true go test -v ./tests -timeout=10m

ci-test-full: ## Run full test suite in CI
	@echo "Running full CI test suite..."
	CI=true go test -v ./tests -timeout=15m

# Default environment variables (matching orchestrator-docker setup)
# These can be overridden by .env file or environment
export POSTGRES_URL ?= postgres://audit_adapter:audit-adapter-db-pass@localhost:5432/trading_ecosystem?sslmode=disable
export REDIS_URL ?= redis://audit-adapter:audit-pass@localhost:6379/0
export MONGO_URL ?= mongodb://localhost:27017/audit_correlator

# Test-specific environment variables
export TEST_POSTGRES_URL ?= $(POSTGRES_URL)
export TEST_REDIS_URL ?= redis://admin:admin-secure-pass@localhost:6379/0
export TEST_MONGO_URL ?= mongodb://localhost:27017/audit_test
export TEST_LOG_LEVEL ?= warn
export TEST_TIMEOUT ?= 10m
export ENVIRONMENT ?= development