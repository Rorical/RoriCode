package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileEditTool edits files using git diff format
type FileEditTool struct {
	confirmator Confirmator
}

// DiffHunk represents a single diff hunk
type DiffHunk struct {
	OldStart int      // Starting line in original file (1-based)
	OldCount int      // Number of lines in original file  
	NewStart int      // Starting line in new file (1-based)
	NewCount int      // Number of lines in new file
	Lines    []string // Diff lines with +/- prefixes
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

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s (use create_file tool to create new files)", path)
	}

	// Ask for confirmation
	if f.confirmator != nil {
		if !f.confirmator.RequestConfirmation("Edit file", fmt.Sprintf("Apply diff to %s", path), true) {
			return map[string]interface{}{
				"output":  "User aborted file edit operation",
				"aborted": true,
			}, nil
		}
	}

	// Read original file
	originalBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read original file: %v", err)
	}

	originalLines := strings.Split(string(originalBytes), "\n")

	// Apply diff
	modifiedLines, err := f.applyDiff(originalLines, diff)
	if err != nil {
		return nil, err
	}

	// Write modified content back to file
	modifiedContent := strings.Join(modifiedLines, "\n")
	if err := os.WriteFile(fullPath, []byte(modifiedContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write modified file: %v", err)
	}

	return map[string]interface{}{
		"path":           path,
		"operation":      "success",
		"original_lines": len(originalLines),
		"modified_lines": len(modifiedLines),
		"diff_applied":   true,
	}, nil
}

// applyDiff applies a unified diff to the original content
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
			// Verify that the line to delete matches
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
			// Add this line
			result = append(result, line[1:])
			// Don't increment contentIdx as we're inserting
			
		default:
			// Treat as context line
			if contentIdx < len(content) {
				result = append(result, content[contentIdx])
				contentIdx++
			}
		}
	}
	
	// Copy remaining lines after the hunk
	result = append(result, content[contentIdx:]...)
	
	return result, nil
}