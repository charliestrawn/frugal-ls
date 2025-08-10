#!/bin/bash

# Development environment setup script for Frugal Language Server
# This script helps new developers get started quickly

set -e

# Colors for output
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
BLUE='\033[34m'
CYAN='\033[36m'
RESET='\033[0m'

echo -e "${BLUE}Frugal Language Server - Development Environment Setup${RESET}"
echo ""

# Check Go version
echo -e "${GREEN}Checking Go installation...${RESET}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Please install Go 1.21+ and try again.${RESET}"
    echo -e "${YELLOW}Visit: https://golang.org/doc/install${RESET}"
    exit 1
fi

GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
echo -e "${GREEN}Found Go version: ${GO_VERSION}${RESET}"

# Check if Make is available
if ! command -v make &> /dev/null; then
    echo -e "${YELLOW}Make is not installed. You can still use individual commands.${RESET}"
    MAKE_AVAILABLE=false
else
    MAKE_AVAILABLE=true
    echo -e "${GREEN}Make is available${RESET}"
fi

# Install development tools
echo ""
echo -e "${GREEN}Installing development tools...${RESET}"

# Install golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${YELLOW}Installing golangci-lint...${RESET}"
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
else
    echo -e "${GREEN}golangci-lint is already installed${RESET}"
fi

# Download Go dependencies
echo -e "${YELLOW}Downloading Go dependencies...${RESET}"
go mod download

# Run initial checks
echo ""
echo -e "${GREEN}Running initial checks...${RESET}"

# Format code
echo -e "${YELLOW}Formatting code...${RESET}"
gofmt -s -w .
go mod tidy

# Run tests
echo -e "${YELLOW}Running tests...${RESET}"
if go test -v ./...; then
    echo -e "${GREEN}✓ All tests passed${RESET}"
else
    echo -e "${RED}✗ Some tests failed${RESET}"
    exit 1
fi

# Build the binary
echo -e "${YELLOW}Building binary...${RESET}"
mkdir -p build
if go build -v -o build/frugal-ls ./cmd/frugal-ls; then
    echo -e "${GREEN}✓ Binary built successfully${RESET}"
else
    echo -e "${RED}✗ Build failed${RESET}"
    exit 1
fi

# Test the binary
if ./build/frugal-ls --help > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Binary runs correctly${RESET}"
else
    echo -e "${YELLOW}⚠ Binary runs but with non-zero exit code (this may be expected)${RESET}"
fi

# Setup complete
echo ""
echo -e "${GREEN}🎉 Development environment setup complete!${RESET}"
echo ""
echo -e "${CYAN}Quick Start:${RESET}"

if [ "$MAKE_AVAILABLE" = true ]; then
    echo -e "${YELLOW}  make help${RESET}          - Show all available commands"
    echo -e "${YELLOW}  make ci-fast${RESET}       - Run essential CI checks"
    echo -e "${YELLOW}  make ci${RESET}            - Run full CI pipeline"
    echo -e "${YELLOW}  make test${RESET}          - Run tests"
    echo -e "${YELLOW}  make build${RESET}         - Build binary"
    echo -e "${YELLOW}  make watch-test${RESET}    - Watch files and run tests (requires fswatch)"
else
    echo -e "${YELLOW}  go test ./...${RESET}              - Run tests"
    echo -e "${YELLOW}  go build -o build/frugal-ls ./cmd/frugal-ls${RESET} - Build binary"
    echo -e "${YELLOW}  golangci-lint run${RESET}          - Run linter"
fi

echo ""
echo -e "${CYAN}VS Code Setup:${RESET}"
echo -e "  1. Open this directory in VS Code"
echo -e "  2. Install the Go extension"
echo -e "  3. Use ${YELLOW}Ctrl+Shift+P${RESET} → 'Go: Install/Update Tools'"
echo ""

# Check for optional tools
echo -e "${CYAN}Optional tools for enhanced development:${RESET}"

if ! command -v fswatch &> /dev/null; then
    echo -e "${YELLOW}  fswatch${RESET} - For file watching (make watch-test, make watch-build)"
    echo -e "    Install: ${YELLOW}brew install fswatch${RESET} (macOS) or ${YELLOW}apt install fswatch${RESET} (Linux)"
fi

if ! command -v bc &> /dev/null; then
    echo -e "${YELLOW}  bc${RESET} - For file size calculations in build script"
    echo -e "    Install: ${YELLOW}brew install bc${RESET} (macOS) or ${YELLOW}apt install bc${RESET} (Linux)"
fi

if [ -d "vscode-extension" ] && ! command -v npm &> /dev/null; then
    echo -e "${YELLOW}  Node.js/npm${RESET} - For VS Code extension development"
    echo -e "    Install: ${YELLOW}https://nodejs.org/${RESET}"
fi

echo ""
echo -e "${GREEN}Happy coding! 🚀${RESET}"