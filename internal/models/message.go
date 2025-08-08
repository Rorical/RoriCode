package models

type MessageType int

const (
	User MessageType = iota
	Assistant
	Program
	ToolCall
	ToolResult
)

type Message struct {
	Content string
	Type    MessageType
	// Additional fields for tool calls and results
	ToolCallID   string // For ToolCall and ToolResult messages
	ToolName     string // For ToolCall and ToolResult messages
	ToolArgs     string // For ToolCall messages (JSON string)
}
