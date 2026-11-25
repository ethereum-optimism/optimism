package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const pcdKonaInteropPrestateArtifact = "rust/kona/prestate-artifacts-cannon-interop/prestate-proof.json"

type pcdPrestateFile struct {
	Pre common.Hash `json:"pre"`
}

func readPCDPrestate(path string) (common.Hash, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return common.Hash{}, pcdPrestateReadError(path, err)
	}

	var artifact pcdPrestateFile
	if err := json.Unmarshal(data, &artifact); err != nil {
		return common.Hash{}, pcdPrestateReadError(path, err)
	}
	if artifact.Pre == (common.Hash{}) {
		return common.Hash{}, pcdPrestateReadError(path, fmt.Errorf("pre is empty"))
	}
	if artifact.Pre == opcm.PermissionedCannonFallbackPrestatePlaceholder {
		return common.Hash{}, pcdPrestateReadError(path, fmt.Errorf("pre uses the reserved hash %s", artifact.Pre.Hex()))
	}
	return artifact.Pre, nil
}

func pcdPrestateReadError(path string, err error) error {
	return fmt.Errorf("read Kona interop prestate artifact %s: %w; run just reproducible-prestate-kona", path, err)
}

func pcdPrestateArtifactPath(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve PCD prestate reader source path")
	root, err := op_service.FindMonorepoRoot(sourceFile)
	require.NoError(t, err)
	return filepath.Join(root, filepath.FromSlash(pcdKonaInteropPrestateArtifact))
}

func requirePCDPrestate(t *testing.T, path string) common.Hash {
	t.Helper()
	prestate, err := readPCDPrestate(path)
	if os.Getenv("CI") == "true" {
		require.NoError(t, err)
	} else if err != nil {
		t.Skip(err.Error())
	}
	return prestate
}
