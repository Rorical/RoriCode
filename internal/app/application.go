package app

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Rorical/RoriCode/internal/config"
	"github.com/Rorical/RoriCode/internal/core"
	"github.com/Rorical/RoriCode/internal/dispatcher"
	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/models"
)

// Application manages the complete application lifecycle
type Application struct {
	config     *config.Config
	eventBus   *eventbus.EventBus
	dispatcher *dispatcher.EventDispatcher
	service    *core.ChatService
	model      *AppModel
}

type AppModel struct {
	appModel   models.AppModel
	dispatcher *dispatcher.EventDispatcher
}

func NewApplication() (*Application, error) {
	// Load configuration
	cfg := config.LoadConfig()

	// Create event bus
	eb := eventbus.NewEventBus()

	// Create dispatcher
	disp := dispatcher.NewEventDispatcher(eb)

	// Initialize chat service (always create, handles invalid config internally)
	chatService, err := core.NewChatService(cfg, eb)
	if err != nil {
		log.Printf("Failed to initialize chat service: %v", err)
		return nil, err
	}

	// Create app model
	model := &AppModel{
		appModel:   createInitialAppModel(chatService),
		dispatcher: disp,
	}

	return &Application{
		config:     cfg,
		eventBus:   eb,
		dispatcher: disp,
		service:    chatService,
		model:      model,
	}, nil
}

func (app *Application) Start() error {
	// Start background services
	app.dispatcher.Start()
	app.service.Start() // Always start since service is always created

	// Run UI
	p := tea.NewProgram(app.model)
	_, err := p.Run()

	return err
}

func (app *Application) Stop() {
	app.service.Stop()    // Always exists
	app.dispatcher.Stop() // Always exists
	app.eventBus.Close()  // Always exists
}

func createInitialAppModel(chatService *core.ChatService) models.AppModel {
	// No initial messages in UI - they come from core as single source of truth
	return models.AppModel{
		Messages:         make([]models.Message, 0), // Start empty, core will send messages
		Status:           "Ready",
		Loading:          false,
		ChatServiceReady: chatService.IsReady(),
	}
}
