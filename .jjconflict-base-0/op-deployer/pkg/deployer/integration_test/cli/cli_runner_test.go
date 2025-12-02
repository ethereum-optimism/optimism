package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCLITestRunnerSmoke tests the CLITestRunner itself
func TestCLITestRunnerSmoke(t *testing.T) {
	runner := NewCLITestRunner(t)

	require.DirExists(t, runner.GetWorkDir())

	// Test basic command execution
	runner.ExpectSuccess(t, []string{"--help"}, nil)
}
