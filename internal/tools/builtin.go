package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ShellTool executes shell commands (use with caution)
type ShellTool struct{
	confirmator Confirmator
}

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

func (s *ShellTool) SetConfirmator(confirmator Confirmator) {
	s.confirmator = confirmator
}

func (s *ShellTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command parameter must be a string")
	}

	// Request user confirmation for shell commands that need it
	if s.confirmator != nil && s.needsConfirmation(command) {
		dangerous := s.isDangerousCommand(command)
		if !s.confirmator.RequestConfirmation("Execute shell command", command, dangerous) {
			return map[string]interface{}{
				"output":    "User aborted execution",
				"exit_code": 1,
				"aborted":   true,
			}, nil
		}
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

// isDangerousCommand checks if a shell command is potentially dangerous
func (s *ShellTool) isDangerousCommand(command string) bool {
	dangerous := []string{
		"rm", "rmdir", "del", "delete", "unlink",
		"format", "fdisk", "mkfs",
		"dd", "shred", "wipe",
		"chmod", "chown", "chgrp",
		"sudo", "su", "doas",
		"passwd", "usermod", "userdel",
		"systemctl", "service", "init",
		"reboot", "shutdown", "halt",
		"kill", "killall", "pkill",
		"mv", "move", "rename",
		"truncate", ">", ">>",
		"curl", "wget", "nc", "netcat",
		"iptables", "firewall",
		"crontab", "at", "batch",
	}
	
	commandLower := strings.ToLower(command)
	for _, danger := range dangerous {
		if strings.Contains(commandLower, danger) {
			return true
		}
	}
	
	// Check for potentially dangerous patterns
	dangerousPatterns := []string{
		">/", ">>", "| rm", "| del",
		"--force", "-f", "--recursive", "-r",
		"--no-preserve-root",
	}
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(commandLower, pattern) {
			return true
		}
	}
	
	return false
}

// needsConfirmation checks if a command needs user confirmation
// Returns false for safe, read-only commands that don't modify the system
func (s *ShellTool) needsConfirmation(command string) bool {
	// List of safe commands that don't need confirmation
	safeCommands := []string{
		// File/directory listing and info
		"ls", "ll", "la", "dir", "tree", "find",
		"pwd", "which", "whereis", "type", "file", "stat",
		
		// Text display and processing
		"cat", "head", "tail", "less", "more", "grep", "egrep", "fgrep",
		"awk", "sed", "sort", "uniq", "wc", "cut", "tr", "fold",
		"diff", "cmp", "comm", "join", "split",
		
		// System info (read-only)
		"ps", "top", "htop", "uptime", "whoami", "id", "groups",
		"uname", "hostname", "date", "cal", "df", "du", "free",
		"lscpu", "lsblk", "lsusb", "lspci", "mount",
		
		// Environment and variables (read-only)
		"env", "printenv", "set", "declare", "export",
		
		// Network info (read-only)
		"ping", "traceroute", "nslookup", "dig", "host",
		"netstat", "ss", "lsof", "ifconfig", "ip",
		
		// Version info
		"--version", "-V", "version",
		
		// Git read operations
		"git status", "git log", "git show", "git diff", "git branch",
		"git remote", "git config --get", "git ls-files",
		
		// Package managers (info only)
		"apt list", "apt show", "apt search",
		"yum list", "yum info", "yum search",
		"brew list", "brew info", "brew search",
		"npm list", "npm info", "npm search",
		
		// Help and manual
		"help", "man", "info", "apropos", "whatis",
	}
	
	commandLower := strings.ToLower(strings.TrimSpace(command))
	
	// Check if it's a safe command
	for _, safe := range safeCommands {
		if strings.HasPrefix(commandLower, safe) {
			return false // No confirmation needed
		}
	}
	
	// Check for read-only patterns
	readOnlyPatterns := []string{
		"--help", "-h", "--version", "-v",
		"cat ", "head ", "tail ", "less ", "more ",
		"grep ", "find ", "ls ", "pwd", "which ",
		"echo ", "printf ", "date", "uptime",
	}
	
	for _, pattern := range readOnlyPatterns {
		if strings.Contains(commandLower, pattern) {
			return false // No confirmation needed
		}
	}
	
	// For everything else, require confirmation
	return true
}

// CurrentTimeTool returns the current time
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

// FileReadTool reads file contents with advanced options
type FileReadTool struct{}

func (f *FileReadTool) Name() string {
	return "read_file"
}

func (f *FileReadTool) Description() string {
	return "Read file or directory contents with advanced filtering options. Supports reading specific line ranges, regex matching with context, and directory listing."
}

func (f *FileReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Relative path to the file or directory from current working directory",
		},
		"lines_from": map[string]interface{}{
			"type":        "number",
			"description": "Start reading from this line number (1-based, optional)",
		},
		"lines_to": map[string]interface{}{
			"type":        "number",
			"description": "Stop reading at this line number (1-based, optional)",
		},
		"regex": map[string]interface{}{
			"type":        "string",
			"description": "Regular expression to search for in the file (optional)",
		},
		"regex_match": map[string]interface{}{
			"type":        "number",
			"description": "Which match to return when using regex (1-based, default: 1, optional)",
		},
		"context_lines": map[string]interface{}{
			"type":        "number",
			"description": "Number of lines to include before and after regex match (default: 3, optional)",
		},
		"max_lines": map[string]interface{}{
			"type":        "number",
			"description": "Maximum number of lines to return (default: 100, max: 1000, optional)",
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

	// Ensure path is relative and safe (no parent directory traversal)
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be relative, not absolute")
	}

	// Prevent directory traversal attacks
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("path cannot contain parent directory references (..)")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %v", err)
	}

	// Construct absolute path safely
	fullPath := filepath.Join(cwd, path)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file or directory not found: %s", path)
		}
		return nil, fmt.Errorf("failed to access path: %v", err)
	}

	// Handle directory listing
	if info.IsDir() {
		return f.listDirectory(fullPath, path)
	}

	// Handle file reading
	return f.readFile(fullPath, path, args)
}

// listDirectory returns directory contents
func (f *FileReadTool) listDirectory(fullPath, relativePath string) (interface{}, error) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var files []map[string]interface{}
	var dirs []string

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name()+"/")
		} else {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			files = append(files, map[string]interface{}{
				"name": entry.Name(),
				"size": info.Size(),
				"type": f.getFileType(entry.Name()),
			})
		}
	}

	return map[string]interface{}{
		"path":        relativePath,
		"type":        "directory",
		"directories": dirs,
		"files":       files,
		"total_items": len(dirs) + len(files),
	}, nil
}

