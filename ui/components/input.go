package components

import (
	"github.com/Rorical/RoriCode/ui/styles"
)

func RenderInput(input string, loading bool, loadingDots int, width int) string {
	inputStyle := styles.InputStyle(width)
	return inputStyle.Render(input)
}
