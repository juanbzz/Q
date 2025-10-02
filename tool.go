package rack

import (
	"context"
	"encoding/json"
)

// Tool represents a single capability that can be executed by an agent
type Tool interface {
	Name() string
	Description() string
	Schema() Schema
	Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}

// Schema defines the input parameters for a tool
type Schema struct {
	Type       string                 `json:"type"`
	Properties map[string]SchemaField `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// SchemaField defines a single parameter in the tool schema
type SchemaField struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// ToolResult represents the output from a tool execution
type ToolResult struct {
	Content  string         `json:"content"`
	Error    string         `json:"error,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToolCall represents a tool invocation with its input
type ToolCall struct {
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}
