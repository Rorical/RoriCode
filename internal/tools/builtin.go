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

// RegisterBuiltinTools registers all builtin tools to a registry
func RegisterBuiltinTools(registry *Registry) {
	registry.Register(&ShellTool{})
	registry.Register(&CurrentTimeTool{})
	registry.Register(&FileReadTool{})
}
