package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/juanbzz/q"
)

// BashExecutor executes bash commands with timeout and validation.
// Implements q.Environment.
type BashExecutor struct {
	timeout    time.Duration
	workingDir string
	validator  CommandValidator
}

type BashExecutorOption func(*BashExecutor)

func WithTimeout(d time.Duration) BashExecutorOption {
	return func(e *BashExecutor) {
		e.timeout = d
	}
}

func WithWorkingDir(dir string) BashExecutorOption {
	return func(e *BashExecutor) {
		e.workingDir = dir
	}
}

func WithValidator(v CommandValidator) BashExecutorOption {
	return func(e *BashExecutor) {
		e.validator = v
	}
}

func WithoutValidation() BashExecutorOption {
	return func(e *BashExecutor) {
		e.validator = nil
	}
}

func NewBashExecutor(opts ...BashExecutorOption) *BashExecutor {
	e := &BashExecutor{
		timeout:   30 * time.Second,
		validator: NewDefaultBlocklistValidator(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *BashExecutor) Execute(action q.Action) (q.Output, error) {
	if action.Type != q.ActionTypeBash {
		return q.Output{}, fmt.Errorf("unsupported action type: %s", action.Type)
	}

	if e.validator != nil {
		if err := e.validator.Validate(action.Command); err != nil {
			return q.Output{}, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", action.Command)

	if e.workingDir != "" {
		cmd.Dir = e.workingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := q.Output{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			output.TimedOut = true
			return output, &q.ProcessErr{
				Type:    q.ProcessErrTimeout,
				Message: fmt.Sprintf("Command timed out after %s. Partial output:\n%s", e.timeout, output.String()),
			}
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitErr.ExitCode()
		}

		return output, &q.ProcessErr{
			Type:    q.ProcessErrExecution,
			Message: fmt.Sprintf("Command failed: %s\nOutput:\n%s", err.Error(), output.String()),
		}
	}

	return output, nil
}
