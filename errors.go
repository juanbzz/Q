package q

import "fmt"

type TerminationReason string

const (
	ReasonComplete  TerminationReason = "complete"
	ReasonStepLimit TerminationReason = "step_limit"
	ReasonCostLimit TerminationReason = "cost_limit"
	ReasonUserAbort TerminationReason = "user_abort"
)

type TerminatingErr struct {
	Reason TerminationReason
	Output string
}

func (e *TerminatingErr) Error() string {
	return fmt.Sprintf("terminating: %s", e.Reason)
}

type ProcessErrType string

const (
	ProcessErrFormat    ProcessErrType = "format"
	ProcessErrTimeout   ProcessErrType = "timeout"
	ProcessErrExecution ProcessErrType = "execution"
)

type ProcessErr struct {
	Type    ProcessErrType
	Message string
}

func (e *ProcessErr) Error() string {
	return fmt.Sprintf("process error [%s]: %s", e.Type, e.Message)
}
