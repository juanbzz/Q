package q

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Agent interface {
	Run(ctx context.Context, task string) (*AgentResponse, error)
	Step(ctx context.Context) (*StepResult, error)
	AddTool(tool Tool) error
}

type AgentResponse struct {
	Content    string         `json:"content"`
	ToolCalls  []ToolCall     `json:"tool_calls"`
	Iterations int            `json:"iterations"`
	TotalUsage *Usage         `json:"total_usage,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type StepResult struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     *Usage     `json:"usage,omitempty"`
	Done      bool       `json:"done"`
}

type AgentEvent struct {
	Type    string    `json:"type"`
	Content string    `json:"content,omitempty"`
	Tool    *ToolCall `json:"tool,omitempty"`
	Error   string    `json:"error,omitempty"`
}

const (
	EventTypeContent  = "content"
	EventTypeToolCall = "tool_call"
	EventTypeDone     = "done"
	EventTypeError    = "error"
)

type AgentConfig struct {
	Model         string      `json:"model"`
	MaxTokens     int         `json:"max_tokens"`
	Temperature   float64     `json:"temperature"`
	MaxIterations int         `json:"max_iterations"`
	SystemPrompt  string      `json:"system_prompt"`
	Environment   Environment `json:"-"`
	Parser        Parser      `json:"-"`
}

// ActionType represents the type of action to execute.
type ActionType string

const (
	ActionTypeBash ActionType = "bash"
)

// Action represents a parsed command to execute.
type Action struct {
	Type    ActionType
	Command string
}

func (a Action) String() string {
	return fmt.Sprintf("%s: %s", a.Type, a.Command)
}

// Output represents the result of command execution.
type Output struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
}

func (o Output) String() string {
	result := o.Stdout
	if o.Stderr != "" {
		result += "\nstderr: " + o.Stderr
	}
	if o.ExitCode != 0 {
		result += fmt.Sprintf("\nexit code: %d", o.ExitCode)
	}
	return result
}

// Environment runs actions. Implemented by the executor package.
type Environment interface {
	Execute(action Action) (Output, error)
}

// Parser extracts executable actions from LLM responses.
// Implemented by the executor package.
type Parser interface {
	ParseAction(response string) (Action, error)
}

// DefaultAgent implements the Agent interface with two execution paths:
// tool-calling (when tools are registered) and bash-only (when Environment
// and Parser are provided in config).
type DefaultAgent struct {
	provider LLMProvider
	registry ToolRegistry
	config   AgentConfig

	messages        []Message
	step            int
	totalTokensUsed int
}

func NewAgent(provider LLMProvider, registry ToolRegistry, config AgentConfig) *DefaultAgent {
	if config.MaxIterations <= 0 {
		config.MaxIterations = 10
	}
	return &DefaultAgent{
		provider: provider,
		registry: registry,
		config:   config,
		messages: []Message{},
	}
}

func (a *DefaultAgent) AddTool(tool Tool) error {
	return a.registry.Register(tool)
}

func (a *DefaultAgent) Run(ctx context.Context, task string) (*AgentResponse, error) {
	a.messages = []Message{}
	a.step = 0
	a.totalTokensUsed = 0

	if a.config.SystemPrompt != "" {
		a.messages = append(a.messages, Message{
			Role:    "system",
			Content: a.config.SystemPrompt,
		})
	}

	a.messages = append(a.messages, Message{
		Role:    "user",
		Content: task,
	})

	if len(a.registry.List()) > 0 {
		return a.runToolCalling(ctx)
	}

	if a.config.Environment != nil && a.config.Parser != nil {
		return a.runBashOnly(ctx)
	}

	return a.runSimple(ctx)
}

func (a *DefaultAgent) Step(ctx context.Context) (*StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	tools := a.registry.List()
	toolDefs := make([]ToolDefinition, len(tools))
	for i, tool := range tools {
		toolDefs[i] = ToolToDefinition(tool)
	}

	response, err := a.provider.Chat(ctx, a.messages, toolDefs)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	a.accumulateUsage(response.Usage)

	if len(response.ToolCalls) == 0 {
		a.messages = append(a.messages, Message{
			Role:    "assistant",
			Content: response.Content,
		})
		return &StepResult{
			Content: response.Content,
			Usage:   response.Usage,
			Done:    true,
		}, nil
	}

	a.messages = append(a.messages, Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: response.ToolCalls,
	})

	var executedCalls []ToolCall
	for _, toolCall := range response.ToolCalls {
		result, err := a.registry.Execute(ctx, toolCall.Name, toolCall.Arguments)
		if err != nil {
			return nil, fmt.Errorf("tool execution error: %w", err)
		}

		executedCalls = append(executedCalls, ToolCall{
			Name:  toolCall.Name,
			Input: toolCall.Arguments,
		})

		toolResultContent := result.Content
		if result.Error != "" {
			toolResultContent = fmt.Sprintf("Error: %s", result.Error)
		}

		a.messages = append(a.messages, Message{
			Role:       "tool",
			Content:    toolResultContent,
			ToolCallID: toolCall.ID,
		})
	}

	return &StepResult{
		Content:   response.Content,
		ToolCalls: executedCalls,
		Usage:     response.Usage,
		Done:      false,
	}, nil
}

func (a *DefaultAgent) runToolCalling(ctx context.Context) (*AgentResponse, error) {
	var allExecutedCalls []ToolCall
	totalUsage := &Usage{}

	for i := 0; i < a.config.MaxIterations; i++ {
		stepResult, err := a.Step(ctx)
		if err != nil {
			var termErr *TerminatingErr
			if errors.As(err, &termErr) {
				return &AgentResponse{
					Content:    termErr.Output,
					ToolCalls:  allExecutedCalls,
					Iterations: i + 1,
					TotalUsage: totalUsage,
				}, nil
			}

			var procErr *ProcessErr
			if errors.As(err, &procErr) {
				a.messages = append(a.messages, Message{
					Role:    "user",
					Content: procErr.Message,
				})
				continue
			}

			return nil, err
		}

		allExecutedCalls = append(allExecutedCalls, stepResult.ToolCalls...)
		addUsage(totalUsage, stepResult.Usage)

		if stepResult.Done {
			return &AgentResponse{
				Content:    stepResult.Content,
				ToolCalls:  allExecutedCalls,
				Iterations: i + 1,
				TotalUsage: totalUsage,
			}, nil
		}
	}

	return nil, &TerminatingErr{Reason: ReasonStepLimit}
}

func (a *DefaultAgent) runBashOnly(ctx context.Context) (*AgentResponse, error) {
	totalUsage := &Usage{}

	for i := 0; i < a.config.MaxIterations; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled: %w", err)
		}

		response, err := a.provider.Chat(ctx, a.messages, nil)
		if err != nil {
			return nil, fmt.Errorf("LLM error: %w", err)
		}

		addUsage(totalUsage, response.Usage)

		action, err := a.config.Parser.ParseAction(response.Content)
		if err != nil {
			var procErr *ProcessErr
			if errors.As(err, &procErr) {
				a.messages = append(a.messages, Message{
					Role:    "user",
					Content: procErr.Message,
				})
				continue
			}
			return nil, err
		}

		a.messages = append(a.messages, Message{
			Role:    "assistant",
			Content: response.Content,
		})

		output, err := a.config.Environment.Execute(action)
		if err != nil {
			var procErr *ProcessErr
			if errors.As(err, &procErr) {
				a.messages = append(a.messages, Message{
					Role:    "user",
					Content: procErr.Message,
				})
				continue
			}
			return nil, err
		}

		if isTaskComplete(output.Stdout) {
			return &AgentResponse{
				Content:    extractFinalOutput(output.Stdout),
				Iterations: i + 1,
				TotalUsage: totalUsage,
			}, nil
		}

		a.messages = append(a.messages, Message{
			Role:    "user",
			Content: formatObservation(output),
		})
	}

	return nil, &TerminatingErr{Reason: ReasonStepLimit}
}

func (a *DefaultAgent) runSimple(ctx context.Context) (*AgentResponse, error) {
	response, err := a.provider.Chat(ctx, a.messages, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	a.messages = append(a.messages, Message{
		Role:    "assistant",
		Content: response.Content,
	})

	return &AgentResponse{
		Content:    response.Content,
		Iterations: 1,
		TotalUsage: response.Usage,
	}, nil
}

func (a *DefaultAgent) accumulateUsage(usage *Usage) {
	if usage != nil {
		a.totalTokensUsed += usage.TotalTokens
	}
}

func (a *DefaultAgent) Messages() []Message {
	return a.messages
}

const completionMarker = "TASK_COMPLETE"

func isTaskComplete(stdout string) bool {
	firstLine := strings.SplitN(strings.TrimSpace(stdout), "\n", 2)[0]
	return strings.TrimSpace(firstLine) == completionMarker
}

func extractFinalOutput(stdout string) string {
	parts := strings.SplitN(stdout, "\n", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func formatObservation(output Output) string {
	if strings.TrimSpace(output.Stdout) == "" && output.ExitCode == 0 {
		return "(no output)"
	}

	result := output.Stdout

	const maxLen = 10000
	if len(result) > maxLen {
		head := result[:maxLen/2]
		tail := result[len(result)-maxLen/2:]
		result = head + "\n\n[... output truncated ...]\n\n" + tail
	}

	if output.ExitCode != 0 {
		result = fmt.Sprintf("[exit code: %d]\n%s", output.ExitCode, result)
	}

	return result
}

func addUsage(total *Usage, step *Usage) {
	if total == nil || step == nil {
		return
	}
	total.PromptTokens += step.PromptTokens
	total.CompletionTokens += step.CompletionTokens
	total.TotalTokens += step.TotalTokens
}
