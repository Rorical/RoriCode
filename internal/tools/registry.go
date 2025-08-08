package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Tool represents a function that can be called by the AI
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{} // JSON schema for parameters
	RequiredParameters() []string       // List of required parameter names
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// ToolCall represents a tool call request
type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Function string                 `json:"function,omitempty"` // For OpenAI compatibility
	Args     map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	CallID string      `json:"call_id"`
	Name   string      `json:"name"`
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// GetTool retrieves a tool by name
func (r *Registry) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, exists := r.tools[name]
	return tool, exists
}

// ListTools returns all registered tools
func (r *Registry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetOpenAIToolsSpec returns OpenAI-compatible tool specifications
func (r *Registry) GetOpenAIToolsSpec() []map[string]interface{} {
	tools := r.ListTools()
	specs := make([]map[string]interface{}, len(tools))
	
	for i, tool := range tools {
		specs[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": tool.Parameters(),
					"required":   tool.RequiredParameters(),
				},
			},
		}
	}
	
	return specs
}

// ExecuteAsync executes a tool call asynchronously
func (r *Registry) ExecuteAsync(ctx context.Context, call ToolCall, resultChan chan<- ToolResult) {
	go func() {
		defer close(resultChan)
		
		tool, exists := r.GetTool(call.Name)
		if !exists {
			resultChan <- ToolResult{
				CallID: call.ID,
				Name:   call.Name,
				Error:  fmt.Sprintf("tool '%s' not found", call.Name),
			}
			return
		}
		
		result, err := tool.Execute(ctx, call.Args)
		toolResult := ToolResult{
			CallID: call.ID,
			Name:   call.Name,
			Result: result,
		}
		
		if err != nil {
			toolResult.Error = err.Error()
		}
		
		resultChan <- toolResult
	}()
}


// ParseToolCallFromJSON parses a tool call from JSON string
func ParseToolCallFromJSON(data string) (*ToolCall, error) {
	var call ToolCall
	err := json.Unmarshal([]byte(data), &call)
	return &call, err
}

// ToJSONString converts a tool result to JSON string
func (tr *ToolResult) ToJSONString() string {
	data, _ := json.Marshal(tr)
	return string(data)
}