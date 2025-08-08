package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// EchoTool is a simple echo tool for testing
type EchoTool struct{}

func (e *EchoTool) Name() string {
	return "echo"
}

func (e *EchoTool) Description() string {
	return "Echo back the provided message"
}

func (e *EchoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]interface{}{
			"type":        "string",
			"description": "The message to echo back",
		},
	}
}

func (e *EchoTool) RequiredParameters() []string {
	return []string{"message"}
}

func (e *EchoTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	message, ok := args["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message parameter must be a string")
	}
	
	return fmt.Sprintf("Echo: %s", message), nil
}

// ShellTool executes shell commands (use with caution)
type ShellTool struct{}

func (s *ShellTool) Name() string {
	return "shell"
}

func (s *ShellTool) Description() string {
	return "Execute a shell command and return its output"
}

func (s *ShellTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"command": map[string]interface{}{
			"type":        "string",
			"description": "The shell command to execute",
		},
		"timeout": map[string]interface{}{
			"type":        "number",
			"description": "Timeout in seconds (default: 30)",
		},
	}
}

func (s *ShellTool) RequiredParameters() []string {
	return []string{"command"}
}

func (s *ShellTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command parameter must be a string")
	}
	
	// Set timeout (default 30 seconds)
	timeout := 30.0
	if t, exists := args["timeout"]; exists {
		if timeoutFloat, ok := t.(float64); ok {
			timeout = timeoutFloat
		}
	}
	
	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	
	// Execute command
	cmd := exec.CommandContext(cmdCtx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	
	result := map[string]interface{}{
		"output":    string(output),
		"exit_code": cmd.ProcessState.ExitCode(),
	}
	
	if err != nil {
		result["error"] = err.Error()
	}
	
	return result, nil
}

// CurrentTimeool returns the current time
type CurrentTimeTool struct{}

func (c *CurrentTimeTool) Name() string {
	return "current_time"
}

func (c *CurrentTimeTool) Description() string {
	return "Get the current date and time"
}

func (c *CurrentTimeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"format": map[string]interface{}{
			"type":        "string",
			"description": "Time format. Common formats: 'iso' (default), 'human', 'date', 'time', 'unix', or Go format string like '2006-01-02 15:04:05'",
		},
	}
}

func (c *CurrentTimeTool) RequiredParameters() []string {
	return []string{} // No required parameters
}

func (c *CurrentTimeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	now := time.Now()
	format := time.RFC3339 // default
	
	if f, exists := args["format"]; exists {
		if formatStr, ok := f.(string); ok {
			// Handle common format names
			switch formatStr {
			case "iso", "":
				format = time.RFC3339
			case "human":
				format = "January 2, 2006 at 3:04 PM MST"
			case "date":
				format = "2006-01-02"
			case "time":
				format = "15:04:05"
			case "unix":
				return now.Unix(), nil
			default:
				// Try to use the format string directly (Go format)
				format = formatStr
			}
		}
	}
	
	return now.Format(format), nil
}

// FileReadTool reads file contents
type FileReadTool struct{}

func (f *FileReadTool) Name() string {
	return "read_file"
}

func (f *FileReadTool) Description() string {
	return "Read the contents of a file"
}

func (f *FileReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Path to the file to read",
		},
	}
}

func (f *FileReadTool) RequiredParameters() []string {
	return []string{"path"}
}

func (f *FileReadTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}
	
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	
	return map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
	}, nil
}

// RegisterBuiltinTools registers all builtin tools to a registry
func RegisterBuiltinTools(registry *Registry) {
	registry.Register(&EchoTool{})
	registry.Register(&ShellTool{})
	registry.Register(&CurrentTimeTool{})
	registry.Register(&FileReadTool{})
}