package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// Result captures the output from a shell command.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner abstracts command execution for testability.
type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, command string, args ...string) (Result, error)
}

// RunError is returned when a command exits non-zero.
type RunError struct {
	Command string
	Args    []string
	Result  Result
}

func (e *RunError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Result.Stderr != "" {
		return fmt.Sprintf("%s %v failed with exit code %d: %s", e.Command, e.Args, e.Result.ExitCode, e.Result.Stderr)
	}
	return fmt.Sprintf("%s %v failed with exit code %d", e.Command, e.Args, e.Result.ExitCode)
}

// RealRunner uses os/exec under the hood.
type RealRunner struct{}

func NewRealRunner() *RealRunner {
	return &RealRunner{}
}

func (r *RealRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (r *RealRunner) Run(ctx context.Context, command string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, &RunError{Command: command, Args: args, Result: result}
	}
	return result, err
}