// readFile handles file content reading with various options
func (f *FileReadTool) readFile(fullPath, relativePath string, args map[string]interface{}) (interface{}, error) {
	// Check if file is text-readable
	if !f.isTextFile(fullPath) {
		return nil, fmt.Errorf("cannot read non-text file: %s (detected as binary)", relativePath)
	}

	// Parse optional parameters
	var linesFrom, linesTo int
	var regexPattern string
	var regexMatch int = 1
	var contextLines int = 3
	var maxLines int = 100

	if val, exists := args["lines_from"]; exists {
		if num, ok := val.(float64); ok {
			linesFrom = int(num)
		}
	}

	if val, exists := args["lines_to"]; exists {
		if num, ok := val.(float64); ok {
			linesTo = int(num)
		}
	}

	if val, exists := args["regex"]; exists {
		if str, ok := val.(string); ok {
			regexPattern = str
		}
	}

	if val, exists := args["regex_match"]; exists {
		if num, ok := val.(float64); ok {
			regexMatch = int(num)
		}
	}

	if val, exists := args["context_lines"]; exists {
		if num, ok := val.(float64); ok {
			contextLines = int(num)
		}
	}

	if val, exists := args["max_lines"]; exists {
		if num, ok := val.(float64); ok {
			if num > 1000 {
				maxLines = 1000
			} else {
				maxLines = int(num)
			}
		}
	}

	// Read file
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Handle regex search
	if regexPattern != "" {
		return f.searchWithRegex(file, relativePath, regexPattern, regexMatch, contextLines, maxLines)
	}

	// Handle line range reading
	if linesFrom > 0 || linesTo > 0 {
		return f.readLineRange(file, relativePath, linesFrom, linesTo, maxLines)
	}

	// Default: read first N lines
	return f.readDefaultLines(file, relativePath, maxLines)
}

// isTextFile checks if a file is text-readable
func (f *FileReadTool) isTextFile(path string) bool {
	// Check by file extension first
	ext := strings.ToLower(filepath.Ext(path))
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".js": true, ".ts": true,
		".py": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".json": true, ".xml": true, ".yaml": true, ".yml": true,
		".html": true, ".css": true, ".sql": true, ".sh": true,
		".dockerfile": true, ".gitignore": true, ".env": true,
		".toml": true, ".ini": true, ".cfg": true, ".conf": true,
		".log": true, ".csv": true, ".tsv": true,
	}

	if textExts[ext] {
		return true
	}

	// Check MIME type
	mimeType := mime.TypeByExtension(ext)
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}

	// Read first 512 bytes to check for binary content
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check if content is valid UTF-8 and doesn't contain null bytes
	content := buffer[:n]
	if !utf8.Valid(content) {
		return false
	}

	// Check for null bytes (common in binary files)
	for _, b := range content {
		if b == 0 {
			return false
		}
	}

	return true
}

// getFileType returns a human-readable file type
func (f *FileReadTool) getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "Go source"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".java":
		return "Java"
	case ".c", ".h":
		return "C source"
	case ".cpp", ".hpp":
		return "C++ source"
	case ".json":
		return "JSON"
	case ".md":
		return "Markdown"
	case ".txt":
		return "Text"
	case ".yaml", ".yml":
		return "YAML"
	case ".xml":
		return "XML"
	default:
		if f.isTextFile(filename) {
			return "Text file"
		}
		return "Binary file"
	}
}

