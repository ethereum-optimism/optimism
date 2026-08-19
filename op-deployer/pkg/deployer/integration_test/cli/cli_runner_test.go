package cli

import (
	"os"
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

func TestCLITestRunnerAppliesOptions(t *testing.T) {
	runner := NewCLITestRunner(t, WithL1RPC("http://l1.example"), WithPrivateKey("test-private-key"))

	require.Equal(t, "http://l1.example", runner.GetL1RPC(), "NewCLITestRunner ignored the L1 RPC option")
	require.Equal(t, "test-private-key", runner.GetPrivateKey(), "NewCLITestRunner ignored the private-key option")
}

func TestCLITestRunnerRestoresEnvironment(t *testing.T) {
	const (
		valueKey = "OP_DEPLOYER_TEST_RUNNER_EXISTING_ENV"
		emptyKey = "OP_DEPLOYER_TEST_RUNNER_EMPTY_ENV"
		unsetKey = "OP_DEPLOYER_TEST_RUNNER_UNSET_ENV"
	)

	t.Setenv(valueKey, "original-value")
	t.Setenv(emptyKey, "")
	previousUnsetValue, unsetKeyExisted := os.LookupEnv(unsetKey)
	require.NoError(t, os.Unsetenv(unsetKey))
	t.Cleanup(func() {
		if unsetKeyExisted {
			require.NoError(t, os.Setenv(unsetKey, previousUnsetValue))
		} else {
			require.NoError(t, os.Unsetenv(unsetKey))
		}
	})

	runner := NewCLITestRunner(t)
	for range 2 {
		_, err := runner.Run(t.Context(), []string{"--help"}, map[string]string{
			valueKey: "temporary-value",
			emptyKey: "temporary-value",
			unsetKey: "temporary-value",
		})
		require.NoError(t, err)

		require.Equal(t, "original-value", os.Getenv(valueKey), "runner destroyed pre-existing environment variable %s", valueKey)
		emptyValue, emptyKeyExists := os.LookupEnv(emptyKey)
		require.True(t, emptyKeyExists, "runner unset pre-existing empty environment variable %s", emptyKey)
		require.Empty(t, emptyValue, "runner changed pre-existing empty environment variable %s", emptyKey)
		_, unsetKeyExists := os.LookupEnv(unsetKey)
		require.False(t, unsetKeyExists, "runner left absent environment variable %s set", unsetKey)
	}
}
