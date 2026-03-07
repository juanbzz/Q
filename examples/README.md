# Q Examples

This directory contains examples showing how to use the Q agentic tooling library with real LLM providers.

## Setup

The examples have their own `go.mod` file to keep development dependencies separate from the core library.

### Environment Variables

Set your OpenRouter API key:
```bash
export OPENROUTER_API_KEY="your-api-key-here"
```

### Install Dependencies

```bash
cd examples/
go mod tidy
```

## Examples

### 1. Simple Agent (`simple_agent.go`)
Basic example using mock LLM provider - no API key required.

```bash
go run simple_agent.go
```

### 2. OpenRouter Agent (`openrouter_agent.go`)
Real LLM integration using LangChain Go with OpenRouter API.

**Requirements:**
- `OPENROUTER_API_KEY` environment variable
- Internet connection

```bash
go run openrouter_agent.go
```

This example:
- Uses Anthropic Claude 3.5 Sonnet via OpenRouter
- Demonstrates real tool usage (file operations)
- Shows conversation flow with multiple iterations
- Provides token usage statistics

## Available Models

The OpenRouter example supports any model available on OpenRouter:
- `anthropic/claude-3.5-sonnet` (default)
- `openai/gpt-4`
- `meta-llama/llama-3.1-8b-instruct`
- `google/gemini-pro`
- And many more...

Change the model in the code:
```go
provider, err := NewOpenRouterProvider("openai/gpt-4")
```

## Creating Your Own Examples

1. Add your example to this directory
2. Import the Q library: `import "github.com/juanbzz/q"`
3. Use `go mod tidy` to update dependencies
4. The local Q library is available via the replace directive in `go.mod`

## Usage Patterns

**Tools Only** (minimal):
```go
import "github.com/juanbzz/q"

registry := q.NewToolRegistry()
registry.Register(q.ReadFileTool())
result, _ := registry.Execute(ctx, "read_file", input)
```

**Agent Building** (full framework):
```go
import "github.com/juanbzz/q"

registry := q.NewToolRegistry()
registry.Register(q.ReadFileTool())

agent := q.NewAgent(provider, registry, config)
response, _ := agent.Execute(ctx, "Analyze this project")
```

## Architecture

The examples demonstrate:
- **Clean separation**: Core library has no LLM dependencies
- **Pluggable providers**: Easy to swap LLM backends
- **Tool composition**: Mix local and external tools
- **Real-world usage**: Practical agent implementations
