# Development Guide

This guide helps developers set up their environment and contribute to the Frugal Language Server project.

> **⚠️ Note**: This is a personal learning project implementing an LSP for Frugal IDL, which was originally an open-source project by Workiva but is no longer open source.

## Quick Start

### Prerequisites

- Go 1.23+
- Git
- Node.js 20+ (for VS Code extension development)

### Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/charliestrawn/frugal-ls
   cd frugal-ls
   ```

2. **Build the language server:**
   ```bash
   go build -o frugal-ls ./cmd/frugal-ls
   ```

3. **Run tests:**
   ```bash
   go test ./...
   ```

## Development Commands

### Go Language Server

```bash
# Build the binary
go build -o frugal-ls ./cmd/frugal-ls

# Run tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run linting (requires golangci-lint)
golangci-lint run

# Format code
go fmt ./...
gofmt -s -w .

# Check for issues
go vet ./...
```

### VS Code Extension

```bash
# Navigate to extension directory
cd vscode-extension

# Install dependencies
npm install

# Compile TypeScript
npm run compile

# Run linting
npm run lint

# Package extension
npm install -g @vscode/vsce
vsce package
```

## Project Structure

```
frugal-ls/
├── cmd/frugal-ls/              # Main executable and CLI
├── internal/
│   ├── document/               # Document lifecycle management
│   ├── features/               # LSP feature implementations
│   │   ├── completion.go       # Code completion
│   │   ├── hover.go            # Hover information
│   │   ├── symbols.go          # Document/workspace symbols
│   │   ├── definition.go       # Go-to-definition
│   │   ├── references.go       # Find references
│   │   ├── rename.go           # Symbol renaming
│   │   ├── codeactions.go      # Code actions and quick fixes
│   │   ├── formatting.go       # Document formatting
│   │   ├── diagnostics.go      # Error diagnostics
│   │   └── semantictokens.go   # Semantic highlighting
│   ├── lsp/                    # LSP protocol server
│   ├── parser/                 # Tree-sitter parser integration
│   └── workspace/              # Cross-file analysis and includes
├── pkg/ast/                    # AST utilities and symbol extraction
├── vscode-extension/           # VS Code extension
├── .github/workflows/          # CI/CD automation
└── sample.frugal              # Example Frugal file for testing
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./internal/features

# Run a specific test
go test -v ./pkg/ast -run TestExtractSymbols

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Test Structure

- Unit tests for each package (`*_test.go` files)
- Integration tests in the main command package
- VS Code extension tests (TypeScript/JavaScript)

## Linting and Code Quality

### Install Tools

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# For VS Code extension
cd vscode-extension
npm install -g @vscode/vsce
```

### Run Checks

```bash
# Go linting
golangci-lint run

# Go formatting check
gofmt -l .

# Go vet
go vet ./...

# VS Code extension linting
cd vscode-extension
npm run lint
```

## CI/CD Pipeline

The project uses GitHub Actions with the following jobs:

### 1. Test (Go Matrix)
- **Go versions**: 1.21, 1.22, 1.23
- **Platform**: Ubuntu
- **Commands**: `go test -v -race -coverprofile=coverage.out ./...`

### 2. Build
- **Platform**: Ubuntu
- **Commands**: Build binary and test execution

### 3. Lint
- **Platform**: Ubuntu  
- **Tools**: golangci-lint, gofmt, go vet, go mod tidy

### 4. VS Code Extension
- **Platform**: Ubuntu
- **Node.js**: 20
- **Commands**: Build and package extension

### 5. Release (on tags)
- **Platforms**: Linux AMD64 (more platforms planned)
- **Artifacts**: Binary + VS Code extension

## Local Development Workflow

### Before Making Changes

```bash
# Ensure everything works
go test ./...
golangci-lint run
```

### During Development

```bash
# Run tests continuously (manual)
watch -n 2 'go test ./...'

# Format code
go fmt ./...
gofmt -s -w .

# Check specific package
go test -v ./internal/features
```

### Before Committing

```bash
# Run full checks
go test -race ./...
golangci-lint run
go vet ./...
go mod tidy

# Check no files changed by go mod tidy
git status --porcelain go.mod go.sum
```

### Testing the Language Server

```bash
# Build and test
go build -o frugal-ls ./cmd/frugal-ls

# Test on sample file
./frugal-ls --test sample.frugal

# Test LSP mode (will wait for input)
./frugal-ls

# Show version
./frugal-ls --version
```

### Testing the VS Code Extension

1. Build the extension:
   ```bash
   cd vscode-extension
   npm install && npm run compile
   vsce package
   ```

2. Install locally:
   ```bash
   code --install-extension frugal-ls-0.1.0.vsix
   ```

3. Test with a `.frugal` file

## Contributing

This is primarily a personal learning project, but contributions are welcome:

1. **Fork** the repository
2. **Create** a feature branch
3. **Make** your changes with tests
4. **Run** all checks locally:
   ```bash
   go test -race ./...
   golangci-lint run
   go vet ./...
   ```
5. **Commit** your changes
6. **Push** and create a pull request

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write tests for new functionality
- Keep functions focused and small
- Document public APIs

## Debugging

### Language Server Protocol

Enable tracing in VS Code:
```json
{
  "frugal-ls.trace.server": "verbose"
}
```

Check the Output panel: View → Output → "Frugal LSP"

### Parser Issues

Test parsing directly:
```bash
./frugal-ls --test problematic-file.frugal
```

### Extension Issues

1. Check Developer Console: Help → Toggle Developer Tools
2. Reload VS Code: Developer → Reload Window
3. Restart language server: Command Palette → "Frugal LS: Restart Server"

## Performance Notes

- Tree-sitter provides incremental parsing for good performance
- LSP features use caching where appropriate
- Cross-file analysis is optimized for typical workspace sizes

## Release Process

Releases are automated via GitHub Actions:

1. **Update versions** in `cmd/frugal-ls/main.go` and `vscode-extension/package.json`
2. **Commit and push** changes
3. **Create and push a tag**: `git tag v0.2.0 && git push origin v0.2.0`
4. **GitHub Actions** will build and create the release automatically

The release includes:
- Linux AMD64 binary
- VS Code extension (.vsix file)
- Checksums file