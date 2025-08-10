package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Rorical/RoriCode/internal/config"
	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/models"
	"github.com/Rorical/RoriCode/internal/tools"
	"github.com/sashabaranov/go-openai"
)

type ChatService struct {
	client           *openai.Client
	config           *config.Config
	state            *ChatState
	eventBus         *eventbus.EventBus
	toolRegistry     *tools.Registry
	ctx              context.Context
	cancel           context.CancelFunc
	lastSentCount    int                            // Track how many messages we've sent to UI
	pendingConfirms  map[string]chan bool           // Track pending confirmations
	confirmMutex     sync.RWMutex                   // Protect pendingConfirms map
}

// NewChatService creates a ChatService regardless of config validity
// This ensures we always have a service to manage state
func NewChatService(cfg *config.Config, eb *eventbus.EventBus) (*ChatService, error) {
	var client *openai.Client

	// Only create OpenAI client if config is valid
	if cfg.IsValid() {
		clientConfig := openai.DefaultConfig(cfg.GetAPIKey())
		if cfg.GetBaseURL() != "" {
			clientConfig.BaseURL = cfg.GetBaseURL()
		}
		client = openai.NewClientWithConfig(clientConfig)
	}

	state := NewChatState()
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize tool registry and register builtin tools
	toolRegistry := tools.NewRegistry()
	tools.RegisterBuiltinTools(toolRegistry)

	service := &ChatService{
		client:          client, // May be nil if config invalid
		config:          cfg,
		state:           state,
		eventBus:        eb,
		toolRegistry:    toolRegistry,
		ctx:             ctx,
		cancel:          cancel,
		pendingConfirms: make(map[string]chan bool),
		lastSentCount:   0,
	}

	// Set the service as the confirmator for tools that need confirmation
	toolRegistry.SetConfirmator(service)

	// Add welcome screen with better formatting
	service.addWelcomeMessages(cfg)

	return service, nil
}

// Start runs the core logic in a goroutine
func (cs *ChatService) Start() {
	// Send initial state to UI immediately
	cs.pushStateToUI()
	go cs.eventLoop()
}

func (cs *ChatService) Stop() {
	cs.cancel()
}

func (cs *ChatService) eventLoop() {
	for {
		select {
		case <-cs.ctx.Done():
			return
		case event, ok := <-cs.eventBus.UIToCore():
			if !ok {
				return
			}
			cs.handleUIEvent(event)
		}
	}
}

func (cs *ChatService) handleUIEvent(event eventbus.UIEvent) {
	switch e := event.(type) {
	case eventbus.SendMessageEvent:
		cs.processMessage(e.Message)
	case eventbus.ConfirmationResponseEvent:
		cs.handleConfirmationResponse(e)
	}
}

func (cs *ChatService) processMessage(userMessage string) {
	// Atomic update: Set processing and add user message
	cs.state.StartProcessingWithUserMessage(userMessage)
	cs.state.ResetRecursion() // Reset recursion depth for new conversation
	cs.pushStateToUI()

	// Start the recursive chat completion process
	cs.continueConversation()
}

// continueConversation handles the recursive chat completion with tool calling
func (cs *ChatService) continueConversation() {
	// If no OpenAI client, just finish processing
	if cs.client == nil {
		cs.state.FinishProcessingWithError(fmt.Errorf("OpenAI integration not available"))
		cs.pushStateToUI()
		return
	}

	// Check recursion depth BEFORE making the call
	if !cs.state.CanRecurse() {
		cs.state.FinishProcessingWithError(fmt.Errorf("maximum tool call recursion depth reached"))
		cs.pushStateToUI()
		return
	}

	// Increment recursion depth for each OpenAI API call to prevent infinite loops
	cs.state.IncrementRecursion()

	// Get chat conversation history with dynamic system prompt
	openaiMessages := cs.state.GetChatHistoryWithSystemPrompt()

	// Call OpenAI API with tool support
	req := openai.ChatCompletionRequest{
		Model:    cs.config.GetModel(),
		Messages: openaiMessages,
		Tools:    cs.getToolsSpec(),
	}

	resp, err := cs.client.CreateChatCompletion(cs.ctx, req)

	if err != nil {
		// Atomic update: Stop processing with error
		cs.state.FinishProcessingWithError(fmt.Errorf("OpenAI API error: %w", err))
		cs.state.ResetRecursion() // Reset on error
		cs.pushStateToUI()
		return
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		message := choice.Message
		
		// Handle assistant message with potential tool calls
		if message.Content != "" || len(message.ToolCalls) > 0 {
			cs.state.AddAssistantMessageWithToolCalls(message.Content, message.ToolCalls)
			cs.pushStateToUI() // Show assistant message immediately
		}
		
		// Handle tool calls if present
		if len(message.ToolCalls) > 0 {
			cs.handleToolCalls(message.ToolCalls)
		} else {
			// No tool calls, conversation is complete
			cs.state.FinishProcessing()
			cs.state.ResetRecursion() // Reset when conversation completes
			cs.pushStateToUI()
		}
	} else {
		// Atomic update: Stop processing without response
		cs.state.FinishProcessing()
		cs.state.ResetRecursion() // Reset when conversation completes
		cs.pushStateToUI()
	}
}

