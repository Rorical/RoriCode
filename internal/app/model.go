package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Rorical/RoriCode/internal/eventbus"
	"github.com/Rorical/RoriCode/internal/models"
	"github.com/Rorical/RoriCode/internal/utils"
)

// Terminal control sequences
func clearLine() {
	fmt.Print("\033[2K") // Clear entire line
}

func moveCursorUp(lines int) {
	fmt.Printf("\033[%dA", lines) // Move cursor up N lines
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
		// Core now only sends new messages, so we can print them all
		newMessages := stateEvent.Messages

		// Print each new message immediately with smart status handling
		for _, msg := range newMessages {
			m.printMessageWithStatusHandling(msg)
		}

		// Update local state - append new messages to existing ones
		m.appModel.Messages = append(m.appModel.Messages, newMessages...)
		m.appModel.Loading = stateEvent.IsProcessing

		// Update status
		if stateEvent.Error != nil {
			m.appModel.Status = "Error: " + stateEvent.Error.Error()
			// Show error status
			m.printStatusBar()
			m.statusShown = true
		} else if stateEvent.IsProcessing {
			if !m.statusShown {
				m.appModel.Status = "Processing..."
				m.printStatusBar()
				m.statusShown = true
			}
		} else {
			// Don't show "Ready" status - just clear any previous status if shown
			if m.statusShown {
				m.clearPreviousStatus()
				m.statusShown = false
			}
			m.appModel.Status = "Ready"
		}

		// Print new prompt after message (only if not processing)
		if !stateEvent.IsProcessing {
			fmt.Print("> ")
		}
	}
}

// printMessageWithStatusHandling handles status clearing/restoring for different message types
func (m *AppModel) printMessageWithStatusHandling(msg models.Message) {
	// For tool calls and tool results, clear status first, print message, then restore status
	if msg.Type == models.ToolCall || msg.Type == models.ToolResult {
		// Clear previous status if shown
		var wasStatusShown bool
		var previousStatus string
		if m.statusShown {
			wasStatusShown = true
			previousStatus = m.appModel.Status
			m.clearPreviousStatus()
			m.statusShown = false
		}

		// Print the tool message
		m.printMessageToScrollArea(msg)

		// Restore status if it was previously shown
		if wasStatusShown {
			m.appModel.Status = previousStatus
			m.printStatusBar()
			m.statusShown = true
		}
	} else {
		// For other message types (User, Assistant, Program), clear status normally
		if m.statusShown {
			m.clearPreviousStatus()
			m.statusShown = false
		}
		m.printMessageToScrollArea(msg)
	}
}

// printMessageToScrollArea prints a message in the scrollable area
func (m *AppModel) printMessageToScrollArea(msg models.Message) {
	switch msg.Type {
	case models.User:
		fmt.Println(utils.UserStyle().Render("> " + msg.Content))
	case models.Assistant:
		// Render markdown for assistant messages
		renderedContent := utils.RenderMarkdown(msg.Content)
		// Add two-space indentation to all lines except the first
		lines := strings.Split(renderedContent, "\n")
		for i := 1; i < len(lines); i++ {
			lines[i] = "   " + lines[i]
		}
		indentedContent := strings.Join(lines, "\n")
		fmt.Println(utils.AssistantStyle().Render(">> " + indentedContent))
	case models.Program:
		fmt.Println(utils.ProgramStyle().Render(msg.Content))
	case models.ToolCall:
		// Format tool call with name and arguments
		toolCallContent := fmt.Sprintf("「%s(%s)」", msg.ToolName, msg.ToolArgs)
		fmt.Println(utils.ToolCallStyle().Render(toolCallContent))
	case models.ToolResult:
		// Format tool result with name and response
		lines := strings.Split(msg.Content, "\n")
		for i := 1; i < len(lines); i++ {
			lines[i] = "  " + lines[i]
		}
		indentedContent := strings.Join(lines, "\n")
		toolResultContent := fmt.Sprintf("·%s → %s", msg.ToolName, indentedContent)
		fmt.Println(utils.ToolResultStyle().Render(toolResultContent))
	}
}

// printStatusBar prints the current status
func (m *AppModel) printStatusBar() {
	fmt.Println(utils.StatusStyle(80).Render(m.appModel.Status))
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

		// Clear the user input line after enter
		moveCursorUp(1)
		clearLine()

		// Send message to core if chat service is ready
		if m.appModel.ChatServiceReady {
			eventBus := m.dispatcher.GetEventBus()
			if err := eventBus.SendToCore(eventbus.SendMessageEvent{Message: input}); err != nil {
				if m.statusShown {
					m.clearPreviousStatus()
					m.statusShown = false
				}
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
