# RoriCode - Go Console Chat Application

## Overview
RoriCode is a console-based chat application built with Go featuring pure fmt-based terminal UI, OpenAI integration with tool support, and robust event-driven architecture. The application has evolved from a Bubble Tea TUI to a streamlined console interface with advanced features.

## Current Status
✅ **Pure Console Interface with Tool System**
- Complete removal of Bubble Tea framework for streamlined console UI
- Comprehensive OpenAI Tools API integration with async execution
- Smart terminal cursor manipulation for clean UI experience
- Real-time tool call and result display with status bar management
- Resource-optimized message transmission system

## Architecture

### Project Structure
```
RoriCode/
├── main.go                             # Minimal application entry point (20 lines)
├── internal/
│   ├── app/
│   │   ├── application.go             # Application lifecycle management
│   │   └── model.go                   # Pure console UI with terminal control
│   ├── config/
│   │   └── config.go                  # Profile-based JSON configuration
│   ├── core/
│   │   ├── service.go                 # ChatService with OpenAI and tool integration
│   │   └── state.go                   # Core state management (single source of truth)
│   ├── dispatcher/
│   │   └── event_dispatcher.go        # Event routing between core and UI
│   ├── eventbus/
│   │   └── eventbus.go                # Channel-based event bus with circuit breaker
│   ├── models/
│   │   ├── app.go                     # UI-specific application state model
│   │   └── message.go                 # Message types: User, Assistant, Program, ToolCall, ToolResult
│   ├── tools/
│   │   ├── registry.go                # Tool registry with async execution
│   │   └── builtin.go                 # Built-in tools (echo, time, random)
│   └── utils/
│       ├── styles.go                  # Lipgloss styling functions
│       └── markdown.go                # Recursive markdown rendering
```

### Event-Driven Architecture
- **Console UI Layer**: Pure fmt-based terminal interface with ANSI cursor control
- **Event Bus**: Channel-based communication with circuit breaker pattern
- **Core Layer**: Business logic, OpenAI API calls, tool execution, conversation state
- **Tool System**: Registry-based async tool execution with OpenAI integration
- **Application Layer**: Dependency injection, lifecycle management

### Data Flow
1. User input → Console scanner → EventBus → Core handlers
2. Core processes message → OpenAI API call (with tools) → Tool execution → State update
3. Core pushes state → EventBus → Console receives update → Direct terminal printing
4. Real-time tool calls/results display with status bar preservation

### Message Types
- **Program**: Welcome messages and program information (purple/magenta, bold, centered)
- **User**: User input messages (blue with left border) - preserved as typed
- **Assistant**: OpenAI assistant responses (orange with left border, markdown rendered)
- **ToolCall**: Tool invocations with name and arguments (styled with special brackets)
- **ToolResult**: Tool execution results (indented with arrow prefix)

### Key Features
- **Pure Console Interface**: Direct terminal printing with fmt, no TUI framework overhead
- **OpenAI Tools Integration**: Full support for OpenAI Tools API with async execution
- **Smart Terminal Control**: ANSI escape sequences for cursor manipulation and status management
- **Real-time Tool Display**: Immediate tool call and result visualization
- **Status Bar Management**: Smart clear-print-restore cycle for clean UI
- **Resource Optimization**: Only sends new messages to reduce bandwidth usage
- **Recursion Protection**: Depth-limited tool calling to prevent infinite loops
- **Profile-based Configuration**: JSON config with multiple profile support
- **Markdown Rendering**: Rich text formatting for assistant responses
- **Terminal Scrollback**: Full integration with terminal scrolling history

### Error Handling
- Circuit breaker pattern prevents event bus overload
- Graceful degradation when OpenAI is unavailable
- Proper context cancellation for goroutine cleanup
- Tool execution error handling with user-friendly messages
- Automatic input line clearing on errors

### Console Interface
- **Message History**: Direct terminal printing using scrollback buffer
- **Input Line**: Simple "> " prompt with automatic clearing after send
- **Status Display**: Dynamic status messages with smart clearing/restoration
- **Tool Visualization**: Real-time tool calls and results with preserved status

### Key Bindings
- **Enter**: Send message to OpenAI (input line auto-clears)
- **Ctrl+C / q / quit / exit**: Quit application
- **Any text**: Direct console input

### Configuration
Profile-based JSON configuration at `~/.roricode/config.json`:
```json
{
  "active_profile": "default",
  "profiles": {
    "default": {
      "api_key": "your-api-key",
      "base_url": "https://api.openai.com/v1",
      "model": "gpt-4"
    }
  }
}
```

CLI profile management:
```bash
# Add new profile
roricode profile add <name>

# List profiles  
roricode profile list

# Switch profile
roricode profile use <name>
```

### Development Commands
```bash
# Run the application
go run main.go

# Build the application  
go build -o roricode main.go

# Run with Go modules
go mod tidy && go run main.go

# Test the application
go test ./...
```

### Tool System
Built-in tools available for OpenAI:
- **echo**: Simple text echo for testing
- **time**: Current timestamp generation  
- **random**: Random number generation

Tool execution features:
- **Async Processing**: Non-blocking tool execution
- **Real-time Display**: Immediate tool call/result visualization
- **Recursion Control**: Maximum depth protection (5 levels)
- **Error Recovery**: Graceful handling of tool failures
- **OpenAI Integration**: Seamless integration with chat completion

## Architecture Benefits
- **Lightweight**: Pure console interface with minimal overhead
- **Real-time**: Immediate tool execution and result display
- **Resilient**: Circuit breaker pattern and error recovery
- **Extensible**: Easy tool registration and message type additions
- **Resource Efficient**: Optimized message transmission and state updates
- **Terminal Native**: Full integration with terminal scrolling and history
- **Tool-Enabled**: Comprehensive OpenAI Tools API support

## Evolution Summary
1. ✅ **Bubble Tea → Pure Console**: Complete framework removal for performance
2. ✅ **Environment → Profile Config**: JSON-based configuration management
3. ✅ **Basic Chat → Tool Integration**: Full OpenAI Tools API support
4. ✅ **Static UI → Dynamic Display**: Real-time tool execution visualization
5. ✅ **Resource Waste → Optimization**: Incremental message transmission
6. ✅ **Simple Status → Smart Management**: Status clear-print-restore cycle