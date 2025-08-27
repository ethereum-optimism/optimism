package forge

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockBinary is a mock implementation of the Binary interface for testing
type mockBinary struct {
	path        string
	ensureError error
}

func (m *mockBinary) Ensure(ctx context.Context) error {
	return m.ensureError
}

func (m *mockBinary) Path() string {
	return m.path
}

func TestNewCLI(t *testing.T) {
	binary := &mockBinary{path: "/usr/bin/forge"}

	t.Run("default configuration", func(t *testing.T) {
		cli := NewCLI(binary)
		require.Equal(t, binary, cli.binary)
		require.Equal(t, os.Stdout, cli.stdout)
		require.Equal(t, os.Stderr, cli.stderr)
		require.Empty(t, cli.workDir)
		require.Empty(t, cli.env)
	})

	t.Run("with options", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		workDir := "/tmp/test"
		env := []string{"FOO=bar"}

		cli := NewCLI(binary,
			WithWorkDir(workDir),
			WithEnv(env),
			WithStdout(&stdout),
			WithStderr(&stderr),
		)

		require.Equal(t, workDir, cli.workDir)
		require.Equal(t, env, cli.env)
		require.Equal(t, &stdout, cli.stdout)
		require.Equal(t, &stderr, cli.stderr)
	})
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "simple command",
			args:     []string{"--version"},
			expected: []string{"--version"},
		},
		{
			name:     "build command",
			args:     []string{"build", "--optimize"},
			expected: []string{"build", "--optimize"},
		},
		{
			name:     "script command",
			args:     []string{"script", "deploy.s.sol", "--rpc-url", "http://localhost:8545"},
			expected: []string{"script", "deploy.s.sol", "--rpc-url", "http://localhost:8545"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Command(tt.args...)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCLI_Execute(t *testing.T) {
	// Create a temporary script that acts like forge
	tmpDir := t.TempDir()
	forgePath := filepath.Join(tmpDir, "forge")

	// Create a simple script that echoes its arguments
	script := `#!/bin/bash
echo "forge called with: $@"
echo "stderr output" >&2
exit 0
`

	require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

	binary := &mockBinary{path: forgePath}

	t.Run("successful execution", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cli := NewCLI(binary,
			WithStdout(&stdout),
			WithStderr(&stderr),
		)

		result, err := cli.Execute(context.Background(), "--version")

		require.NoError(t, err)
		require.Equal(t, 0, result.ExitCode)
		require.Contains(t, result.Stdout, "forge called with: --version")
		require.Contains(t, result.Stderr, "stderr output")

		// Check that output was also written to the configured writers
		require.Contains(t, stdout.String(), "forge called with: --version")
		require.Contains(t, stderr.String(), "stderr output")
	})

	t.Run("binary ensure error", func(t *testing.T) {
		binary := &mockBinary{
			path:        forgePath,
			ensureError: io.ErrUnexpectedEOF,
		}
		cli := NewCLI(binary)

		_, err := cli.Execute(context.Background(), "--version")

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ensure forge binary")
	})

	t.Run("with working directory", func(t *testing.T) {
		workDir := tmpDir
		cli := NewCLI(binary, WithWorkDir(workDir))

		result, err := cli.Execute(context.Background(), "--version")

		require.NoError(t, err)
		require.Equal(t, 0, result.ExitCode)
	})

	t.Run("with environment variables", func(t *testing.T) {
		env := []string{"TEST_VAR=test_value"}
		cli := NewCLI(binary, WithEnv(env))

		result, err := cli.Execute(context.Background(), "--version")

		require.NoError(t, err)
		require.Equal(t, 0, result.ExitCode)
	})
}

func TestCLI_ExecuteQuiet(t *testing.T) {
	tmpDir := t.TempDir()
	forgePath := filepath.Join(tmpDir, "forge")

	script := `#!/bin/bash
echo "stdout output"
echo "stderr output" >&2
exit 0
`

	require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

	binary := &mockBinary{path: forgePath}

	var stdout, stderr bytes.Buffer
	cli := NewCLI(binary,
		WithStdout(&stdout),
		WithStderr(&stderr),
	)

	result, err := cli.ExecuteQuiet(context.Background(), "--version")

	require.NoError(t, err)
	require.Equal(t, 0, result.ExitCode)
	require.Contains(t, result.Stdout, "stdout output")
	require.Contains(t, result.Stderr, "stderr output")

	// Check that no output was written to the configured writers
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestCLI_ExecuteWithInput(t *testing.T) {
	tmpDir := t.TempDir()
	forgePath := filepath.Join(tmpDir, "forge")

	// Create a script that reads from stdin and echoes it
	script := `#!/bin/bash
echo "Reading from stdin:"
cat
exit 0
`

	require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

	binary := &mockBinary{path: forgePath}
	cli := NewCLI(binary)

	input := "test input data"
	result, err := cli.ExecuteWithInput(context.Background(), input, "--version")

	require.NoError(t, err)
	require.Equal(t, 0, result.ExitCode)
	require.Contains(t, result.Stdout, "Reading from stdin:")
	require.Contains(t, result.Stdout, input)
}

func TestCLI_Version(t *testing.T) {
	tmpDir := t.TempDir()
	forgePath := filepath.Join(tmpDir, "forge")

	t.Run("successful version check", func(t *testing.T) {
		script := `#!/bin/bash
if [ "$1" = "--version" ]; then
    echo "forge 0.2.0 (abc123 2023-01-01T00:00:00.000000000Z)"
    exit 0
fi
exit 1
`

		require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

		binary := &mockBinary{path: forgePath}
		cli := NewCLI(binary)

		version, err := cli.Version(context.Background())

		require.NoError(t, err)
		require.Equal(t, "forge 0.2.0 (abc123 2023-01-01T00:00:00.000000000Z)", version)
	})

	t.Run("version command fails", func(t *testing.T) {
		script := `#!/bin/bash
echo "error message" >&2
exit 1
`

		require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

		binary := &mockBinary{path: forgePath}
		cli := NewCLI(binary)

		_, err := cli.Version(context.Background())

		require.Error(t, err)
		require.Contains(t, err.Error(), "forge --version failed with exit code 1")
	})
}

func TestCLI_IsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	forgePath := filepath.Join(tmpDir, "forge")

	t.Run("forge is installed", func(t *testing.T) {
		script := `#!/bin/bash
if [ "$1" = "--version" ]; then
    echo "forge 0.2.0"
    exit 0
fi
exit 1
`

		require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

		binary := &mockBinary{path: forgePath}
		cli := NewCLI(binary)

		require.True(t, cli.IsInstalled(context.Background()))
	})

	t.Run("forge is not installed", func(t *testing.T) {
		binary := &mockBinary{path: "/nonexistent/forge"}
		cli := NewCLI(binary)

		require.False(t, cli.IsInstalled(context.Background()))
	})
}

func TestExecuteResult(t *testing.T) {
	tmpDir := t.TempDir()
	forgePath := filepath.Join(tmpDir, "forge")

	t.Run("non-zero exit code", func(t *testing.T) {
		script := `#!/bin/bash
echo "command failed" >&2
exit 42
`

		require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

		binary := &mockBinary{path: forgePath}
		cli := NewCLI(binary)

		result, err := cli.Execute(context.Background(), "failing-command")

		require.NoError(t, err) // Execute should not return error for non-zero exit codes
		require.Equal(t, 42, result.ExitCode)
		require.Contains(t, result.Stderr, "command failed")
	})
}

// Integration test with real binary interface
func TestCLI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary binary that we can use for testing
	binary, err := AutodetectBinary()
	require.NoError(t, err)

	cli := NewCLI(binary)

	// Test that we can check if forge is available
	ctx := context.Background()

	// This will either find forge on PATH or download it
	err = binary.Ensure(ctx)
	if err != nil {
		t.Skipf("Could not ensure forge binary: %v", err)
	}

	// Test version command
	version, err := cli.Version(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, version)
	require.True(t, strings.Contains(version, "forge"))

	// Test IsInstalled
	require.True(t, cli.IsInstalled(ctx))
}

