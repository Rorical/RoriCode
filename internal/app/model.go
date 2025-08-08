package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// Terminal control sequences
func clearLine() {
	fmt.Print("\033[2K") // Clear entire line
}

func moveCursorUp(lines int) {
	fmt.Printf("\033[%dA", lines) // Move cursor up N lines
}

// Style functions
func statusStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(width)
}

func systemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("555")).
		Padding(0, 2)
}

func userStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("343")).
		Padding(0, 2)
}

func assistantStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		BorderForeground(lipgloss.Color("214")).
		Padding(0, 1)
}

func programStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Bold(true).
		Padding(0, 2).
		Align(lipgloss.Center)
}

// Start the simple fmt-based UI loop
func (m *AppModel) Start() {
	// Initialize basic state
	m.appModel.Status = "Ready"

	// Start listening for core events in background
	go m.listenForCoreEvents()

	// Start input loop
	m.inputLoop()
}

// listenForCoreEvents handles events from core and prints messages
func (m *AppModel) listenForCoreEvents() {
	eventBus := m.dispatcher.GetEventBus()

	for {
		select {
		case coreEvent, ok := <-eventBus.CoreToUI():
			if !ok {
				return
			}
			m.handleCoreEvent(coreEvent)
		}
	}
}

// handleCoreEvent processes events from core and prints new messages
func (m *AppModel) handleCoreEvent(coreEvent eventbus.CoreEvent) {
	if stateEvent, ok := coreEvent.(eventbus.StateUpdateEvent); ok {
		// Check if we have new messages to print
		currentCount := len(m.appModel.Messages)
		newMessages := stateEvent.Messages[currentCount:]

		m.clearPreviousStatus()
		// Print each new message
		for _, msg := range newMessages {
			m.printMessageToScrollArea(msg)
		}

		// Update local state
		m.appModel.Messages = stateEvent.Messages
		m.appModel.Loading = stateEvent.IsProcessing

		// Update status
		if stateEvent.Error != nil {
			m.appModel.Status = "Error: " + stateEvent.Error.Error()
			// Show error status
			m.printStatusBar()
		} else if stateEvent.IsProcessing {
			m.appModel.Status = "Processing..."
			m.printStatusBar()
		} else {
			// Don't show "Ready" status - just clear any previous status
			m.appModel.Status = "Ready"
		}

		// Print new prompt after message (only if not processing)
		if !stateEvent.IsProcessing {
			fmt.Print("> ")
		}
	}
}

// printMessageToScrollArea prints a message in the scrollable area
func (m *AppModel) printMessageToScrollArea(msg models.Message) {
	switch msg.Type {
	case models.System:
		fmt.Println(systemStyle().Render(msg.Content))
	case models.User:
		fmt.Println(userStyle().Render("> " + msg.Content))
	case models.Assistant:
		fmt.Println(assistantStyle().Render(">> " + msg.Content))
	case models.Program:
		fmt.Println(programStyle().Render(msg.Content))
	}
}

// printStatusBar prints the current status
func (m *AppModel) printStatusBar() {
	fmt.Println(statusStyle(80).Render(m.appModel.Status))
}

// clearPreviousStatus clears the previous status line
func (m *AppModel) clearPreviousStatus() {
	// Move up one line and clear it (where the status was)
	moveCursorUp(1)
	clearLine()
}

// inputLoop handles user input with simple console interface
func (m *AppModel) inputLoop() {
	scanner := bufio.NewScanner(os.Stdin)

	// Print initial prompt
	fmt.Print("> ")

	for {
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Print("> ")
			continue
		}

		// Handle quit commands
		if input == "q" || input == "quit" || input == "exit" {
			break
		}

		// Keep the input line as is - don't clear or reformat it

		// Send message to core if chat service is ready
		if m.appModel.ChatServiceReady {
			eventBus := m.dispatcher.GetEventBus()
			if err := eventBus.SendToCore(eventbus.SendMessageEvent{Message: input}); err != nil {
				m.clearPreviousStatus()
				fmt.Printf("Error sending message: %s\n", err.Error())
				fmt.Print("> ")
			}
			// Don't print prompt here - it will be printed when response comes back
		} else {
			fmt.Println("Chat service not available")
			fmt.Print("> ")
		}
	}
}
