package rack_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/j0lvera/rack"
)

func TestToolRegistry(t *testing.T) {
	registry := rack.NewToolRegistry()

	// Test registering a tool
	tool := rack.ReadFileTool()
	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Test getting a tool
	retrievedTool, exists := registry.Get("read_file")
	if !exists {
		t.Fatal("Tool not found after registration")
	}

	if retrievedTool.Name() != "read_file" {
		t.Errorf("Expected tool name 'read_file', got '%s'", retrievedTool.Name())
	}

	// Test listing tools
	toolsList := registry.List()
	if len(toolsList) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolsList))
	}

	// Test duplicate registration
	err = registry.Register(tool)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}
}

func TestReadFileTool(t *testing.T) {
	tool := rack.ReadFileTool()

	// Test tool metadata
	if tool.Name() != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}

	schema := tool.Schema()
	if schema.Type != "object" {
		t.Errorf("Expected schema type 'object', got '%s'", schema.Type)
	}

	// Test execution with valid input
	ctx := context.Background()
	input := json.RawMessage(`{"path": "go.mod"}`)

	result, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	// Should either succeed or fail gracefully
	if result.Error != "" && result.Content != "" {
		t.Error("Tool result should have either content or error, not both")
	}

	// Test execution with invalid input
	invalidInput := json.RawMessage(`{"invalid": "parameter"}`)
	result, err = tool.Execute(ctx, invalidInput)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error for invalid input")
	}
}

func TestListFilesTool(t *testing.T) {
	tool := rack.ListFilesTool()

	ctx := context.Background()
	input := json.RawMessage(`{"path": "."}`)

	result, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}

	if result.Content == "" {
		t.Error("Expected content from list_files tool")
	}

	// Check metadata
	if result.Metadata == nil {
		t.Error("Expected metadata from list_files tool")
	}
}

func TestMockLLMProvider(t *testing.T) {
	responses := []string{
		"Hello, I'm a mock LLM",
		"I can help with various tasks. TOOL_CALL: read_file",
		"Task completed successfully",
	}

	provider := rack.NewMockProvider(responses)

	ctx := context.Background()
	messages := []rack.Message{
		{Role: "user", Content: "Hello"},
	}

	// Test first response
	response, err := provider.Chat(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response.Content != responses[0] {
		t.Errorf("Expected '%s', got '%s'", responses[0], response.Content)
	}

	// Test tool call detection
	response, err = provider.Chat(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(response.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(response.ToolCalls))
	}

	if response.ToolCalls[0].Name != "read_file" {
		t.Errorf("Expected tool call 'read_file', got '%s'", response.ToolCalls[0].Name)
	}
}

func TestAgent(t *testing.T) {
	// Create registry with tools
	registry := rack.NewToolRegistry()
	registry.Register(rack.ReadFileTool())
	registry.Register(rack.ListFilesTool())

	// Create mock provider
	provider := rack.NewMockProvider([]string{
		"I'll help you. Let me list the files first. TOOL_CALL: list_files",
		"I can see the project structure. Here's my analysis.",
	})

	// Create agent
	config := rack.AgentConfig{
		Model:       "test-model",
		MaxTokens:   1000,
		Temperature: 0.1,
	}

	agentInstance := rack.NewAgent(provider, registry, config)

	// Test execution
	ctx := context.Background()
	response, err := agentInstance.Execute(ctx, "Analyze the project")
	if err != nil {
		t.Fatalf("Agent execution failed: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected content in agent response")
	}

	// Should have used tools
	if len(response.ToolCalls) == 0 {
		t.Error("Expected agent to use tools")
	}

	// Check metadata
	if response.Metadata == nil {
		t.Error("Expected metadata in agent response")
	}

	if iterations, ok := response.Metadata["iterations"]; !ok || iterations == nil {
		t.Error("Expected iterations in metadata")
	}
}

func TestToolToDefinition(t *testing.T) {
	tool := rack.ReadFileTool()
	definition := rack.ToolToDefinition(tool)

	if definition.Name != tool.Name() {
		t.Errorf("Expected name '%s', got '%s'", tool.Name(), definition.Name)
	}

	if definition.Description != tool.Description() {
		t.Errorf("Expected description '%s', got '%s'", tool.Description(), definition.Description)
	}

	if definition.Parameters == nil {
		t.Error("Expected parameters in tool definition")
	}

	// Check parameters structure
	params, ok := definition.Parameters["properties"]
	if !ok {
		t.Error("Expected 'properties' in parameters")
	}

	if params == nil {
		t.Error("Properties should not be nil")
	}
}