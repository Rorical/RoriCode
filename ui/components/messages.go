package components

import (
	"strings"

	"github.com/Rorical/RoriCode/internal/models"
	"github.com/Rorical/RoriCode/ui/styles"
)

func RenderMessages(messages []models.Message) string {
	var b strings.Builder

	systemStyle := styles.SystemStyle()
	userStyle := styles.UserStyle()
	assistantStyle := styles.AssistantStyle()
	programStyle := styles.ProgramStyle()

	for _, msg := range messages {
		switch msg.Type {
		case models.System:
			b.WriteString(systemStyle.Render(msg.Content) + "\n\n")
		case models.User:
			b.WriteString(userStyle.Render("You: "+msg.Content) + "\n\n")
		case models.Assistant:
			b.WriteString(assistantStyle.Render("Assistant: "+msg.Content) + "\n\n")
		case models.Program:
			b.WriteString(programStyle.Render(msg.Content) + "\n\n")
		}
	}

	return b.String()
}
