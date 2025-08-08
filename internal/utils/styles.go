package utils

import "github.com/charmbracelet/lipgloss"

// UI Styles
func StatusStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(width)
}

func UserStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("343")).
		Padding(0, 2)
}

func AssistantStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		BorderForeground(lipgloss.Color("214")).
		Padding(0, 1)
}

func ProgramStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Bold(true).
		Padding(0, 2).
		Align(lipgloss.Center)
}

func ToolCallStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("166")).
		BorderForeground(lipgloss.Color("166")).
		Padding(0, 1).
		MarginLeft(2)
}

func ToolResultStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("72")).
		BorderForeground(lipgloss.Color("72")).
		Padding(0, 1).
		MarginLeft(2)
}
