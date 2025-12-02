package verify

import (
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
	testTzstPath := filepath.Join("testdata", "test-with-foundry.toml.tar.zst")
	f, err := os.Open(testTzstPath)
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
	ctx := context.Background()
	loc := artifacts.EmbeddedLocator

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)
	artifactsFS, err := artifacts.Download(ctx, loc, nil, testCacheDir)
	require.NoError(t, err)

	_, err = createTestForgeVerifier(artifactsFS)
	require.NoError(t, err, "should successfully initialize forge verifier with embedded locator")
}

func TestNewForgeVerifier_FileLocator(t *testing.T) {
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	projectDir := filepath.Join(testCacheDir, "test-project")
	forgeArtifactsDir := filepath.Join(projectDir, "forge-artifacts")
	require.NoError(t, os.MkdirAll(forgeArtifactsDir, 0755))

	foundryTomlPath := filepath.Join(projectDir, "foundry.toml")
	require.NoError(t, os.WriteFile(foundryTomlPath, []byte(testFoundryTomlContent), 0644))

	artifactDir := filepath.Join(forgeArtifactsDir, "Test.sol")
	require.NoError(t, os.MkdirAll(artifactDir, 0755))
	artifactFile := filepath.Join(artifactDir, "Test.json")
	require.NoError(t, os.WriteFile(artifactFile, []byte(`{"abi":[]}`), 0644))

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
