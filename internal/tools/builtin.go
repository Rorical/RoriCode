package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// RegisterBuiltinTools registers all builtin tools to a registry
func RegisterBuiltinTools(registry *Registry) {
	registry.Register(&ShellTool{})
	registry.Register(&CurrentTimeTool{})
	registry.Register(&FileReadTool{})
	registry.Register(&FileEditTool{})
	registry.Register(&FileCreationTool{})
}
