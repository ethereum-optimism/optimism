package cli

import (
	"context"
	"log/slog"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/stretchr/testify/require"
)

var cliTestBinary string

// CLITestRunner provides utilities for running op-deployer CLI commands in tests
type CLITestRunner struct {
	workDir       string
	l1RPC         string
	privateKeyHex string
	lgr           log.Logger
}

// CLITestRunnerOption is a functional option for configuring CLITestRunner
type CLITestRunnerOption func(*CLITestRunner)

func WithL1RPC(rpcURL string) CLITestRunnerOption {
	return func(r *CLITestRunner) {
		r.l1RPC = rpcURL
	}
}

func WithPrivateKey(pkHex string) CLITestRunnerOption {
	return func(r *CLITestRunner) {
		r.privateKeyHex = pkHex
	}
}

func NewCLITestRunner(t *testing.T, opts ...CLITestRunnerOption) *CLITestRunner {
	workDir, err := filepath.Abs(testutils.IsolatedTestDirWithAutoCleanup(t))
	require.NoError(t, err)
	runner := &CLITestRunner{
		workDir: workDir,
		lgr:     testlog.Logger(t, slog.LevelDebug),
	}
	for _, opt := range opts {
		opt(runner)
	}
	return runner
}

// NewCLITestRunnerWithNetwork creates a new CLI test runner with default network setup.
// Defaults can be overridden using functional options.
func NewCLITestRunnerWithNetwork(t *testing.T, opts ...CLITestRunnerOption) *CLITestRunner {
	workDir, err := filepath.Abs(testutils.IsolatedTestDirWithAutoCleanup(t))
	require.NoError(t, err)

	// Set up defaults
	lgr := testlog.Logger(t, slog.LevelDebug)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	pkHex, _, _ := shared.DefaultPrivkey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Poll until we can get the chain ID, maximum 10 seconds
	// Helps prevent race condition where anvil env is accessed before its ready
	var anvilReady bool
	for range 25 {
		if _, err := l1Client.ChainID(ctx); err == nil {
			anvilReady = true
			lgr.Info("Anvil is ready and responding")
			break
		}
		// Exit early if context expired
		if ctx.Err() != nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	require.True(t, anvilReady, "Anvil did not become ready in time")

	runner := &CLITestRunner{
		workDir:       workDir,
		l1RPC:         l1RPC,
		privateKeyHex: pkHex,
		lgr:           lgr,
	}

	// Apply options to override defaults
	for _, opt := range opts {
		opt(runner)
	}

	return runner
}

// GetWorkDir returns the working directory for this test runner
func (r *CLITestRunner) GetWorkDir() string {
	return r.workDir
}

// GetL1RPC returns the L1 RPC URL for this test runner
func (r *CLITestRunner) GetL1RPC() string {
	return r.l1RPC
}

// GetPrivateKey returns the private key hex for this test runner
func (r *CLITestRunner) GetPrivateKey() string {
	return r.privateKeyHex
}

// Run executes a CLI command and returns the output
func (r *CLITestRunner) Run(ctx context.Context, args []string, env map[string]string) (string, error) {
	if len(args) > 0 && args[0] == "op-deployer" {
		args = args[1:]
	}
	args = append([]string{"--cache-dir", filepath.Join(r.workDir, ".cache")}, args...)
	cmd := exec.CommandContext(ctx, cliTestBinary, args...)
	// Forge descendants must stop with the CLI when its context expires.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 5 * time.Second
	cmd.Dir = r.workDir
	cmd.Env = cmd.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
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
	r.lgr.Info("Running cli command, expecting success")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, err := r.Run(ctx, args, env)
	require.NoError(t, err, "Command failed: %s", output)
	return output
}

// ExpectSuccessWithNetwork runs a command with network parameters expecting it to succeed
func (r *CLITestRunner) ExpectSuccessWithNetwork(t *testing.T, args []string, env map[string]string) string {
	r.lgr.Info("Running cli command with network, expecting success")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, err := r.RunWithNetwork(ctx, args, env)
	require.NoError(t, err, "Command failed: %s", output)
	return output
}

// ExpectErrorContains runs a command expecting it to fail with specific error text
func (r *CLITestRunner) ExpectErrorContains(t *testing.T, args []string, env map[string]string, contains string) string {
	r.lgr.Info("Running cli command, expecting error")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, err := r.Run(ctx, args, env)
	require.Error(t, err, "Expected command to fail but it succeeded")
	require.Contains(t, output, contains, "Error message should contain expected text")
	return output
}

// ExpectErrorContainsWithNetwork runs a command with network parameters expecting it to fail with specific error text
func (r *CLITestRunner) ExpectErrorContainsWithNetwork(t *testing.T, args []string, env map[string]string, contains string) string {
	r.lgr.Info("Running cli command with network, expecting error")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	output, err := r.RunWithNetwork(ctx, args, env)
	require.Error(t, err, "Expected command to fail but it succeeded")
	require.Contains(t, output, contains, "Error message should contain expected text")
	return output
}
