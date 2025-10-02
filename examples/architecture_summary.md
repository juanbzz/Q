# Rack Framework Examples - Technical Architecture

## Project Overview
This repository demonstrates the implementation of AI agents using the Rack framework, showcasing both a simple mock implementation and a production-ready OpenRouter integration.

## Core Components

### 1. Simple Agent Implementation (`simple_agent.go`)
The simple agent implementation demonstrates the basic framework capabilities:

- **Tool Registry System**
  - Built-in file operation tools (read, write, list)
  - Custom tool implementation (ProjectStatsTool)
  - Extensible registration mechanism

- **Mock LLM Provider**
  - Simulated intelligent responses
  - Predefined response patterns
  - Tool call simulation

- **Configuration**
  ```go
  config := rack.AgentConfig{
      Model:       "gpt-4",
      MaxTokens:   4096,
      Temperature: 0.1,
  }
  ```

### 2. OpenRouter Integration (`openrouter_agent.go`)
Production-ready implementation featuring:

- **LangChain Integration**
  - OpenRouter API support
  - Model configuration
  - Token usage tracking

- **Message Handling**
  - Support for multiple message types:
    - System messages
    - User messages
    - Assistant messages
    - Tool messages
  - Structured message formatting

- **Streaming Support**
  - Event-based streaming interface
  - Error propagation
  - Tool call streaming

## Key Interfaces

### LLM Provider Interface
```go
type OpenRouterProvider struct {
    llm   llms.Model
    model string
}

// Core methods
func (p *OpenRouterProvider) Chat(ctx context.Context, messages []rack.Message, tools []rack.ToolDefinition) (*rack.LLMResponse, error)
func (p *OpenRouterProvider) Stream(ctx context.Context, messages []rack.Message, tools []rack.ToolDefinition) (<-chan *rack.StreamEvent, error)
```

### Tool Interface
```go
type ProjectStatsTool struct{}

func (t *ProjectStatsTool) Name() string
func (t *ProjectStatsTool) Description() string
func (t *ProjectStatsTool) Schema() rack.Schema
func (t *ProjectStatsTool) Execute(ctx context.Context, input json.RawMessage) (*rack.ToolResult, error)
```

## Implementation Features

### 1. Error Handling
- Comprehensive error checking
- Structured error responses
- Context-aware error propagation
- Environment validation

### 2. Configuration Management
- Environment-based configuration (OPENROUTER_API_KEY)
- Model configuration
- Tool registry configuration
- Agent configuration options

### 3. Tool System
- JSON-based tool definitions
- Parameter validation
- Structured responses
- Metadata support

### 4. Monitoring
- Token usage tracking
- Tool call logging
- Execution metadata
- Stream event monitoring

## Best Practices

### 1. Code Organization
- Clear separation of concerns
- Modular design
- Interface-based architecture
- Clean error handling

### 2. Configuration
- Environment variable usage
- Structured configuration objects
- Default values handling
- Validation checks

### 3. Testing Support
- Mock provider implementation
- Testable interfaces
- Isolated components
- Error scenario handling

## Usage Examples

### Basic Agent Usage
```go
// Create and configure agent
registry := rack.NewToolRegistry()
registry.Register(rack.ReadFileTool())
registry.Register(rack.ListFilesTool())
registry.Register(rack.WriteFileTool())

agent := rack.NewAgent(provider, registry, config)

// Execute agent
response, err := agent.Execute(ctx, "Please analyze this project")
```

### Direct Provider Usage
```go
provider, err := NewOpenRouterProvider("anthropic/claude-3.5-sonnet")
response, err := provider.Chat(ctx, messages, nil)
```

### Custom Tool Implementation
```go
type CustomTool struct{}

func (t *CustomTool) Execute(ctx context.Context, input json.RawMessage) (*rack.ToolResult, error) {
    // Tool implementation
    return &rack.ToolResult{
        Content: "Result",
        Metadata: map[string]any{
            "key": "value",
        },
    }, nil
}
```

## Conclusion
The Rack framework examples demonstrate a well-architected system for building AI agents in Go. The code showcases:
- Clean architecture principles
- Proper error handling
- Extensible design
- Production-ready patterns

The modular design allows for easy extension and customization of both tools and LLM providers, while maintaining robust error handling and monitoring capabilities.