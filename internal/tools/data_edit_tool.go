package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DataEditTool handles JSON and YAML data manipulation
type DataEditTool struct {
	confirmator Confirmator
}

func (d *DataEditTool) Name() string {
	return "data_edit"
}

func (d *DataEditTool) Description() string {
	return "Edit JSON and YAML files: read, modify, validate, and format structured data"
}

func (d *DataEditTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"operation": map[string]interface{}{
			"type":        "string",
			"description": "Operation to perform: 'read', 'write', 'validate', 'format', 'query', 'merge'",
			"enum":        []string{"read", "write", "validate", "format", "query", "merge"},
		},
		"path": map[string]interface{}{
			"type":        "string",
			"description": "File path (relative to current working directory)",
		},
		"format": map[string]interface{}{
			"type":        "string",
			"description": "Data format: 'json' or 'yaml' (auto-detected if not specified)",
			"enum":        []string{"json", "yaml"},
		},
		"data": map[string]interface{}{
			"type":        "object",
			"description": "Data to write (for write and merge operations)",
		},
		"query": map[string]interface{}{
			"type":        "string",
			"description": "JSONPath or YAML path query (for query operation)",
		},
		"indent": map[string]interface{}{
			"type":        "number",
			"description": "Indentation level for formatting (default: 2)",
		},
	}
}

func (d *DataEditTool) RequiredParameters() []string {
	return []string{"operation", "path"}
}

func (d *DataEditTool) SetConfirmator(confirmator Confirmator) {
	d.confirmator = confirmator
}

func (d *DataEditTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter must be a string")
	}

	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter must be a string")
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

	// Determine format
	format := ""
	if val, exists := args["format"]; exists {
		if f, ok := val.(string); ok {
			format = strings.ToLower(f)
		}
	}
	if format == "" {
		format = d.detectFormat(path)
	}

	switch operation {
	case "read":
		return d.readData(fullPath, path, format)
	case "write":
		data := args["data"]
		return d.writeData(fullPath, path, format, data)
	case "validate":
		return d.validateData(fullPath, path, format)
	case "format":
		indent := 2
		if val, exists := args["indent"]; exists {
			if i, ok := val.(float64); ok {
				indent = int(i)
			}
		}
		return d.formatData(fullPath, path, format, indent)
	case "query":
		query, ok := args["query"].(string)
		if !ok {
			return nil, fmt.Errorf("query parameter must be a string for query operation")
		}
		return d.queryData(fullPath, path, format, query)
	case "merge":
		data := args["data"]
		return d.mergeData(fullPath, path, format, data)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

func (d *DataEditTool) detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "json" // Default to JSON
	}
}

func (d *DataEditTool) readData(fullPath, relativePath, format string) (interface{}, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var parsed interface{}
	switch format {
	case "json":
		err = json.Unmarshal(data, &parsed)
	case "yaml":
		err = yaml.Unmarshal(data, &parsed)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", format, err)
	}

	return map[string]interface{}{
		"operation": "read",
		"path":      relativePath,
		"format":    format,
		"data":      parsed,
		"size":      len(data),
	}, nil
}

