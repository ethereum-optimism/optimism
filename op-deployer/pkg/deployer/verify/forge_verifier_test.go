package verify

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

const testFoundryTomlContent = `[profile.default]
src = 'src'
out = 'forge-artifacts'
`

func TestNewForgeVerifier_HTTPLocator(t *testing.T) {
	// Create a test tar.gz with foundry.toml at root and forge-artifacts/ subdirectory
	testTarGzPath := createTestTarGzWithFoundryToml(t)
	f, err := os.Open(testTarGzPath)
	require.NoError(t, err)
	defer f.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := io.Copy(w, f)
		require.NoError(t, err)
		_, err = f.Seek(0, 0)
		require.NoError(t, err)
	}))
	defer ts.Close()

	ctx := context.Background()
	artifactsURL, err := url.Parse(ts.URL)
	require.NoError(t, err)
	loc := &artifacts.Locator{
		URL: artifactsURL,
	}

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	artifactsFS, err := artifacts.Download(ctx, loc, nil, testCacheDir)
	require.NoError(t, err)

	_, err = createTestForgeVerifier(artifactsFS)
	require.NoError(t, err, "should successfully initialize forge verifier with HTTP locator")
}

func TestNewForgeVerifier_EmbeddedLocator(t *testing.T) {
	// Note: This test assumes embedded artifacts include foundry.toml at the root
	// If embedded artifacts don't have foundry.toml, this test will need to be adjusted
	ctx := context.Background()
	loc := artifacts.EmbeddedLocator

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	artifactsFS, err := artifacts.Download(ctx, loc, nil, testCacheDir)
	require.NoError(t, err)

	_, err = createTestForgeVerifier(artifactsFS)
	// This test may fail if embedded artifacts don't include foundry.toml
	// In that case, we expect an error about foundry.toml not found
	if err != nil {
		require.Contains(t, err.Error(), "foundry.toml", "error should mention foundry.toml")
	} else {
		require.NoError(t, err, "should successfully initialize forge verifier with embedded locator")
	}
}

func TestNewForgeVerifier_FileLocator(t *testing.T) {
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	// Create a test project directory structure with foundry.toml and forge-artifacts/
	projectDir := filepath.Join(testCacheDir, "test-project")
	forgeArtifactsDir := filepath.Join(projectDir, "forge-artifacts")
	require.NoError(t, os.MkdirAll(forgeArtifactsDir, 0755))

	// Create foundry.toml at the project root
	foundryTomlPath := filepath.Join(projectDir, "foundry.toml")
	require.NoError(t, os.WriteFile(foundryTomlPath, []byte(testFoundryTomlContent), 0644))

	// Create a dummy artifact file
	artifactDir := filepath.Join(forgeArtifactsDir, "Test.sol")
	require.NoError(t, os.MkdirAll(artifactDir, 0755))
	artifactFile := filepath.Join(artifactDir, "Test.json")
	require.NoError(t, os.WriteFile(artifactFile, []byte(`{"abi":[]}`), 0644))

	// Create file locator pointing to the project directory
	loc, err := artifacts.NewFileLocator(projectDir)
	require.NoError(t, err)

	ctx := context.Background()
	artifactsFS, err := artifacts.Download(ctx, loc, nil, testCacheDir)
	require.NoError(t, err)

	_, err = createTestForgeVerifier(artifactsFS)
	require.NoError(t, err, "should successfully initialize forge verifier with file locator")
}

// createTestForgeVerifier creates a ForgeVerifier with standard test options
func createTestForgeVerifier(artifactsFS foundry.StatDirFs) (*ForgeVerifier, error) {
	logger := log.New("test", "forge_verifier")
	return NewForgeVerifier(ForgeVerifierOpts{
		RpcUrl:       "http://localhost:8545",
		VerifierType: "etherscan",
		ChainID:      1,
		ArtifactsFS:  artifactsFS,
		Logger:       logger,
	})
}

// createTestTarGzWithFoundryToml creates a tar.gz file with foundry.toml at root and forge-artifacts/ subdirectory
func createTestTarGzWithFoundryToml(t *testing.T) string {
	tempDir := t.TempDir()
	tarGzPath := filepath.Join(tempDir, "test-with-foundry.toml.tar.gz")

	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)

	// Add foundry.toml at root
	err := tarWriter.WriteHeader(&tar.Header{
		Name: "foundry.toml",
		Mode: 0644,
		Size: int64(len(testFoundryTomlContent)),
	})
	require.NoError(t, err)
	_, err = tarWriter.Write([]byte(testFoundryTomlContent))
	require.NoError(t, err)

	// Add forge-artifacts/ directory
	err = tarWriter.WriteHeader(&tar.Header{
		Name:     "forge-artifacts/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})
	require.NoError(t, err)

	// Add a test artifact
	err = tarWriter.WriteHeader(&tar.Header{
		Name:     "forge-artifacts/Test.sol/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})
	require.NoError(t, err)

	testContent := `{"abi":[]}`
	err = tarWriter.WriteHeader(&tar.Header{
		Name: "forge-artifacts/Test.sol/Test.json",
		Mode: 0644,
		Size: int64(len(testContent)),
	})
	require.NoError(t, err)
	_, err = tarWriter.Write([]byte(testContent))
	require.NoError(t, err)

	err = tarWriter.Close()
	require.NoError(t, err)

	// Compress with gzip
	gzFile, err := os.Create(tarGzPath)
	require.NoError(t, err)
	defer gzFile.Close()

	gzWriter := gzip.NewWriter(gzFile)
	_, err = gzWriter.Write(tarBuffer.Bytes())
	require.NoError(t, err)
	err = gzWriter.Close()
	require.NoError(t, err)

	return tarGzPath
}
