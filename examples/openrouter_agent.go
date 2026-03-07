package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/juanbzz/q"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// OpenRouterProvider implements q.LLMProvider using langchaingo with OpenRouter
type OpenRouterProvider struct {
	llm   llms.Model
	model string
}

// NewOpenRouterProvider creates a new OpenRouter provider using langchaingo
func NewOpenRouterProvider(model string) (*OpenRouterProvider, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY environment variable is required")
	}

	// Create OpenAI-compatible client pointing to OpenRouter
	llm, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenRouter client: %w", err)
	}

	return &OpenRouterProvider{
		llm:   llm,
		model: model,
	}, nil
}

// Chat implements q.LLMProvider interface
func (p *OpenRouterProvider) Chat(ctx context.Context, messages []q.Message, tools []q.ToolDefinition) (*q.LLMResponse, error) {
	// Convert q messages to langchaingo format
	var content strings.Builder
	
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			content.WriteString(fmt.Sprintf("System: %s\n", msg.Content))
		case "user":
			content.WriteString(fmt.Sprintf("Human: %s\n", msg.Content))
		case "assistant":
			content.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
		case "tool":
			content.WriteString(fmt.Sprintf("Tool result: %s\n", msg.Content))
		}
	}

	// Add tool definitions if tools are available
	if len(tools) > 0 {
		toolsJSON, _ := json.MarshalIndent(tools, "", "  ")
		toolPrompt := fmt.Sprintf(`\n\nYou have access to the following q. When you need to use a tool, respond with "TOOL_CALL: tool_name" followed by the JSON arguments on the next line.

Available tools:
%s

Use tools when appropriate to help answer the user's request.`, string(toolsJSON))
		content.WriteString(toolPrompt)
	}

	content.WriteString("\nAssistant: ")

	// Generate response using langchaingo
	response, err := p.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, content.String()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Get the response text
	responseText := response.Choices[0].Content

	// Parse response for tool calls
	var toolCalls []q.LLMToolCall

	// Simple tool call detection
	if strings.Contains(responseText, "TOOL_CALL:") {
		lines := strings.Split(responseText, "\n")
		for i, line := range lines {
			if strings.Contains(line, "TOOL_CALL:") {
				parts := strings.Split(line, "TOOL_CALL:")
				if len(parts) > 1 {
					toolName := strings.TrimSpace(parts[1])
					
					// Look for JSON arguments in the next lines
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

	return &q.LLMResponse{
		Content:   responseText,
		ToolCalls: toolCalls,
		Model:     p.model,
		Usage: &q.Usage{
			PromptTokens:     0, // langchaingo doesn't always provide usage stats
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}, nil
}

// Stream implements q.LLMProvider interface (simplified implementation)
func (p *OpenRouterProvider) Stream(ctx context.Context, messages []q.Message, tools []q.ToolDefinition) (<-chan *q.StreamEvent, error) {
	ch := make(chan *q.StreamEvent, 1)
	go func() {
		defer close(ch)
		response, err := p.Chat(ctx, messages, tools)
		if err != nil {
			ch <- &q.StreamEvent{Type: "error", Error: err.Error()}
			return
		}
		ch <- &q.StreamEvent{Type: "content", Content: response.Content}
		for _, toolCall := range response.ToolCalls {
			ch <- &q.StreamEvent{Type: "tool_call", ToolCall: &toolCall}
		}
		ch <- &q.StreamEvent{Type: "done"}
	}()
	return ch, nil
}

func main() {
	fmt.Println("=== Q + LangChain Go + OpenRouter Example ===")

	// Check for API key
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		log.Fatal("Please set OPENROUTER_API_KEY environment variable")
	}

	// Create OpenRouter provider
	provider, err := NewOpenRouterProvider("anthropic/claude-3.5-sonnet")
	if err != nil {
		log.Fatalf("Failed to create OpenRouter provider: %v", err)
	}

	// Create tool registry
	registry := q.NewToolRegistry()
	registry.Register(q.ReadFileTool())
	registry.Register(q.ListFilesTool())
	registry.Register(q.WriteFileTool())

	// Create agent config
	config := q.AgentConfig{
		Model:       "anthropic/claude-3.5-sonnet",
		MaxTokens:   4096,
		Temperature: 0.1,
	}

	// Create agent with real LLM
	agent := q.NewAgent(provider, registry, config)

	// Example 1: Simple analysis
	fmt.Println("\n🤖 Analyzing project with real LLM...")
	ctx := context.Background()
	
	response, err := agent.Execute(ctx, "Please analyze the current Go project structure. List the files first, then read the go.mod file to understand what this project does.")
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	fmt.Printf("\n📝 Agent Response:\n%s\n", response.Content)
	fmt.Printf("\n🔧 Tools Used: %d\n", len(response.ToolCalls))
	for i, call := range response.ToolCalls {
		fmt.Printf("  %d. %s\n", i+1, call.Name)
	}

	if metadata, ok := response.Metadata["iterations"]; ok {
		fmt.Printf("\n⚡ Completed in %v iterations\n", metadata)
	}

	if usageInterface, ok := response.Metadata["usage"]; ok {
		if usage, ok := usageInterface.(*q.Usage); ok && usage != nil {
			fmt.Printf("📊 Token Usage: %d prompt + %d completion = %d total\n", 
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
		}
	}

	// Example 2: Code analysis task
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🔍 Performing detailed code analysis...")
	
	response2, err := agent.Execute(ctx, "Now analyze the Go source files in this project. Read a few key files like tool.go and q.go to understand the architecture. Then write a brief technical summary to 'architecture_summary.md'.")
	if err != nil {
		log.Printf("Second analysis failed: %v", err)
	} else {
		fmt.Printf("\n📝 Technical Analysis:\n%s\n", response2.Content)
		fmt.Printf("\n🔧 Additional Tools Used: %d\n", len(response2.ToolCalls))
		
		// Check if summary was created
		if _, err := os.Stat("architecture_summary.md"); err == nil {
			fmt.Println("✅ Architecture summary created successfully!")
		}
	}
}

// Example of using the provider directly (without agent)
func directProviderExample() {
	provider, err := NewOpenRouterProvider("anthropic/claude-3.5-sonnet")
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	messages := []q.Message{
		{Role: "system", Content: "You are a helpful Go programming assistant."},
		{Role: "user", Content: "Explain what an interface is in Go programming."},
	}

	ctx := context.Background()
	response, err := provider.Chat(ctx, messages, nil)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}

	fmt.Printf("Direct provider response: %s\n", response.Content)
}