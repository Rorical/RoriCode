package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileReplaceLinesTool replaces specific line ranges in files
type FileReplaceLinesTool struct {
	confirmator Confirmator
}

func (f *FileReplaceLinesTool) Name() string {
	return "replace_lines"
}

func (f *FileReplaceLinesTool) Description() string {
	return "Replace specific line ranges in existing files with new content"
}

func (f *FileReplaceLinesTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Relative path to the file to edit from current working directory",
		},
		"start_line": map[string]interface{}{
			"type":        "number",
			"description": "Starting line number to replace (1-based indexing)",
		},
		"end_line": map[string]interface{}{
			"type":        "number", 
			"description": "Ending line number to replace (1-based indexing, inclusive). If not provided, only replaces start_line",
		},
		"content": map[string]interface{}{
			"type":        "string",
			"description": "New content to replace the specified lines. Use \\n for line breaks",
		},
	}
}

func (f *FileReplaceLinesTool) RequiredParameters() []string {
	return []string{"path", "start_line", "content"}
}

func (f *FileReplaceLinesTool) SetConfirmator(confirmator Confirmator) {
	f.confirmator = confirmator
}

func (f *FileReplaceLinesTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	startLine, ok := args["start_line"].(float64)
	if !ok {
		return nil, fmt.Errorf("start_line parameter must be a number")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter must be a string")
	}

	// Handle optional end_line
	endLine := startLine
	if val, exists := args["end_line"]; exists {
		if line, ok := val.(float64); ok {
			endLine = line
		}
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

	// Validate line numbers
	if int(startLine) < 1 || int(endLine) < 1 {
		return nil, fmt.Errorf("line numbers must be 1-based (start with 1)")
	}
	if int(endLine) < int(startLine) {
		return nil, fmt.Errorf("end_line (%d) must be >= start_line (%d)", int(endLine), int(startLine))
	}

	// Ask for confirmation
	if f.confirmator != nil {
		linesCount := int(endLine) - int(startLine) + 1
		message := fmt.Sprintf("Replace %d line(s) in %s (lines %d-%d)", 
			linesCount, path, int(startLine), int(endLine))
		operation := "Replace lines"
		dangerous := true // File modification is potentially dangerous
		if !f.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Read file
	originalBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	originalLines := strings.Split(string(originalBytes), "\n")
	
	// Validate line range against file
	if int(startLine) > len(originalLines) {
		return nil, fmt.Errorf("start_line (%d) exceeds file length (%d lines)", int(startLine), len(originalLines))
	}
	if int(endLine) > len(originalLines) {
		return nil, fmt.Errorf("end_line (%d) exceeds file length (%d lines)", int(endLine), len(originalLines))
	}

	// Handle newlines in content
	content = strings.ReplaceAll(content, "\\n", "\n")
	newLines := strings.Split(content, "\n")

	// Build result
	var result []string
	
	// Copy lines before replacement range
	result = append(result, originalLines[:int(startLine)-1]...)
	
	// Add new content
	result = append(result, newLines...)
	
	// Copy lines after replacement range
	result = append(result, originalLines[int(endLine):]...)

	// Write file
	newContent := strings.Join(result, "\n")
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	linesReplaced := int(endLine) - int(startLine) + 1
	return fmt.Sprintf("Successfully replaced %d line(s) in %s (lines %d-%d) with %d new line(s)", 
		linesReplaced, path, int(startLine), int(endLine), len(newLines)), nil
}