func (cs *ChatService) pushStateToUI() {
	allMessages := cs.state.GetMessages()
	isProcessing := cs.state.IsProcessing()
	lastError := cs.state.GetLastError()

	// Only send new messages to reduce resource usage
	newMessages := allMessages[cs.lastSentCount:]
	cs.lastSentCount = len(allMessages)

	if err := cs.eventBus.SendToUI(eventbus.StateUpdateEvent{
		Messages:     newMessages, // Only new messages
		IsProcessing: isProcessing,
		Error:        lastError,
	}); err != nil {
		// If we can't send to UI, log the error and continue
		// In a production app, you might want to implement retry logic
		// or fallback mechanisms here
		fmt.Printf("Error sending state to UI: %v\n", err)
	}
}

func (cs *ChatService) IsReady() bool {
	return cs.config.IsValid() && cs.state.IsConversationReady()
}

// GetInitialMessages returns the initial messages for printing to terminal
func (cs *ChatService) GetInitialMessages() []models.Message {
	return cs.state.GetMessages()
}

func (cs *ChatService) addWelcomeMessages(cfg *config.Config) {
	// Welcome header
	cs.state.AddProgramMessage("-- RORICODE --")

	// Profile information with status
	if cfg.IsValid() {
		cs.state.AddProgramMessage(fmt.Sprintf("Active Profile: %s [OK]", cfg.ActiveProfile))
	} else {
		cs.state.AddProgramMessage(fmt.Sprintf("Active Profile: %s [NOT CONFIGURED]", cfg.ActiveProfile))
	}
	// Instructions
	if cfg.IsValid() {
		cs.state.AddProgramMessage("Ready to chat! Type your message and press Enter")
	} else {
		cs.state.AddProgramMessage("Configure your profile to start chatting:")
		cs.state.AddProgramMessage("• Run: roricode profile add <name>")
		cs.state.AddProgramMessage("• Or edit: ~/.roricode/config.json")
	}

	cs.state.AddProgramMessage("Controls: Ctrl+C or 'q' to exit")
	cs.state.AddProgramMessage("")
}

// getToolsSpec returns OpenAI tools specification from registry
func (cs *ChatService) getToolsSpec() []openai.Tool {
	toolSpecs := cs.toolRegistry.GetOpenAIToolsSpec()
	openaiTools := make([]openai.Tool, len(toolSpecs))
	
	for i, spec := range toolSpecs {
		openaiTools[i] = openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        spec["function"].(map[string]interface{})["name"].(string),
				Description: spec["function"].(map[string]interface{})["description"].(string),
				Parameters:  spec["function"].(map[string]interface{})["parameters"],
			},
		}
	}
	
	return openaiTools
}

