package core

import (
	"sync"

	"github.com/Rorical/RoriCode/internal/models"
	"github.com/sashabaranov/go-openai"
)

// ChatState manages the conversation state for event-driven architecture
type ChatState struct {
	mu                sync.RWMutex
	uiMessages        []models.Message                // Messages for UI display
	openaiHistory     []openai.ChatCompletionMessage // Direct OpenAI conversation history
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
		uiMessages:        make([]models.Message, 0),
		openaiHistory:     make([]openai.ChatCompletionMessage, 0),
		isProcessing:      false,
		lastError:         nil,
		conversationReady: true,
		pendingToolCalls:  make(map[string]bool),
		recursionDepth:    0,
		maxRecursionDepth: 5, // Prevent infinite recursion
	}
}

func (cs *ChatState) AddUserMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Add to UI messages
	userMsg := models.Message{
		Content: content,
		Type:    models.User,
	}
	cs.uiMessages = append(cs.uiMessages, userMsg)
	
	// Add to OpenAI history
	openaiMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	}
	cs.openaiHistory = append(cs.openaiHistory, openaiMsg)
}

func (cs *ChatState) AddAssistantMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Add to UI messages
	assistantMsg := models.Message{
		Content: content,
		Type:    models.Assistant,
	}
	cs.uiMessages = append(cs.uiMessages, assistantMsg)
	
	// Add to OpenAI history
	openaiMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	}
	cs.openaiHistory = append(cs.openaiHistory, openaiMsg)
}

func (cs *ChatState) GetOpenAIHistory() []openai.ChatCompletionMessage {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	result := make([]openai.ChatCompletionMessage, len(cs.openaiHistory))
	copy(result, cs.openaiHistory)
	return result
}

func (cs *ChatState) GetMessages() []models.Message {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	result := make([]models.Message, len(cs.uiMessages))
	copy(result, cs.uiMessages)
	return result
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
	cs.uiMessages = append(cs.uiMessages, programMsg)
}


// Atomic operations for event ordering
func (cs *ChatState) StartProcessingWithUserMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Atomic: set processing and add user message
	cs.isProcessing = true
	cs.lastError = nil
	
	// Add to UI messages
	userMsg := models.Message{
		Content: content,
		Type:    models.User,
	}
	cs.uiMessages = append(cs.uiMessages, userMsg)
	
	// Add to OpenAI history
	openaiMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	}
	cs.openaiHistory = append(cs.openaiHistory, openaiMsg)
}

func (cs *ChatState) FinishProcessingWithAssistantMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Atomic: stop processing and add assistant message
	cs.isProcessing = false
	cs.lastError = nil
	
	// Add to UI messages
	assistantMsg := models.Message{
		Content: content,
		Type:    models.Assistant,
	}
	cs.uiMessages = append(cs.uiMessages, assistantMsg)
	
	// Add to OpenAI history
	openaiMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	}
	cs.openaiHistory = append(cs.openaiHistory, openaiMsg)
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


// AddToolResultMessage adds a tool result message to UI and OpenAI history
func (cs *ChatState) AddToolResultMessage(callID, toolName, result string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Add to UI messages
	toolResultMsg := models.Message{
		Content:    result,
		Type:       models.ToolResult,
		ToolCallID: callID,
		ToolName:   toolName,
	}
	cs.uiMessages = append(cs.uiMessages, toolResultMsg)
	
	// Add to OpenAI history as tool message
	openaiMsg := openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    result,
		ToolCallID: callID,
	}
	cs.openaiHistory = append(cs.openaiHistory, openaiMsg)
}

// AddAssistantMessageWithToolCalls adds an assistant message with tool calls to both UI and OpenAI history
func (cs *ChatState) AddAssistantMessageWithToolCalls(content string, toolCalls []openai.ToolCall) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Add to UI messages if there's content
	if content != "" {
		assistantMsg := models.Message{
			Content: content,
			Type:    models.Assistant,
		}
		cs.uiMessages = append(cs.uiMessages, assistantMsg)
	}
	
	// Add to OpenAI history as single message with content and tool calls
	openaiMsg := openai.ChatCompletionMessage{
		Role:      openai.ChatMessageRoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
	cs.openaiHistory = append(cs.openaiHistory, openaiMsg)
}

// AddToolCallMessageToUI adds a tool call message only to UI (not OpenAI history)
func (cs *ChatState) AddToolCallMessageToUI(callID, toolName, args string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	toolCallMsg := models.Message{
		Content:    args,
		Type:       models.ToolCall,
		ToolCallID: callID,
		ToolName:   toolName,
		ToolArgs:   args,
	}
	cs.uiMessages = append(cs.uiMessages, toolCallMsg)
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