package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/juanbzz/q"
)

// Example: Building a simple file analysis agent
func main() {
	fmt.Println("=== Simple File Analysis Agent ===")
	
	// Create tool registry
	registry := q.NewToolRegistry()
	
	// Register file operation tools
	registry.Register(q.ReadFileTool())
	registry.Register(q.ListFilesTool())
	registry.Register(q.WriteFileTool())
	
	// Create a mock LLM provider that simulates intelligent responses
	provider := q.NewMockProvider([]string{
		"I'll analyze your project structure. Let me start by listing the files. TOOL_CALL: list_files",
		"Now let me examine the main module file to understand the project. TOOL_CALL: read_file",
		"Based on my analysis, this is a Go project called 'Q' - an agentic tooling library. Let me create a summary report. TOOL_CALL: write_file",
		"Analysis complete! I've created a project summary in 'analysis_report.txt'. The project appears to be well-structured with clean interfaces for building AI agents.",
	})
	
	// Configure the agent
	config := q.AgentConfig{
		Model:       "gpt-4",
		MaxTokens:   4096,
		Temperature: 0.1,
	}
	
	// Create the agent
	agent := q.NewAgent(provider, registry, config)
	
	// Execute the analysis
	ctx := context.Background()
	response, err := agent.Run(ctx, "Please analyze this Go project and create a summary report")
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}
	
	// Display results
	fmt.Printf("\n🤖 Agent Response:\n%s\n", response.Content)
	fmt.Printf("\n🔧 Tools Used: %d\n", len(response.ToolCalls))
	for i, call := range response.ToolCalls {
		fmt.Printf("  %d. %s\n", i+1, call.Name)
	}
	
	fmt.Printf("\n⚡ Completed in %d iterations\n", response.Iterations)
	
	// Demonstrate direct tool usage
	fmt.Println("\n=== Direct Tool Usage ===")
	
	// List files directly
	input := json.RawMessage(`{"path": "."}`)
	result, err := registry.Execute(ctx, "list_files", input)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	
	fmt.Printf("📁 Directory listing:\n%s\n", result.Content)
	
	// Read the generated report if it exists
	readInput := json.RawMessage(`{"path": "analysis_report.txt"}`)
	readResult, err := registry.Execute(ctx, "read_file", readInput)
	if err != nil {
		log.Printf("Error reading report: %v", err)
	} else if readResult.Error == "" {
		fmt.Printf("📄 Generated Report:\n%s\n", readResult.Content)
	}
}

// Example: Custom tool implementation
type ProjectStatsTool struct{}

func (t *ProjectStatsTool) Name() string {
	return "project_stats"
}

func (t *ProjectStatsTool) Description() string {
	return "Analyze project statistics like file count, lines of code, etc."
}

func (t *ProjectStatsTool) Schema() q.Schema {
	return q.Schema{
		Type: "object",
		Properties: map[string]q.SchemaField{
			"path": {
				Type:        "string",
				Description: "Project path to analyze",
				Default:     ".",
			},
		},
	}
}

func (t *ProjectStatsTool) Execute(ctx context.Context, input json.RawMessage) (*q.ToolResult, error) {
	var params struct {
		Path string `json:"path"`
	}
	
	if err := json.Unmarshal(input, &params); err != nil {
		return &q.ToolResult{Error: "invalid input parameters"}, nil
	}
	
	if params.Path == "" {
		params.Path = "."
	}
	
	// Simple implementation - count .go files
	// In a real implementation, you'd walk the directory tree
	stats := map[string]interface{}{
		"go_files":     8, // Simulated count
		"total_lines":  1500,
		"test_files":   1,
		"directories":  3,
	}
	
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	
	return &q.ToolResult{
		Content: fmt.Sprintf("Project Statistics:\n%s", string(statsJSON)),
		Metadata: map[string]any{
			"analyzed_path": params.Path,
			"stats":         stats,
		},
	}, nil
}

// Example usage with custom tool
func customToolExample() {
	registry := q.NewToolRegistry()
	
	// Register built-in and custom tools
	registry.Register(q.ReadFileTool())
	registry.Register(&ProjectStatsTool{})
	
	ctx := context.Background()
	
	// Use the custom tool
	input := json.RawMessage(`{"path": "."}`)
	result, err := registry.Execute(ctx, "project_stats", input)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	
	fmt.Printf("Custom tool result:\n%s\n", result.Content)
}