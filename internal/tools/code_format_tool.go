package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CodeFormatterTool runs code formatters and linters
type CodeFormatterTool struct {
	confirmator Confirmator
}

func (c *CodeFormatterTool) Name() string {
	return "code_format"
}

func (c *CodeFormatterTool) Description() string {
	return "Format code using various formatters and linters (prettier, gofmt, black, etc.)"
}

func (c *CodeFormatterTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"tool": map[string]interface{}{
			"type":        "string",
			"description": "Formatter to use: prettier, gofmt, goimports, black, autopep8, rustfmt, clang-format",
			"enum":        []string{"prettier", "gofmt", "goimports", "black", "autopep8", "rustfmt", "clang-format"},
		},
		"path": map[string]interface{}{
			"type":        "string",
			"description": "File or directory path to format (relative to current working directory)",
		},
		"fix": map[string]interface{}{
			"type":        "boolean",
			"description": "Apply fixes automatically (default: true). If false, only check for issues",
		},
		"config": map[string]interface{}{
			"type":        "string",
			"description": "Path to configuration file (optional)",
		},
		"options": map[string]interface{}{
			"type":        "object",
			"description": "Additional formatter-specific options",
		},
	}
}

func (c *CodeFormatterTool) RequiredParameters() []string {
	return []string{"tool", "path"}
}

func (c *CodeFormatterTool) SetConfirmator(confirmator Confirmator) {
	c.confirmator = confirmator
}

func (c *CodeFormatterTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	tool, ok := args["tool"].(string)
	if !ok {
		return nil, fmt.Errorf("tool parameter must be a string")
	}

	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	fix := true
	if val, exists := args["fix"]; exists {
		if b, ok := val.(bool); ok {
			fix = b
		}
	}

	var config string
	if val, exists := args["config"]; exists {
		if c, ok := val.(string); ok {
			config = c
		}
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

	// Check if path exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	// Ask for confirmation
	if c.confirmator != nil {
		operation := "Check"
		if fix {
			operation = "Format"
		}
		message := fmt.Sprintf("%s %s with %s", operation, path, tool)
		dangerous := fix // Fixing code is more dangerous than just checking
		if !c.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Run the formatter
	return c.runFormatter(tool, fullPath, path, fix, config)
}

func (c *CodeFormatterTool) runFormatter(tool, fullPath, relativePath string, fix bool, config string) (interface{}, error) {
	var cmd *exec.Cmd
	
	switch tool {
	case "gofmt":
		if fix {
			cmd = exec.Command("gofmt", "-w", fullPath)
		} else {
			cmd = exec.Command("gofmt", "-d", fullPath)
		}
		
	case "goimports":
		if fix {
			cmd = exec.Command("goimports", "-w", fullPath)
		} else {
			cmd = exec.Command("goimports", "-d", fullPath)
		}
		
	case "prettier":
		args := []string{}
		if fix {
			args = append(args, "--write")
		} else {
			args = append(args, "--check")
		}
		if config != "" {
			args = append(args, "--config", config)
		}
		args = append(args, fullPath)
		cmd = exec.Command("prettier", args...)
		
	case "black":
		args := []string{}
		if !fix {
			args = append(args, "--check", "--diff")
		}
		if config != "" {
			args = append(args, "--config", config)
		}
		args = append(args, fullPath)
		cmd = exec.Command("black", args...)
		
	case "autopep8":
		args := []string{}
		if fix {
			args = append(args, "--in-place")
		} else {
			args = append(args, "--diff")
		}
		args = append(args, fullPath)
		cmd = exec.Command("autopep8", args...)
		
	case "rustfmt":
		if fix {
			cmd = exec.Command("rustfmt", fullPath)
		} else {
			cmd = exec.Command("rustfmt", "--check", fullPath)
		}
		
	case "clang-format":
		if fix {
			cmd = exec.Command("clang-format", "-i", fullPath)
		} else {
			cmd = exec.Command("clang-format", "--dry-run", "--Werror", fullPath)
		}
		
	default:
		return nil, fmt.Errorf("unsupported formatter: %s", tool)
	}

	// Set working directory
	cmd.Dir = filepath.Dir(fullPath)
	
	// Capture output
	output, err := cmd.CombinedOutput()
	
	// Parse result
	success := err == nil
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	result := map[string]interface{}{
		"tool":      tool,
		"path":      relativePath,
		"fix":       fix,
		"success":   success,
		"exit_code": exitCode,
		"output":    strings.TrimSpace(string(output)),
	}

	if !success {
		result["error"] = err.Error()
	}

	// Check if tool is available
	if exitCode == 127 || strings.Contains(err.Error(), "executable file not found") {
		return nil, fmt.Errorf("formatter not found: %s (make sure it's installed and in PATH)", tool)
	}

	return result, nil
}