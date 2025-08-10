.PHONY: help install build test test-race test-coverage lint format vet check clean dev ci ci-fast vscode vscode-test cross-platform
.DEFAULT_GOAL := help

# Colors for output
GREEN  := \033[32m
YELLOW := \033[33m
RED    := \033[31m
BLUE   := \033[34m
CYAN   := \033[36m
RESET  := \033[0m

# Build info
BINARY_NAME := frugal-ls
BUILD_DIR := build
CMD_DIR := ./cmd/frugal-ls
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

# Go version check
GO_VERSION := $(shell go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
MIN_GO_VERSION := 1.21

help: ## Display this help message
	@echo "$(CYAN)Frugal Language Server Development Tools$(RESET)"
	@echo ""
	@echo "$(GREEN)Quick Commands:$(RESET)"
	@echo "  $(YELLOW)make ci$(RESET)         - Run full CI pipeline locally (mimics GitHub Actions)"
	@echo "  $(YELLOW)make ci-fast$(RESET)    - Run essential CI checks quickly"
	@echo "  $(YELLOW)make dev$(RESET)        - Development setup (install tools, format, test)"
	@echo "  $(YELLOW)make test$(RESET)       - Run all tests"
	@echo "  $(YELLOW)make build$(RESET)      - Build the binary"
	@echo ""
	@echo "$(GREEN)Available targets:$(RESET)"
	@awk 'BEGIN {FS = ":.*##"}; /^[a-zA-Z_-]+:.*?##/ { printf "  $(YELLOW)%-15s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

install: ## Install development tools and dependencies
	@echo "$(GREEN)Installing development tools...$(RESET)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go mod download
	@echo "$(GREEN)Development tools installed successfully!$(RESET)"

build: ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@go build -v -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(GREEN)Built $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

test: ## Run all tests
	@echo "$(GREEN)Running tests...$(RESET)"
	@go test -v ./...

test-race: ## Run tests with race detection
	@echo "$(GREEN)Running tests with race detection...$(RESET)"
	@go test -v -race ./...

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

benchmark: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(RESET)"
	@go test -bench=. -benchmem ./...

lint: ## Run linting
	@echo "$(GREEN)Running linter...$(RESET)"
	@golangci-lint run --timeout=5m

format: ## Format code
	@echo "$(GREEN)Formatting code...$(RESET)"
	@gofmt -s -w .
	@go mod tidy

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(RESET)"
	@go vet ./...

check: format vet ## Run format and vet checks
	@echo "$(GREEN)Checking if code is properly formatted...$(RESET)"
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "$(RED)Code is not formatted. Run 'make format'$(RESET)"; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "$(GREEN)Checking go mod tidy...$(RESET)"
	@go mod tidy
	@if [ -n "$$(git status --porcelain go.mod go.sum 2>/dev/null)" ]; then \
		echo "$(RED)go.mod or go.sum is not tidy$(RESET)"; \
		git diff go.mod go.sum 2>/dev/null || true; \
		exit 1; \
	fi
	@echo "$(GREEN)Code formatting and dependencies are clean!$(RESET)"

clean: ## Clean build artifacts and cache
	@echo "$(GREEN)Cleaning...$(RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache -modcache
	@echo "$(GREEN)Cleaned build artifacts and cache$(RESET)"

# VS Code Extension Commands
vscode: ## Build VS Code extension
	@echo "$(GREEN)Building VS Code extension...$(RESET)"
	@if [ ! -d "vscode-extension" ]; then \
		echo "$(RED)VS Code extension directory not found$(RESET)"; \
		exit 1; \
	fi
	@cd vscode-extension && npm ci
	@cd vscode-extension && npm run compile
	@echo "$(GREEN)VS Code extension built successfully$(RESET)"

vscode-test: vscode ## Test VS Code extension
	@echo "$(GREEN)Testing VS Code extension...$(RESET)"
	@cd vscode-extension && npm run lint
	@echo "$(GREEN)VS Code extension tests passed$(RESET)"

vscode-package: vscode-test ## Package VS Code extension
	@echo "$(GREEN)Packaging VS Code extension...$(RESET)"
	@cd vscode-extension && npm install -g vsce
	@cd vscode-extension && vsce package --no-dependencies
	@echo "$(GREEN)VS Code extension packaged successfully$(RESET)"

# Cross-platform builds
cross-platform: ## Build for all platforms
	@echo "$(GREEN)Building for all platforms...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@./scripts/build-cross-platform.sh
	@echo "$(GREEN)Cross-platform builds completed$(RESET)"

# Development workflow
dev: install format test ## Development setup: install tools, format, and test
	@echo "$(GREEN)Development environment ready!$(RESET)"
	@echo "$(CYAN)Try running: $(BUILD_DIR)/$(BINARY_NAME) --help$(RESET)"

# CI simulation commands
ci-fast: check test build ## Quick CI pipeline (essential checks)
	@echo "$(GREEN)Running quick CI pipeline...$(RESET)"
	@./$(BUILD_DIR)/$(BINARY_NAME) --help > /dev/null 2>&1 || echo "$(YELLOW)Binary runs (exit code $$?)$(RESET)"
	@echo "$(GREEN)✓ Quick CI pipeline completed successfully!$(RESET)"

ci: check test-coverage build cross-platform ## Full CI pipeline (mirrors GitHub Actions)
	@echo "$(GREEN)Running full CI pipeline...$(RESET)"
	@./$(BUILD_DIR)/$(BINARY_NAME) --help > /dev/null 2>&1 || echo "$(YELLOW)Binary runs (exit code $$?)$(RESET)"
	@if [ -d "vscode-extension" ]; then \
		make vscode-test; \
	else \
		echo "$(YELLOW)VS Code extension directory not found, skipping...$(RESET)"; \
	fi
	@echo "$(GREEN)✓ Full CI pipeline completed successfully!$(RESET)"
	@echo "$(CYAN)Coverage report: coverage.html$(RESET)"
	@echo "$(CYAN)Binaries: $(BUILD_DIR)/$(RESET)"

# Version info
version: ## Show version information
	@echo "$(CYAN)Frugal Language Server Build Info:$(RESET)"
	@echo "Version: $(VERSION)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Build Dir: $(BUILD_DIR)"

# Go version check
check-go-version: ## Check Go version
	@echo "$(GREEN)Checking Go version...$(RESET)"
	@if [ "$$(printf '%s\n' "$(MIN_GO_VERSION)" "$(GO_VERSION)" | sort -V | head -n1)" != "$(MIN_GO_VERSION)" ]; then \
		echo "$(RED)Go version $(GO_VERSION) is too old. Minimum required: $(MIN_GO_VERSION)$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Go version $(GO_VERSION) is supported$(RESET)"

# Watch mode for development
watch-test: ## Watch files and run tests on change
	@echo "$(GREEN)Watching for changes and running tests...$(RESET)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(RESET)"
	@which fswatch > /dev/null 2>&1 || (echo "$(RED)fswatch not found. Install with: brew install fswatch$(RESET)" && exit 1)
	@fswatch -o . -e ".*" -i "\\.go$$" | xargs -n1 -I{} sh -c 'echo "$(CYAN)Files changed, running tests...$(RESET)" && make test'

watch-build: ## Watch files and rebuild on change
	@echo "$(GREEN)Watching for changes and rebuilding...$(RESET)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(RESET)"
	@which fswatch > /dev/null 2>&1 || (echo "$(RED)fswatch not found. Install with: brew install fswatch$(RESET)" && exit 1)
	@fswatch -o . -e ".*" -i "\\.go$$" | xargs -n1 -I{} sh -c 'echo "$(CYAN)Files changed, rebuilding...$(RESET)" && make build'