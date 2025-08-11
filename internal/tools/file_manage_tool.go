package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileManageTool handles file operations (copy/move/rename)
type FileManageTool struct {
	confirmator Confirmator
}

func (f *FileManageTool) Name() string {
	return "file_manage"
}

func (f *FileManageTool) Description() string {
	return "Perform file operations: copy, move, or rename files"
}

func (f *FileManageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"operation": map[string]interface{}{
			"type":        "string",
			"description": "Operation to perform: 'copy', 'move', or 'rename'",
			"enum":        []string{"copy", "move", "rename"},
		},
		"source": map[string]interface{}{
			"type":        "string",
			"description": "Source file path (relative to current working directory)",
		},
		"destination": map[string]interface{}{
			"type":        "string",
			"description": "Destination file path (relative to current working directory)",
		},
		"overwrite": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether to overwrite destination if it exists (default: false)",
		},
	}
}

func (f *FileManageTool) RequiredParameters() []string {
	return []string{"operation", "source", "destination"}
}

func (f *FileManageTool) SetConfirmator(confirmator Confirmator) {
	f.confirmator = confirmator
}

func (f *FileManageTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter must be a string")
	}

	source, ok := args["source"].(string)
	if !ok {
		return nil, fmt.Errorf("source parameter must be a string")
	}

	destination, ok := args["destination"].(string)
	if !ok {
		return nil, fmt.Errorf("destination parameter must be a string")
	}

	overwrite := false
	if val, exists := args["overwrite"]; exists {
		if b, ok := val.(bool); ok {
			overwrite = b
		}
	}

	// Validate operation
	validOps := map[string]bool{"copy": true, "move": true, "rename": true}
	if !validOps[operation] {
		return nil, fmt.Errorf("operation must be one of: copy, move, rename")
	}

	// Validate path safety
	for _, path := range []string{source, destination} {
		if filepath.IsAbs(path) {
			return nil, fmt.Errorf("paths must be relative, not absolute: %s", path)
		}
		if strings.Contains(path, "..") {
			return nil, fmt.Errorf("paths cannot contain parent directory references (..): %s", path)
		}
	}

	// Get absolute paths
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}

	sourcePath := filepath.Join(cwd, source)
	destPath := filepath.Join(cwd, destination)

	// Check if source exists
	sourceInfo, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("source file does not exist: %s", source)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check source file: %v", err)
	}

	// Check if destination exists
	destExists := false
	if _, err := os.Stat(destPath); err == nil {
		destExists = true
		if !overwrite {
			return nil, fmt.Errorf("destination file already exists: %s (use overwrite: true to replace)", destination)
		}
	}

	// Create destination directory if needed
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Ask for confirmation
	if f.confirmator != nil {
		var message string
		switch operation {
		case "copy":
			message = fmt.Sprintf("Copy %s to %s", source, destination)
		case "move":
			message = fmt.Sprintf("Move %s to %s", source, destination)
		case "rename":
			message = fmt.Sprintf("Rename %s to %s", source, destination)
		}
		if destExists {
			message += " (will overwrite existing file)"
		}
		
		dangerous := destExists || operation == "move"
		operationName := "File management"
		
		if !f.confirmator.RequestConfirmation(operationName, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Perform the operation
	switch operation {
	case "copy":
		err = f.copyFile(sourcePath, destPath)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file: %v", err)
		}
		
		return map[string]interface{}{
			"operation":   "copy",
			"source":      source,
			"destination": destination,
			"size":        sourceInfo.Size(),
			"success":     true,
		}, nil

	case "move", "rename":
		err = os.Rename(sourcePath, destPath)
		if err != nil {
			return nil, fmt.Errorf("failed to %s file: %v", operation, err)
		}
		
		return map[string]interface{}{
			"operation":   operation,
			"source":      source,
			"destination": destination,
			"size":        sourceInfo.Size(),
			"success":     true,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// copyFile copies a file from source to destination
func (f *FileManageTool) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	return os.Chmod(dst, sourceInfo.Mode())
}