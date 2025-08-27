package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// CLI provides a clean interface for executing forge commands
type CLI struct {
	binary  Binary
	workDir string
	env     []string
	stdout  io.Writer
	stderr  io.Writer
}

// CLIOption configures the CLI wrapper
type CLIOption func(*CLI)

// WithWorkDir sets the working directory for forge commands
func WithWorkDir(dir string) CLIOption {
	return func(c *CLI) {
		c.workDir = dir
	}
}

// WithEnv sets environment variables for forge commands
func WithEnv(env []string) CLIOption {
	return func(c *CLI) {
		c.env = env
	}
}

// WithStdout sets the stdout writer for forge commands
func WithStdout(w io.Writer) CLIOption {
	return func(c *CLI) {
		c.stdout = w
	}
}

// WithStderr sets the stderr writer for forge commands
func WithStderr(w io.Writer) CLIOption {
	return func(c *CLI) {
		c.stderr = w
	}
}

// NewCLI creates a new CLI wrapper around the forge binary
func NewCLI(binary Binary, opts ...CLIOption) *CLI {
	cli := &CLI{
		binary: binary,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	for _, opt := range opts {
		opt(cli)
	}

	return cli
}

// Command represents a forge command with its arguments
func Command(args ...string) []string {
	return args
}

// ExecuteResult contains the result of command execution
type ExecuteResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// ParseJSON attempts to parse the stdout as JSON into the provided interface
func (r *ExecuteResult) ParseJSON(v interface{}) error {
	if r.Stdout == "" {
		return fmt.Errorf("no stdout to parse")
	}

	return json.Unmarshal([]byte(r.Stdout), v)
}

// Execute runs a forge command and returns the result
func (c *CLI) Execute(ctx context.Context, args ...string) (*ExecuteResult, error) {
	// Ensure the binary is available
	if err := c.binary.Ensure(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure forge binary: %w", err)
	}

	// Create the exec command
	execCmd := exec.CommandContext(ctx, c.binary.Path(), args...)

	// Set working directory if specified
	if c.workDir != "" {
		execCmd.Dir = c.workDir
	}

	// Set environment variables
	if len(c.env) > 0 {
		execCmd.Env = append(os.Environ(), c.env...)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	// Use MultiWriter to write to both our buffer and the configured writers
	execCmd.Stdout = io.MultiWriter(&stdout, c.stdout)
	execCmd.Stderr = io.MultiWriter(&stderr, c.stderr)

	// Execute the command
	err := execCmd.Run()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Get exit code
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			// Command failed to start
			return result, fmt.Errorf("failed to execute forge command: %w", err)
		}
	}

	return result, nil
}

// ExecuteQuiet runs a forge command without output to stdout/stderr
func (c *CLI) ExecuteQuiet(ctx context.Context, args ...string) (*ExecuteResult, error) {
	// Create a temporary CLI with no output
	quietCLI := &CLI{
		binary:  c.binary,
		workDir: c.workDir,
		env:     c.env,
		stdout:  io.Discard,
		stderr:  io.Discard,
	}

	return quietCLI.Execute(ctx, args...)
}

// ExecuteWithInput runs a forge command with stdin input
func (c *CLI) ExecuteWithInput(ctx context.Context, input string, args ...string) (*ExecuteResult, error) {
	// Ensure the binary is available
	if err := c.binary.Ensure(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure forge binary: %w", err)
	}

	// Create the exec command
	execCmd := exec.CommandContext(ctx, c.binary.Path(), args...)

	// Set working directory if specified
	if c.workDir != "" {
		execCmd.Dir = c.workDir
	}

	// Set environment variables
	if len(c.env) > 0 {
		execCmd.Env = append(os.Environ(), c.env...)
	}

	// Set stdin
	execCmd.Stdin = strings.NewReader(input)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	// Use MultiWriter to write to both our buffer and the configured writers
	execCmd.Stdout = io.MultiWriter(&stdout, c.stdout)
	execCmd.Stderr = io.MultiWriter(&stderr, c.stderr)

	// Execute the command
	err := execCmd.Run()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Get exit code
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			// Command failed to start
			return result, fmt.Errorf("failed to execute forge command: %w", err)
		}
	}

	return result, nil
}

// Version returns the forge version
func (c *CLI) Version(ctx context.Context) (string, error) {
	result, err := c.ExecuteQuiet(ctx, "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get forge version: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("forge --version failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// IsInstalled checks if forge is properly installed and accessible
func (c *CLI) IsInstalled(ctx context.Context) bool {
	_, err := c.Version(ctx)
	return err == nil
}

// ExecuteForJSON executes a forge command with --json flag and parses the result
// This is useful for commands that support JSON output as mentioned in the design doc
func (c *CLI) ExecuteForJSON(ctx context.Context, result interface{}, args ...string) error {
	// Add --json flag if not already present
	hasJSONFlag := false
	for _, arg := range args {
		if arg == "--json" {
			hasJSONFlag = true
			break
		}
	}

	if !hasJSONFlag {
		args = append(args, "--json")
	}

	execResult, err := c.ExecuteQuiet(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to execute forge command: %w", err)
	}

	if execResult.ExitCode != 0 {
		return fmt.Errorf("forge command failed with exit code %d: %s", execResult.ExitCode, execResult.Stderr)
	}

	if err := execResult.ParseJSON(result); err != nil {
		return fmt.Errorf("failed to parse JSON output: %w", err)
	}

	return nil
}
