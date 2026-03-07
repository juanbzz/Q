# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Q is a minimalist agentic tooling library for Go that provides a unified interface for agents to use tools—from simple file operations to complex external services via Model Context Protocol (MCP). Named after James Bond's quartermaster who equips agents with tools before each mission, it enables building coding assistants, automation tools, and intelligent CLI applications.

## Build and Test Commands

### Testing
- **Run all tests**: `go test .`
- **Test with coverage**: `go test -cover .`
- **Test with race detection**: `go test -race .`
- **Test verbose**: `go test -v .`

### Code Quality
- **Format code**: `go fmt .`
- **Vet code**: `go vet .`
- **Build library**: `go build .`

### Examples
- **Run simple example** (no API key needed): `cd examples && go run simple_agent.go`
- **Run OpenRouter example**: `cd examples && OPENROUTER_API_KEY=xxx go run openrouter_agent.go`
- **Test examples compile**: `cd examples && go build simple_agent.go && go build openrouter_agent.go`

## Architecture

### Core Layers
1. **Tool Layer** (`tool.go`) - Individual capabilities via 4-method interface: Name, Description, Schema, Execute
2. **Registry Layer** (`registry.go`) - Central tool management with Register/Get/List/Execute pattern
3. **Agent Layer** (`agent.go`) - Agent interface definitions and conversation management
4. **LLM Layer** (`llm.go`) - LLM provider abstraction and DefaultAgent implementation
5. **MCP Layer** (`mcp.go`) - External MCP server integration via JSON-RPC over stdin/stdout

### Package Organization
Single package design (`package q`) with logical file separation:
- `tool.go` - Core Tool interface and types (Schema, SchemaField, ToolResult, ToolCall)
- `registry.go` - ToolRegistry interface and DefaultToolRegistry implementation
- `tools.go` - Built-in tools (ReadFile, WriteFile, ListFiles, Exec)
- `agent.go` - Agent interface and response types
- `llm.go` - LLMProvider interface, DefaultAgent, mock provider
- `mcp.go` - MCPServer management, MCPTool wrapper
- `factory.go` - Convenience factory functions
- `q_test.go` - All tests (uses `package q_test`)

### Key Interfaces

**Tool Interface** - Every tool must implement:
```go
type Tool interface {
    Name() string
    Description() string
    Schema() Schema
    Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}
```

**LLMProvider Interface** - Connect any LLM:
```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
    Stream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan *StreamEvent, error)
}
```

**ToolRegistry Interface** - Central tool management:
```go
type ToolRegistry interface {
    Register(tool Tool) error
    Get(name string) (Tool, bool)
    List() []Tool
    Execute(ctx context.Context, name string, input json.RawMessage) (*ToolResult, error)
}
```

### Agent Execution Flow
1. User prompt added to conversation history
2. Agent loop (max 10 iterations):
   - Send messages + tool definitions to LLM
   - If no tool calls: return final response
   - If tool calls: execute each via registry, add results to conversation
   - Continue loop with updated conversation
3. Return AgentResponse with content, executed tool calls, and metadata

### MCP Integration
- External MCP servers run as separate processes (deno, uvx, etc.)
- JSON-RPC 2.0 protocol over stdin/stdout
- Connection lifecycle: Start → Initialize → LoadTools → CallTool → Stop
- MCPTool wraps external tools with standard Tool interface for seamless integration
- Popular servers: @pydantic/mcp-run-python, @upstash/context7-mcp, awslabs.core-mcp-server

## Development Principles

### Dependencies
- **Core library has zero dependencies** - keep it lightweight
- Examples can have dependencies (e.g., langchaingo) but not core
- Depend on abstractions (interfaces), not concrete implementations
- Prefer small, single-method interfaces

### Error Handling
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Tools return errors in `ToolResult.Error` field, not Go errors
- Agent errors should be meaningful for debugging

### Code Style
- Clarity over brevity - prioritize readability
- Keep functions small and focused on one task
- No comments unless explicitly requested
- Use clear, descriptive names (ReadFileTool, NewToolRegistry)
- Export types and functions that users need

### Testing
- Use table-driven tests for multiple scenarios
- Test both success and error cases
- Mock external dependencies (LLM providers, MCP servers)
- All tests in `q_test.go` with `package q_test`
