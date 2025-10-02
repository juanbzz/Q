# Rack - Agentic Tooling Library for Go

**A minimalist tooling library for building Go agents that get things done.**

Rack provides a clean, unified interface for agents to use tools—from simple file operations to complex external services. Built on Unix philosophy: small, focused tools that compose beautifully.

```go
// Local tools
registry.Register(rack.ReadFileTool())
registry.Register(rack.ExecTool())

// External MCP servers  
registry.Register(rack.PythonMCPTool())
registry.Register(rack.AWSMCPTool())

// Your agent uses them all the same way
agent := rack.NewAgent(provider, registry)
response, _ := agent.Execute(ctx, "Fix the failing tests")
```

## Features

- **Simple Tool Interface** - Easy to implement and extend with just 4 methods
- **MCP Support** - Connect to external Model Context Protocol servers
- **Multiple Execution Strategies** - Local, sandboxed, remote tool execution
- **Built-in Middleware** - Logging, auth, rate limiting support
- **Minimal Dependencies** - Just what you need, nothing more
- **Type Safety** - Leverage Go's type system with JSON flexibility

`rack` helps you build coding assistants, automation tools, and intelligent CLI applications.

## Examples

See the [`examples/`](examples/) directory for complete working examples:

- **`simple_agent.go`** - Basic usage with mock LLM (no API key required)
- **`openrouter_agent.go`** - Real LLM integration using LangChain Go + OpenRouter

```bash
cd examples/
export OPENROUTER_API_KEY="your-key"
go run openrouter_agent.go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    
    "github.com/j0lvera/rack"
)

func main() {
    // Create a tool registry
    registry := rack.NewToolRegistry()
    
    // Register built-in tools
    registry.Register(rack.ReadFileTool())
    registry.Register(rack.WriteFileTool())
    registry.Register(rack.ListFilesTool())
    
    // Use tools directly
    ctx := context.Background()
    input := json.RawMessage(`{"path": "."}`)
    result, err := registry.Execute(ctx, "list_files", input)
    if err != nil {
        log.Fatal(err)
    }
    
    println(result.Content)
}
```

### Agent with LLM Provider

```go
// Create registry with tools
registry := rack.NewToolRegistry()
registry.Register(rack.ReadFileTool())
registry.Register(rack.ListFilesTool())

// Create LLM provider (implement rack.LLMProvider interface)
provider := NewOpenAIProvider("gpt-4", apiKey)

// Create agent
config := rack.AgentConfig{
    Model:       "gpt-4",
    MaxTokens:   4096,
    Temperature: 0.1,
}

agent := rack.NewAgent(provider, registry, config)

// Execute with natural language
response, err := agent.Execute(ctx, "Analyze the current project structure")
```

## Core Interfaces

### Tool Interface

Every tool implements this simple interface:

```go
type Tool interface {
    Name() string
    Description() string
    Schema() Schema
    Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}
```

### LLM Provider Interface

Connect any LLM provider:

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
    Stream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan *StreamEvent, error)
}
```

## Built-in Tools

### File Operations
- `ReadFileTool()` - Read file contents
- `WriteFileTool()` - Write content to files
- `ListFilesTool()` - List directory contents

### Command Execution
- `ExecTool()` - Execute shell commands (security-restricted)

## MCP Integration

Connect to external Model Context Protocol servers for advanced capabilities:

```go
// Python execution server
pythonConfig := rack.MCPServerConfig{
    Name:    "python_runner",
    Command: "deno",
    Args: []string{
        "run", "-N", "-R=node_modules", "-W=node_modules",
        "--node-modules-dir=auto",
        "jsr:@pydantic/mcp-run-python",
        "stdio",
    },
}

pythonTools, err := rack.NewMCPToolFromServer(pythonConfig)
if err != nil {
    log.Fatal(err)
}

for _, tool := range pythonTools {
    registry.Register(tool)
}
```

### Popular MCP Servers

- **Python Execution**: `@pydantic/mcp-run-python` - Sandboxed Python code execution
- **Documentation**: `@upstash/context7-mcp` - Up-to-date library documentation  
- **AWS Integration**: `awslabs.core-mcp-server` - AWS service interaction
- **Desktop Commander**: File system operations with advanced editing
- **Git Operations**: Git workflow automation

## Creating Custom Tools

```go
type MyCustomTool struct{}

func (t *MyCustomTool) Name() string {
    return "my_tool"
}

func (t *MyCustomTool) Description() string {
    return "Does something useful"
}

func (t *MyCustomTool) Schema() rack.Schema {
    return rack.Schema{
        Type: "object",
        Properties: map[string]rack.SchemaField{
            "input": {
                Type:        "string",
                Description: "Input parameter",
            },
        },
        Required: []string{"input"},
    }
}

func (t *MyCustomTool) Execute(ctx context.Context, input json.RawMessage) (*rack.ToolResult, error) {
    var params struct {
        Input string `json:"input"`
    }
    
    if err := json.Unmarshal(input, &params); err != nil {
        return &rack.ToolResult{Error: "invalid input"}, nil
    }
    
    // Do your work here
    result := processInput(params.Input)
    
    return &rack.ToolResult{
        Content: result,
        Metadata: map[string]any{
            "processed_at": time.Now(),
        },
    }, nil
}
```

## Configuration-Driven Setup

Load MCP servers from configuration:

```json
{
  "mcp_servers": [
    {
      "name": "python_runner",
      "command": "deno",
      "args": ["run", "jsr:@pydantic/mcp-run-python", "stdio"]
    },
    {
      "name": "aws_tools", 
      "command": "uvx",
      "args": ["awslabs.core-mcp-server@latest"]
    }
  ]
}
```

## Testing

Run the test suite:

```bash
go test ./...
```

Run the demo:

```bash
go run ./cmd/main
```

## Architecture

Rack follows clean architecture principles:

- **Tool Layer**: Individual capabilities (read file, execute command, etc.)
- **Registry Layer**: Tool discovery and execution management
- **Agent Layer**: LLM integration and conversation management  
- **MCP Layer**: External server integration via Model Context Protocol

## Design Principles

1. **Unix Philosophy**: Small, focused tools that compose well
2. **Interface Segregation**: Minimal, single-purpose interfaces
3. **Dependency Inversion**: Depend on abstractions, not implementations
4. **Fail Fast**: Clear error messages and graceful degradation
5. **Type Safety**: Leverage Go's type system while maintaining JSON flexibility

## Contributing

1. Follow the existing code style (see `guides/go.md`)
2. Add tests for new functionality
3. Update documentation
4. Ensure `go vet` and `go fmt` pass

## License

MIT License - see LICENSE file for details.

---

Built with ❤️ for the Go community. Perfect for building coding assistants, automation tools, and intelligent CLI applications.