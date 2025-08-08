package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Rorical/RoriCode/internal/update"
	"github.com/Rorical/RoriCode/ui/components"
)

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		update.TickCmd(),
		m.dispatcher.ListenForUIEvents(),
	)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle core events and continue listening
	if coreEvent, ok := msg.(update.CoreEventMsg); ok {
		cmd := update.HandleCoreEvent(&m.appModel, coreEvent)
		return m, tea.Batch(cmd, m.dispatcher.ListenForUIEvents())
	}
	
	// Handle other events through the event bus
	eventBus := m.dispatcher.GetEventBus()
	chatReady := m.appModel.ChatServiceReady
	cmd := update.HandleUpdateWithEventBus(&m.appModel, msg, eventBus, chatReady)
	
	return m, cmd
}

func (m *AppModel) View() string {
	var b strings.Builder

	b.WriteString(components.RenderMessages(m.appModel.Messages))
	b.WriteString(components.RenderInput(m.appModel.Input, m.appModel.Loading, m.appModel.LoadingDots, m.appModel.Width))
	b.WriteString("\n")
	b.WriteString(components.RenderStatus(m.appModel.Status, m.appModel.Loading, m.appModel.LoadingDots, m.appModel.Width))

	return b.String()
}