func TestExecuteResult_ParseJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		result := &ExecuteResult{
			Stdout: `{"name": "test", "value": 42}`,
		}

		var data struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		err := result.ParseJSON(&data)
		require.NoError(t, err)
		require.Equal(t, "test", data.Name)
		require.Equal(t, 42, data.Value)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		result := &ExecuteResult{
			Stdout: `invalid json`,
		}

		var data map[string]interface{}
		err := result.ParseJSON(&data)
		require.Error(t, err)
	})

	t.Run("empty stdout", func(t *testing.T) {
		result := &ExecuteResult{
			Stdout: "",
		}

		var data map[string]interface{}
		err := result.ParseJSON(&data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no stdout to parse")
	})
}

func TestCLI_ExecuteForJSON(t *testing.T) {
	tmpDir := t.TempDir()
	forgePath := filepath.Join(tmpDir, "forge")

	t.Run("successful JSON execution", func(t *testing.T) {
		script := `#!/bin/bash
if [[ "$*" == *"--json"* ]]; then
    echo '{"success": true, "message": "test"}'
    exit 0
fi
echo "non-json output"
exit 0
`

		require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

		binary := &mockBinary{path: forgePath}
		cli := NewCLI(binary)

		var result struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		err := cli.ExecuteForJSON(context.Background(), &result, "test-command")
		require.NoError(t, err)
		require.True(t, result.Success)
		require.Equal(t, "test", result.Message)
	})

	t.Run("command failure", func(t *testing.T) {
		script := `#!/bin/bash
echo "error message" >&2
exit 1
`

		require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

		binary := &mockBinary{path: forgePath}
		cli := NewCLI(binary)

		var result map[string]interface{}
		err := cli.ExecuteForJSON(context.Background(), &result, "failing-command")
		require.Error(t, err)
		require.Contains(t, err.Error(), "forge command failed with exit code 1")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		script := `#!/bin/bash
echo "not json output"
exit 0
`

		require.NoError(t, os.WriteFile(forgePath, []byte(script), 0755))

		binary := &mockBinary{path: forgePath}
		cli := NewCLI(binary)

		var result map[string]interface{}
		err := cli.ExecuteForJSON(context.Background(), &result, "test-command")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse JSON output")
	})
}
