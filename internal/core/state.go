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
}

func NewChatState() *ChatState {
	return &ChatState{
		uiMessages:        make([]models.Message, 0),
		openaiHistory:     make([]openai.ChatCompletionMessage, 0),
		isProcessing:      false,
		lastError:         nil,
		conversationReady: true,
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

// AddSystemMessage adds a system message (configuration info, instructions)
func (cs *ChatState) AddSystemMessage(content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	systemMsg := models.Message{
		Content: content,
		Type:    models.System,
	}
	cs.uiMessages = append(cs.uiMessages, systemMsg)
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