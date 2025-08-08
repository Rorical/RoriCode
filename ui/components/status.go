package components

import (
	"strings"

	"github.com/Rorical/RoriCode/ui/styles"
)

func RenderStatus(status string, loading bool, loadingDots int, width int) string {
	statusStyle := styles.StatusStyle(width)
	
	statusContent := status
	if loading {
		statusContent += strings.Repeat(".", loadingDots)
	}
	
	return statusStyle.Render(statusContent)
}