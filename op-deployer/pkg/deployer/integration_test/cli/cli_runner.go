package cli

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/stretchr/testify/require"
)

// CLITestRunner provides utilities for running op-deployer CLI commands in tests
type CLITestRunner struct {
	workDir       string
	l1RPC         string
	privateKeyHex string
}

// NewCLITestRunner creates a new CLI test runner
func NewCLITestRunner(t *testing.T) *CLITestRunner {
	// Create a temporary working directory for tests
	workDir := t.TempDir()

	return &CLITestRunner{
		workDir: workDir,
	}
}

// NewCLITestRunnerWithNetwork creates a new CLI test runner with network setup
func NewCLITestRunnerWithNetwork(t *testing.T) *CLITestRunner {
	workDir := t.TempDir()

	lgr := testlog.Logger(t, slog.LevelDebug)
	l1RPC, _ := devnet.DefaultAnvilRPC(t, lgr)

	// Get private key
	pkHex, _, _ := shared.DefaultPrivkey(t)

	return &CLITestRunner{
		workDir:       workDir,
		l1RPC:         l1RPC,
		privateKeyHex: pkHex,
	}
}

// GetWorkDir returns the working directory for this test runner
func (r *CLITestRunner) GetWorkDir() string {
	return r.workDir
}

// captureOutputWriter captures output written to it for testing
type captureOutputWriter struct {
	buf *bytes.Buffer
}

func (w *captureOutputWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func newCaptureOutputWriter() *captureOutputWriter {
	return &captureOutputWriter{buf: &bytes.Buffer{}}
}

// Run executes a CLI command and returns the output
func (r *CLITestRunner) Run(ctx context.Context, args []string, env map[string]string) (string, error) {
	// Set up environment variables
	for key, value := range env {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Change to the working directory for the test
	originalDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(r.workDir); err != nil {
		return "", err
	}

	// Capture output
	stdout := newCaptureOutputWriter()
	stderr := newCaptureOutputWriter()

	// Add "op-deployer" as the first argument if not already present
	fullArgs := args
	if len(args) == 0 || args[0] != "op-deployer" {
		fullArgs = append([]string{"op-deployer"}, args...)
	}

	// Run the CLI command using the testable interface
	err = RunCLI(ctx, stdout, stderr, fullArgs)
	output := stdout.buf.String() + stderr.buf.String()
	if err != nil {
		return output, err
	}

	return output, nil
}

// RunWithNetwork executes a CLI command with network parameters if available
func (r *CLITestRunner) RunWithNetwork(ctx context.Context, args []string, env map[string]string) (string, error) {
	if r.l1RPC != "" {
		args = append(args, "--l1-rpc-url", r.l1RPC)
	}
	if r.privateKeyHex != "" {
		args = append(args, "--private-key", r.privateKeyHex)
	}

	return r.Run(ctx, args, env)
}

// ExpectSuccess runs a command expecting it to succeed
func (r *CLITestRunner) ExpectSuccess(t *testing.T, args []string, env map[string]string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := r.Run(ctx, args, env)
	require.NoError(t, err, "Command failed: %s", output)
	return output
}

// ExpectSuccessWithNetwork runs a command with network parameters expecting it to succeed
func (r *CLITestRunner) ExpectSuccessWithNetwork(t *testing.T, args []string, env map[string]string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := r.RunWithNetwork(ctx, args, env)
	require.NoError(t, err, "Command failed: %s", output)
	return output
}

// ExpectErrorContains runs a command expecting it to fail with specific error text
func (r *CLITestRunner) ExpectErrorContains(t *testing.T, args []string, env map[string]string, contains string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := r.Run(ctx, args, env)
	require.Error(t, err, "Expected command to fail but it succeeded")
	require.Contains(t, output, contains, "Error message should contain expected text")
	return output
}
