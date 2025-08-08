package styles

import "github.com/charmbracelet/lipgloss"

func InputStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(width - 4)
}

func StatusStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(width)
}

func SystemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 2)
}

func UserStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("39")).
		Padding(0, 1).
		MarginLeft(2)
}

func AssistantStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("214")).
		Padding(0, 1).
		MarginLeft(2)
}

func ProgramStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Bold(true).
		Padding(0, 2).
		Align(lipgloss.Center)
}