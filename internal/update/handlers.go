package update

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/models"
)

// HandleKeyMsgWithEventBus handles keyboard input using event bus
func HandleKeyMsgWithEventBus(appModel *models.AppModel, keyMsg tea.KeyMsg, eb *eventbus.EventBus, chatReady bool) tea.Cmd {
	switch keyMsg.String() {
	case "ctrl+c", "q":
		return tea.Quit
	case "enter":
		if strings.TrimSpace(appModel.Input) != "" && chatReady {
			// Send event to core via event bus with error handling
			if err := eb.SendToCore(eventbus.SendMessageEvent{Message: appModel.Input}); err != nil {
				appModel.Status = "Error sending message: " + err.Error()
				return nil
			}

			// Only manage local UI state - clear input
			appModel.Input = ""
			return nil
		} else if strings.TrimSpace(appModel.Input) != "" {
			// Fallback when chat service is not ready
			appModel.Input = ""
			appModel.Status = "Chat service not available"
		}
	case "backspace":
		if len(appModel.Input) > 0 {
			appModel.Input = appModel.Input[:len(appModel.Input)-1]
		}
	default:
		if len(keyMsg.String()) == 1 {
			appModel.Input += keyMsg.String()
		}
	}
	return nil
}

// CoreEventMsg wraps core events for Bubble Tea
type CoreEventMsg struct {
	Event eventbus.CoreEvent
}

// HandleCoreEvent processes events from the core
func HandleCoreEvent(appModel *models.AppModel, coreEventMsg CoreEventMsg) tea.Cmd {
	switch event := coreEventMsg.Event.(type) {
	case eventbus.StateUpdateEvent:
		// Update UI state from core state
		appModel.Messages = event.Messages
		appModel.Loading = event.IsProcessing

		// Update status based on core state
		if event.Error != nil {
			appModel.Status = "Error: " + event.Error.Error()
		} else if event.IsProcessing {
			appModel.Status = "Processing"
		} else {
			appModel.Status = "Ready"
		}
	}

	return nil
}

type TickMsg time.Time

func TickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func HandleWindowSizeMsg(appModel *models.AppModel, sizeMsg tea.WindowSizeMsg) {
	appModel.Width = sizeMsg.Width
	appModel.Height = sizeMsg.Height
}

func HandleTickMsg(appModel *models.AppModel) tea.Cmd {
	// Only handle UI animations - loading dots
	if appModel.Loading {
		appModel.LoadingDots = (appModel.LoadingDots + 1) % 4
	}
	return TickCmd()
}
