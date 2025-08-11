package tools

import (
	"context"
	"time"
)

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