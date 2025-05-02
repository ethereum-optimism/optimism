package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kurtosis-tech/kurtosis/api/golang/kurtosis_version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngineManager(t *testing.T) {
	tests := []struct {
		name           string
		opts           []Option
		expectedBinary string
		expectedVer    string
	}{
		{
			name:           "default options",
			opts:           []Option{},
			expectedBinary: "kurtosis",
			expectedVer:    kurtosis_version.KurtosisVersion,
		},
		{
			name:           "custom binary path",
			opts:           []Option{WithKurtosisBinary("/custom/path/kurtosis")},
			expectedBinary: "/custom/path/kurtosis",
			expectedVer:    kurtosis_version.KurtosisVersion,
		},
		{
			name:           "custom version",
			opts:           []Option{WithVersion("1.0.0")},
			expectedBinary: "kurtosis",
			expectedVer:    "1.0.0",
		},
		{
			name:           "custom binary and version",
			opts:           []Option{WithKurtosisBinary("/custom/path/kurtosis"), WithVersion("1.0.0")},
			expectedBinary: "/custom/path/kurtosis",
			expectedVer:    "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewEngineManager(tt.opts...)
			assert.Equal(t, tt.expectedBinary, manager.kurtosisBinary)
			assert.Equal(t, tt.expectedVer, manager.version)
		})
	}
}

func TestEnsureRunning(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kurtosis-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock kurtosis binary that captures and verifies the arguments
	mockBinary := filepath.Join(tempDir, "kurtosis")
	mockScript := `#!/bin/sh
if [ "$1" = "engine" ] && [ "$2" = "start" ] && [ "$3" = "--version" ] && [ "$4" = "test-version" ]; then
    echo "Engine started with version test-version"
    exit 0
else
    echo "Invalid arguments: $@"
    exit 1
fi`
	err = os.WriteFile(mockBinary, []byte(mockScript), 0755)
	require.NoError(t, err)

	manager := NewEngineManager(
		WithKurtosisBinary(mockBinary),
		WithVersion("test-version"),
	)
	err = manager.EnsureRunning()
	assert.NoError(t, err)
}

func TestGetEngineType(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kurtosis-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock kurtosis binary that simulates cluster commands
	mockBinary := filepath.Join(tempDir, "kurtosis")
	mockScript := `#!/bin/sh
if [ "$1" = "cluster" ] && [ "$2" = "get" ]; then
    echo "test-cluster"
elif [ "$1" = "config" ] && [ "$2" = "path" ]; then
    echo "` + tempDir + `/config.yaml"
else
    exit 1
fi`
	err = os.WriteFile(mockBinary, []byte(mockScript), 0755)
	require.NoError(t, err)

	// Create a mock config file
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
kurtosis-clusters:
  test-cluster:
    type: docker
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	manager := NewEngineManager(WithKurtosisBinary(mockBinary))
	engineType, err := manager.GetEngineType()
	assert.NoError(t, err)
	assert.Equal(t, "docker", engineType)
}

func TestGetEngineType_NoCluster(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kurtosis-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock kurtosis binary that simulates no cluster set
	mockBinary := filepath.Join(tempDir, "kurtosis")
	mockScript := `#!/bin/sh
if [ "$1" = "cluster" ] && [ "$2" = "get" ]; then
    exit 1
elif [ "$1" = "cluster" ] && [ "$2" = "ls" ]; then
    echo "default-cluster"
else
    exit 1
fi`
	err = os.WriteFile(mockBinary, []byte(mockScript), 0755)
	require.NoError(t, err)

	manager := NewEngineManager(WithKurtosisBinary(mockBinary))
	engineType, err := manager.GetEngineType()
	assert.NoError(t, err)
	assert.Equal(t, "default-cluster", engineType)
}