func (d *DataEditTool) writeData(fullPath, relativePath, format string, data interface{}) (interface{}, error) {
	if data == nil {
		return nil, fmt.Errorf("data parameter is required for write operation")
	}

	// Ask for confirmation
	if d.confirmator != nil {
		operation := "Write data"
		message := fmt.Sprintf("Write %s data to %s", format, relativePath)
		dangerous := true // Writing files is potentially dangerous
		if !d.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	var output []byte
	var err error

	switch format {
	case "json":
		output, err = json.MarshalIndent(data, "", "  ")
	case "yaml":
		output, err = yaml.Marshal(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %v", format, err)
	}

	err = os.WriteFile(fullPath, output, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	return map[string]interface{}{
		"operation": "write",
		"path":      relativePath,
		"format":    format,
		"size":      len(output),
		"success":   true,
	}, nil
}

func (d *DataEditTool) validateData(fullPath, relativePath, format string) (interface{}, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var parsed interface{}
	var validationErr error

	switch format {
	case "json":
		validationErr = json.Unmarshal(data, &parsed)
	case "yaml":
		validationErr = yaml.Unmarshal(data, &parsed)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	result := map[string]interface{}{
		"operation": "validate",
		"path":      relativePath,
		"format":    format,
		"valid":     validationErr == nil,
		"size":      len(data),
	}

	if validationErr != nil {
		result["error"] = validationErr.Error()
	}

	return result, nil
}

func (d *DataEditTool) formatData(fullPath, relativePath, format string, indent int) (interface{}, error) {
	// Ask for confirmation
	if d.confirmator != nil {
		operation := "Format data"
		message := fmt.Sprintf("Format %s file %s", format, relativePath)
		dangerous := true // Modifying files is potentially dangerous
		if !d.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var parsed interface{}
	switch format {
	case "json":
		err = json.Unmarshal(data, &parsed)
	case "yaml":
		err = yaml.Unmarshal(data, &parsed)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", format, err)
	}

	var output []byte
	switch format {
	case "json":
		indentStr := strings.Repeat(" ", indent)
		output, err = json.MarshalIndent(parsed, "", indentStr)
	case "yaml":
		output, err = yaml.Marshal(parsed)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to format %s: %v", format, err)
	}

	err = os.WriteFile(fullPath, output, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write formatted file: %v", err)
	}

	return map[string]interface{}{
		"operation": "format",
		"path":      relativePath,
		"format":    format,
		"old_size":  len(data),
		"new_size":  len(output),
		"success":   true,
	}, nil
}

func (d *DataEditTool) queryData(fullPath, relativePath, format, query string) (interface{}, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var parsed interface{}
	switch format {
	case "json":
		err = json.Unmarshal(data, &parsed)
	case "yaml":
		err = yaml.Unmarshal(data, &parsed)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", format, err)
	}

	// Simple query implementation (could be enhanced with JSONPath library)
	result := d.simpleQuery(parsed, query)

	return map[string]interface{}{
		"operation": "query",
		"path":      relativePath,
		"format":    format,
		"query":     query,
		"result":    result,
	}, nil
}

func (d *DataEditTool) mergeData(fullPath, relativePath, format string, newData interface{}) (interface{}, error) {
	if newData == nil {
		return nil, fmt.Errorf("data parameter is required for merge operation")
	}

	// Ask for confirmation
	if d.confirmator != nil {
		operation := "Merge data"
		message := fmt.Sprintf("Merge data into %s file %s", format, relativePath)
		dangerous := true // Modifying files is potentially dangerous
		if !d.confirmator.RequestConfirmation(operation, message, dangerous) {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Read existing data
	existingData := make(map[string]interface{})
	if _, err := os.Stat(fullPath); err == nil {
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read existing file: %v", err)
		}

		switch format {
		case "json":
			err = json.Unmarshal(data, &existingData)
		case "yaml":
			err = yaml.Unmarshal(data, &existingData)
		default:
			return nil, fmt.Errorf("unsupported format: %s", format)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to parse existing %s: %v", format, err)
		}
	}

	// Merge data
	if newDataMap, ok := newData.(map[string]interface{}); ok {
		for key, value := range newDataMap {
			existingData[key] = value
		}
	} else {
		existingData["merged_data"] = newData
	}

	// Write merged data
	var output []byte
	var err error
	switch format {
	case "json":
		output, err = json.MarshalIndent(existingData, "", "  ")
	case "yaml":
		output, err = yaml.Marshal(existingData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged %s: %v", format, err)
	}

	err = os.WriteFile(fullPath, output, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write merged file: %v", err)
	}

	return map[string]interface{}{
		"operation": "merge",
		"path":      relativePath,
		"format":    format,
		"size":      len(output),
		"success":   true,
	}, nil
}

func (d *DataEditTool) simpleQuery(data interface{}, query string) interface{} {
	// Simple query implementation - just return the data for now
	// This could be enhanced with a proper JSONPath or YAML path library
	if query == "." || query == "" {
		return data
	}

	// Basic dot notation support for maps
	if dataMap, ok := data.(map[string]interface{}); ok {
		parts := strings.Split(query, ".")
		current := dataMap
		for _, part := range parts {
			if part == "" {
				continue
			}
			if val, exists := current[part]; exists {
				if nextMap, ok := val.(map[string]interface{}); ok {
					current = nextMap
				} else {
					return val
				}
			} else {
				return nil
			}
		}
		return current
	}

	return data
}