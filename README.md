# RoriCode - Another Terminal Coding Agent

RoriCode is a powerful terminal-based coding assistant that brings the power of AI directly to your command line. Built with Go, it provides a streamlined console interface with advanced features for developers who prefer working in terminal environments.

## 🚀 Overview

RoriCode is designed for developers who want a lightweight, efficient coding assistant that lives directly in their terminal. It combines the power of OpenAI's API with a suite of built-in tools to help you code, explore files, and execute commands - all without leaving your terminal.

## ✨ Key Features

### 💬 AI-Powered Chat Interface
- Direct integration with OpenAI's GPT models
- Markdown rendering for rich text responses
- Real-time conversation flow with streaming responses

### 🛠️ Built-in Tools System
RoriCode comes with several built-in tools that extend the AI's capabilities:

- **Shell Commands**, **File Operations**, **Time Utilities**, **Directory Browsing**
- See [Built-in Tools](#️-built-in-tools) section for detailed information

### 🎨 Enhanced Terminal Experience
- **Color-coded Messages**: Different message types with distinct styling
- **Smart Status Management**: Dynamic status bar with automatic clearing/restoring
- **Terminal Integration**: Full support for scrolling and terminal history
- **Real-time Tool Display**: Immediate visualization of tool calls and results

### ⚙️ Profile-based Configuration
- Multiple API profiles for different providers (OpenAI, local models, etc.)
- JSON-based configuration management
- Easy profile switching and management via CLI commands

### 🏗️ Technical Architecture
- **Event-driven Design**: Clean separation between UI, business logic, and AI service
- **Resource Optimized**: Incremental message transmission to reduce bandwidth
- **Extensible Tool System**: Easy registration of custom tools
- **Error Resilience**: Circuit breaker pattern and graceful error handling

## 📦 Installation

### Prerequisites

- **Go 1.24.4 or later**: RoriCode is built with Go. Download from [golang.org](https://golang.org/dl/)
- **Git**: For cloning the repository
- **OpenAI API Key**: Sign up at [OpenAI](https://platform.openai.com/) to get your API key

### Method 1: From Source (Recommended)

```bash
# Clone the repository
git clone https://github.com/Rorical/RoriCode.git
cd RoriCode

# Download dependencies
go mod tidy

# Build for your platform
go build -o roricode main.go
```

### Method 2: Direct Installation with Go

If you have Go installed, you can install directly:

```bash
go install github.com/Rorical/RoriCode@latest
```

### Method 3: Cross-Platform Builds

Build for different platforms:

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o roricode.exe main.go

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o roricode-mac-intel main.go

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o roricode-mac-arm main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o roricode-linux main.go
```

### Installing to System PATH

To use `roricode` from anywhere:

**On Windows:**
```cmd
# Copy to a directory in your PATH, or add current directory to PATH
copy roricode.exe C:\Windows\System32\
```

**On macOS/Linux:**
```bash
# Copy to /usr/local/bin (requires sudo)
sudo cp roricode /usr/local/bin/

# Or copy to ~/bin and add to PATH
cp roricode ~/bin/
echo 'export PATH=$PATH:~/bin' >> ~/.bashrc  # or ~/.zshrc
```

### Development Installation

For development purposes:

```bash
# Or run directly
go run main.go
```

## 🚀 Quick Start

### Verify Installation

After installation, verify RoriCode is working:

```bash
# Check if the binary is accessible
roricode --help

# Test the application (will prompt for profile setup on first run)
roricode
```

If you encounter any issues, see the [Troubleshooting](#-troubleshooting) section below.

1. **Configure your API key**:
   ```bash
   # Add a new profile
   roricode profile add default
   
   # The tool will prompt for your OpenAI API key and other settings
   ```

2. **List and manage profiles**:
   ```bash
   # List all profiles
   roricode profile list
   
   # Show specific profile details
   roricode profile show default
   ```

3. **Start chatting**:
   ```bash
   # Run the application
   roricode
   ```

## 🎯 Message Types

- **Program Messages** (Purple): Welcome messages and system information
- **User Messages** (Blue): Your input with a left border
- **Assistant Messages** (Orange): AI responses with markdown rendering
- **Tool Calls** (Red): Tool invocation requests with name and arguments
- **Tool Results** (Green): Results from executed tools

## 🛠️ Built-in Tools

### Shell Tool (`shell`)
Execute shell commands with safety features:
- User confirmation for potentially dangerous commands
- Timeout control for command execution
- Error handling and exit code reporting

### File Read Tool (`read_file`)
Read file contents with advanced filtering:
- Read specific line ranges
- Search with regex patterns and context lines
- Directory listing with file information
- Security protection against directory traversal

### File Edit Tool (`edit_file`)
Modify existing files using git diff format:
- Apply unified diff patches to files
- Context-aware editing for accuracy

### File Creation Tool (`create_file`)
Create new files with content:
- Write content to new files
- Overwrite protection

### Current Time Tool (`current_time`)
Get current date and time information:
- Various time format options

## ⌨️ Key Bindings

- **Enter**: Send message to AI
- **Ctrl+C / q / quit / exit**: Quit the application
- **Any text**: Direct console input

## 📁 Configuration

Configuration is stored in `~/.roricode/config.json`:

```json
{
  "active_profile": "default",
  "profiles": {
    "default": {
      "api_key": "your-api-key",
      "base_url": "https://api.openai.com/v1",
      "model": "gpt-4o-mini"
    }
  }
}
```

## 🧪 Development

```bash
# Run tests
go test ./...

# Update dependencies (also used in installation)
# See Installation section for build and run instructions
```

## 🏗️ Architecture Overview

```
RoriCode/
├── main.go                 # Application entry point
├── cmd/                    # CLI commands
│   ├── root.go            # Main command
│   ├── profile.go         # Profile management
│   └── use.go             # Profile switching
├── internal/
│   ├── app/               # Application lifecycle and UI
│   ├── config/            # Configuration management
│   ├── core/              # Core service and state management
│   ├── dispatcher/        # Event dispatching
│   ├── eventbus/          # Event bus system
│   ├── models/            # Data models
│   ├── tools/             # Built-in tools and registry
│   └── utils/             # Utility functions
```

## 🌟 Benefits

- **Terminal Native**: Works seamlessly with your existing terminal workflow
- **Lightweight**: No heavy GUI frameworks, minimal resource usage
- **Extensible**: Easy to add new tools and capabilities
- **Secure**: Built-in protections against dangerous operations
- **Customizable**: Multiple profiles for different use cases
- **Real-time**: Immediate feedback for tool operations

## 🤝 Contributing
## 🔧 Troubleshooting

### Common Issues

**"roricode: command not found"**
- Make sure the binary is in your system PATH
- Try running with full path to the executable
- Verify the build was successful

**"API key not configured"**
- Run `roricode profile add default` to set up your first profile
- Ensure your OpenAI API key is valid and has sufficient credits
- Check your internet connection

**"Permission denied" on macOS/Linux**
- Make the binary executable: `chmod +x roricode`
- On macOS, you may need to allow the app in Security & Privacy settings

**Build errors**
- Ensure you have Go 1.24.4 or later: `go version`
- Run `go mod tidy` to download dependencies
- Clear Go module cache: `go clean -modcache`

**"Tool execution failed"**
- Check file permissions for file operations
- Ensure you have necessary system permissions for shell commands
- Some operations may require confirmation prompts

For other issues, please check the [GitHub Issues](https://github.com/Rorical/RoriCode/issues) or create a new issue.

Contributions are welcome! Feel free to submit issues, feature requests, or pull requests to help improve RoriCode.

## 📄 License

This project is licensed under the Apache 2.0 License - see the LICENSE file for details.