// handleToolCalls processes tool calls from OpenAI and executes them
func (cs *ChatService) handleToolCalls(toolCalls []openai.ToolCall) {
	// Add all tool calls to pending tracker
	for _, call := range toolCalls {
		cs.state.AddPendingToolCall(call.ID)
	}
	
	for _, call := range toolCalls {
		// Tool calls are automatically displayed via GetMessages() conversion
		cs.pushStateToUI() // Show tool call immediately
		
		// Parse arguments
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			// Add error result and complete the tool call
			cs.state.AddToolResultMessage(call.ID, call.Function.Name, fmt.Sprintf("Error parsing arguments: %v", err))
			allComplete := cs.state.CompletePendingToolCall(call.ID)
			cs.pushStateToUI() // Show error result immediately
			if allComplete {
				cs.continueAfterAllToolsComplete()
			}
			continue
		}
		
		// Execute tool asynchronously
		toolCall := tools.ToolCall{
			ID:   call.ID,
			Name: call.Function.Name,
			Args: args,
		}
		
		resultChan := make(chan tools.ToolResult, 1)
		cs.toolRegistry.ExecuteAsync(cs.ctx, toolCall, resultChan)
		
		// Handle result asynchronously
		go cs.handleToolResult(resultChan)
	}
}

// handleToolResult processes tool execution results
func (cs *ChatService) handleToolResult(resultChan <-chan tools.ToolResult) {
	result := <-resultChan
	
	var resultContent string
	if result.Error != "" {
		resultContent = fmt.Sprintf("Error: %s", result.Error)
	} else {
		// Convert result to JSON string for display
		if resultBytes, err := json.MarshalIndent(result.Result, "", "  "); err == nil {
			resultContent = string(resultBytes)
		} else {
			resultContent = fmt.Sprintf("%v", result.Result)
		}
	}
	
	// Add tool result message
	cs.state.AddToolResultMessage(result.CallID, result.Name, resultContent)
	cs.pushStateToUI() // Show result immediately
	
	// Mark this tool call as complete and check if all are done
	allComplete := cs.state.CompletePendingToolCall(result.CallID)
	if allComplete {
		cs.continueAfterAllToolsComplete()
	}
}

// continueAfterAllToolsComplete continues the conversation after all tool calls are complete
func (cs *ChatService) continueAfterAllToolsComplete() {
	// All tool calls completed, continue the conversation recursively
	cs.continueConversation()
}

// generateConfirmationID generates a unique ID for confirmation requests
func (cs *ChatService) generateConfirmationID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// requestUserConfirmation sends a confirmation request to the UI and waits for response
func (cs *ChatService) requestUserConfirmation(operation, command string, dangerous bool) bool {
	// Generate unique ID for this confirmation
	id := cs.generateConfirmationID()
	
	// Create response channel
	responseChan := make(chan bool, 1)
	
	// Store the channel in pending confirmations
	cs.confirmMutex.Lock()
	cs.pendingConfirms[id] = responseChan
	cs.confirmMutex.Unlock()
	
	// Send confirmation request to UI
	request := eventbus.ConfirmationRequestEvent{
		ID:        id,
		Operation: operation,
		Command:   command,
		Dangerous: dangerous,
	}
	
	if err := cs.eventBus.SendToUI(request); err != nil {
		// Clean up and return false on error
		cs.confirmMutex.Lock()
		delete(cs.pendingConfirms, id)
		cs.confirmMutex.Unlock()
		return false
	}
	
	// Wait for user response
	select {
	case approved := <-responseChan:
		// Clean up
		cs.confirmMutex.Lock()
		delete(cs.pendingConfirms, id)
		cs.confirmMutex.Unlock()
		return approved
	case <-cs.ctx.Done():
		// Context cancelled, clean up
		cs.confirmMutex.Lock()
		delete(cs.pendingConfirms, id)
		cs.confirmMutex.Unlock()
		return false
	}
}

// handleConfirmationResponse handles confirmation responses from the UI
func (cs *ChatService) handleConfirmationResponse(response eventbus.ConfirmationResponseEvent) {
	cs.confirmMutex.RLock()
	responseChan, exists := cs.pendingConfirms[response.ID]
	cs.confirmMutex.RUnlock()
	
	if exists {
		// Send the response to the waiting goroutine
		select {
		case responseChan <- response.Approved:
		default:
			// Channel might be full or closed, ignore
		}
	}
}

// RequestConfirmation implements the Confirmator interface
func (cs *ChatService) RequestConfirmation(operation, command string, dangerous bool) bool {
	return cs.requestUserConfirmation(operation, command, dangerous)
}