// searchWithRegex searches for regex pattern and returns matches with context
func (f *FileReadTool) searchWithRegex(file *os.File, relativePath, pattern string, matchNum, contextLines, maxLines int) (interface{}, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	scanner := bufio.NewScanner(file)
	var lines []string
	var matchingLines []int

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		if regex.MatchString(line) {
			matchingLines = append(matchingLines, lineNum-1) // 0-based index
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	if len(matchingLines) == 0 {
		return map[string]interface{}{
			"path":          relativePath,
			"type":          "file",
			"matches_found": 0,
			"content":       "",
			"message":       "No matches found for the regex pattern",
		}, nil
	}

	if matchNum > len(matchingLines) {
		return map[string]interface{}{
			"path":          relativePath,
			"type":          "file",
			"matches_found": len(matchingLines),
			"content":       "",
			"message":       fmt.Sprintf("Only %d matches found, but requested match %d", len(matchingLines), matchNum),
		}, nil
	}

	// Get the requested match (1-based to 0-based)
	matchLineIdx := matchingLines[matchNum-1]

	// Calculate context range
	startIdx := matchLineIdx - contextLines
	if startIdx < 0 {
		startIdx = 0
	}

	endIdx := matchLineIdx + contextLines + 1
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	// Limit total lines returned
	if endIdx-startIdx > maxLines {
		endIdx = startIdx + maxLines
	}

	var resultLines []string
	for i := startIdx; i < endIdx; i++ {
		prefix := "    "
		if i == matchLineIdx {
			prefix = ">>> " // Mark the matching line
		}
		resultLines = append(resultLines, fmt.Sprintf("%d: %s%s", i+1, prefix, lines[i]))
	}

	return map[string]interface{}{
		"path":          relativePath,
		"type":          "file",
		"matches_found": len(matchingLines),
		"match_number":  matchNum,
		"match_line":    matchLineIdx + 1,
		"context_lines": contextLines,
		"content":       strings.Join(resultLines, "\n"),
		"total_lines":   len(lines),
	}, nil
}

// readLineRange reads a specific range of lines
func (f *FileReadTool) readLineRange(file *os.File, relativePath string, linesFrom, linesTo, maxLines int) (interface{}, error) {
	if linesFrom < 1 {
		linesFrom = 1
	}

	scanner := bufio.NewScanner(file)
	var resultLines []string

	lineNum := 0
	for scanner.Scan() {
		lineNum++

		if lineNum < linesFrom {
			continue
		}

		if linesTo > 0 && lineNum > linesTo {
			break
		}

		if len(resultLines) >= maxLines {
			break
		}

		line := scanner.Text()
		resultLines = append(resultLines, fmt.Sprintf("%d: %s", lineNum, line))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return map[string]interface{}{
		"path":       relativePath,
		"type":       "file",
		"lines_from": linesFrom,
		"lines_to":   linesTo,
		"content":    strings.Join(resultLines, "\n"),
		"lines_read": len(resultLines),
	}, nil
}

// readDefaultLines reads the first N lines of a file
func (f *FileReadTool) readDefaultLines(file *os.File, relativePath string, maxLines int) (interface{}, error) {
	scanner := bufio.NewScanner(file)
	var resultLines []string

	lineNum := 0
	for scanner.Scan() && lineNum < maxLines {
		lineNum++
		line := scanner.Text()
		resultLines = append(resultLines, fmt.Sprintf("%d: %s", lineNum, line))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Check if there are more lines
	moreLines := scanner.Scan()

	return map[string]interface{}{
		"path":       relativePath,
		"type":       "file",
		"content":    strings.Join(resultLines, "\n"),
		"lines_read": len(resultLines),
		"more_lines": moreLines,
		"max_lines":  maxLines,
	}, nil
}

// FileEditTool edits files using git diff format
type FileEditTool struct {
	confirmator Confirmator
}

func (f *FileEditTool) Name() string {
	return "edit_file"
}

func (f *FileEditTool) Description() string {
	return "Edit existing files using git diff format. Applies unified diff patches to modify file content."
}

func (f *FileEditTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Relative path to the file to edit from current working directory",
		},
		"diff": map[string]interface{}{
			"type":        "string",
			"description": "Git unified diff format content. Use @@ headers and +/- line prefixes. Context lines should be included for accurate application.",
		},
	}
}

func (f *FileEditTool) RequiredParameters() []string {
	return []string{"path", "diff"}
}

func (f *FileEditTool) SetConfirmator(confirmator Confirmator) {
	f.confirmator = confirmator
}

func (f *FileEditTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	diff, ok := args["diff"].(string)
	if !ok {
		return nil, fmt.Errorf("diff parameter must be a string")
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

	// Request confirmation for file modifications
	if f.confirmator != nil {
		if !f.confirmator.RequestConfirmation("Edit file", fmt.Sprintf("Modify %s with diff", path), false) {
			return map[string]interface{}{
				"output":  "User aborted file edit operation",
				"aborted": true,
			}, nil
		}
	}

	// Check if file exists
	var originalContent []string
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s (use create_file tool to create new files)", path)
	} else if err != nil {
		return nil, fmt.Errorf("failed to access file: %v", err)
	} else {
		// Read existing file
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		originalContent = strings.Split(string(content), "\n")
		// Remove last empty line if file ends with newline
		if len(originalContent) > 0 && originalContent[len(originalContent)-1] == "" {
			originalContent = originalContent[:len(originalContent)-1]
		}
	}

	// Parse and apply diff
	modifiedContent, err := f.applyDiff(originalContent, diff)
	if err != nil {
		return nil, fmt.Errorf("failed to apply diff: %v", err)
	}

	// Write the result
	finalContent := strings.Join(modifiedContent, "\n")
	if len(modifiedContent) > 0 {
		finalContent += "\n" // Add final newline
	}

	err = os.WriteFile(fullPath, []byte(finalContent), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	return map[string]interface{}{
		"path":           path,
		"operation":      "success",
		"lines_before":   len(originalContent),
		"lines_after":    len(modifiedContent),
		"changes_applied": true,
	}, nil
}

// DiffHunk represents a single diff hunk
type DiffHunk struct {
	OldStart int      // Starting line in original file (1-based)
	OldCount int      // Number of lines in original file
	NewStart int      // Starting line in new file (1-based)  
	NewCount int      // Number of lines in new file
	Lines    []string // Diff lines with +/- prefixes
}

// applyDiff parses and applies a unified diff to content
func (f *FileEditTool) applyDiff(originalLines []string, diff string) ([]string, error) {
	hunks, err := f.parseDiff(diff)
	if err != nil {
		return nil, err
	}

	if len(hunks) == 0 {
		return nil, fmt.Errorf("no valid diff hunks found")
	}

	// Apply hunks in reverse order to maintain line numbers
	result := make([]string, len(originalLines))
	copy(result, originalLines)

	// Sort hunks by starting line (descending) to apply from bottom to top
	for i := len(hunks) - 1; i >= 0; i-- {
		hunk := hunks[i]
		var err error
		result, err = f.applyHunk(result, hunk)
		if err != nil {
			return nil, fmt.Errorf("failed to apply hunk at line %d: %v", hunk.OldStart, err)
		}
	}

	return result, nil
}

// parseDiff parses unified diff format with support for various formats
func (f *FileEditTool) parseDiff(diff string) ([]DiffHunk, error) {
	// Handle escaped newlines in diff content
	diff = strings.ReplaceAll(diff, "\\n", "\n")
	
	lines := strings.Split(diff, "\n")
	var hunks []DiffHunk

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Look for hunk header: @@ -oldStart,oldCount +newStart,newCount @@
		if strings.HasPrefix(strings.TrimSpace(line), "@@") {
			hunk, err := f.parseHunkHeader(line, lines, i)
			if err != nil {
				continue // Skip invalid hunks but continue processing
			}
			if hunk != nil {
				hunks = append(hunks, *hunk)
			}
			
			// Skip to end of this hunk
			i = f.findNextHunkStart(lines, i+1) - 1
		}
	}

	return hunks, nil
}

// parseHunkHeader parses a single hunk header and its content
func (f *FileEditTool) parseHunkHeader(headerLine string, lines []string, startIdx int) (*DiffHunk, error) {
	line := strings.TrimSpace(headerLine)
	
	// Extract content between @@ markers
	if !strings.HasPrefix(line, "@@") {
		return nil, fmt.Errorf("invalid hunk header")
	}
	
	// Find the closing @@
	secondAt := strings.Index(line[2:], "@@")
	if secondAt == -1 {
		return nil, fmt.Errorf("malformed hunk header")
	}
	secondAt += 2
	
	// Parse header ranges
	header := strings.TrimSpace(line[2:secondAt])
	parts := strings.Fields(header)
	if len(parts) < 2 {
		return nil, fmt.Errorf("insufficient header parts")
	}
	
	var oldStart, oldCount, newStart, newCount int
	
	// Parse old range (-oldStart,oldCount)
	oldPart := strings.TrimPrefix(parts[0], "-")
	if strings.Contains(oldPart, ",") {
		fmt.Sscanf(oldPart, "%d,%d", &oldStart, &oldCount)
	} else {
		fmt.Sscanf(oldPart, "%d", &oldStart)
		oldCount = 1
	}
	
	// Parse new range (+newStart,newCount)  
	newPart := strings.TrimPrefix(parts[1], "+")
	if strings.Contains(newPart, ",") {
		fmt.Sscanf(newPart, "%d,%d", &newStart, &newCount)
	} else {
		fmt.Sscanf(newPart, "%d", &newStart)
		newCount = 1
	}
	
	var hunkLines []string
	
	// Get context from header if present
	if secondAt+2 < len(line) {
		context := strings.TrimSpace(line[secondAt+2:])
		if context != "" {
			hunkLines = f.parseInlineContext(context)
		}
	}
	
	// Get lines from body (if any)
	bodyLines := f.getHunkBodyLines(lines, startIdx+1)
	hunkLines = append(hunkLines, bodyLines...)
	
	// If we still don't have proper diff lines, treat any remaining content as changes
	if len(hunkLines) == 0 {
		return nil, fmt.Errorf("no hunk content found")
	}
	
	return &DiffHunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
		Lines:    hunkLines,
	}, nil
}

// parseInlineContext handles context that appears after the @@ header
func (f *FileEditTool) parseInlineContext(context string) []string {
	var lines []string
	
	// Handle the case where the diff format embeds content in the header
	// Look for deletion markers (-)
	if strings.Contains(context, "-") {
		parts := strings.Split(context, "-")
		
		// Everything before the first "-" is context
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			contextWords := strings.Fields(parts[0])
			for _, word := range contextWords {
				if word != "" {
					lines = append(lines, " "+word) // Context line
				}
			}
		}
		
		// Everything after "-" is the removal
		if len(parts) > 1 {
			removal := strings.TrimSpace(parts[1])
			if removal != "" {
				lines = append(lines, "-"+removal)
			}
		}
	} else {
		// No explicit changes, treat as context
		words := strings.Fields(context)
		for _, word := range words {
			if word != "" {
				lines = append(lines, " "+word)
			}
		}
	}
	
	return lines
}

// getHunkBodyLines extracts lines that are part of this hunk body
func (f *FileEditTool) getHunkBodyLines(lines []string, startIdx int) []string {
	var bodyLines []string
	
	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		
		// Stop at next hunk header
		if strings.HasPrefix(trimmed, "@@") {
			break
		}
		
		// Include lines that start with diff prefixes or non-empty lines
		if len(line) > 0 && (line[0] == '+' || line[0] == '-' || line[0] == ' ') {
			bodyLines = append(bodyLines, line)
		} else if strings.TrimSpace(line) != "" {
			// Non-prefixed content might be context
			bodyLines = append(bodyLines, line)
		}
	}
	
	return bodyLines
}

