package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/enclave"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCmd struct {
	output []byte
	err    error
	dir    string
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func (c *testCmd) CombinedOutput() ([]byte, error) {
	return c.output, c.err
}

func (c *testCmd) Dir() string {
	return c.dir
}

func (c *testCmd) SetDir(dir string) {
	c.dir = dir
}

func (c *testCmd) Run() error {
	return c.err
}

func (c *testCmd) SetOutput(stdout, stderr *bytes.Buffer) {
	c.stdout = stdout
	c.stderr = stderr
}

func testCmdFactory(output []byte, err error) cmdFactory {
	return func(name string, arg ...string) cmdRunner {
		return &testCmd{output: output, err: err}
	}
}

type testEnclaveFS struct {
	fs       afero.Fs
	artifact string
}

func (e *testEnclaveFS) GetAllFilesArtifactNamesAndUuids(ctx context.Context) ([]*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid, error) {
	if e.artifact == "" {
		return []*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid{}, nil
	}
	return []*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid{
		{
			FileName: e.artifact,
			FileUuid: "test-uuid",
		},
	}, nil
}

func (e *testEnclaveFS) DownloadFilesArtifact(ctx context.Context, name string) ([]byte, error) {
	return []byte("test content"), nil
}

func (e *testEnclaveFS) UploadFiles(pathToUpload string, artifactName string) (services.FilesArtifactUUID, services.FileArtifactName, error) {
	e.artifact = artifactName
	return "test-uuid", services.FileArtifactName(artifactName), nil
}

func (m *testEnclaveFS) GetAllArtifactNames(ctx context.Context) ([]string, error) {
	if m.artifact == "" {
		return []string{}, nil
	}
	return []string{m.artifact}, nil
}

func (m *testEnclaveFS) PutArtifact(ctx context.Context, artifactName string, content []byte) error {
	reader := bytes.NewReader(content)
	file, err := m.fs.Create(artifactName)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}
	m.artifact = artifactName
	return nil
}

func (m *testEnclaveFS) GetArtifact(ctx context.Context, name string) (*ktfs.Artifact, error) {
	return nil, fmt.Errorf("not implemented")
}

type testEnclaveContext struct{}

func (e *testEnclaveContext) RunStarlarkPackage(ctx context.Context, pkg string, config *starlark_run_config.StarlarkRunConfig) (<-chan interfaces.StarlarkResponse, string, error) {
	return nil, "", nil
}

func (e *testEnclaveContext) RunStarlarkScript(ctx context.Context, script string, config *starlark_run_config.StarlarkRunConfig) error {
	return nil
}

type testKurtosisContext struct{}

func (c *testKurtosisContext) CreateEnclave(ctx context.Context, name string) (interfaces.EnclaveContext, error) {
	return &testEnclaveContext{}, nil
}

func (c *testKurtosisContext) GetEnclave(ctx context.Context, name string) (interfaces.EnclaveContext, error) {
	return &testEnclaveContext{}, nil
}

func setupTestFS(t *testing.T) (*testEnclaveFS, afero.Fs) {
	fs := afero.NewMemMapFs()

	// Create the contracts directory structure
	contractsDir := filepath.Join(".", relativeContractsPath)
	require.NoError(t, fs.MkdirAll(contractsDir, 0755))

	// Create a mock solidity cache file
	cacheDir := filepath.Join(contractsDir, "cache")
	require.NoError(t, fs.MkdirAll(cacheDir, 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(cacheDir, "solidity-files-cache.json"), []byte("test cache"), 0644))

	// Create forge-artifacts directory with test files
	forgeDir := filepath.Join(contractsDir, "forge-artifacts")
	require.NoError(t, fs.MkdirAll(forgeDir, 0755))

	// Create some test contract artifacts
	contractDirs := []string{"Contract1.sol", "Contract2.sol"}
	for _, dir := range contractDirs {
		artifactDir := filepath.Join(forgeDir, dir)
		require.NoError(t, fs.MkdirAll(artifactDir, 0755))
		require.NoError(t, afero.WriteFile(fs, filepath.Join(artifactDir, "artifact.json"), []byte("test artifact"), 0644))
	}

	// Create a test contract directory
	testContractDir := filepath.Join(forgeDir, "TestContract.t.sol")
	require.NoError(t, fs.MkdirAll(testContractDir, 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(testContractDir, "artifact.json"), []byte("test artifact"), 0644))

	return &testEnclaveFS{fs: fs}, fs
}

