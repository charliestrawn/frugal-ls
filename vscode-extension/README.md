# Frugal IDL Language Support

This VS Code extension provides comprehensive language support for Frugal IDL files, including:

## Features

- **Syntax Highlighting** - Rich syntax highlighting for Frugal IDL files
- **IntelliSense** - Context-aware code completion and suggestions
- **Hover Information** - Detailed information on hover for symbols and types
- **Go to Definition** - Navigate to symbol definitions across files
- **Document Symbols** - Outline view with hierarchical symbol structure
- **Workspace Symbols** - Search symbols across the entire workspace
- **Code Actions** - Quick fixes and refactoring suggestions
- **Document Formatting** - Automatic code formatting
- **Diagnostics** - Real-time syntax error detection and reporting
- **Cross-file Support** - Full support for include statements and cross-file navigation

## Requirements

The extension requires the `frugal-ls` language server binary to be installed and available in your PATH, or configured via the `frugal-ls.server.path` setting.

## Installation

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/charliestrawn/frugal-ls
   cd frugal-ls
   ```

2. Build the language server:
   ```bash
   go build -o frugal-ls ./cmd/frugal-ls
   ```

3. Install the VS Code extension:
   ```bash
   cd vscode-extension
   npm install
   npm run compile
   code --install-extension .
   ```

## Configuration

The extension can be configured via VS Code settings:

- `frugal-ls.server.path`: Path to the frugal-ls executable (default: "frugal-ls")
- `frugal-ls.server.args`: Arguments to pass to the language server (default: [])
- `frugal-ls.trace.server`: Enable server communication tracing (default: "off")

## Commands

- `Frugal LS: Restart Server` - Restart the language server

## Development

To set up the development environment:

1. Install dependencies:
   ```bash
   npm install
   ```

2. Compile TypeScript:
   ```bash
   npm run compile
   ```

3. Launch VS Code with the extension:
   ```bash
   code --extensionDevelopmentPath=. .
   ```

## About Frugal

Frugal is an extension to Apache Thrift that provides additional features for pub/sub messaging patterns. It adds "scope" constructs that enable efficient publish/subscribe communication while maintaining compatibility with standard Thrift.

## License

This project is licensed under the MIT License.