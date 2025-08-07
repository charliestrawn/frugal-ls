# Frugal Language Server Protocol (LSP) Implementation

A comprehensive Language Server Protocol implementation for Frugal IDL files, built with Go and tree-sitter parsing.

## Overview

Frugal is an extension to Apache Thrift that adds pub/sub "scope" constructs for efficient publish/subscribe messaging patterns. This LSP implementation provides full language support for Frugal IDL files including syntax highlighting, code completion, hover information, go-to-definition, and more.

## Features

### Core Language Features
- **Syntax Error Detection** - Real-time diagnostics with syntax error reporting
- **Code Completion** - Context-aware completions for types, services, and identifiers
- **Hover Information** - Rich documentation on hover with type information
- **Go to Definition** - Navigate to symbol definitions across files
- **Document Symbols** - Hierarchical outline view of file structure
- **Workspace Symbols** - Search symbols across the entire workspace

### Advanced Features (Phase 4)
- **Cross-file Includes Resolution** - Full support for include statements and dependency tracking
- **Code Actions & Quick Fixes** - Automated fixes for common syntax errors and refactoring actions
- **Document Formatting** - Automatic code formatting with configurable indentation
- **VS Code Extension** - Complete VS Code integration with syntax highlighting

### Code Actions Include:
- Fix missing semicolons and parentheses
- Extract methods from services
- Add fields to structs
- Generate constructors
- Add missing includes
- Organize includes
- Generate service and scope templates

## Installation

### Build from Source

1. **Clone the repository:**
   ```bash
   git clone https://github.com/charliestrawn/frugal-ls
   cd frugal-ls
   ```

2. **Build the language server:**
   ```bash
   go build -o frugal-ls ./cmd/frugal-ls
   ```

3. **Install the VS Code extension:**
   ```bash
   cd vscode-extension
   npm install
   npm run compile
   ```

## Usage

### Command Line

Run the language server directly:
```bash
./frugal-ls
```

Test parsing on a Frugal file:
```bash
./frugal-ls -test sample.frugal
```

### VS Code Integration

1. Install the extension from the `vscode-extension` directory
2. Configure the path to the `frugal-ls` binary in VS Code settings
3. Open any `.frugal` file to activate language support

### Neovim Integration

#### Quick Setup (Kickstart.nvim)
If you're using kickstart.nvim, add this to your servers table in `init.lua`:

```lua
local servers = {
  -- ... your existing servers
  frugal_ls = {
    filetypes = { 'frugal' },
  },
}
```

Then copy the filetype detection:
```bash
mkdir -p ~/.config/nvim/ftdetect/
cp nvim-integration/ftdetect/frugal.lua ~/.config/nvim/ftdetect/
```

#### Full Installation
Use the automated installer:
```bash
cd nvim-integration
./install.sh
```

Or see `nvim-integration/README.md` for detailed manual setup instructions.

### Configuration

VS Code settings:
- `frugal-ls.server.path`: Path to the frugal-ls executable
- `frugal-ls.server.args`: Additional arguments for the server
- `frugal-ls.trace.server`: Enable communication tracing

## Architecture

### Project Structure
```
frugal-ls/
├── cmd/frugal-ls/          # Main executable
├── internal/
│   ├── document/            # Document lifecycle management
│   ├── features/            # Language feature providers
│   │   ├── completion.go    # Code completion
│   │   ├── hover.go         # Hover information
│   │   ├── symbols.go       # Document/workspace symbols
│   │   ├── definition.go    # Go-to-definition
│   │   ├── codeactions.go   # Code actions and quick fixes
│   │   └── formatting.go    # Document formatting
│   ├── lsp/                 # LSP server implementation
│   ├── parser/              # Tree-sitter parser wrapper
│   └── workspace/           # Cross-file dependency resolution
├── pkg/ast/                 # AST utilities and symbol extraction
├── vscode-extension/        # VS Code extension
├── nvim-integration/        # Neovim integration files
│   ├── ftdetect/           # Filetype detection
│   ├── after/syntax/       # Syntax highlighting
│   ├── lua/plugins/        # Plugin configurations
│   └── install.sh          # Automated installer
└── sample.frugal           # Example Frugal file
```

### Technology Stack
- **Go 1.21+** - Server implementation
- **tree-sitter** - Syntax parsing with existing grammar
- **GLSP** - LSP 3.16 protocol implementation
- **TypeScript** - VS Code extension
- **TextMate Grammar** - Syntax highlighting

## Development Phases

This project was built iteratively through 4 phases:

1. **Phase 1** - Project setup and tree-sitter integration
2. **Phase 2** - Core LSP server with document lifecycle and diagnostics
3. **Phase 3** - Language features (completion, hover, symbols, definition)
4. **Phase 4** - Advanced features (includes resolution, code actions, formatting, VS Code extension)

## Example Usage

Given a Frugal file like:
```frugal
include "common.frugal"

namespace go example

service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound error),
    void updateUser(1: User user)
}

scope UserEvents prefix "user" {
    UserCreated: User,
    UserUpdated: User
}

struct User {
    1: required i64 id,
    2: optional string name,
    3: optional string email
}

exception UserNotFound {
    1: string message
}
```

The LSP provides:
- Syntax highlighting for all Frugal constructs
- Code completion for types, services, and methods
- Hover information showing full signatures and documentation
- Go-to-definition across files (following includes)
- Real-time syntax error detection
- Code actions for common improvements
- Automatic formatting

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## About Frugal

Frugal extends Apache Thrift with additional constructs for pub/sub messaging:
- **Scopes** - Define publish/subscribe event channels with prefixes
- **Backward Compatibility** - Works with existing Thrift tooling
- **Multi-language Support** - Generates code for Go, Java, Dart, Python, and more

For more information about Frugal, visit the [Frugal documentation](https://github.com/Workiva/frugal).