package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileSearchReplaceTool performs search and replace operations
type FileSearchReplaceTool struct {
	confirmator Confirmator
}

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
				result = re.ReplaceAllString(content, replace)
			} else {
				result = content
			}
		}
	} else {
		// Literal text mode
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
	if f.confirmator != nil {
		message := fmt.Sprintf("Replace %d occurrence(s) of '%s' in %s", count, search, path)
		operation := "Search and replace"
		dangerous := true // File modification is potentially dangerous
		if !f.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(result), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	return fmt.Sprintf("Successfully replaced %d occurrence(s) of '%s' in %s", count, search, path), nil
}