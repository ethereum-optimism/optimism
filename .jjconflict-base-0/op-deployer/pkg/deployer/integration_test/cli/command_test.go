package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/stretchr/testify/require"
)

func TestInitCommandFlagValidation(t *testing.T) {
	runner := NewCLITestRunner(t)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
	}{
		{
			name:           "valid init command",
			args:           []string{"init", "--l1-chain-id", "11155111", "--l2-chain-ids", "123"},
			expectError:    false,
			expectContains: []string{}, // Success is validated by lack of error
		},
		{
			name:           "missing required l2-chain-ids",
			args:           []string{"init", "--l1-chain-id", "11155111"},
			expectError:    true,
			expectContains: []string{"must specify at least one L2 chain ID"},
		},
		{
			name:           "invalid l1-chain-id format",
			args:           []string{"init", "--l1-chain-id", "invalid", "--l2-chain-ids", "123"},
			expectError:    true,
			expectContains: []string{"invalid value \"invalid\" for flag -l1-chain-id: parse error"},
		},
		{
			name:           "invalid intent-type",
			args:           []string{"init", "--l1-chain-id", "11155111", "--l2-chain-ids", "123", "--intent-type", "invalid"},
			expectError:    true,
			expectContains: []string{"intent type not supported: invalid"},
		},
		{
			name:           "valid intent-type custom",
			args:           []string{"init", "--l1-chain-id", "11155111", "--l2-chain-ids", "123", "--intent-type", "custom"},
			expectError:    false,
			expectContains: []string{}, // Success is validated by lack of error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := runner.GetWorkDir()
			args := append(tt.args, "--workdir", workDir)

			if tt.expectError {
				runner.ExpectErrorContains(t, args, nil, strings.Join(tt.expectContains, " "))
			} else {
				runner.ExpectSuccess(t, args, nil)
				// For successful cases, validate output if expectContains is not empty
				if len(tt.expectContains) > 0 {
					output := runner.ExpectSuccess(t, args, nil)
					for _, expected := range tt.expectContains {
						require.Contains(t, output, expected, "Output should contain expected string: %s", expected)
					}
				}
			}
		})
	}
}

func TestApplyCommandFlagValidation(t *testing.T) {
	runner := NewCLITestRunner(t)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
	}{
		{
			name:           "invalid deployment target",
			args:           []string{"apply", "--deployment-target", "invalid"},
			expectError:    true,
			expectContains: []string{"failed to parse deployment target: invalid deployment target: invalid"},
		},
		{
			name:           "valid deployment targets",
			args:           []string{"apply", "--deployment-target", "live"},
			expectError:    true, // Will fail because no RPC URL, but tests flag parsing
			expectContains: []string{"l1 RPC URL must be specified"},
		},
		{
			name:           "valid noop deployment target",
			args:           []string{"apply", "--deployment-target", "noop"},
			expectError:    true, // Will fail because no intent file, but tests flag parsing
			expectContains: []string{"failed to read intent file"},
		},
		{
			name:           "valid calldata deployment target",
			args:           []string{"apply", "--deployment-target", "calldata"},
			expectError:    true, // Will fail because no intent file, but tests flag parsing
			expectContains: []string{"failed to read intent file"},
		},
		{
			name:           "valid genesis deployment target",
			args:           []string{"apply", "--deployment-target", "genesis"},
			expectError:    true, // Will fail because no intent file, but tests flag parsing
			expectContains: []string{"failed to read intent file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := runner.GetWorkDir()
			args := append(tt.args, "--workdir", workDir)

			if tt.expectError {
				runner.ExpectErrorContains(t, args, nil, strings.Join(tt.expectContains, " "))
			} else {
				runner.ExpectSuccess(t, args, nil)
				// For successful cases, validate output if expectContains is not empty
				if len(tt.expectContains) > 0 {
					output := runner.ExpectSuccess(t, args, nil)
					for _, expected := range tt.expectContains {
						require.Contains(t, output, expected, "Output should contain expected string: %s", expected)
					}
				}
			}
		})
	}
}

func TestGlobalFlagValidation(t *testing.T) {
	runner := NewCLITestRunner(t)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
	}{
		{
			name:           "invalid global flag",
			args:           []string{"--invalid-global-flag", "init", "--l1-chain-id", "11155111", "--l2-chain-ids", "123"},
			expectError:    true,
			expectContains: []string{"flag provided but not defined: -invalid-global-flag"},
		},
		{
			name:           "valid cache dir flag",
			args:           []string{"--cache-dir", "/tmp/cache", "init", "--l1-chain-id", "11155111", "--l2-chain-ids", "123"},
			expectError:    false,
			expectContains: []string{}, // Success is validated by lack of error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := runner.GetWorkDir()
			args := append(tt.args, "--workdir", workDir)

			if tt.expectError {
				runner.ExpectErrorContains(t, args, nil, strings.Join(tt.expectContains, " "))
			} else {
				runner.ExpectSuccess(t, args, nil)
				// For successful cases, validate output if expectContains is not empty
				if len(tt.expectContains) > 0 {
					output := runner.ExpectSuccess(t, args, nil)
					for _, expected := range tt.expectContains {
						require.Contains(t, output, expected, "Output should contain expected string: %s", expected)
					}
				}
			}
		})
	}
}

func TestCommandParsing(t *testing.T) {
	runner := NewCLITestRunner(t)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
	}{
		{
			name:           "help command",
			args:           []string{"--help"},
			expectError:    false,
			expectContains: []string{"op-deployer", "Tool to configure and deploy OP Chains"}, // From cli/app.go app.Usage
		},
		{
			name:           "version command",
			args:           []string{"--version"},
			expectError:    false,
			expectContains: []string{"op-deployer", "v0.0.0"}, // Version should contain app name and version format
		},
		{
			name:           "no command",
			args:           []string{},
			expectError:    false,                                                             // Shows help
			expectContains: []string{"op-deployer", "Tool to configure and deploy OP Chains"}, // From cli/app.go app.Usage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				runner.ExpectErrorContains(t, tt.args, nil, strings.Join(tt.expectContains, " "))
			} else {
				output := runner.ExpectSuccess(t, tt.args, nil)
				// Verify expected content in output
				for _, expected := range tt.expectContains {
					require.Contains(t, output, expected, "Output should contain expected string: %s", expected)
				}
			}
		})
	}
}

// TestCLIApplyMissingIntent tests apply when intent.toml is missing (uses real CLI binary)
func TestCLIApplyMissingIntent(t *testing.T) {
	runner := NewCLITestRunner(t)

	workDir := runner.GetWorkDir()

	runner.ExpectErrorContains(t, []string{
		"apply",
		"--deployment-target", "noop",
		"--workdir", workDir,
	}, nil, "failed to read intent file")
}

// TestCLIApplyMissingState tests apply when state.json is missing (uses real CLI binary)
func TestCLIApplyMissingState(t *testing.T) {
	runner := NewCLITestRunner(t)

	workDir := runner.GetWorkDir()

	// Create intent.toml but not state.json
	intent := &state.Intent{
		ConfigType: state.IntentTypeCustom,
		L1ChainID:  11155111,
	}
	require.NoError(t, intent.WriteToFile(filepath.Join(workDir, "intent.toml")))

	runner.ExpectErrorContains(t, []string{
		"apply",
		"--deployment-target", "noop",
		"--workdir", workDir,
	}, nil, "failed to read state file")
}
