# Development Guide

This guide helps developers quickly set up their environment and run CI checks locally before pushing to GitHub.

## Quick Start

### 1. Setup Development Environment

```bash
# One-time setup (installs tools, formats code, runs tests)
./scripts/dev-setup.sh

# Or manually:
make dev
```

### 2. Run CI Locally

```bash
# Quick CI (essential checks only - faster)
make ci-fast

# Full CI pipeline (mirrors GitHub Actions exactly)
make ci

# Or use the detailed CI script with options
./scripts/ci-local.sh --quick          # Quick mode
./scripts/ci-local.sh --skip-vscode    # Skip VS Code extension
```

## Available Commands

### Makefile Commands

```bash
make help               # Show all available commands
make install           # Install development tools
make build             # Build the binary  
make test              # Run all tests
make test-coverage     # Run tests with coverage report
make lint              # Run linting
make format            # Format code
make check             # Run format and vet checks
make clean             # Clean build artifacts
make ci-fast           # Quick CI pipeline
make ci                # Full CI pipeline (mirrors GitHub Actions)
make cross-platform    # Build for all platforms
make watch-test        # Watch files and run tests on change
make watch-build       # Watch files and rebuild on change
```

### Development Scripts

```bash
./scripts/dev-setup.sh          # Initial development environment setup
./scripts/ci-local.sh           # Run full CI pipeline locally
./scripts/ci-local.sh --quick   # Run essential CI checks quickly
./scripts/build-cross-platform.sh  # Build for all platforms
```

## CI Pipeline Overview

Our CI pipeline consists of 5 jobs that run in parallel on GitHub Actions:

### 1. Test Job
- **Go versions tested**: 1.21, 1.22, 1.23
- **What it does**: Runs tests with race detection and generates coverage report
- **Local equivalent**: `make test-coverage` or `go test -v -race -coverprofile=coverage.out ./...`

### 2. Build Job  
- **What it does**: Builds the binary and verifies it runs
- **Local equivalent**: `make build` or `go build -o build/frugal-ls ./cmd/frugal-ls`

### 3. Lint Job
- **What it does**: Runs golangci-lint, checks formatting, runs go vet, verifies go mod tidy
- **Local equivalent**: `make check` + `make lint`

### 4. VS Code Extension Job
- **What it does**: Builds and tests the VS Code extension (if present)
- **Local equivalent**: `make vscode-test`

### 5. Cross-platform Build Job
- **Platforms**: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- **What it does**: Builds static binaries for all supported platforms
- **Local equivalent**: `make cross-platform` or `./scripts/build-cross-platform.sh`

## Development Workflow

### Before Making Changes

```bash
# Make sure everything works
make ci-fast
```

### During Development

```bash
# Format code automatically
make format

# Run tests continuously (requires fswatch)
make watch-test

# Or run tests manually
make test
```

### Before Committing

```bash
# Run the same checks as CI
make ci-fast

# Or run the full CI pipeline
make ci
```

### Before Pushing

```bash
# Final check - run everything CI does
make ci
```

## Troubleshooting

### Common Issues

**1. Tests failing**
```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v ./pkg/ast -run TestExtractSymbols
```

**2. Linting errors**
```bash
# Run linter to see specific issues
make lint

# Auto-fix formatting issues
make format
```

**3. Build failures**
```bash
# Clean and rebuild
make clean
make build
```

**4. Coverage issues**
```bash
# Generate coverage report
make test-coverage
# Open coverage.html in browser
```

### Tool Installation

If you're missing development tools:

```bash
# Install/update all tools
make install

# Or install individually:
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Optional Tools

For enhanced development experience:

```bash
# macOS
brew install fswatch bc

# Linux  
apt install fswatch bc

# For VS Code extension development
npm install -g vsce
```

## Performance

### CI Timing Comparison

| Command | Time | What it does |
|---------|------|--------------|
| `make ci-fast` | ~30s | Essential checks only |
| `make ci` | ~2-3min | Full pipeline with cross-platform builds |
| `./scripts/ci-local.sh --quick` | ~30s | Same as ci-fast with detailed output |
| `./scripts/ci-local.sh` | ~2-3min | Full pipeline with job separation |

### Parallel Execution

The Makefile is optimized for parallel execution. You can speed up builds:

```bash
# Use all CPU cores
make -j$(nproc) ci
```

## Integration with IDEs

### VS Code

1. Install the Go extension
2. Use `Ctrl+Shift+P` → "Go: Install/Update Tools"
3. The workspace is pre-configured for Go development
4. Use `Ctrl+`` to open integrated terminal and run `make` commands

### Other IDEs

The Makefile and scripts work with any editor. Key commands:
- `make test` - Run tests
- `make build` - Build binary  
- `make lint` - Check code quality
- `make format` - Format code

## Contributing

Before submitting a PR:

1. Run `make ci` to ensure all checks pass
2. Commit your changes
3. The GitHub Actions CI will run the same checks
4. All CI jobs must pass before merging

This ensures that the main branch always builds and passes all tests across all supported platforms and Go versions.