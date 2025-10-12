package rack

import (
	"context"
)

// Agent represents an AI agent that can execute tools
type Agent interface {
	AddTool(tool Tool) error
	Execute(ctx context.Context, prompt string) (*AgentResponse, error)
	Stream(ctx context.Context, prompt string) (<-chan *AgentEvent, error)
}

// AgentResponse represents the response from an agent execution
type AgentResponse struct {
	Content   string            `json:"content"`
	ToolCalls []ToolCall  `json:"tool_calls"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

// AgentEvent represents a streaming event from an agent
type AgentEvent struct {
	Type    string           `json:"type"` // "content", "tool_call", "done", "error"
	Content string           `json:"content,omitempty"`
	Tool    *ToolCall  `json:"tool,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// Event types for streaming
const (
	EventTypeContent  = "content"
	EventTypeToolCall = "tool_call"
	EventTypeDone     = "done"
	EventTypeError    = "error"
)

// AgentConfig holds configuration for an agent
type AgentConfig struct {
	Model         string  `json:"model"`
	MaxTokens     int     `json:"max_tokens"`
	Temperature   float64 `json:"temperature"`
	MaxIterations int     `json:"max_iterations"` // Maximum agent loop iterations (default: 10)
}
