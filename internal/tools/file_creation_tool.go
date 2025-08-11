package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileCreationTool creates new files with specified content
type FileCreationTool struct {
	confirmator Confirmator
}

func (f *FileCreationTool) Name() string {
	return "create_file"
}

func (f *FileCreationTool) Description() string {
	return "Create new files with specified content. Can optionally overwrite existing files."
}

func (f *FileCreationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Relative path to the file to create from current working directory",
		},
		"content": map[string]interface{}{
			"type":        "string",
			"description": "Content to write to the file",
		},
		"overwrite": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether to overwrite existing file (default: false)",
		},
	}
}

func (f *FileCreationTool) RequiredParameters() []string {
	return []string{"path", "content"}
}

func (f *FileCreationTool) SetConfirmator(confirmator Confirmator) {
	f.confirmator = confirmator
}

func (f *FileCreationTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter must be a string")
	}

	overwrite := false
	if val, exists := args["overwrite"]; exists {
		if b, ok := val.(bool); ok {
			overwrite = b
		}
	}

	// Validate path safety
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be relative, not absolute")
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path cannot contain parent directory references (..)")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %v", err)
	}

	fullPath := filepath.Join(cwd, path)

	// Check if file exists
	if _, err := os.Stat(fullPath); err == nil {
		if !overwrite {
			return nil, fmt.Errorf("file already exists: %s (use overwrite: true to replace)", path)
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	// Request confirmation for file creation
	if f.confirmator != nil {
		operation := "Create file"
		if overwrite {
			operation = "Create/Overwrite file"
		}
		dangerous := overwrite // Overwriting is considered more dangerous
		if !f.confirmator.RequestConfirmation(operation, fmt.Sprintf("Create %s", path), dangerous) {
			return map[string]interface{}{
				"output":  "User aborted file creation operation",
				"aborted": true,
			}, nil
		}
	}

	// Write the file
	err = os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	// Count lines in content
	lines := strings.Split(content, "\n")
	lineCount := len(lines)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lineCount-- // Don't count final empty line
	}

	return map[string]interface{}{
		"path":      path,
		"operation": "success",
		"lines":     lineCount,
		"bytes":     len(content),
		"created":   true,
	}, nil
}