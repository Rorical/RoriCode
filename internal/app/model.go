package app

import (
	"bufio"
	"encoding/json"
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

	for coreEvent := range eventBus.CoreToUI() {
		m.handleCoreEvent(coreEvent)
	}
}

// handleCoreEvent processes events from core and prints new messages
func (m *AppModel) handleCoreEvent(coreEvent eventbus.CoreEvent) {
	// Handle confirmation requests
	if confirmationEvent, ok := coreEvent.(eventbus.ConfirmationRequestEvent); ok {
		m.handleConfirmationRequest(confirmationEvent)
		return
	}

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
		fmt.Print(utils.AssistantStyle().Render(">> "+indentedContent) + "\n")
	case models.Program:
		fmt.Println(utils.ProgramStyle().Render(msg.Content))
	case models.ToolCall:
		// Format tool call with name and arguments
		toolCallContent := fmt.Sprintf("「%s(%s)」", msg.ToolName, msg.ToolArgs)
		fmt.Println(utils.ToolCallStyle().Render(toolCallContent))
	case models.ToolResult:
		// Format tool result with user-friendly summary instead of raw JSON
		formattedResult := m.formatToolResult(msg.ToolName, msg.Content)
		toolResultContent := fmt.Sprintf("·%s → %s", msg.ToolName, formattedResult)
		fmt.Println(utils.ToolResultStyle().Render(toolResultContent))
	}
}

// formatToolResult creates user-friendly summaries for tool results
func (m *AppModel) formatToolResult(toolName, content string) string {
	// Try to parse as JSON first
	var result map[string]any
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If not JSON, return content as-is but truncated if too long
		if len(content) > 200 {
			return content[:200] + "..."
		}
		return content
	}

	switch toolName {
	case "read_file":
		return m.formatFileReadResult(result)
	case "current_time":
		// For simple string results, return as-is
		return strings.Trim(content, "\"") // Remove JSON quotes if present
	case "shell":
		return m.formatShellResult(result)
	default:
		// For unknown tools, provide a generic summary
		return m.formatGenericResult(result)
	}
}

// formatFileReadResult creates a summary for file reading operations
func (m *AppModel) formatFileReadResult(result map[string]any) string {
	fileType, _ := result["type"].(string)
	path, _ := result["path"].(string)

	if fileType == "directory" {
		totalItems, _ := result["total_items"].(float64)
		return fmt.Sprintf("Listed %s directory with %.0f items", path, totalItems)
	}

	// File reading result
	linesRead, _ := result["lines_read"].(float64)
	matchesFound, hasMatches := result["matches_found"].(float64)

	if hasMatches {
		if matchesFound == 0 {
			return fmt.Sprintf("No matches found in %s", path)
		}
		matchLine, _ := result["match_line"].(float64)
		return fmt.Sprintf("Found %.0f matches in %s (showing match at line %.0f)", matchesFound, path, matchLine)
	}

	moreLines, _ := result["more_lines"].(bool)
	if moreLines {
		return fmt.Sprintf("Read first %.0f lines of %s (more content available)", linesRead, path)
	}
	return fmt.Sprintf("Read %.0f lines from %s", linesRead, path)
}

// formatShellResult creates a summary for shell command execution
func (m *AppModel) formatShellResult(result map[string]any) string {
	output, _ := result["output"].(string)
	exitCode, _ := result["exit_code"].(float64)
	errorMsg, hasError := result["error"].(string)

	if hasError {
		return fmt.Sprintf("Command failed (exit %d): %s", int(exitCode), errorMsg)
	}

	if exitCode != 0 {
		return fmt.Sprintf("Command completed with exit code %d", int(exitCode))
	}

	// Truncate long output
	if len(output) > 150 {
		lines := strings.Split(output, "\n")
		if len(lines) > 3 {
			return fmt.Sprintf("Command successful (%d lines of output)", len(lines))
		}
		return fmt.Sprintf("Command successful: %s...", output[:100])
	}

	if output == "" {
		return "Command completed successfully"
	}
	return fmt.Sprintf("Command successful: %s", strings.TrimSpace(output))
}

// formatGenericResult provides a fallback summary for unknown tool results
func (m *AppModel) formatGenericResult(result map[string]any) string {
	// Look for common fields that might indicate success or provide summary info
	if message, ok := result["message"].(string); ok {
		return message
	}

	if summary, ok := result["summary"].(string); ok {
		return summary
	}

	if content, ok := result["content"].(string); ok {
		if len(content) > 100 {
			return content[:100] + "..."
		}
		return content
	}

	// Count the keys to give some indication of result size
	return fmt.Sprintf("Result with %d fields", len(result))
}

// handleConfirmationRequest handles confirmation requests from core
func (m *AppModel) handleConfirmationRequest(request eventbus.ConfirmationRequestEvent) {
	// Convert to local type to avoid import cycle
	m.appModel.PendingConfirmation = &models.ConfirmationRequest{
		ID:        request.ID,
		Operation: request.Operation,
		Command:   request.Command,
		Dangerous: request.Dangerous,
	}

	// Clear any existing status
	if m.statusShown {
		m.clearPreviousStatus()
		m.statusShown = false
	}

	// Show the confirmation prompt using the local copy
	fmt.Printf("\n%s\n", utils.ProgramStyle().Render("CONFIRMATION REQUIRED"))
	fmt.Printf("Operation: %s\n", utils.BoldStyle().Render(m.appModel.PendingConfirmation.Operation))
	if m.appModel.PendingConfirmation.Command != "" {
		fmt.Printf("Content: %s\n", utils.CodeBlockStyle().Render(m.appModel.PendingConfirmation.Command))
	}
	if m.appModel.PendingConfirmation.Dangerous {
		fmt.Printf("%s\n", utils.DangerStyle().Render("This operation may be potentially dangerous"))
	}
	fmt.Print("Do you still want to proceed? (y/N): ")
}

// handleConfirmationInput processes user input when a confirmation is pending
func (m *AppModel) handleConfirmationInput(input string) {
	if m.appModel.PendingConfirmation == nil {
		return
	}

	// Clear the input line
	moveCursorUp(1)
	clearLine()

	input = strings.ToLower(strings.TrimSpace(input))
	approved := input == "y" || input == "yes"

	// Send response back to core
	eventBus := m.dispatcher.GetEventBus()
	response := eventbus.ConfirmationResponseEvent{
		ID:       m.appModel.PendingConfirmation.ID,
		Approved: approved,
	}

	if err := eventBus.SendToCore(response); err != nil {
		fmt.Printf("Error sending confirmation response: %s\n", err.Error())
	}

	// Show user's decision
	if approved {
		fmt.Printf("%s\n", utils.ListStyle().Render("✓ Approved - proceeding with operation"))
	} else {
		fmt.Printf("%s\n", utils.ListStyle().Render("✗ Denied - operation aborted"))
	}

	// Clear the pending confirmation
	m.appModel.PendingConfirmation = nil

	// Print new prompt
	fmt.Print("> ")
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

		// Check if we're waiting for a confirmation response
		if m.appModel.PendingConfirmation != nil {
			m.handleConfirmationInput(input)
			continue
		}

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
