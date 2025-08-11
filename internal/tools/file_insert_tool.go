package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInsertTool inserts content at specific positions
type FileInsertTool struct {
	confirmator Confirmator
}

func (f *FileInsertTool) Name() string {
	return "insert_content"
}

func (f *FileInsertTool) Description() string {
	return "Insert content at specific positions in files (beginning, end, or after a specific line)"
}

func (f *FileInsertTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Relative path to the file to edit from current working directory",
		},
		"content": map[string]interface{}{
			"type":        "string",
			"description": "Content to insert. Use \\n for line breaks",
		},
		"position": map[string]interface{}{
			"type":        "string",
			"description": "Where to insert: 'beginning', 'end', or 'after_line'",
			"enum":        []string{"beginning", "end", "after_line"},
		},
		"line_number": map[string]interface{}{
			"type":        "number",
			"description": "Line number to insert after (1-based, required when position='after_line')",
		},
	}
}

func (f *FileInsertTool) RequiredParameters() []string {
	return []string{"path", "content", "position"}
}

func (f *FileInsertTool) SetConfirmator(confirmator Confirmator) {
	f.confirmator = confirmator
}

func (f *FileInsertTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter must be a string")
	}

	position, ok := args["position"].(string)
	if !ok {
		return nil, fmt.Errorf("position parameter must be a string")
	}

	var lineNumber float64
	if position == "after_line" {
		if val, exists := args["line_number"]; exists {
			if num, ok := val.(float64); ok {
				lineNumber = num
			} else {
				return nil, fmt.Errorf("line_number must be a number when position='after_line'")
			}
		} else {
			return nil, fmt.Errorf("line_number is required when position='after_line'")
		}
	}

	// Validate position
	validPositions := map[string]bool{"beginning": true, "end": true, "after_line": true}
	if !validPositions[position] {
		return nil, fmt.Errorf("position must be one of: beginning, end, after_line")
	}

	// Validate path safety
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be relative, not absolute")
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path cannot contain parent directory references (..)")
	}

	fullPath := path
	if !filepath.IsAbs(fullPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %v", err)
		}
		fullPath = filepath.Join(cwd, path)
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	// Read file
	originalBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	originalLines := strings.Split(string(originalBytes), "\n")

	// Validate line number if needed
	if position == "after_line" {
		if int(lineNumber) < 1 {
			return nil, fmt.Errorf("line_number must be 1-based (start with 1)")
		}
		if int(lineNumber) > len(originalLines) {
			return nil, fmt.Errorf("line_number (%d) exceeds file length (%d lines)", int(lineNumber), len(originalLines))
		}
	}

	// Ask for confirmation
	if f.confirmator != nil {
		var message string
		switch position {
		case "beginning":
			message = fmt.Sprintf("Insert content at beginning of %s", path)
		case "end":
			message = fmt.Sprintf("Insert content at end of %s", path)
		case "after_line":
			message = fmt.Sprintf("Insert content after line %d in %s", int(lineNumber), path)
		}
		operation := "Insert content"
		dangerous := true // File modification is potentially dangerous
		if !f.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Handle newlines in content
	content = strings.ReplaceAll(content, "\\n", "\n")
	newLines := strings.Split(content, "\n")

	// Build result based on position
	var result []string

	switch position {
	case "beginning":
		result = append(result, newLines...)
		result = append(result, originalLines...)
	case "end":
		result = append(result, originalLines...)
		result = append(result, newLines...)
	case "after_line":
		insertIdx := int(lineNumber)
		result = append(result, originalLines[:insertIdx]...)
		result = append(result, newLines...)
		result = append(result, originalLines[insertIdx:]...)
	}

	// Write file
	newContent := strings.Join(result, "\n")
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	var positionDesc string
	switch position {
	case "beginning":
		positionDesc = "at beginning"
	case "end":
		positionDesc = "at end"
	case "after_line":
		positionDesc = fmt.Sprintf("after line %d", int(lineNumber))
	}

	return fmt.Sprintf("Successfully inserted %d line(s) %s of %s", len(newLines), positionDesc, path), nil
}