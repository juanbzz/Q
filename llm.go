package q

import (
	"context"
	"encoding/json"
	"strings"
)

// LLMProvider interface for different LLM implementations
type LLMProvider interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
	Stream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan *StreamEvent, error)
}

// Message represents a conversation message
type Message struct {
	Role       string        `json:"role"` // "system", "user", "assistant", "tool"
	Content    string        `json:"content"`
	ToolCallID string        `json:"tool_call_id,omitempty"` // For tool responses
	ToolCalls  []LLMToolCall `json:"tool_calls,omitempty"`   // For assistant messages with tool calls
}

// LLMResponse represents a response from an LLM
type LLMResponse struct {
	Content   string        `json:"content"`
	ToolCalls []LLMToolCall `json:"tool_calls,omitempty"`
	Usage     *Usage        `json:"usage,omitempty"`
	Model     string        `json:"model"`
}

// LLMToolCall represents a tool call from the LLM
type LLMToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamEvent represents a streaming event from an LLM
type StreamEvent struct {
	Type     string       `json:"type"` // "content", "tool_call", "done", "error"
	Content  string       `json:"content,omitempty"`
	ToolCall *LLMToolCall `json:"tool_call,omitempty"`
	Error    string       `json:"error,omitempty"`
}

// ToolDefinition represents a tool definition for the LLM
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolToDefinition converts a Tool to an LLM ToolDefinition
func ToolToDefinition(tool Tool) ToolDefinition {
	schema := tool.Schema()
	parameters := map[string]interface{}{
		"type":       schema.Type,
		"properties": schema.Properties,
	}
	if len(schema.Required) > 0 {
		parameters["required"] = schema.Required
	}

	return ToolDefinition{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters:  parameters,
	}
}

// MockLLMProvider for testing
type MockLLMProvider struct {
	responses []string
	index     int
}

func NewMockProvider(responses []string) *MockLLMProvider {
	return &MockLLMProvider{
		responses: responses,
		index:     0,
	}
}

func (m *MockLLMProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error) {
	if m.index >= len(m.responses) {
		return &LLMResponse{
			Content: "I don't have any more responses.",
			Model:   "mock-model",
		}, nil
	}

	response := m.responses[m.index]
	m.index++

	var toolCalls []LLMToolCall
	if strings.Contains(response, "TOOL_CALL:") {
		parts := strings.Split(response, "TOOL_CALL:")
		if len(parts) > 1 {
			toolName := strings.TrimSpace(parts[1])
			toolCalls = append(toolCalls, LLMToolCall{
				ID:        "test-call-1",
				Name:      toolName,
				Arguments: json.RawMessage(`{}`),
			})
		}
	}

	return &LLMResponse{
		Content:   response,
		ToolCalls: toolCalls,
		Model:     "mock-model",
	}, nil
}

func (m *MockLLMProvider) Stream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan *StreamEvent, error) {
	ch := make(chan *StreamEvent, 1)
	go func() {
		defer close(ch)
		response, err := m.Chat(ctx, messages, tools)
		if err != nil {
			ch <- &StreamEvent{Type: "error", Error: err.Error()}
			return
		}
		ch <- &StreamEvent{Type: "content", Content: response.Content}
		ch <- &StreamEvent{Type: "done"}
	}()
	return ch, nil
}
