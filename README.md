# Q

**A minimalist tooling library for building Go agents that get things done.**

In the Bond universe, Q is the one who equips agents with everything they need before a mission — tools, gadgets, tech. This library does the same: it gives your AI agents a unified interface to use tools, from simple file operations to complex external services via MCP.

```go
import "github.com/juanbzz/q"

// Equip your agent
registry := q.NewToolRegistry()
registry.Register(q.ReadFileTool())
registry.Register(q.ExecTool())

// Send it on a mission
agent := q.NewAgent(provider, registry, config)
response, _ := agent.Run(ctx, "Fix the failing tests")
```

## Why Q?

Every agent needs equipment. Q handles the boring but critical part — registering tools, converting schemas, calling MCP servers, managing the tool-call loop — so you can focus on the mission.

- **4-method Tool interface** — dead simple to implement
- **MCP support** — connect external tool servers over stdin/stdout
- **Minimal dependencies** — core library depends only on langchaingo
- **Any LLM** — bring your own provider via the `LLMProvider` interface

## Quick Start

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/juanbzz/q"
)

func main() {
    registry := q.NewToolRegistry()
    registry.Register(q.ReadFileTool())
    registry.Register(q.WriteFileTool())
    registry.Register(q.ListFilesTool())

    ctx := context.Background()
    input := json.RawMessage(`{"path": "."}`)
    result, err := registry.Execute(ctx, "list_files", input)
    if err != nil {
        log.Fatal(err)
    }

    println(result.Content)
}
```

### With an LLM

```go
registry := q.NewToolRegistry()
registry.Register(q.ReadFileTool())
registry.Register(q.ListFilesTool())

provider := NewOpenAIProvider("gpt-4", apiKey)

agent := q.NewAgent(provider, registry, q.AgentConfig{
    Model:       "gpt-4",
    MaxTokens:   4096,
    Temperature: 0.1,
})

response, err := agent.Run(ctx, "Analyze the current project structure")
```

## Core Interfaces

**Tool** — every gadget in Q's arsenal:

```go
type Tool interface {
    Name() string
    Description() string
    Schema() Schema
    Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}
```

**LLMProvider** — plug in any LLM:

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
    Stream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan *StreamEvent, error)
}
```

## Built-in Tools

| Tool | Description |
|------|-------------|
| `ReadFileTool()` | Read file contents |
| `WriteFileTool()` | Write content to files |
| `ListFilesTool()` | List directory contents |
| `ExecTool()` | Execute shell commands (security-restricted) |

## MCP Integration

Connect external tool servers — Q speaks JSON-RPC 2.0 over stdin/stdout:

```go
pythonTools, err := q.NewMCPToolFromServer(q.MCPServerConfig{
    Name:    "python_runner",
    Command: "deno",
    Args:    []string{"run", "jsr:@pydantic/mcp-run-python", "stdio"},
})

for _, tool := range pythonTools {
    registry.Register(tool)
}
```

Works with popular MCP servers like `@pydantic/mcp-run-python`, `@upstash/context7-mcp`, `awslabs.core-mcp-server`, and any other server that speaks the protocol.

## Custom Tools

```go
type MyTool struct{}

func (t *MyTool) Name() string        { return "my_tool" }
func (t *MyTool) Description() string { return "Does something useful" }

func (t *MyTool) Schema() q.Schema {
    return q.Schema{
        Type: "object",
        Properties: map[string]q.SchemaField{
            "input": {Type: "string", Description: "Input parameter"},
        },
        Required: []string{"input"},
    }
}

func (t *MyTool) Execute(ctx context.Context, input json.RawMessage) (*q.ToolResult, error) {
    var params struct{ Input string `json:"input"` }
    json.Unmarshal(input, &params)
    return &q.ToolResult{Content: process(params.Input)}, nil
}
```

## Architecture

```
Q Branch HQ
├── Tool Layer      — individual gadgets (read file, exec, etc.)
├── Registry Layer  — the armory: tool discovery & execution
├── Agent Layer     — mission control: LLM integration & conversation loop
└── MCP Layer       — field comms: external server integration
```

The agent loop: prompt → LLM → tool calls → execute → repeat (max 10 iterations) → final response.

## Examples

See [`examples/`](examples/) for working code. All examples use `models/openai` which works with OpenAI, OpenRouter, or any compatible endpoint.

Set your environment:

```bash
export OPENAI_API_KEY=sk-...
export OPENAI_BASE_URL=https://openrouter.ai/api/v1  # optional, for OpenRouter
export MODEL=anthropic/claude-sonnet-4-5-20250929     # optional, this is the default
```

### basic

Bash-only agent — the LLM generates shell commands, Q executes them.

```bash
cd examples && go run ./basic
```

### tools

Tool-calling agent — uses Q's built-in tools (`read_file`, `write_file`, `list_files`, `exec`) via the LLM's native tool-calling protocol.

```bash
cd examples && go run ./tools
```

### cli

Cobra-based CLI wrapping the bash-only agent with flags for working directory, timeout, and max steps.

```bash
cd examples && go run ./cli run "list all Go files in the project"
cd examples && go run ./cli run --working-dir /tmp --timeout 60s --max-steps 10 "create a hello.txt file"
```

## Testing

```bash
go test ./...
```

## License

Apache 2.0
