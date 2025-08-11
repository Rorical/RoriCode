package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ShellTool executes shell commands (use with caution)
type ShellTool struct {
	confirmator Confirmator
}

func (s *ShellTool) Name() string {
	return "shell"
}

func (s *ShellTool) Description() string {
	return "Execute shell commands with safety features and timeout control"
}

func (s *ShellTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"command": map[string]interface{}{
			"type":        "string",
			"description": "The shell command to execute",
		},
		"timeout": map[string]interface{}{
			"type":        "number",
			"description": "Timeout in seconds (default: 30, max: 300)",
		},
		"working_dir": map[string]interface{}{
			"type":        "string",
			"description": "Working directory for the command (optional)",
		},
	}
}

func (s *ShellTool) RequiredParameters() []string {
	return []string{"command"}
}

func (s *ShellTool) SetConfirmator(confirmator Confirmator) {
	s.confirmator = confirmator
}

func (s *ShellTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command parameter must be a string")
	}

	// Handle timeout
	timeout := 30.0
	if val, exists := args["timeout"]; exists {
		if t, ok := val.(float64); ok {
			if t > 300 {
				timeout = 300 // Max 5 minutes
			} else if t > 0 {
				timeout = t
			}
		}
	}

	// Handle working directory
	var workingDir string
	if val, exists := args["working_dir"]; exists {
		if wd, ok := val.(string); ok {
			workingDir = wd
		}
	}

	// Check if confirmation is needed
	if s.confirmator != nil {
		dangerous := s.isDangerousCommand(command)
		if !s.confirmator.RequestConfirmation("Execute command", command, dangerous) {
			return map[string]interface{}{
				"output":  "User aborted command execution",
				"aborted": true,
			}, nil
		}
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(timeoutCtx, "cmd", "/c", command)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"command":     command,
		"output":      string(output),
		"working_dir": workingDir,
		"timeout":     timeout,
	}

	if err != nil {
		result["error"] = err.Error()
		result["success"] = false
		
		// Check if it was a timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			result["timed_out"] = true
		}
		
		// Get exit code if available
		if exitError, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitError.ExitCode()
		}
	} else {
		result["success"] = true
		result["exit_code"] = 0
	}

	return result, nil
}

// isDangerousCommand returns true if the command might be dangerous
func (s *ShellTool) isDangerousCommand(command string) bool {
	commandLower := strings.ToLower(command)
	
	// Commands that are generally safe and don't need confirmation
	readOnlyPatterns := []string{
		"dir", "ls", "pwd", "echo", "cat", "type", "find", "grep", 
		"head", "tail", "wc", "sort", "uniq", "which", "where",
		"git status", "git log", "git diff", "git show",
		"npm list", "pip list", "go version", "node --version",
	}
	
	for _, pattern := range readOnlyPatterns {
		if strings.Contains(commandLower, pattern) {
			return false // No confirmation needed
		}
	}
	
	// For everything else, require confirmation
	return true
}