// findNextHunkStart finds the start of the next hunk
func (f *FileEditTool) findNextHunkStart(lines []string, startIdx int) int {
	for i := startIdx; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "@@") {
			return i
		}
	}
	return len(lines)
}

// applyHunk applies a single diff hunk to content
func (f *FileEditTool) applyHunk(content []string, hunk DiffHunk) ([]string, error) {
	// Convert to 0-based indexing
	startIdx := hunk.OldStart - 1
	if startIdx < 0 {
		startIdx = 0
	}

	// Verify context and apply changes
	var result []string
	
	// Copy lines before the hunk
	result = append(result, content[:startIdx]...)
	
	// Apply the hunk
	contentIdx := startIdx
	for _, line := range hunk.Lines {
		if len(line) == 0 {
			continue
		}
		
		switch line[0] {
		case ' ': // Context line
			// Verify context matches
			expectedLine := line[1:]
			if contentIdx < len(content) {
				if content[contentIdx] != expectedLine {
					return nil, fmt.Errorf("context mismatch at line %d: expected '%s', got '%s'", 
						contentIdx+1, expectedLine, content[contentIdx])
				}
			}
			result = append(result, expectedLine)
			contentIdx++
			
		case '-': // Deletion
			// Verify line to delete matches
			expectedLine := line[1:]
			if contentIdx < len(content) {
				if content[contentIdx] != expectedLine {
					return nil, fmt.Errorf("deletion mismatch at line %d: expected '%s', got '%s'", 
						contentIdx+1, expectedLine, content[contentIdx])
				}
			}
			// Skip this line (delete it)
			contentIdx++
			
		case '+': // Addition  
			// Add new line
			newLine := line[1:]
			result = append(result, newLine)
			// Don't increment contentIdx for additions
		}
	}
	
	// Copy remaining lines after the hunk
	if contentIdx < len(content) {
		result = append(result, content[contentIdx:]...)
	}

	return result, nil
}

// FileCreationTool creates new files with specified content
type FileCreationTool struct {
	confirmator Confirmator
}

// FileReplaceLinesTool replaces specific line ranges in files
type FileReplaceLinesTool struct {
	confirmator Confirmator
}

// FileSearchReplaceTool performs search and replace operations
type FileSearchReplaceTool struct {
	confirmator Confirmator
}

// FileInsertTool inserts content at specific positions
type FileInsertTool struct {
	confirmator Confirmator
}

// FileManageTool handles file operations (copy/move/rename)
type FileManageTool struct {
	confirmator Confirmator
}

// DirectoryManageTool handles directory operations (create/delete/list)
type DirectoryManageTool struct {
	confirmator Confirmator
}

// CodeFormatterTool runs code formatters and linters
type CodeFormatterTool struct {
	confirmator Confirmator
}

// DataEditTool manipulates JSON/YAML files
type DataEditTool struct {
	confirmator Confirmator
}

// DataProcessTool processes CSV and structured data
type DataProcessTool struct {
	confirmator Confirmator
}

// FileDiffTool compares files and shows differences
type FileDiffTool struct {
	confirmator Confirmator
}

// HttpRequestTool makes HTTP requests
type HttpRequestTool struct {
	confirmator Confirmator
}

// EnvManageTool manages environment variables
type EnvManageTool struct {
	confirmator Confirmator
}

// ProcessManageTool manages system processes
type ProcessManageTool struct {
	confirmator Confirmator
}

func (f *FileCreationTool) Name() string {
	return "create_file"
}

func (f *FileCreationTool) Description() string {
	return "Create a new file with specified content. Fails if file already exists."
}

func (f *FileCreationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Relative path to the file to create from current working directory",
		},
		"content": map[string]interface{}{
			"type":        "string",
			"description": "Content to write to the new file",
		},
		"overwrite": map[string]interface{}{
			"type":        "boolean",
			"description": "Allow overwriting existing file (default: false)",
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

// FileReplaceLinesTool implementation
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
	if f.confirmator != nil && f.confirmator.ShouldConfirm("replace_lines") {
		linesCount := int(endLine) - int(startLine) + 1
		message := fmt.Sprintf("Replace %d line(s) in %s (lines %d-%d)", 
			linesCount, path, int(startLine), int(endLine))
		if !f.confirmator.Confirm("replace_lines", message, "high") {
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

// FileSearchReplaceTool implementation
func (f *FileSearchReplaceTool) Name() string {
	return "search_replace"
}

func (f *FileSearchReplaceTool) Description() string {
	return "Find and replace text patterns in files. Supports literal text and regex patterns."
}

func (f *FileSearchReplaceTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "Relative path to the file to edit from current working directory",
		},
		"search": map[string]interface{}{
			"type":        "string",
			"description": "Text or pattern to search for",
		},
		"replace": map[string]interface{}{
			"type":        "string",
			"description": "Replacement text. Use \\n for line breaks",
		},
		"regex": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether to treat search as regex pattern (default: false)",
		},
		"global": map[string]interface{}{
			"type":        "boolean",
			"description": "Replace all occurrences (default: true). If false, only replaces first occurrence",
		},
		"case_sensitive": map[string]interface{}{
			"type":        "boolean",
			"description": "Case-sensitive matching (default: true)",
		},
	}
}

func (f *FileSearchReplaceTool) RequiredParameters() []string {
	return []string{"path", "search", "replace"}
}

func (f *FileSearchReplaceTool) SetConfirmator(confirmator Confirmator) {
	f.confirmator = confirmator
}

