package update

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/models"
)

func HandleUpdateWithEventBus(appModel *models.AppModel, msg tea.Msg, eb *eventbus.EventBus, chatReady bool) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return HandleKeyMsgWithEventBus(appModel, msg, eb, chatReady)
	case tea.WindowSizeMsg:
		HandleWindowSizeMsg(appModel, msg)
		return nil
	case TickMsg:
		return HandleTickMsg(appModel)
	case CoreEventMsg:
		return HandleCoreEvent(appModel, msg)
	}
	return nil
}