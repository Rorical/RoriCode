package core

import (
	"context"
	"fmt"

	"github.com/Rorical/RoriCode/internal/config"
	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/models"
	"github.com/sashabaranov/go-openai"
)

type ChatService struct {
	client   *openai.Client
	config   *config.Config
	state    *ChatState
	eventBus *eventbus.EventBus
	ctx      context.Context
	cancel   context.CancelFunc
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

	service := &ChatService{
		client:   client, // May be nil if config invalid
		config:   cfg,
		state:    state,
		eventBus: eb,
		ctx:      ctx,
		cancel:   cancel,
	}

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
	}
}

func (cs *ChatService) processMessage(userMessage string) {
	// Atomic update: Set processing and add user message
	cs.state.StartProcessingWithUserMessage(userMessage)
	cs.pushStateToUI()

	// If no OpenAI client, just finish processing
	if cs.client == nil {
		cs.state.FinishProcessingWithError(fmt.Errorf("OpenAI integration not available"))
		cs.pushStateToUI()
		return
	}

	// Get OpenAI conversation history
	openaiMessages := cs.state.GetOpenAIHistory()

	// Call OpenAI API
	resp, err := cs.client.CreateChatCompletion(
		cs.ctx, // Use service context for cancellation
		openai.ChatCompletionRequest{
			Model:    cs.config.GetModel(),
			Messages: openaiMessages,
		},
	)

	if err != nil {
		// Atomic update: Stop processing with error
		cs.state.FinishProcessingWithError(fmt.Errorf("OpenAI API error: %w", err))
		cs.pushStateToUI()
		return
	}

	if len(resp.Choices) > 0 {
		// Atomic update: Stop processing with assistant message
		cs.state.FinishProcessingWithAssistantMessage(resp.Choices[0].Message.Content)
		cs.pushStateToUI()
	} else {
		// Atomic update: Stop processing without response
		cs.state.FinishProcessing()
		cs.pushStateToUI()
	}
}

func (cs *ChatService) pushStateToUI() {
	messages := cs.state.GetMessages()
	isProcessing := cs.state.IsProcessing()
	lastError := cs.state.GetLastError()

	if err := cs.eventBus.SendToUI(eventbus.StateUpdateEvent{
		Messages:     messages,
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
		cs.state.AddSystemMessage(fmt.Sprintf("Active Profile: %s [OK]", cfg.ActiveProfile))
	} else {
		cs.state.AddSystemMessage(fmt.Sprintf("Active Profile: %s [NOT CONFIGURED]", cfg.ActiveProfile))
	}
	// Instructions
	if cfg.IsValid() {
		cs.state.AddProgramMessage("Ready to chat! Type your message and press Enter")
	} else {
		cs.state.AddSystemMessage("Configure your profile to start chatting:")
		cs.state.AddSystemMessage("• Run: roricode profile add <name>")
		cs.state.AddSystemMessage("• Or edit: ~/.roricode/config.json")
	}

	cs.state.AddSystemMessage("Controls: Ctrl+C or 'q' to exit")
	cs.state.AddProgramMessage("")
}