func (f *FileSearchReplaceTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	search, ok := args["search"].(string)
	if !ok {
		return nil, fmt.Errorf("search parameter must be a string")
	}

	replace, ok := args["replace"].(string)
	if !ok {
		return nil, fmt.Errorf("replace parameter must be a string")
	}

	// Handle optional parameters
	useRegex := false
	if val, exists := args["regex"]; exists {
		if b, ok := val.(bool); ok {
			useRegex = b
		}
	}

	global := true
	if val, exists := args["global"]; exists {
		if b, ok := val.(bool); ok {
			global = b
		}
	}

	caseSensitive := true
	if val, exists := args["case_sensitive"]; exists {
		if b, ok := val.(bool); ok {
			caseSensitive = b
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

	// Read file
	originalBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	content := string(originalBytes)
	
	// Handle newlines in replacement
	replace = strings.ReplaceAll(replace, "\\n", "\n")

	var result string
	var count int

	if useRegex {
		// Regex mode
		var re *regexp.Regexp
		if caseSensitive {
			re, err = regexp.Compile(search)
		} else {
			re, err = regexp.Compile("(?i)" + search)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %v", err)
		}

		if global {
			matches := re.FindAllString(content, -1)
			count = len(matches)
			result = re.ReplaceAllString(content, replace)
		} else {
			match := re.FindString(content)
			if match != "" {
				count = 1
				result = re.ReplaceString(content, replace)
			} else {
				result = content
			}
		}
	} else {
		// Literal text mode
		searchText := search
		if !caseSensitive {
			content = strings.ToLower(content)
			searchText = strings.ToLower(search)
		}

		if global {
			if caseSensitive {
				count = strings.Count(content, search)
				result = strings.ReplaceAll(content, search, replace)
			} else {
				// Case insensitive replacement is more complex
				result = content
				searchLower := strings.ToLower(search)
				for {
					idx := strings.Index(strings.ToLower(result), searchLower)
					if idx == -1 {
						break
					}
					result = result[:idx] + replace + result[idx+len(search):]
					count++
				}
			}
		} else {
			if caseSensitive {
				if strings.Contains(content, search) {
					count = 1
					result = strings.Replace(content, search, replace, 1)
				} else {
					result = content
				}
			} else {
				idx := strings.Index(strings.ToLower(content), strings.ToLower(search))
				if idx != -1 {
					count = 1
					result = content[:idx] + replace + content[idx+len(search):]
				} else {
					result = content
				}
			}
		}
	}

	if count == 0 {
		return fmt.Sprintf("No matches found for '%s' in %s", search, path), nil
	}

	// Ask for confirmation
	if f.confirmator != nil && f.confirmator.ShouldConfirm("search_replace") {
		message := fmt.Sprintf("Replace %d occurrence(s) of '%s' in %s", count, search, path)
		if !f.confirmator.Confirm("search_replace", message, "medium") {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(result), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	return fmt.Sprintf("Successfully replaced %d occurrence(s) of '%s' in %s", count, search, path), nil
}

// FileInsertTool implementation
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
	if f.confirmator != nil && f.confirmator.ShouldConfirm("insert_content") {
		var message string
		switch position {
		case "beginning":
			message = fmt.Sprintf("Insert content at beginning of %s", path)
		case "end":
			message = fmt.Sprintf("Insert content at end of %s", path)
		case "after_line":
			message = fmt.Sprintf("Insert content after line %d in %s", int(lineNumber), path)
		}
		if !f.confirmator.Confirm("insert_content", message, "medium") {
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

// FileManageTool implementation
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
	if f.confirmator != nil && f.confirmator.ShouldConfirm("file_manage") {
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
		
		dangerLevel := "medium"
		if destExists || operation == "move" {
			dangerLevel = "high"
		}
		
		if !f.confirmator.Confirm("file_manage", message, dangerLevel) {
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

// DirectoryManageTool implementation
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
	if d.confirmator != nil && d.confirmator.ShouldConfirm("dir_manage") {
		message := fmt.Sprintf("Create directory %s", relativePath)
		if recursive {
			message += " (with parent directories)"
		}
		if !d.confirmator.Confirm("dir_manage", message, "low") {
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
	if d.confirmator != nil && d.confirmator.ShouldConfirm("dir_manage") {
		message := fmt.Sprintf("Delete directory %s", relativePath)
		if recursive {
			message += " (recursively, including all contents)"
		}
		if !d.confirmator.Confirm("dir_manage", message, "high") {
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

// HttpRequestTool implementation
func (h *HttpRequestTool) Name() string {
	return "http_request"
}

func (h *HttpRequestTool) Description() string {
	return "Make HTTP requests to APIs and web services"
}

func (h *HttpRequestTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"url": map[string]interface{}{
			"type":        "string",
			"description": "Target URL for the HTTP request",
		},
		"method": map[string]interface{}{
			"type":        "string",
			"description": "HTTP method: GET, POST, PUT, DELETE, PATCH (default: GET)",
			"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
		},
		"headers": map[string]interface{}{
			"type":        "object",
			"description": "HTTP headers as key-value pairs",
		},
		"body": map[string]interface{}{
			"type":        "string",
			"description": "Request body (for POST, PUT, PATCH methods)",
		},
		"json": map[string]interface{}{
			"type":        "object",
			"description": "JSON data to send (automatically sets Content-Type: application/json)",
		},
		"timeout": map[string]interface{}{
			"type":        "number",
			"description": "Request timeout in seconds (default: 30)",
		},
	}
}

func (h *HttpRequestTool) RequiredParameters() []string {
	return []string{"url"}
}

func (h *HttpRequestTool) SetConfirmator(confirmator Confirmator) {
	h.confirmator = confirmator
}

func (h *HttpRequestTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter must be a string")
	}

	method := "GET"
	if val, exists := args["method"]; exists {
		if m, ok := val.(string); ok {
			method = strings.ToUpper(m)
		}
	}

	timeout := 30.0
	if val, exists := args["timeout"]; exists {
		if t, ok := val.(float64); ok {
			timeout = t
		}
	}

	// Validate method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[method] {
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	// Ask for confirmation for potentially dangerous requests
	if h.confirmator != nil && h.confirmator.ShouldConfirm("http_request") {
		dangerLevel := "low"
		if method != "GET" && method != "HEAD" && method != "OPTIONS" {
			dangerLevel = "medium"
		}
		
		message := fmt.Sprintf("Make %s request to %s", method, url)
		if !h.confirmator.Confirm("http_request", message, dangerLevel) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Prepare request body
	var reqBody io.Reader
	if val, exists := args["json"]; exists {
		jsonData, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %v", err)
		}
		reqBody = bytes.NewReader(jsonData)
	} else if val, exists := args["body"]; exists {
		if body, ok := val.(string); ok {
			reqBody = strings.NewReader(body)
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	if val, exists := args["headers"]; exists {
		if headers, ok := val.(map[string]interface{}); ok {
			for key, value := range headers {
				if strVal, ok := value.(string); ok {
					req.Header.Set(key, strVal)
				}
			}
		}
	}

	// Set JSON content type if JSON body was provided
	if _, exists := args["json"]; exists {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "RoriCode-HttpTool/1.0")

	// Create client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	// Try to parse JSON response
	var jsonResponse interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		json.Unmarshal(respBody, &jsonResponse)
	}

	result := map[string]interface{}{
		"url":            url,
		"method":         method,
		"status_code":    resp.StatusCode,
		"status":         resp.Status,
		"headers":        responseHeaders,
		"body":           string(respBody),
		"content_length": len(respBody),
	}

	if jsonResponse != nil {
		result["json"] = jsonResponse
	}

	return result, nil
}

// CodeFormatterTool implementation
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
	if c.confirmator != nil && c.confirmator.ShouldConfirm("code_format") {
		operation := "Check"
		if fix {
			operation = "Format"
		}
		message := fmt.Sprintf("%s %s with %s", operation, path, tool)
		if !c.confirmator.Confirm("code_format", message, "medium") {
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

// DataEditTool implementation
func (d *DataEditTool) Name() string {
	return "data_edit"
}

func (d *DataEditTool) Description() string {
	return "Manipulate JSON and YAML files: get, set, delete values by key path"
}

func (d *DataEditTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "File path (relative to current working directory)",
		},
		"format": map[string]interface{}{
			"type":        "string",
			"description": "File format: json or yaml (auto-detected from extension if not provided)",
			"enum":        []string{"json", "yaml", "yml"},
		},
		"operation": map[string]interface{}{
			"type":        "string",
			"description": "Operation: get, set, delete, list",
			"enum":        []string{"get", "set", "delete", "list"},
		},
		"key": map[string]interface{}{
			"type":        "string",
			"description": "Key path (e.g., 'server.port', 'users[0].name', 'config.database.host')",
		},
		"value": map[string]interface{}{
			"type":        "string",
			"description": "Value to set (for set operation). Will be parsed as JSON first, then as string",
		},
		"create_path": map[string]interface{}{
			"type":        "boolean",
			"description": "Create intermediate keys if they don't exist (default: false)",
		},
	}
}

func (d *DataEditTool) RequiredParameters() []string {
	return []string{"path", "operation"}
}

func (d *DataEditTool) SetConfirmator(confirmator Confirmator) {
	d.confirmator = confirmator
}

func (d *DataEditTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter must be a string")
	}

	var key string
	if val, exists := args["key"]; exists {
		if k, ok := val.(string); ok {
			key = k
		}
	}

	var value string
	if val, exists := args["value"]; exists {
		if v, ok := val.(string); ok {
			value = v
		}
	}

	createPath := false
	if val, exists := args["create_path"]; exists {
		if b, ok := val.(bool); ok {
			createPath = b
		}
	}

	// Validate operation
	validOps := map[string]bool{"get": true, "set": true, "delete": true, "list": true}
	if !validOps[operation] {
		return nil, fmt.Errorf("operation must be one of: get, set, delete, list")
	}

	// Key is required for get, set, delete operations
	if (operation == "get" || operation == "set" || operation == "delete") && key == "" {
		return nil, fmt.Errorf("key parameter is required for %s operation", operation)
	}

	// Value is required for set operation
	if operation == "set" && value == "" {
		return nil, fmt.Errorf("value parameter is required for set operation")
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

	// Detect format from file extension if not provided
	format := ""
	if val, exists := args["format"]; exists {
		if f, ok := val.(string); ok {
			format = f
		}
	}
	
	if format == "" {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".json":
			format = "json"
		case ".yaml", ".yml":
			format = "yaml"
		default:
			return nil, fmt.Errorf("unable to detect format from extension %s, please specify format parameter", ext)
		}
	}

	// Ask for confirmation for write operations
	if (operation == "set" || operation == "delete") && d.confirmator != nil && d.confirmator.ShouldConfirm("data_edit") {
		message := fmt.Sprintf("%s key '%s' in %s", strings.Title(operation), key, path)
		if !d.confirmator.Confirm("data_edit", message, "medium") {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Execute operation
	switch operation {
	case "get":
		return d.getValue(fullPath, path, format, key)
	case "set":
		return d.setValue(fullPath, path, format, key, value, createPath)
	case "delete":
		return d.deleteValue(fullPath, path, format, key)
	case "list":
		return d.listKeys(fullPath, path, format)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

func (d *DataEditTool) loadData(fullPath, format string) (map[string]interface{}, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var result map[string]interface{}
	
	switch format {
	case "json":
		err = json.Unmarshal(data, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %v", err)
		}
	case "yaml", "yml":
		// For YAML support, we'll implement a basic parser or return an error
		return nil, fmt.Errorf("YAML support not implemented yet (requires gopkg.in/yaml.v3 dependency)")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return result, nil
}

func (d *DataEditTool) saveData(fullPath, format string, data map[string]interface{}) error {
	var output []byte
	var err error

	switch format {
	case "json":
		output, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
	case "yaml", "yml":
		return fmt.Errorf("YAML support not implemented yet (requires gopkg.in/yaml.v3 dependency)")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	return os.WriteFile(fullPath, output, 0644)
}

func (d *DataEditTool) getValue(fullPath, relativePath, format, keyPath string) (interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", relativePath)
	}

	data, err := d.loadData(fullPath, format)
	if err != nil {
		return err, nil
	}

	value, exists := d.getValueByPath(data, keyPath)
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyPath)
	}

	return map[string]interface{}{
		"operation": "get",
		"path":      relativePath,
		"key":       keyPath,
		"value":     value,
		"exists":    true,
	}, nil
}

func (d *DataEditTool) setValue(fullPath, relativePath, format, keyPath, value string, createPath bool) (interface{}, error) {
	// Parse value (try JSON first, then string)
	var parsedValue interface{}
	if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
		// Not JSON, treat as string
		parsedValue = value
	}

	// Load existing data or create new
	var data map[string]interface{}
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		data = make(map[string]interface{})
	} else {
		var err error
		data, err = d.loadData(fullPath, format)
		if err != nil {
			return nil, err
		}
	}

	// Set the value
	oldValue, existed := d.setValueByPath(data, keyPath, parsedValue, createPath)

	// Save the file
	if err := d.saveData(fullPath, format, data); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	return map[string]interface{}{
		"operation": "set",
		"path":      relativePath,
		"key":       keyPath,
		"value":     parsedValue,
		"old_value": oldValue,
		"existed":   existed,
	}, nil
}

func (d *DataEditTool) deleteValue(fullPath, relativePath, format, keyPath string) (interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", relativePath)
	}

	data, err := d.loadData(fullPath, format)
	if err != nil {
		return nil, err
	}

	deletedValue, existed := d.deleteValueByPath(data, keyPath)
	if !existed {
		return nil, fmt.Errorf("key not found: %s", keyPath)
	}

	// Save the file
	if err := d.saveData(fullPath, format, data); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	return map[string]interface{}{
		"operation":     "delete",
		"path":          relativePath,
		"key":           keyPath,
		"deleted_value": deletedValue,
	}, nil
}

func (d *DataEditTool) listKeys(fullPath, relativePath, format string) (interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", relativePath)
	}

	data, err := d.loadData(fullPath, format)
	if err != nil {
		return nil, err
	}

	keys := d.getAllKeys(data, "")

	return map[string]interface{}{
		"operation": "list",
		"path":      relativePath,
		"keys":      keys,
		"count":     len(keys),
	}, nil
}

// Helper functions for key path navigation
func (d *DataEditTool) getValueByPath(data map[string]interface{}, keyPath string) (interface{}, bool) {
	keys := d.parseKeyPath(keyPath)
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key
			value, exists := current[key]
			return value, exists
		}

		// Intermediate key
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
}

func (d *DataEditTool) setValueByPath(data map[string]interface{}, keyPath string, value interface{}, createPath bool) (interface{}, bool) {
	keys := d.parseKeyPath(keyPath)
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key - set the value
			oldValue, existed := current[key]
			current[key] = value
			return oldValue, existed
		}

		// Intermediate key
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else if createPath {
			// Create intermediate map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		} else {
			return nil, false
		}
	}

	return nil, false
}

func (d *DataEditTool) deleteValueByPath(data map[string]interface{}, keyPath string) (interface{}, bool) {
	keys := d.parseKeyPath(keyPath)
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key - delete it
			value, existed := current[key]
			if existed {
				delete(current, key)
			}
			return value, existed
		}

		// Intermediate key
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
}

func (d *DataEditTool) parseKeyPath(keyPath string) []string {
	// Simple key path parser (supports dot notation: "a.b.c")
	// TODO: Add support for array indices: "a[0].b"
	return strings.Split(keyPath, ".")
}

func (d *DataEditTool) getAllKeys(data map[string]interface{}, prefix string) []string {
	var keys []string

	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		keys = append(keys, fullKey)

		// Recursively get keys from nested objects
		if nested, ok := value.(map[string]interface{}); ok {
			nestedKeys := d.getAllKeys(nested, fullKey)
			keys = append(keys, nestedKeys...)
		}
	}

	return keys
}

// DataProcessTool implementation
func (d *DataProcessTool) Name() string {
	return "data_process"
}

func (d *DataProcessTool) Description() string {
	return "Process CSV and structured data: filter, sort, transform, and analyze data"
}

func (d *DataProcessTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"type":        "string",
			"description": "CSV file path (relative to current working directory)",
		},
		"operation": map[string]interface{}{
			"type":        "string",
			"description": "Operation: filter, sort, transform, stats, head, tail",
			"enum":        []string{"filter", "sort", "transform", "stats", "head", "tail"},
		},
		"output": map[string]interface{}{
			"type":        "string",
			"description": "Output file path (optional, prints to stdout if not provided)",
		},
		"filter": map[string]interface{}{
			"type":        "object",
			"description": "Filter conditions: {\"column\": \"age\", \"operator\": \">\", \"value\": \"18\"}",
		},
		"sort_by": map[string]interface{}{
			"type":        "string",
			"description": "Column name to sort by",
		},
		"sort_order": map[string]interface{}{
			"type":        "string",
			"description": "Sort order: asc or desc (default: asc)",
			"enum":        []string{"asc", "desc"},
		},
		"columns": map[string]interface{}{
			"type":        "array",
			"description": "Column names to select/transform (for transform operation)",
		},
		"limit": map[string]interface{}{
			"type":        "number",
			"description": "Limit number of rows (for head/tail operations)",
		},
		"has_header": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether CSV has header row (default: true)",
		},
	}
}

func (d *DataProcessTool) RequiredParameters() []string {
	return []string{"path", "operation"}
}

func (d *DataProcessTool) SetConfirmator(confirmator Confirmator) {
	d.confirmator = confirmator
}

func (d *DataProcessTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
	}

	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter must be a string")
	}

	// Handle optional parameters
	var output string
	if val, exists := args["output"]; exists {
		if o, ok := val.(string); ok {
			output = o
		}
	}

	hasHeader := true
	if val, exists := args["has_header"]; exists {
		if h, ok := val.(bool); ok {
			hasHeader = h
		}
	}

	limit := 10
	if val, exists := args["limit"]; exists {
		if l, ok := val.(float64); ok {
			limit = int(l)
		}
	}

	// Validate operation
	validOps := map[string]bool{"filter": true, "sort": true, "transform": true, "stats": true, "head": true, "tail": true}
	if !validOps[operation] {
		return nil, fmt.Errorf("operation must be one of: filter, sort, transform, stats, head, tail")
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

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	// Ask for confirmation for write operations
	if output != "" && d.confirmator != nil && d.confirmator.ShouldConfirm("data_process") {
		message := fmt.Sprintf("Process %s and save to %s", path, output)
		if !d.confirmator.Confirm("data_process", message, "low") {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Load CSV data
	data, err := d.loadCSV(fullPath, hasHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to load CSV: %v", err)
	}

	// Execute operation
	var result interface{}
	var processedData [][]string

	switch operation {
	case "filter":
		processedData, result, err = d.filterData(data, args)
	case "sort":
		processedData, result, err = d.sortData(data, args)
	case "transform":
		processedData, result, err = d.transformData(data, args)
	case "stats":
		result, err = d.getStats(data)
	case "head":
		processedData, result, err = d.headTail(data, limit, true)
	case "tail":
		processedData, result, err = d.headTail(data, limit, false)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	if err != nil {
		return nil, err
	}

	// Save to output file if specified
	if output != "" && processedData != nil {
		err = d.saveCSV(filepath.Join(cwd, output), processedData)
		if err != nil {
			return nil, fmt.Errorf("failed to save output: %v", err)
		}
		
		if resultMap, ok := result.(map[string]interface{}); ok {
			resultMap["output_file"] = output
			resultMap["saved"] = true
		}
	}

	return result, nil
}

func (d *DataProcessTool) loadCSV(path string, hasHeader bool) (*CSVData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	data := &CSVData{
		Records:   records,
		HasHeader: hasHeader,
	}

	if hasHeader && len(records) > 0 {
		data.Headers = records[0]
		data.Data = records[1:]
	} else {
		data.Data = records
		// Generate default headers
		if len(records) > 0 {
			for i := 0; i < len(records[0]); i++ {
				data.Headers = append(data.Headers, fmt.Sprintf("col_%d", i))
			}
		}
	}

	return data, nil
}

func (d *DataProcessTool) saveCSV(path string, records [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	return writer.WriteAll(records)
}

func (d *DataProcessTool) filterData(data *CSVData, args map[string]interface{}) ([][]string, interface{}, error) {
	filterMap, exists := args["filter"].(map[string]interface{})
	if !exists {
		return nil, nil, fmt.Errorf("filter parameter is required for filter operation")
	}

	column, ok := filterMap["column"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("filter.column must be a string")
	}

	operator, ok := filterMap["operator"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("filter.operator must be a string")
	}

	value, ok := filterMap["value"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("filter.value must be a string")
	}

	// Find column index
	colIndex := -1
	for i, header := range data.Headers {
		if header == column {
			colIndex = i
			break
		}
	}
	if colIndex == -1 {
		return nil, nil, fmt.Errorf("column not found: %s", column)
	}

	// Filter data
	var filtered [][]string
	if data.HasHeader {
		filtered = append(filtered, data.Headers)
	}

	for _, row := range data.Data {
		if colIndex >= len(row) {
			continue
		}

		cellValue := row[colIndex]
		match, err := d.evaluateCondition(cellValue, operator, value)
		if err != nil {
			return nil, nil, err
		}

		if match {
			filtered = append(filtered, row)
		}
	}

	result := map[string]interface{}{
		"operation":     "filter",
		"total_rows":    len(data.Data),
		"filtered_rows": len(filtered) - 1, // Subtract header if present
		"filter":        filterMap,
	}

	return filtered, result, nil
}

func (d *DataProcessTool) sortData(data *CSVData, args map[string]interface{}) ([][]string, interface{}, error) {
	sortBy, exists := args["sort_by"].(string)
	if !exists {
		return nil, nil, fmt.Errorf("sort_by parameter is required for sort operation")
	}

	sortOrder := "asc"
	if val, exists := args["sort_order"].(string); exists {
		sortOrder = val
	}

	// Find column index
	colIndex := -1
	for i, header := range data.Headers {
		if header == sortBy {
			colIndex = i
			break
		}
	}
	if colIndex == -1 {
		return nil, nil, fmt.Errorf("column not found: %s", sortBy)
	}

	// Sort data
	sortedData := make([][]string, len(data.Data))
	copy(sortedData, data.Data)

	sort.Slice(sortedData, func(i, j int) bool {
		if colIndex >= len(sortedData[i]) || colIndex >= len(sortedData[j]) {
			return false
		}

		val1 := sortedData[i][colIndex]
		val2 := sortedData[j][colIndex]

		// Try numeric comparison first
		if num1, err1 := strconv.ParseFloat(val1, 64); err1 == nil {
			if num2, err2 := strconv.ParseFloat(val2, 64); err2 == nil {
				if sortOrder == "desc" {
					return num1 > num2
				}
				return num1 < num2
			}
		}

		// Fall back to string comparison
		if sortOrder == "desc" {
			return val1 > val2
		}
		return val1 < val2
	})

	// Rebuild records with header
	var result [][]string
	if data.HasHeader {
		result = append(result, data.Headers)
	}
	result = append(result, sortedData...)

	resultInfo := map[string]interface{}{
		"operation":  "sort",
		"sort_by":    sortBy,
		"sort_order": sortOrder,
		"total_rows": len(sortedData),
	}

	return result, resultInfo, nil
}

func (d *DataProcessTool) transformData(data *CSVData, args map[string]interface{}) ([][]string, interface{}, error) {
	columnsInterface, exists := args["columns"]
	if !exists {
		return nil, nil, fmt.Errorf("columns parameter is required for transform operation")
	}

	columns, ok := columnsInterface.([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("columns must be an array")
	}

	// Convert to string slice and find column indices
	var columnNames []string
	var columnIndices []int

	for _, col := range columns {
		colName, ok := col.(string)
		if !ok {
			return nil, nil, fmt.Errorf("all column names must be strings")
		}
		columnNames = append(columnNames, colName)

		// Find column index
		colIndex := -1
		for i, header := range data.Headers {
			if header == colName {
				colIndex = i
				break
			}
		}
		if colIndex == -1 {
			return nil, nil, fmt.Errorf("column not found: %s", colName)
		}
		columnIndices = append(columnIndices, colIndex)
	}

	// Transform data
	var transformed [][]string
	if data.HasHeader {
		transformed = append(transformed, columnNames)
	}

	for _, row := range data.Data {
		var newRow []string
		for _, colIndex := range columnIndices {
			if colIndex < len(row) {
				newRow = append(newRow, row[colIndex])
			} else {
				newRow = append(newRow, "")
			}
		}
		transformed = append(transformed, newRow)
	}

	result := map[string]interface{}{
		"operation":      "transform",
		"selected_columns": columnNames,
		"total_rows":     len(data.Data),
	}

	return transformed, result, nil
}

func (d *DataProcessTool) getStats(data *CSVData) (interface{}, error) {
	stats := map[string]interface{}{
		"operation":    "stats",
		"total_rows":   len(data.Data),
		"total_columns": len(data.Headers),
		"headers":      data.Headers,
		"column_stats": make(map[string]interface{}),
	}

	// Calculate statistics for each column
	for i, header := range data.Headers {
		colStats := d.getColumnStats(data.Data, i)
		stats["column_stats"].(map[string]interface{})[header] = colStats
	}

	return stats, nil
}

func (d *DataProcessTool) headTail(data *CSVData, limit int, isHead bool) ([][]string, interface{}, error) {
	var result [][]string
	if data.HasHeader {
		result = append(result, data.Headers)
	}

	var selectedRows [][]string
	if isHead {
		if limit > len(data.Data) {
			limit = len(data.Data)
		}
		selectedRows = data.Data[:limit]
	} else {
		start := len(data.Data) - limit
		if start < 0 {
			start = 0
		}
		selectedRows = data.Data[start:]
	}

	result = append(result, selectedRows...)

	operation := "head"
	if !isHead {
		operation = "tail"
	}

	resultInfo := map[string]interface{}{
		"operation":      operation,
		"limit":          limit,
		"returned_rows":  len(selectedRows),
		"total_rows":     len(data.Data),
	}

	return result, resultInfo, nil
}

// Helper functions
func (d *DataProcessTool) evaluateCondition(cellValue, operator, value string) (bool, error) {
	switch operator {
	case "=", "==":
		return cellValue == value, nil
	case "!=":
		return cellValue != value, nil
	case ">":
		return d.compareNumeric(cellValue, value, func(a, b float64) bool { return a > b })
	case ">=":
		return d.compareNumeric(cellValue, value, func(a, b float64) bool { return a >= b })
	case "<":
		return d.compareNumeric(cellValue, value, func(a, b float64) bool { return a < b })
	case "<=":
		return d.compareNumeric(cellValue, value, func(a, b float64) bool { return a <= b })
	case "contains":
		return strings.Contains(strings.ToLower(cellValue), strings.ToLower(value)), nil
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(cellValue), strings.ToLower(value)), nil
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(cellValue), strings.ToLower(value)), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

func (d *DataProcessTool) compareNumeric(cellValue, value string, compare func(float64, float64) bool) (bool, error) {
	cellNum, err1 := strconv.ParseFloat(cellValue, 64)
	valueNum, err2 := strconv.ParseFloat(value, 64)
	
	if err1 != nil || err2 != nil {
		// Fall back to string comparison
		return strings.Compare(cellValue, value) > 0, nil
	}
	
	return compare(cellNum, valueNum), nil
}

func (d *DataProcessTool) getColumnStats(data [][]string, colIndex int) map[string]interface{} {
	stats := map[string]interface{}{
		"non_empty_count": 0,
		"empty_count":     0,
	}

	var numericValues []float64
	var allValues []string

	for _, row := range data {
		if colIndex >= len(row) {
			stats["empty_count"] = stats["empty_count"].(int) + 1
			continue
		}

		value := row[colIndex]
		if value == "" {
			stats["empty_count"] = stats["empty_count"].(int) + 1
			continue
		}

		stats["non_empty_count"] = stats["non_empty_count"].(int) + 1
		allValues = append(allValues, value)

		// Try to parse as number
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			numericValues = append(numericValues, num)
		}
	}

	// If we have numeric values, calculate numeric stats
	if len(numericValues) > 0 {
		sort.Float64s(numericValues)
		stats["is_numeric"] = true
		stats["numeric_count"] = len(numericValues)
		stats["min"] = numericValues[0]
		stats["max"] = numericValues[len(numericValues)-1]
		
		// Calculate mean
		sum := 0.0
		for _, v := range numericValues {
			sum += v
		}
		stats["mean"] = sum / float64(len(numericValues))
		
		// Calculate median
		mid := len(numericValues) / 2
		if len(numericValues)%2 == 0 {
			stats["median"] = (numericValues[mid-1] + numericValues[mid]) / 2
		} else {
			stats["median"] = numericValues[mid]
		}
	} else {
		stats["is_numeric"] = false
	}

	// String statistics
	if len(allValues) > 0 {
		// Unique values count
		unique := make(map[string]bool)
		for _, v := range allValues {
			unique[v] = true
		}
		stats["unique_count"] = len(unique)
	}

	return stats
}

// CSVData holds parsed CSV data
type CSVData struct {
	Records   [][]string
	Headers   []string
	Data      [][]string
	HasHeader bool
}

// RegisterBuiltinTools registers all builtin tools to a registry
func RegisterBuiltinTools(registry *Registry) {
	registry.Register(&ShellTool{})
	registry.Register(&CurrentTimeTool{})
	registry.Register(&FileReadTool{})
	registry.Register(&FileEditTool{})
	registry.Register(&FileCreationTool{})
	registry.Register(&FileReplaceLinesTool{})
	registry.Register(&FileSearchReplaceTool{})
	registry.Register(&FileInsertTool{})
	registry.Register(&FileManageTool{})
	registry.Register(&DirectoryManageTool{})
	registry.Register(&HttpRequestTool{})
	registry.Register(&CodeFormatterTool{})
	registry.Register(&DataEditTool{})
	registry.Register(&DataProcessTool{})
}
