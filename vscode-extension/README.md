# Frugal IDL Language Support

> **⚠️ Important Note**: This extension supports Frugal IDL, which was originally an open-source project by Workiva that extended Apache Thrift with pub/sub messaging. However, Frugal is no longer open source. This is a personal learning project based on my fork of the original grammar and specification.

Comprehensive language support for Frugal IDL files in Visual Studio Code, powered by a custom Language Server Protocol (LSP) implementation.

## Features

### Core Language Features
- **🎨 Syntax Highlighting** - Rich syntax highlighting with semantic tokens
- **✨ IntelliSense** - Context-aware code completion and suggestions
- **📖 Hover Information** - Detailed documentation and type information on hover
- **🔍 Go to Definition** - Navigate to symbol definitions across files
- **🔎 Find References** - Find all references to symbols throughout the workspace
- **📋 Document Symbols** - Hierarchical outline view of file structure
- **🌐 Workspace Symbols** - Search symbols across the entire workspace
- **✏️ Rename Symbol** - Rename symbols with validation and conflict detection
- **🔧 Code Actions** - Quick fixes and refactoring suggestions:
  - Extract method parameters to struct
  - Add missing fields to structs
  - Generate method stubs
  - Organize includes
- **📝 Document Formatting** - Automatic code formatting with consistent style
- **⚠️ Diagnostics** - Real-time syntax error detection with detailed messages
- **🔗 Cross-file Support** - Full include statement resolution and navigation
- **💡 Document Highlights** - Highlight all occurrences of the symbol under cursor

## Installation

### Option 1: Download from GitHub Release (Recommended)

1. **Install the language server:**
   ```bash
   # Download the binary (Linux AMD64)
   wget https://github.com/charliestrawn/frugal-ls/releases/download/v0.1.0/frugal-ls-linux-amd64
   
   # Make it executable and install
   chmod +x frugal-ls-linux-amd64
   sudo mv frugal-ls-linux-amd64 /usr/local/bin/frugal-ls
   ```

2. **Install the VS Code extension:**
   - Download `frugal-ls-0.1.0.vsix` from [GitHub Releases](https://github.com/charliestrawn/frugal-ls/releases)
   - Install via command line: `code --install-extension frugal-ls-0.1.0.vsix`
   - Or install via VS Code UI: Command Palette → "Extensions: Install from VSIX"

### Option 2: Build from Source

1. **Clone and build the language server:**
   ```bash
   git clone https://github.com/charliestrawn/frugal-ls
   cd frugal-ls
   go build -o frugal-ls ./cmd/frugal-ls
   ```

2. **Build and install the extension:**
   ```bash
   cd vscode-extension
   npm install
   npm run compile
   npm install -g @vscode/vsce
   vsce package
   code --install-extension frugal-ls-0.1.0.vsix
   ```

## Configuration

Configure the extension via VS Code settings:

- `frugal-ls.server.path`: Path to the frugal-ls executable (default: "frugal-ls")
- `frugal-ls.server.args`: Arguments to pass to the language server (default: [])
- `frugal-ls.trace.server`: Enable server communication tracing (default: "off", options: "off", "messages", "verbose")

### Example settings.json:
```json
{
  "frugal-ls.server.path": "/usr/local/bin/frugal-ls",
  "frugal-ls.trace.server": "messages"
}
```

## Usage

1. Install the extension and language server (see above)
2. Open any `.frugal` file
3. The extension will automatically activate and provide language features

### Example Frugal file:
```frugal
include "common.frugal"

namespace go example

// User management service
service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound error),
    void updateUser(1: User user),
    list<User> getAllUsers()
}

// User events for pub/sub messaging
scope UserEvents prefix "user.events" {
    UserCreated: User,
    UserUpdated: User,
    UserDeleted: UserDeletion
}

struct User {
    1: required i64 id,
    2: optional string name,
    3: optional string email
}

exception UserNotFound {
    1: string message = "User not found"
}
```

## Commands

- `Frugal LS: Restart Server` - Restart the language server if needed

## Troubleshooting

### Language server not starting
- Ensure `frugal-ls` is installed and in your PATH, or configure `frugal-ls.server.path`
- Check the Output panel (View → Output → "Frugal LSP") for error messages
- Try restarting VS Code or running the "Frugal LS: Restart Server" command

### No syntax highlighting
- Ensure the file has a `.frugal` extension
- Try reloading VS Code (Developer → Reload Window)

### Features not working
- Enable tracing with `"frugal-ls.trace.server": "verbose"` to debug communication issues
- Check that the language server binary is the correct version for your platform

## Development

To contribute to the extension:

```bash
# Install dependencies
npm install

# Compile TypeScript
npm run compile

# Run linting
npm run lint

# Package the extension
vsce package
```

## About This Project

This VS Code extension is part of a personal learning project that implements a comprehensive Language Server Protocol for Frugal IDL. It demonstrates modern LSP architecture and provides a full-featured IDE experience for Frugal files.

The original Frugal specification and grammar were created by Workiva as an extension to Apache Thrift for pub/sub messaging patterns.

## License

MIT License - see [LICENSE](../LICENSE) file for details.