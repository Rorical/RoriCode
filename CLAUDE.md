# RoriCode - Go TUI Chat Application

## Overview
RoriCode is a Terminal User Interface (TUI) chat application built with Go and the Bubble Tea framework. It features a clean, event-driven architecture with proper separation of concerns, OpenAI integration, and robust error handling.

## Current Status
✅ **Fully Refactored Event-Driven Architecture**
- Eliminated all 6 major anti-patterns identified in the codebase
- Implemented proper dependency injection and circuit breaker patterns
- Single source of truth for state management
- Clean separation between UI and business logic

## Architecture

### Project Structure
```
RoriCode/
├── main.go                             # Minimal application entry point (20 lines)
├── internal/
│   ├── app/
│   │   ├── application.go             # Application lifecycle management
│   │   └── model.go                   # Bubble Tea model implementation
│   ├── config/
│   │   └── config.go                  # Environment configuration
│   ├── core/
│   │   ├── service.go                 # ChatService with OpenAI integration
│   │   └── state.go                   # Core state management (single source of truth)
│   ├── dispatcher/
│   │   └── event_dispatcher.go        # Event routing between core and UI
│   ├── eventbus/
│   │   └── eventbus.go                # Channel-based event bus with circuit breaker
│   ├── models/
│   │   ├── app.go                     # UI-specific application state model
│   │   └── message.go                 # Message types and structures
│   └── update/
│       ├── handlers.go                # Event handlers with error handling
│       └── update.go                  # Main update dispatcher
└── ui/
    ├── components/
    │   ├── input.go                   # Input box rendering
    │   ├── messages.go                # Message history rendering
    │   └── status.go                  # Status bar rendering
    └── styles/
        └── styles.go                  # UI styling functions
```

### Event-Driven Architecture
- **UI Layer**: Handles user input, renders components, manages local UI state
- **Event Bus**: Channel-based communication with circuit breaker pattern
- **Core Layer**: Business logic, OpenAI API calls, conversation state management
- **Application Layer**: Dependency injection, lifecycle management

### Data Flow
1. User input → UI handlers → EventBus → Core handlers
2. Core processes message → OpenAI API call → State update
3. Core pushes state → EventBus → UI receives update → Re-render

### Message Types
- **Program**: Welcome messages and program information (purple/magenta, bold, centered)
- **System**: System notifications and instructions (gray)
- **User**: User input messages (blue with left border)
- **Assistant**: OpenAI assistant responses (orange with left border)

### Key Features
- **Event-Driven Architecture**: UI and core communicate via channel-based event bus
- **Circuit Breaker Pattern**: Automatic failure handling and recovery
- **Single Source of Truth**: Core state manages all conversation history
- **Atomic State Updates**: Race-condition-free state management
- **Natural Scrolling**: Messages expand naturally, allowing terminal scroll navigation
- **Styled Components**: Each message type has distinct visual styling
- **Loading Animations**: Animated dots during OpenAI processing
- **Responsive Layout**: Adapts to terminal size changes
- **OpenAI Integration**: Supports custom API keys and base URLs

### Error Handling
- Circuit breaker pattern prevents event bus overload
- Graceful degradation when OpenAI is unavailable
- Proper context cancellation for goroutine cleanup
- Error propagation with user-friendly status messages

### UI Layout
1. **Message History**: Scrollable area at top with all chat messages
2. **Input Box**: Rounded rectangle with ">" prompt at bottom
3. **Status Bar**: Shows current status, errors, and loading animations at very bottom

### Key Bindings
- **Enter**: Send message to OpenAI
- **Backspace**: Delete character
- **Ctrl+C / q**: Quit application
- **Any character**: Add to input

### Configuration
Set environment variables for OpenAI integration:
```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # Optional
export OPENAI_MODEL="gpt-3.5-turbo"  # Optional, defaults to gpt-3.5-turbo
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

## Architecture Benefits
- **Event-Driven**: Proper separation of UI and business logic
- **Resilient**: Circuit breaker pattern handles failures gracefully  
- **Testable**: Each component can be tested independently
- **Maintainable**: Clean dependency injection and separation of concerns
- **Extensible**: Easy to add new message types, handlers, or UI components
- **Performance**: Efficient channel-based communication and atomic state updates
- **Robust**: Proper error handling and context-based cancellation

## Anti-Patterns Eliminated
1. ✅ **Global State Variables**: Replaced with dependency injection
2. ✅ **Mixed Responsibilities**: Separated UI, business logic, and infrastructure  
3. ✅ **Inefficient Event Loops**: Proper blocking with context cancellation
4. ✅ **Missing Event Ordering**: Atomic state operations guarantee consistency
5. ✅ **Poor Error Handling**: Circuit breaker pattern with error callbacks
6. ✅ **State Duplication**: Single source of truth in core state management