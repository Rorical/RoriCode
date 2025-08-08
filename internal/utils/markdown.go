package utils

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Markdown styles
func CodeBlockStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		MarginLeft(2)
}

func BoldStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true)
}

func ItalicStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Italic(true)
}

func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true)
}

func SubtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true)
}

func LinkStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Underline(true)
}

func ListStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		MarginLeft(2)
}

// RenderMarkdown applies basic markdown rendering to text
func RenderMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder
	
	inCodeBlock := false
	
	for _, line := range lines {
		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				result.WriteString(CodeBlockStyle().Render("┌─ Code Block ─┐") + "\n")
			} else {
				result.WriteString(CodeBlockStyle().Render("└──────────────┘") + "\n")
			}
			continue
		}

		if inCodeBlock {
			result.WriteString(CodeBlockStyle().Render(line) + "\n")
			continue
		}

		// Handle titles (# ## ###) - remove marks for cleaner visual
		if title, found := strings.CutPrefix(line, "### "); found {
			// Recursively parse inline markdown within headings
			parsedTitle := processInlineMarkdown(title)
			result.WriteString(SubtitleStyle().Render(parsedTitle) + "\n")
			continue
		} else if title, found := strings.CutPrefix(line, "## "); found {
			// Recursively parse inline markdown within headings
			parsedTitle := processInlineMarkdown(title)
			result.WriteString(TitleStyle().Render(parsedTitle) + "\n")
			continue
		} else if title, found := strings.CutPrefix(line, "# "); found {
			// Recursively parse inline markdown within headings
			parsedTitle := processInlineMarkdown(title)
			result.WriteString(TitleStyle().Render(parsedTitle) + "\n")
			continue
		}

		// Handle unordered lists (- or *)
		if listItem, found := strings.CutPrefix(line, "- "); found {
			// Recursively parse inline markdown within list items
			parsedItem := processInlineMarkdown(listItem)
			result.WriteString(ListStyle().Render("• " + parsedItem) + "\n")
			continue
		} else if listItem, found := strings.CutPrefix(line, "* "); found {
			// Recursively parse inline markdown within list items
			parsedItem := processInlineMarkdown(listItem)
			result.WriteString(ListStyle().Render("• " + parsedItem) + "\n")
			continue
		}

		// Handle ordered lists (1. 2. etc.)
		orderedListRegex := regexp.MustCompile(`^(\d+)\.\s+(.*)`)
		if matches := orderedListRegex.FindStringSubmatch(line); len(matches) == 3 {
			// Recursively parse inline markdown within ordered list items
			parsedItem := processInlineMarkdown(matches[2])
			result.WriteString(ListStyle().Render(matches[1] + ". " + parsedItem) + "\n")
			continue
		}

		line = processInlineMarkdown(line)
		result.WriteString(line + "\n")
	}
	
	return strings.TrimSuffix(result.String(), "\n")
}

// processInlineMarkdown handles inline markdown elements recursively
func processInlineMarkdown(line string) string {
	// Process in order of precedence: code first (to avoid conflicts), then links, then formatting
	
	// Handle inline code first (outer to inner - match longest first)
	// This prevents code content from being processed as other markdown
	codeRegex := regexp.MustCompile("```([^`]|`[^`]|``[^`])*```|``[^`]*``|`[^`]*`")
	line = codeRegex.ReplaceAllStringFunc(line, func(match string) string {
		code := strings.Trim(match, "`")
		return CodeBlockStyle().Render(code)
	})

	// Handle links [text](url) - recursively process link text
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	line = linkRegex.ReplaceAllStringFunc(line, func(match string) string {
		matches := linkRegex.FindStringSubmatch(match)
		if len(matches) == 3 {
			// Recursively process the link text for nested formatting
			linkText := processNestedFormatting(matches[1])
			return LinkStyle().Render(linkText)
		}
		return match
	})

	// Process remaining formatting (bold, italic)
	line = processNestedFormatting(line)
	
	return line
}

// processNestedFormatting handles bold and italic formatting recursively
func processNestedFormatting(text string) string {
	// Handle bold text first (outer to inner) - remove ** marks
	boldRegex := regexp.MustCompile(`\*\*([^*]|\*[^*])*\*\*`)
	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := strings.Trim(match, "*")
		// Recursively process content inside bold for nested italic
		processedContent := processItalicText(content)
		return BoldStyle().Render(processedContent)
	})

	// Handle italic text with underscores and single asterisks - remove marks
	text = processItalicText(text)
	
	return text
}

// processItalicText handles italic text formatting
func processItalicText(line string) string {
	// First handle underscores for italic
	italicUnderscoreRegex := regexp.MustCompile(`_([^_]+)_`)
	line = italicUnderscoreRegex.ReplaceAllStringFunc(line, func(match string) string {
		text := strings.Trim(match, "_")
		return ItalicStyle().Render(text)
	})
	
	// Then handle single asterisks that aren't part of bold (avoid conflicts)
	italicAsteriskRegex := regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*(?:[^*]|$)`)
	line = italicAsteriskRegex.ReplaceAllStringFunc(line, func(match string) string {
		// Extract just the italic content between single asterisks
		parts := regexp.MustCompile(`\*([^*]+)\*`).FindStringSubmatch(match)
		if len(parts) == 2 {
			before := ""
			after := ""
			if len(match) > len(parts[0]) {
				if match[0] != '*' {
					before = string(match[0])
				}
				if match[len(match)-1] != '*' {
					after = string(match[len(match)-1])
				}
			}
			return before + ItalicStyle().Render(parts[1]) + after
		}
		return match
	})
	
	return line
}