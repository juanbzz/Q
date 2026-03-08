package main

import (
	"fmt"
	"os"
	"time"

	"github.com/juanbzz/q"
	"github.com/juanbzz/q/executor"
	"github.com/juanbzz/q/models/openai"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "agent",
		Short: "An LLM-powered command execution agent",
	}

	runCmd := &cobra.Command{
		Use:   "run [task]",
		Short: "Run the agent with a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task := args[0]

			modelName := os.Getenv("MODEL")
			if modelName == "" {
				modelName = "anthropic/claude-sonnet-4-5-20250929"
			}

			model, err := openai.New(modelName, openai.Config{
				APIKey:  os.Getenv("OPENAI_API_KEY"),
				BaseURL: os.Getenv("OPENAI_BASE_URL"),
			})
			if err != nil {
				return fmt.Errorf("failed to create model: %w", err)
			}

			workingDir, _ := cmd.Flags().GetString("working-dir")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			maxSteps, _ := cmd.Flags().GetInt("max-steps")

			env := executor.NewBashExecutor(
				executor.WithWorkingDir(workingDir),
				executor.WithTimeout(timeout),
			)
			parser := executor.NewBashParser()

			agent := q.NewAgent(model, q.NewToolRegistry(), q.AgentConfig{
				MaxIterations: maxSteps,
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

			result, err := agent.Run(cmd.Context(), task)
			if err != nil {
				return err
			}

			fmt.Println(result.Content)
			return nil
		},
	}

	runCmd.Flags().String("working-dir", ".", "Working directory for commands")
	runCmd.Flags().Duration("timeout", 30*time.Second, "Command timeout")
	runCmd.Flags().Int("max-steps", 25, "Maximum number of agent steps")

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
