package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

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
	relativePath := path

	// Check if path exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", relativePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %v", err)
	}

	// Handle directory
	if info.IsDir() {
		return f.listDirectory(fullPath, relativePath)
	}

	// Handle file
	if !f.isTextFile(fullPath) {
		return map[string]interface{}{
			"path":    relativePath,
			"type":    "binary",
			"size":    info.Size(),
			"error":   "Cannot read binary file",
			"message": "File appears to be binary and cannot be displayed as text",
		}, nil
	}

	// Extract parameters
	var linesFrom, linesTo, regexMatch int
	var regexPattern string
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

// listDirectory lists directory contents
func (f *FileReadTool) listDirectory(fullPath, relativePath string) (interface{}, error) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var files []map[string]interface{}
	var dirs []map[string]interface{}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := map[string]interface{}{
			"name":     entry.Name(),
			"size":     info.Size(),
			"modified": info.ModTime().Format("2006-01-02 15:04:05"),
		}

		if entry.IsDir() {
			item["type"] = "directory"
			dirs = append(dirs, item)
		} else {
			item["type"] = f.getFileType(entry.Name())
			files = append(files, item)
		}
	}

	return map[string]interface{}{
		"path":        relativePath,
		"type":        "directory",
		"directories": dirs,
		"files":       files,
		"total_dirs":  len(dirs),
		"total_files": len(files),
	}, nil
}

// searchWithRegex performs regex search with context
func (f *FileReadTool) searchWithRegex(file *os.File, relativePath, pattern string, matchNum, contextLines, maxLines int) (interface{}, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	if matchNum < 1 {
		matchNum = 1
	}

	scanner := bufio.NewScanner(file)
	var allLines []string
	lineNum := 0

	// Read all lines first
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
		lineNum++
		if lineNum > 10000 { // Prevent memory issues
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Find matches
	var matches []map[string]interface{}
	matchCount := 0

	for i, line := range allLines {
		if regex.MatchString(line) {
			matchCount++
			if matchCount == matchNum {
				// Extract context
				start := i - contextLines
				end := i + contextLines + 1

				if start < 0 {
					start = 0
				}
				if end > len(allLines) {
					end = len(allLines)
				}

				contextLinesData := make([]map[string]interface{}, 0)
				for j := start; j < end; j++ {
					contextLinesData = append(contextLinesData, map[string]interface{}{
						"line_number": j + 1,
						"content":     allLines[j],
						"is_match":    j == i,
					})
				}

				matches = append(matches, map[string]interface{}{
					"match_number": matchCount,
					"line_number":  i + 1,
					"content":      line,
					"context":      contextLinesData,
				})
				break
			}
		}
	}

	return map[string]interface{}{
		"path":         relativePath,
		"type":         "regex_search",
		"pattern":      pattern,
		"match_number": matchNum,
		"total_lines":  len(allLines),
		"matches":      matches,
		"found":        len(matches) > 0,
	}, nil
}

// readLineRange reads a specific range of lines
func (f *FileReadTool) readLineRange(file *os.File, relativePath string, linesFrom, linesTo, maxLines int) (interface{}, error) {
	scanner := bufio.NewScanner(file)
	var lines []map[string]interface{}
	lineNum := 0
	
	if linesFrom < 1 {
		linesFrom = 1
	}
	if linesTo < 1 {
		linesTo = linesFrom + maxLines - 1
	}

	for scanner.Scan() {
		lineNum++
		
		if lineNum < linesFrom {
			continue
		}
		if lineNum > linesTo {
			break
		}
		if len(lines) >= maxLines {
			break
		}

		lines = append(lines, map[string]interface{}{
			"line_number": lineNum,
			"content":     scanner.Text(),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	return map[string]interface{}{
		"path":       relativePath,
		"type":       "line_range",
		"lines_from": linesFrom,
		"lines_to":   linesTo,
		"lines":      lines,
		"count":      len(lines),
	}, nil
}

// readDefaultLines reads the first N lines of the file
func (f *FileReadTool) readDefaultLines(file *os.File, relativePath string, maxLines int) (interface{}, error) {
	scanner := bufio.NewScanner(file)
	var lines []map[string]interface{}
	lineNum := 0

	for scanner.Scan() && len(lines) < maxLines {
		lineNum++
		lines = append(lines, map[string]interface{}{
			"line_number": lineNum,
			"content":     scanner.Text(),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Get file info
	info, _ := file.Stat()
	
	return map[string]interface{}{
		"path":       relativePath,
		"type":       f.getFileType(relativePath),
		"size":       info.Size(),
		"lines":      lines,
		"count":      len(lines),
		"truncated":  len(lines) == maxLines,
	}, nil
}