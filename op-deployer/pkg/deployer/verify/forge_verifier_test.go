package verify

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts/bundle"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

const testFoundryTomlContent = `[profile.default]
src = 'src'
out = 'forge-artifacts'
`

func TestNewForgeVerifier_HTTPLocator(t *testing.T) {
	var archive bytes.Buffer
	zw, err := zstd.NewWriter(&archive)
	require.NoError(t, err)
	require.NoError(t, bundle.Write(zw, writeTestForgeProject(t), nil, ""))
	require.NoError(t, zw.Close())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archive.Bytes())
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

	projectDir := writeTestForgeProject(t)

	loc, err := artifacts.NewFileLocator(projectDir)
	require.NoError(t, err)

	ctx := context.Background()
	artifactsFS, err := artifacts.Download(ctx, loc, nil, testCacheDir)
	require.NoError(t, err)

	_, err = createTestForgeVerifier(artifactsFS)
	require.NoError(t, err, "should successfully initialize forge verifier with file locator")
}

func TestNewForgeVerifier_RejectsArtifactsOutsideProject(t *testing.T) {
	artifactsDir := filepath.Join(t.TempDir(), "forge-artifacts")
	require.NoError(t, os.Mkdir(artifactsDir, 0o755))
	_, err := createTestForgeVerifier(os.DirFS(artifactsDir).(foundry.StatDirFs))
	require.ErrorContains(t, err, "not a forge project")
}

func writeTestForgeProject(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	for name, content := range map[string]string{
		"foundry.toml":                       testFoundryTomlContent,
		bundle.CompileCacheFile:              `{"paths":{"artifacts":"forge-artifacts"},"files":{"src/Test.sol":{}}}`,
		"src/Test.sol":                       "contract Test {}",
		"forge-artifacts/Test.sol/Test.json": `{"abi":[]}`,
	} {
		path := filepath.Join(project, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return project
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
