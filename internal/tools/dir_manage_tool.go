package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DirectoryManageTool handles directory operations (create/delete/list)
type DirectoryManageTool struct {
	confirmator Confirmator
}

func (d *DirectoryManageTool) Name() string {
	return "dir_manage"
}

func (d *DirectoryManageTool) Description() string {
	return "Manage directories: create, delete, or list directory contents"
}

func (d *DirectoryManageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"operation": map[string]interface{}{
			"type":        "string",
			"description": "Operation to perform: 'create', 'delete', or 'list'",
			"enum":        []string{"create", "delete", "list"},
		},
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Directory path (relative to current working directory)",
		},
		"recursive": map[string]interface{}{
			"type":        "boolean",
			"description": "For delete: remove directory recursively. For create: create parent directories (default: false)",
		},
		"show_hidden": map[string]interface{}{
			"type":        "boolean",
			"description": "For list: show hidden files/directories (default: false)",
		},
		"details": map[string]interface{}{
			"type":        "boolean",
			"description": "For list: show detailed information (size, permissions, modified time) (default: false)",
		},
	}
}

func (d *DirectoryManageTool) RequiredParameters() []string {
	return []string{"operation", "path"}
}

func (d *DirectoryManageTool) SetConfirmator(confirmator Confirmator) {
	d.confirmator = confirmator
}

func (d *DirectoryManageTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter must be a string")
	}

	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	// Handle optional parameters
	recursive := false
	if val, exists := args["recursive"]; exists {
		if b, ok := val.(bool); ok {
			recursive = b
		}
	}

	showHidden := false
	if val, exists := args["show_hidden"]; exists {
		if b, ok := val.(bool); ok {
			showHidden = b
		}
	}

	details := false
	if val, exists := args["details"]; exists {
		if b, ok := val.(bool); ok {
			details = b
		}
	}

	// Validate operation
	validOps := map[string]bool{"create": true, "delete": true, "list": true}
	if !validOps[operation] {
		return nil, fmt.Errorf("operation must be one of: create, delete, list")
	}

	// Validate path safety
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be relative, not absolute: %s", path)
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path cannot contain parent directory references (..): %s", path)
	}

	// Get absolute path
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}

	fullPath := filepath.Join(cwd, path)

	switch operation {
	case "create":
		return d.createDirectory(fullPath, path, recursive)
	case "delete":
		return d.deleteDirectory(fullPath, path, recursive)
	case "list":
		return d.listDirectory(fullPath, path, showHidden, details)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

func (d *DirectoryManageTool) createDirectory(fullPath, relativePath string, recursive bool) (interface{}, error) {
	// Check if directory already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Sprintf("Directory already exists: %s", relativePath), nil
	}

	// Ask for confirmation
	if d.confirmator != nil {
		operation := "Create directory"
		message := fmt.Sprintf("Create directory %s", relativePath)
		if recursive {
			message += " (with parent directories)"
		}
		if !d.confirmator.RequestConfirmation(operation, message, false) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Create directory
	var err error
	if recursive {
		err = os.MkdirAll(fullPath, 0755)
	} else {
		err = os.Mkdir(fullPath, 0755)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	return map[string]interface{}{
		"operation": "create",
		"path":      relativePath,
		"recursive": recursive,
		"success":   true,
	}, nil
}

func (d *DirectoryManageTool) deleteDirectory(fullPath, relativePath string, recursive bool) (interface{}, error) {
	// Check if directory exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", relativePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check directory: %v", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", relativePath)
	}

	// Check if directory is empty (unless recursive)
	if !recursive {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %v", err)
		}
		if len(entries) > 0 {
			return nil, fmt.Errorf("directory is not empty: %s (use recursive: true to force delete)", relativePath)
		}
	}

	// Ask for confirmation
	if d.confirmator != nil {
		operation := "Delete directory"
		message := fmt.Sprintf("Delete directory %s", relativePath)
		if recursive {
			message += " (recursively, including all contents)"
		}
		dangerous := true // Directory deletion is always dangerous
		if !d.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Delete directory
	if recursive {
		err = os.RemoveAll(fullPath)
	} else {
		err = os.Remove(fullPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to delete directory: %v", err)
	}

	return map[string]interface{}{
		"operation": "delete",
		"path":      relativePath,
		"recursive": recursive,
		"success":   true,
	}, nil
}

func (d *DirectoryManageTool) listDirectory(fullPath, relativePath string, showHidden, details bool) (interface{}, error) {
	// Check if directory exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", relativePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check directory: %v", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", relativePath)
	}

	// Read directory contents
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var result []map[string]interface{}
	
	for _, entry := range entries {
		name := entry.Name()
		
		// Skip hidden files unless requested
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		entryInfo := map[string]interface{}{
			"name": name,
			"type": "file",
		}

		if entry.IsDir() {
			entryInfo["type"] = "directory"
		}

		// Add detailed information if requested
		if details {
			info, err := entry.Info()
			if err == nil {
				entryInfo["size"] = info.Size()
				entryInfo["mode"] = info.Mode().String()
				entryInfo["modified"] = info.ModTime().Format(time.RFC3339)
			}
		}

		result = append(result, entryInfo)
	}

	return map[string]interface{}{
		"operation": "list",
		"path":      relativePath,
		"count":     len(result),
		"entries":   result,
	}, nil
}