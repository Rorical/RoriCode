package core

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/Rorical/RoriCode/internal/models"
	"github.com/sashabaranov/go-openai"
)

// ChatState manages the conversation state for event-driven architecture
type ChatState struct {
	mu                sync.RWMutex
	chatHistory       []openai.ChatCompletionMessage // Single source of truth for conversation
	programMessages   []models.Message               // Program messages (welcome, status, etc.)
	isProcessing      bool
	lastError         error
	conversationReady bool
	// Tool call tracking
	pendingToolCalls  map[string]bool // Track pending tool calls by ID
	recursionDepth    int             // Current recursion depth for tool calls
	maxRecursionDepth int             // Maximum allowed recursion depth
}

func NewChatState() *ChatState {
	return &ChatState{
		chatHistory:       make([]openai.ChatCompletionMessage, 0),
		programMessages:   make([]models.Message, 0),
		isProcessing:      false,
		lastError:         nil,
		conversationReady: true,
		pendingToolCalls:  make(map[string]bool),
		recursionDepth:    0,
		maxRecursionDepth: 5, // Prevent infinite recursion
	}
}

// generateSystemPrompt creates a dynamic system prompt with current environment context
func (cs *ChatState) generateSystemPrompt() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	// Get system information
	systemOS := runtime.GOOS
	systemArch := runtime.GOARCH

	// Map OS names to user-friendly names
	osName := systemOS
	switch systemOS {
	case "darwin":
		osName = "macOS"
	case "windows":
		osName = "Windows"
	case "linux":
		osName = "Linux"
	}

	// Create system prompt template
	return fmt.Sprintf(`You are an active coding assistant agent named RoriCode. Your role is to explore, understand, and cooperate with the user to complete coding tasks efficiently.

## Environment Context
- **Current Working Directory**: %s
- **Operating System**: %s (%s)
- **Architecture**: %s

## Your Capabilities
You have access to various tools that allow you to:
- Execute shell commands and scripts
- Read and analyze file contents
- Interact with the local development environment
- Search for information you need, locally or remotely.

## Your Role & Behavior
1. **Active Agent**: Proactively suggest solutions, explore the codebase, and ask clarifying questions
2. **Collaborative Partner**: Work alongside the user to understand requirements and implement solutions
3. **Problem Solver**: Break down complex tasks into manageable steps and execute them systematically
4. **Code Explorer**: Navigate and understand project structures, dependencies, and existing implementations
5. **Best Practices Advocate**: Suggest improvements, follow coding standards, and ensure code quality

## Guidelines
- Be proactive in exploring the codebase to understand context
- Use available tools to gather information before making recommendations  
- Provide clear explanations of your actions and reasoning
- Ask for clarification when requirements are ambiguous
- Suggest multiple approaches when appropriate
- Focus on practical, working solutions

## Communication Style
- Be concise but thorough in explanations
- Acknowledge limitations and ask for help when needed
- Maintain a collaborative and helpful tone`, cwd, osName, systemOS, systemArch)
}

func (cs *ChatState) GetChatHistory() []openai.ChatCompletionMessage {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	result := make([]openai.ChatCompletionMessage, len(cs.chatHistory))
	copy(result, cs.chatHistory)
	return result
}

// GetChatHistoryWithSystemPrompt returns chat history with dynamic system prompt prepended
func (cs *ChatState) GetChatHistoryWithSystemPrompt() []openai.ChatCompletionMessage {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Generate dynamic system prompt
	systemPrompt := cs.generateSystemPrompt()

	// Create system message
	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	}

	// Prepend system message to chat history
	result := make([]openai.ChatCompletionMessage, 0, len(cs.chatHistory)+1)
	result = append(result, systemMessage)
	result = append(result, cs.chatHistory...)

	return result
}

