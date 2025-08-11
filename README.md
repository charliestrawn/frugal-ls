# Frugal Language Server Protocol (LSP) Implementation

> **⚠️ Important Note**: Frugal was originally an open-source project by Workiva that extended Apache Thrift with pub/sub messaging. However, Frugal is no longer open source. This project is a personal learning exercise and toy implementation based on my fork of the original grammar and specification. It is not affiliated with or endorsed by Workiva.

A comprehensive Language Server Protocol implementation for Frugal IDL files, providing modern IDE features like syntax highlighting, code completion, diagnostics, and more.

## Features

### Core Language Features
- **Syntax Error Detection** - Real-time diagnostics with detailed error reporting
- **Code Completion** - Context-aware completions for types, services, and identifiers
- **Hover Information** - Rich documentation on hover with type information
- **Go to Definition** - Navigate to symbol definitions across files
- **Find References** - Find all references to symbols throughout the workspace
- **Document Symbols** - Hierarchical outline view of file structure
- **Workspace Symbols** - Search symbols across the entire workspace
- **Rename Symbol** - Rename symbols with validation and conflict detection
- **Document Formatting** - Automatic code formatting with consistent style
- **Semantic Syntax Highlighting** - Enhanced syntax highlighting based on semantic analysis
- **Document Highlights** - Highlight all occurrences of the symbol under cursor

### Advanced Features
- **Cross-file Include Resolution** - Full support for include statements and dependency tracking
- **Code Actions & Quick Fixes** - Automated refactoring and code improvements:
  - Extract method parameters to struct
  - Add missing fields to structs
  - Generate method stubs
  - Organize includes
- **VS Code Extension** - Complete VS Code integration with syntax highlighting and language features

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release from [GitHub Releases](https://github.com/charliestrawn/frugal-ls/releases):

```bash
# Download the binary (Linux AMD64)
wget https://github.com/charliestrawn/frugal-ls/releases/download/v0.1.0/frugal-ls-linux-amd64

# Make it executable and install
chmod +x frugal-ls-linux-amd64
sudo mv frugal-ls-linux-amd64 /usr/local/bin/frugal-ls
```

### Option 2: Build from Source

1. **Prerequisites:**
   - Go 1.23+ 
   - Git

2. **Clone and build:**
   ```bash
   git clone https://github.com/charliestrawn/frugal-ls
   cd frugal-ls
   go build -o frugal-ls ./cmd/frugal-ls
   ```

### VS Code Extension

#### Option 1: Download from Release
1. Download `frugal-ls-0.1.0.vsix` from [GitHub Releases](https://github.com/charliestrawn/frugal-ls/releases)
2. Install via VS Code: `code --install-extension frugal-ls-0.1.0.vsix`
3. Or install via VS Code UI: Command Palette → "Extensions: Install from VSIX"

#### Option 2: Build from Source
```bash
cd vscode-extension
npm install
npm run compile
npm install -g @vscode/vsce
vsce package
code --install-extension frugal-ls-0.1.0.vsix
```

## Usage

### VS Code
1. Install the extension (see above)
2. Ensure `frugal-ls` is in your PATH, or configure the path in settings
3. Open any `.frugal` file to activate language support

### Other Editors

#### Neovim with nvim-lspconfig
```lua
-- Add to your LSP configuration
require('lspconfig').frugal_ls.setup({
  cmd = { "frugal-ls" },
  filetypes = { "frugal" },
  root_dir = require('lspconfig.util').root_pattern(".git"),
})
```

#### Vim/Neovim with vim-lsp
```vim
if executable('frugal-ls')
    autocmd User lsp_setup call lsp#register_server({
        \ 'name': 'frugal-ls',
        \ 'cmd': {server_info->['frugal-ls']},
        \ 'allowlist': ['frugal'],
        \ })
endif
```

#### Emacs with lsp-mode
```elisp
(add-to-list 'lsp-language-id-configuration '(frugal-mode . "frugal"))
(lsp-register-client
 (make-lsp-client :new-connection (lsp-stdio-connection "frugal-ls")
                  :major-modes '(frugal-mode)
                  :server-id 'frugal-ls))
```

### Command Line Testing
```bash
# Test the language server
frugal-ls --version

# Test parsing a file
frugal-ls --test sample.frugal
```

## Configuration

### VS Code Settings
- `frugal-ls.server.path`: Path to the frugal-ls executable (default: "frugal-ls")
- `frugal-ls.server.args`: Additional arguments for the server (default: [])
- `frugal-ls.trace.server`: Enable communication tracing (default: "off")

## Example Frugal Code

```frugal
include "common.frugal"

namespace go example

// User service for managing user data
service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound error),
    void updateUser(1: User user),
    list<User> getAllUsers()
}

// User events scope for pub/sub messaging
scope UserEvents prefix "user.events" {
    UserCreated: User,
    UserUpdated: User,
    UserDeleted: UserDeletion
}

struct User {
    1: required i64 id,
    2: optional string name,
    3: optional string email,
    4: optional i64 createdAt
}

struct UserDeletion {
    1: required i64 userId,
    2: optional string reason
}

exception UserNotFound {
    1: string message = "User not found"
}
```

## Architecture

Built with modern tooling and best practices:

- **Go 1.23+** - Server implementation with comprehensive test coverage
- **Tree-sitter** - Fast, incremental parsing with syntax highlighting
- **LSP 3.16** - Full Language Server Protocol compliance
- **TypeScript** - VS Code extension with type safety
- **GitHub Actions** - Automated CI/CD with cross-platform releases

### Project Structure
```
frugal-ls/
├── cmd/frugal-ls/          # Main executable and CLI
├── internal/
│   ├── document/           # Document lifecycle management
│   ├── features/           # LSP feature implementations
│   ├── lsp/               # LSP protocol server
│   ├── parser/            # Tree-sitter parser integration
│   └── workspace/         # Cross-file analysis and includes
├── pkg/ast/               # AST utilities and symbol extraction
├── vscode-extension/      # VS Code extension
└── .github/workflows/     # CI/CD automation
```

## Development

```bash
# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Build for development
go build -o frugal-ls ./cmd/frugal-ls

# Build VS Code extension
cd vscode-extension
npm install && npm run compile
```

## Contributing

This is primarily a personal learning project, but contributions are welcome:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## About This Project

This language server implementation is a personal project created for learning purposes. It demonstrates:

- Modern LSP server architecture
- Tree-sitter parser integration
- Comprehensive IDE feature implementation
- Cross-platform distribution with GitHub Actions
- Multi-editor support (VS Code, Neovim, Vim, Emacs)

The original Frugal specification and grammar were created by Workiva as an extension to Apache Thrift for pub/sub messaging patterns.