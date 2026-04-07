.PHONY: help test test-integration test-contract test-all lint build build-all coverage clean fmt vet install-tools

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run unit tests (fast, no external deps)
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

test-integration: ## Run integration tests (requires Docker / testcontainers)
	go test -race -tags=integration ./...

test-contract: ## Run Go/Python contract parity tests (requires ../contract-fixtures)
	go test -race -tags=contract ./internal/session/contract/...

test-all: ## Run unit + integration + contract tests
	$(MAKE) test
	$(MAKE) test-integration
	$(MAKE) test-contract

coverage: test ## Generate HTML coverage report
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linters
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: make install-tools"; exit 1; }
	golangci-lint run --timeout=10m ./...

fmt: ## Format code
	gofmt -s -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

build: ## Build simple_agent example
	go build -o bin/simple_agent ./cmd/examples/simple_agent
	@echo "Binary built: bin/simple_agent"

build-all: ## Build all example binaries
	@mkdir -p bin
	@for dir in cmd/examples/*/; do \
		name=$$(basename $$dir); \
		echo "Building $$name..."; \
		go build -o bin/$$name ./$$dir || exit 1; \
	done
	@echo "All binaries built in bin/"

clean: ## Clean build artifacts
	rm -rf bin/ coverage.txt coverage.html

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

.DEFAULT_GOAL := help
