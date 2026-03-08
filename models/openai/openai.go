package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/juanbzz/q"
	"github.com/tmc/langchaingo/llms"
	lcOpenAI "github.com/tmc/langchaingo/llms/openai"
)

type Config struct {
	APIKey  string
	BaseURL string
}

type Provider struct {
	client llms.Model
	model  string
}

func New(model string, cfg Config) (*Provider, error) {
	opts := []lcOpenAI.Option{
		lcOpenAI.WithModel(model),
	}
	if cfg.APIKey != "" {
		opts = append(opts, lcOpenAI.WithToken(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, lcOpenAI.WithBaseURL(cfg.BaseURL))
	}

	client, err := lcOpenAI.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &Provider{client: client, model: model}, nil
}

func (p *Provider) Chat(ctx context.Context, messages []q.Message, tools []q.ToolDefinition) (*q.LLMResponse, error) {
	llmMessages := make([]llms.MessageContent, 0, len(messages))

	for _, msg := range messages {
		var msgType llms.ChatMessageType
		switch msg.Role {
		case "system":
			msgType = llms.ChatMessageTypeSystem
		case "user":
			msgType = llms.ChatMessageTypeHuman
		case "assistant":
			msgType = llms.ChatMessageTypeAI
		case "tool":
			msgType = llms.ChatMessageTypeTool
		default:
			continue
		}

		var parts []llms.ContentPart

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			if msg.Content != "" {
				parts = append(parts, llms.TextPart(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				parts = append(parts, llms.ToolCall{
					ID:   tc.ID,
					Type: "function",
					FunctionCall: &llms.FunctionCall{
						Name:      tc.Name,
						Arguments: string(tc.Arguments),
					},
				})
			}
		} else if msg.Role == "tool" && msg.ToolCallID != "" {
			parts = []llms.ContentPart{llms.ToolCallResponse{
				ToolCallID: msg.ToolCallID,
				Content:    msg.Content,
			}}
		} else {
			parts = []llms.ContentPart{llms.TextPart(msg.Content)}
		}

		llmMessages = append(llmMessages, llms.MessageContent{
			Role:  msgType,
			Parts: parts,
		})
	}

	var callOpts []llms.CallOption
	if len(tools) > 0 {
		lcTools := make([]llms.Tool, 0, len(tools))
		for _, t := range tools {
			lcTools = append(lcTools, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			})
		}
		callOpts = append(callOpts, llms.WithTools(lcTools))
	}

	resp, err := p.client.GenerateContent(ctx, llmMessages, callOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from model")
	}

	choice := resp.Choices[0]

	var toolCalls []q.LLMToolCall
	if choice.ToolCalls != nil {
		for _, tc := range choice.ToolCalls {
			args := json.RawMessage(tc.FunctionCall.Arguments)
			toolCalls = append(toolCalls, q.LLMToolCall{
				ID:        tc.ID,
				Name:      tc.FunctionCall.Name,
				Arguments: args,
			})
		}
	}

	// Fallback: parse TOOL_CALL: from text if no native tool calls
	if len(toolCalls) == 0 && strings.Contains(choice.Content, "TOOL_CALL:") {
		lines := strings.Split(choice.Content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "TOOL_CALL:") {
				parts := strings.Split(line, "TOOL_CALL:")
				if len(parts) > 1 {
					toolName := strings.TrimSpace(parts[1])
					args := json.RawMessage(`{}`)
					if i+1 < len(lines) {
						nextLine := strings.TrimSpace(lines[i+1])
						if strings.HasPrefix(nextLine, "{") {
							args = json.RawMessage(nextLine)
						}
					}
					toolCalls = append(toolCalls, q.LLMToolCall{
						ID:        fmt.Sprintf("call_%d", len(toolCalls)+1),
						Name:      toolName,
						Arguments: args,
					})
				}
			}
		}
	}

	var usage *q.Usage
	if info := choice.GenerationInfo; info != nil {
		usage = &q.Usage{}
		if v, ok := info["PromptTokens"].(int); ok {
			usage.PromptTokens = v
		}
		if v, ok := info["CompletionTokens"].(int); ok {
			usage.CompletionTokens = v
		}
		if v, ok := info["TotalTokens"].(int); ok {
			usage.TotalTokens = v
		}
	}

	return &q.LLMResponse{
		Content:   choice.Content,
		ToolCalls: toolCalls,
		Usage:     usage,
		Model:     p.model,
	}, nil
}

func (p *Provider) Stream(ctx context.Context, messages []q.Message, tools []q.ToolDefinition) (<-chan *q.StreamEvent, error) {
	ch := make(chan *q.StreamEvent, 1)
	go func() {
		defer close(ch)
		response, err := p.Chat(ctx, messages, tools)
		if err != nil {
			ch <- &q.StreamEvent{Type: "error", Error: err.Error()}
			return
		}
		ch <- &q.StreamEvent{Type: "content", Content: response.Content}
		for _, tc := range response.ToolCalls {
			tc := tc
			ch <- &q.StreamEvent{Type: "tool_call", ToolCall: &tc}
		}
		ch <- &q.StreamEvent{Type: "done"}
	}()
	return ch, nil
}
