package main

import (
	"context"
	"fmt"
	"os"

	"github.com/juanbzz/q"
	"github.com/juanbzz/q/executor"
	"github.com/juanbzz/q/models/openai"
)

func main() {
	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "anthropic/claude-sonnet-4-5-20250929"
	}

	model, err := openai.New(modelName, openai.Config{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create model: %v\n", err)
		os.Exit(1)
	}

	env := executor.NewBashExecutor()
	parser := executor.NewBashParser()

	agent := q.NewAgent(model, q.NewToolRegistry(), q.AgentConfig{
		MaxIterations: 25,
		SystemPrompt: `You are an autonomous agent that completes tasks by executing bash commands.

Rules:
- To run a command, respond with exactly ONE command inside a bash code block.
- When the task is complete, respond with TASK_COMPLETE on the first line, followed by a summary.
- Do not include multiple code blocks in a single response.`,
		Environment: env,
		Parser:      parser,
		OnStep: func(e q.AgentEvent) {
			switch e.Type {
			case q.EventTypeContent:
				fmt.Fprintln(os.Stderr, e.Content)
			case q.EventTypeToolCall:
				fmt.Fprintf(os.Stderr, "$ %s\n", e.Tool.Name)
			case q.EventTypeError:
				fmt.Fprintf(os.Stderr, "error: %s\n", e.Error)
			}
		},
	})

	result, err := agent.Run(context.Background(), "Create a file called hello.txt with 'Hello, World!' inside")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Result:", result.Content)
}
