package core

import (
	"sync"

	"github.com/Rorical/RoriCode/internal/models"
	"github.com/sashabaranov/go-openai"
)

// ChatState manages the conversation state for event-driven architecture
type ChatState struct {
	mu                sync.RWMutex
	chatHistory       []openai.ChatCompletionMessage // Single source of truth for conversation
	programMessages   []models.Message                // Program messages (welcome, status, etc.)
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


func (cs *ChatState) GetChatHistory() []openai.ChatCompletionMessage {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	result := make([]openai.ChatCompletionMessage, len(cs.chatHistory))
	copy(result, cs.chatHistory)
	return result
}

func (cs *ChatState) GetMessages() []models.Message {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	var result []models.Message
	
	// First add program messages
	result = append(result, cs.programMessages...)
	
	// Convert OpenAI history to UI messages
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

func (cs *ChatState) SetProcessing(processing bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.isProcessing = processing
}

func (cs *ChatState) IsProcessing() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.isProcessing
}

func (cs *ChatState) SetError(err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.lastError = err
}

func (cs *ChatState) GetLastError() error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.lastError
}

func (cs *ChatState) ClearError() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.lastError = nil
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

func (cs *ChatState) FinishProcessingWithAssistantMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Atomic: stop processing and add assistant message
	cs.isProcessing = false
	cs.lastError = nil
	
	// Add to chat history (single source of truth)
	openaiMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
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


// AddToolResultMessage adds a tool result message to OpenAI history
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

// AddAssistantMessageWithToolCalls adds an assistant message with tool calls to OpenAI history
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