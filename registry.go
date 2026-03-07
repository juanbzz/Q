package q

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolRegistry manages a collection of tools
type ToolRegistry interface {
	Register(tool Tool) error
	Get(name string) (Tool, bool)
	List() []Tool
	Execute(ctx context.Context, name string, input json.RawMessage) (*ToolResult, error)
}

// DefaultToolRegistry implements ToolRegistry
type DefaultToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() ToolRegistry {
	return &DefaultToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *DefaultToolRegistry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name
func (r *DefaultToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools
func (r *DefaultToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Execute runs a tool by name with the given input
func (r *DefaultToolRegistry) Execute(ctx context.Context, name string, input json.RawMessage) (*ToolResult, error) {
	tool, exists := r.Get(name)
	if !exists {
		return &ToolResult{
			Error: fmt.Sprintf("tool %s not found", name),
		}, nil
	}

	return tool.Execute(ctx, input)
}