func TestContractBuilder_Build(t *testing.T) {
	tests := []struct {
		name           string
		setupCmd       func() *testCmd
		expectError    bool
		expectedOutput string
	}{
		{
			name: "successful build",
			setupCmd: func() *testCmd {
				return &testCmd{
					output: []byte("build successful"),
					err:    nil,
					dir:    ".",
				}
			},
			expectError:    false,
			expectedOutput: "artifact://contracts-ce0456a3c5a930d170e08492989cf52b416562106c8040bc384548bfe142eaa2", // hash of "test cache"
		},
		{
			name: "build command fails",
			setupCmd: func() *testCmd {
				return &testCmd{
					output: []byte("build failed"),
					err:    fmt.Errorf("command failed"),
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test filesystem
			testFS, memFS := setupTestFS(t)

			// Create mock command
			mockCmd := tt.setupCmd()

			// Create mock enclave manager
			enclaveManager, err := enclave.NewKurtosisEnclaveManager(
				enclave.WithKurtosisContext(&testKurtosisContext{}),
			)
			require.NoError(t, err)

			// Create contract builder with mocks
			builder := NewContractBuilder(
				WithContractFS(memFS),
				WithContractBaseDir("."),
				WithContractDryRun(false),
			)
			builder.cmdFactory = testCmdFactory(mockCmd.output, mockCmd.err)
			builder.enclaveFS, err = ktfs.NewEnclaveFS(context.Background(), "test-enclave", ktfs.WithEnclaveCtx(testFS), ktfs.WithFs(memFS))
			require.NoError(t, err)
			builder.enclaveManager = enclaveManager

			// Execute build
			output, err := builder.Build("")

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, output)
			assert.Equal(t, mockCmd.dir, ".")
		})
	}
}

func TestContractBuilder_createContractsArtifact(t *testing.T) {
	testFS, memFS := setupTestFS(t)

	// Create mock enclave manager
	enclaveManager, err := enclave.NewKurtosisEnclaveManager(
		enclave.WithKurtosisContext(&testKurtosisContext{}),
	)
	require.NoError(t, err)

	builder := NewContractBuilder(
		WithContractFS(memFS),
		WithContractBaseDir("."),
	)
	builder.enclaveFS, err = ktfs.NewEnclaveFS(context.Background(), "test-enclave", ktfs.WithEnclaveCtx(testFS), ktfs.WithFs(memFS))
	require.NoError(t, err)
	builder.enclaveManager = enclaveManager

	// Create the artifact
	name, err := builder.createContractsArtifact()
	require.NoError(t, err)

	// Verify the artifact was created
	artifacts, err := builder.enclaveFS.GetAllArtifactNames(context.Background())
	require.NoError(t, err)
	assert.Contains(t, artifacts, name)

	// Verify it skips test contracts
	for _, artifact := range artifacts {
		assert.NotContains(t, artifact, "TestContract.t.sol")
	}
}

func TestContractBuilder_buildHash(t *testing.T) {
	_, memFS := setupTestFS(t)

	builder := NewContractBuilder(
		WithContractFS(memFS),
		WithContractBaseDir("."),
	)

	// Get the hash
	hash := builder.buildHash()

	// Verify it's not empty or "error"
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "error", hash)

	// Verify it's consistent
	hash2 := builder.buildHash()
	assert.Equal(t, hash, hash2)

	// Modify the cache file and verify the hash changes
	cacheFile := filepath.Join(".", relativeContractsPath, solidityCachePath)
	err := afero.WriteFile(memFS, cacheFile, []byte("modified cache"), 0644)
	require.NoError(t, err)

	hash3 := builder.buildHash()
	assert.NotEqual(t, hash, hash3)
}

func TestContractBuilder_populateContractsArtifact(t *testing.T) {
	_, memFS := setupTestFS(t)

	builder := NewContractBuilder(
		WithContractFS(memFS),
		WithContractBaseDir("."),
	)

	// Create a temporary directory for the test
	tempDir, err := afero.TempDir(memFS, "", "test-artifacts-*")
	require.NoError(t, err)
	defer func() {
		_ = memFS.RemoveAll(tempDir)
	}()

	// Populate the artifacts
	contractsDir := filepath.Join(".", relativeContractsPath)
	err = builder.populateContractsArtifact(contractsDir, tempDir)
	require.NoError(t, err)

	// Verify the directory structure
	exists, err := afero.DirExists(memFS, filepath.Join(tempDir, "Contract1.sol"))
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = afero.DirExists(memFS, filepath.Join(tempDir, "Contract2.sol"))
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify test contracts are not copied
	exists, err = afero.DirExists(memFS, filepath.Join(tempDir, "TestContract.t.sol"))
	assert.NoError(t, err)
	assert.False(t, exists)

	// Verify file contents
	content, err := afero.ReadFile(memFS, filepath.Join(tempDir, "Contract1.sol", "artifact.json"))
	require.NoError(t, err)
	assert.Equal(t, "test artifact", string(content))
}
