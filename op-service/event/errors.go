package event

import "errors"

var (
	// ErrNoSystemInContext indicates that a context did not contain an event System.
	ErrNoSystemInContext = errors.New("event: no system in context")
	// ErrUnexpectedSystemType indicates the System retrieved from context was not a *Sys.
	ErrUnexpectedSystemType = errors.New("event: system is not *Sys")
	// ErrExecutorNotCooperative indicates the executor does not support cooperative awaiting.
	ErrExecutorNotCooperative = errors.New("event: executor is not CooperativeExec")
	// ErrTaskRunnerInit indicates a failure to prepare the task runner for spawning work.
	ErrTaskRunnerInit = errors.New("event: failed to ensure task runner")
	// ErrExecutorNotDrivable indicates the underlying executor cannot be driven by the System.
	ErrExecutorNotDrivable = errors.New("event: executor cannot be driven")
)
