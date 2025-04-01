package deploy

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/spec"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployFileserver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs := afero.NewMemMapFs()

	// Create test directories
	sourceDir := "/source"
	require.NoError(t, fs.MkdirAll(sourceDir, 0755))

	// Create required directory structure
	nginxDir := filepath.Join(sourceDir, "static_files", "nginx")
	require.NoError(t, fs.MkdirAll(nginxDir, 0755))

	// Create a mock deployer function
	mockDeployerFunc := func(opts ...kurtosis.KurtosisDeployerOptions) (deployer, error) {
		return &mockDeployer{}, nil
	}

	testCases := []struct {
		name         string
		setup        func(t *testing.T, fs afero.Fs, sourceDir, nginxDir string, state *fileserverState)
		state        *fileserverState
		shouldError  bool
		shouldDeploy bool
	}{
		{
			name: "empty source directory - no deployment needed",
			setup: func(t *testing.T, fs afero.Fs, sourceDir, nginxDir string, state *fileserverState) {
				// No files to create
			},
			state:        &fileserverState{},
			shouldError:  false,
			shouldDeploy: false,
		},
		{
			name: "new files to deploy",
			setup: func(t *testing.T, fs afero.Fs, sourceDir, nginxDir string, state *fileserverState) {
				require.NoError(t, afero.WriteFile(
					fs,
					filepath.Join(sourceDir, "test.txt"),
					[]byte("test content"),
					0644,
				))
			},
			state:        &fileserverState{},
			shouldError:  false,
			shouldDeploy: true,
		},
		{
			name: "no changes - deployment skipped",
			setup: func(t *testing.T, fs afero.Fs, sourceDir, nginxDir string, state *fileserverState) {
				require.NoError(t, afero.WriteFile(
					fs,
					filepath.Join(sourceDir, "test.txt"),
					[]byte("test content"),
					0644,
				))

				// Calculate actual hash for the test file
				hash, err := calculateDirHashWithFs(sourceDir, fs)
				require.NoError(t, err)

				// Calculate nginx config hash
				configHash, err := calculateDirHashWithFs(nginxDir, fs)
				require.NoError(t, err)

				// Update state with actual hashes
				state.contentHash = hash
				state.configHash = configHash
			},
			state:        &fileserverState{},
			shouldError:  false,
			shouldDeploy: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up and recreate source directory for each test
			require.NoError(t, fs.RemoveAll(sourceDir))
			require.NoError(t, fs.MkdirAll(sourceDir, 0755))

			// Recreate nginx directory
			require.NoError(t, fs.MkdirAll(nginxDir, 0755))

			// Setup test files
			tc.setup(t, fs, sourceDir, nginxDir, tc.state)

			// Create a separate directory for the fileserver deployment
			deployBaseDir := "/deploy"
			require.NoError(t, fs.MkdirAll(deployBaseDir, 0755))

			fileServer := &FileServer{
				baseDir:  deployBaseDir,
				enclave:  "test-enclave",
				dryRun:   true,
				deployer: mockDeployerFunc,
				fs:       fs,
			}

			// Create state channel and send test state
			ch := make(chan *fileserverState, 1)
			ch <- tc.state
			close(ch)

			err := fileServer.Deploy(ctx, sourceDir, ch)
			if tc.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify deployment directory was created only if deployment was needed
			deployDir := filepath.Join(deployBaseDir, FILESERVER_PACKAGE)
			exists, err := afero.Exists(fs, deployDir)
			require.NoError(t, err)
			if tc.shouldDeploy {
				assert.True(t, exists)
			}
		})
	}
}

// mockDeployer implements the deployer interface for testing
type mockDeployer struct{}

func (m *mockDeployer) Deploy(ctx context.Context, input io.Reader) (*spec.EnclaveSpec, error) {
	return &spec.EnclaveSpec{}, nil
}

func (m *mockDeployer) GetEnvironmentInfo(ctx context.Context, spec *spec.EnclaveSpec) (*kurtosis.KurtosisEnvironment, error) {
	return &kurtosis.KurtosisEnvironment{}, nil
}
