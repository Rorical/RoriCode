package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HttpRequestTool makes HTTP requests
type HttpRequestTool struct {
	confirmator Confirmator
}

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
	if h.confirmator != nil {
		dangerous := method != "GET" && method != "HEAD" && method != "OPTIONS"
		operation := "HTTP Request"
		message := fmt.Sprintf("Make %s request to %s", method, url)
		if !h.confirmator.RequestConfirmation(operation, message, dangerous) {
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