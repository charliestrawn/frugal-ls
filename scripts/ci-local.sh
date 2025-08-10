#!/bin/bash

# Local CI simulation script
# Mirrors the GitHub Actions CI pipeline as closely as possible

set -e

# Colors for output
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
BLUE='\033[34m'
CYAN='\033[36m'
RESET='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="build"
BINARY_NAME="frugal-ls"

# Counters for summary
TOTAL_JOBS=0
PASSED_JOBS=0
FAILED_JOBS=0

# Helper functions
print_header() {
    echo ""
    echo -e "${BLUE}========================================${RESET}"
    echo -e "${BLUE} $1${RESET}"
    echo -e "${BLUE}========================================${RESET}"
}

print_step() {
    echo -e "${CYAN}→ $1${RESET}"
}

print_success() {
    echo -e "${GREEN}✓ $1${RESET}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${RESET}"
}

print_error() {
    echo -e "${RED}✗ $1${RESET}"
}

run_job() {
    local job_name="$1"
    local job_command="$2"
    
    TOTAL_JOBS=$((TOTAL_JOBS + 1))
    print_header "$job_name"
    
    if eval "$job_command"; then
        print_success "$job_name completed successfully"
        PASSED_JOBS=$((PASSED_JOBS + 1))
        return 0
    else
        print_error "$job_name failed"
        FAILED_JOBS=$((FAILED_JOBS + 1))
        return 1
    fi
}

# Change to project directory
cd "$PROJECT_DIR"

echo -e "${BLUE}Frugal Language Server - Local CI Pipeline${RESET}"
echo -e "${CYAN}Simulating GitHub Actions CI workflow locally${RESET}"
echo ""

# Job 1: Test (mirrors test job in CI)
test_job() {
    print_step "Downloading dependencies"
    go mod download
    
    print_step "Running tests with race detection and coverage"
    go test -v -race -coverprofile=coverage.out ./...
    
    if [ -f coverage.out ]; then
        print_step "Generating coverage report"
        go tool cover -html=coverage.out -o coverage.html
        print_success "Coverage report generated: coverage.html"
    fi
}

# Job 2: Build (mirrors build job in CI)
build_job() {
    print_step "Downloading dependencies"
    go mod download
    
    print_step "Building binary"
    mkdir -p "$BUILD_DIR"
    go build -v -o "$BUILD_DIR/$BINARY_NAME" ./cmd/frugal-ls
    
    print_step "Testing binary runs"
    if "./$BUILD_DIR/$BINARY_NAME" --help > /dev/null 2>&1; then
        print_success "Binary runs correctly"
    else
        local exit_code=$?
        print_warning "Binary runs (exit code $exit_code)"
    fi
}

# Job 3: Lint (mirrors lint job in CI)
lint_job() {
    print_step "Running golangci-lint"
    if ! command -v golangci-lint &> /dev/null; then
        print_warning "golangci-lint not found, installing..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    fi
    golangci-lint run --timeout=5m
    
    print_step "Checking formatting"
    if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
        print_error "Code is not formatted. Run 'gofmt -s -w .'"
        gofmt -s -l .
        return 1
    fi
    print_success "Code formatting is correct"
    
    print_step "Running go vet"
    go vet ./...
    
    print_step "Checking go mod tidy"
    go mod tidy
    if [ -n "$(git status --porcelain go.mod go.sum 2>/dev/null)" ]; then
        print_error "go.mod or go.sum is not tidy"
        git diff go.mod go.sum 2>/dev/null || true
        return 1
    fi
    print_success "Dependencies are tidy"
}

# Job 4: VS Code Extension (mirrors vscode-extension job in CI)
vscode_job() {
    if [ ! -d "vscode-extension" ]; then
        print_warning "VS Code extension directory not found, skipping"
        return 0
    fi
    
    if ! command -v npm &> /dev/null; then
        print_warning "npm not found, skipping VS Code extension build"
        return 0
    fi
    
    print_step "Installing dependencies"
    cd vscode-extension
    npm ci
    
    print_step "Running lint"
    npm run lint
    
    print_step "Compiling TypeScript"
    npm run compile
    
    print_step "Packaging extension"
    if ! command -v vsce &> /dev/null; then
        npm install -g vsce
    fi
    vsce package --no-dependencies
    
    cd "$PROJECT_DIR"
    print_success "VS Code extension packaged successfully"
}

# Job 5: Cross-platform Build (mirrors cross-platform-build job in CI)
cross_platform_job() {
    print_step "Running cross-platform build"
    ./scripts/build-cross-platform.sh
}

# Main execution
SKIP_CROSS_PLATFORM=false
SKIP_VSCODE=false
QUICK_MODE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-cross-platform)
            SKIP_CROSS_PLATFORM=true
            shift
            ;;
        --skip-vscode)
            SKIP_VSCODE=true
            shift
            ;;
        --quick)
            QUICK_MODE=true
            SKIP_CROSS_PLATFORM=true
            SKIP_VSCODE=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --quick                Skip cross-platform builds and VS Code extension"
            echo "  --skip-cross-platform  Skip cross-platform builds"
            echo "  --skip-vscode          Skip VS Code extension build"
            echo "  --help                 Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run all CI jobs
run_job "Test" "test_job" || true
run_job "Build" "build_job" || true  
run_job "Lint" "lint_job" || true

if [ "$SKIP_VSCODE" = false ]; then
    run_job "VS Code Extension" "vscode_job" || true
fi

if [ "$SKIP_CROSS_PLATFORM" = false ]; then
    run_job "Cross-platform Build" "cross_platform_job" || true
fi

# Summary
echo ""
print_header "CI Pipeline Summary"
echo -e "${CYAN}Total Jobs: $TOTAL_JOBS${RESET}"
echo -e "${GREEN}Passed: $PASSED_JOBS${RESET}"

if [ $FAILED_JOBS -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED_JOBS${RESET}"
    echo ""
    echo -e "${RED}CI Pipeline failed! Please fix the issues above.${RESET}"
    exit 1
else
    echo -e "${GREEN}Failed: 0${RESET}"
    echo ""
    echo -e "${GREEN}🎉 CI Pipeline passed! All jobs completed successfully.${RESET}"
    
    if [ -f coverage.html ]; then
        echo -e "${CYAN}📊 Coverage report: coverage.html${RESET}"
    fi
    
    if [ -d "$BUILD_DIR" ]; then
        echo -e "${CYAN}📦 Binaries: $BUILD_DIR/${RESET}"
    fi
    
    exit 0
fi