func (cs *ChatState) GetMessages() []models.Message {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var result []models.Message

	// First add program messages
	result = append(result, cs.programMessages...)

	// Convert chat history to UI messages
	for _, openaiMsg := range cs.chatHistory {
		switch openaiMsg.Role {
		case openai.ChatMessageRoleUser:
			result = append(result, models.Message{
				Content: openaiMsg.Content,
				Type:    models.User,
			})
		case openai.ChatMessageRoleAssistant:
			if openaiMsg.Content != "" {
				result = append(result, models.Message{
					Content: openaiMsg.Content,
					Type:    models.Assistant,
				})
			}
			// Handle tool calls
			for _, toolCall := range openaiMsg.ToolCalls {
				result = append(result, models.Message{
					Content:    toolCall.Function.Arguments,
					Type:       models.ToolCall,
					ToolCallID: toolCall.ID,
					ToolName:   toolCall.Function.Name,
					ToolArgs:   toolCall.Function.Arguments,
				})
			}
		case openai.ChatMessageRoleTool:
			result = append(result, models.Message{
				Content:    openaiMsg.Content,
				Type:       models.ToolResult,
				ToolCallID: openaiMsg.ToolCallID,
				ToolName:   extractToolNameFromHistory(cs.chatHistory, openaiMsg.ToolCallID),
			})
		}
	}

	return result
}

// extractToolNameFromHistory finds the tool name for a given tool call ID
func extractToolNameFromHistory(history []openai.ChatCompletionMessage, toolCallID string) string {
	for _, msg := range history {
		if msg.Role == openai.ChatMessageRoleAssistant {
			for _, toolCall := range msg.ToolCalls {
				if toolCall.ID == toolCallID {
					return toolCall.Function.Name
				}
			}
		}
	}
	return "unknown"
}

func (cs *ChatState) IsProcessing() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.isProcessing
}

func (cs *ChatState) GetLastError() error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.lastError
}

func (cs *ChatState) IsConversationReady() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.conversationReady
}

// AddProgramMessage adds a program message (system notifications)
func (cs *ChatState) AddProgramMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	programMsg := models.Message{
		Content: content,
		Type:    models.Program,
	}
	cs.programMessages = append(cs.programMessages, programMsg)
}

// Atomic operations for event ordering
func (cs *ChatState) StartProcessingWithUserMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Atomic: set processing and add user message
	cs.isProcessing = true
	cs.lastError = nil

	// Add to chat history (single source of truth)
	openaiMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	}
	cs.chatHistory = append(cs.chatHistory, openaiMsg)
}

func (cs *ChatState) FinishProcessingWithError(err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Atomic: stop processing with error
	cs.isProcessing = false
	cs.lastError = err
}

func (cs *ChatState) FinishProcessing() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Atomic: stop processing without changes
	cs.isProcessing = false
	cs.lastError = nil
}

// AddToolResultMessage adds a tool result message to chat history
func (cs *ChatState) AddToolResultMessage(callID, toolName, result string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Add to chat history as tool message (UI messages generated on-demand)
	openaiMsg := openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result,
		ToolCallID: callID,
	}
	cs.chatHistory = append(cs.chatHistory, openaiMsg)
}

// AddAssistantMessageWithToolCalls adds an assistant message with tool calls to chat history
func (cs *ChatState) AddAssistantMessageWithToolCalls(content string, toolCalls []openai.ToolCall) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Add to chat history as single message with content and tool calls (UI messages generated on-demand)
	openaiMsg := openai.ChatCompletionMessage{
		Role:      openai.ChatMessageRoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
	cs.chatHistory = append(cs.chatHistory, openaiMsg)
}

// Tool call tracking methods
func (cs *ChatState) AddPendingToolCall(callID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.pendingToolCalls[callID] = true
}

func (cs *ChatState) CompletePendingToolCall(callID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.pendingToolCalls, callID)
	return len(cs.pendingToolCalls) == 0 // Return true if all calls complete
}

func (cs *ChatState) HasPendingToolCalls() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.pendingToolCalls) > 0
}

func (cs *ChatState) CanRecurse() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.recursionDepth < cs.maxRecursionDepth
}

func (cs *ChatState) IncrementRecursion() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.recursionDepth++
}

func (cs *ChatState) ResetRecursion() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.recursionDepth = 0
}
