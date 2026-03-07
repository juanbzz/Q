package q

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// LLMProvider interface for different LLM implementations
type LLMProvider interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
	Stream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan *StreamEvent, error)
}

// Message represents a conversation message
type Message struct {
	Role       string `json:"role"` // "system", "user", "assistant", "tool"
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"` // For tool responses
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

// DefaultAgent implements the Agent interface
type DefaultAgent struct {
	provider LLMProvider
	registry ToolRegistry
	messages []Message
	config   AgentConfig
}

// NewAgent creates a new agent with the given provider and registry
func NewAgent(provider LLMProvider, registry ToolRegistry, config AgentConfig) *DefaultAgent {
	return &DefaultAgent{
		provider: provider,
		registry: registry,
		config:   config,
		messages: []Message{},
	}
}

// AddTool adds a tool to the agent's registry
func (a *DefaultAgent) AddTool(tool Tool) error {
	return a.registry.Register(tool)
}

// Execute runs the agent with the given prompt
func (a *DefaultAgent) Execute(ctx context.Context, prompt string) (*AgentResponse, error) {
	// Add user message
	a.messages = append(a.messages, Message{
		Role:    "user",
		Content: prompt,
	})

	// Prepare tool definitions
	tools := a.registry.List()
	toolDefs := make([]ToolDefinition, len(tools))
	for i, tool := range tools {
		toolDefs[i] = ToolToDefinition(tool)
	}

	// Main agent loop
	maxIterations := a.config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10 // Default to 10 if not configured
	}
	var finalResponse *AgentResponse
	var allExecutedCalls []ToolCall

	for i := 0; i < maxIterations; i++ {
		// Get LLM response
		response, err := a.provider.Chat(ctx, a.messages, toolDefs)
		if err != nil {
			return nil, fmt.Errorf("LLM error: %w", err)
		}

		// If no tool calls, we're done - add content and return
		if len(response.ToolCalls) == 0 {
			// Only add assistant message when there are no tool calls (final response)
			a.messages = append(a.messages, Message{
				Role:    "assistant",
				Content: response.Content,
			})

			finalResponse = &AgentResponse{
				Content:   response.Content,
				ToolCalls: allExecutedCalls,
				Metadata: map[string]any{
					"iterations": i + 1,
					"usage":      response.Usage,
				},
			}
			break
		}

		// If there are tool calls, add assistant message with empty content
		// This prevents mixed content+tool_calls from polluting conversation history
		a.messages = append(a.messages, Message{
			Role:    "assistant",
			Content: "",
		})

		// Execute tool calls
		for _, toolCall := range response.ToolCalls {
			result, err := a.registry.Execute(ctx, toolCall.Name, toolCall.Arguments)
			if err != nil {
				return nil, fmt.Errorf("tool execution error: %w", err)
			}

			allExecutedCalls = append(allExecutedCalls, ToolCall{
				Name:  toolCall.Name,
				Input: toolCall.Arguments,
			})

			// Add tool result message
			toolResultContent := result.Content
			if result.Error != "" {
				toolResultContent = fmt.Sprintf("Error: %s", result.Error)
			}

			a.messages = append(a.messages, Message{
				Role:       "tool",
				Content:    toolResultContent,
				ToolCallID: toolCall.ID,
			})
		}

		// Continue loop to get next response
	}

	if finalResponse == nil {
		return nil, fmt.Errorf("max iterations reached without completion")
	}

	return finalResponse, nil
}

// Stream provides streaming execution (simplified implementation)
func (a *DefaultAgent) Stream(ctx context.Context, prompt string) (<-chan *AgentEvent, error) {
	eventChan := make(chan *AgentEvent, 10)

	go func() {
		defer close(eventChan)

		response, err := a.Execute(ctx, prompt)
		if err != nil {
			eventChan <- &AgentEvent{
				Type:  EventTypeError,
				Error: err.Error(),
			}
			return
		}

		// Send content
		if response.Content != "" {
			eventChan <- &AgentEvent{
				Type:    EventTypeContent,
				Content: response.Content,
			}
		}

		// Send tool calls
		for _, toolCall := range response.ToolCalls {
			eventChan <- &AgentEvent{
				Type: EventTypeToolCall,
				Tool: &toolCall,
			}
		}

		eventChan <- &AgentEvent{Type: EventTypeDone}
	}()

	return eventChan, nil
}

// MockLLMProvider for testing
type MockLLMProvider struct {
	responses []string
	index     int
}

// NewMockProvider creates a new mock LLM provider
func NewMockProvider(responses []string) *MockLLMProvider {
	return &MockLLMProvider{
		responses: responses,
		index:     0,
	}
}

// Chat implements LLMProvider interface
func (m *MockLLMProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error) {
	if m.index >= len(m.responses) {
		return &LLMResponse{
			Content: "I don't have any more responses.",
			Model:   "mock-model",
		}, nil
	}

	response := m.responses[m.index]
	m.index++

	// Simple tool call detection for testing
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

// Stream implements LLMProvider interface
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
