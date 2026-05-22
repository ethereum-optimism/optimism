package legacy

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/stretchr/testify/require"
)

func TestStaticMappingsMatchArtifacts(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	dir := filepath.Dir(file)
	repoRoot := filepath.Clean(filepath.Join(dir, "..", "..", ".."))
	artifactsDir := foundry.OpenArtifactsDir(filepath.Join(repoRoot, "packages/contracts-bedrock/forge-artifacts"))

	matches, err := filepath.Glob(filepath.Join(dir, "*.input.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, matches)

	for _, mappingPath := range matches {
		t.Run(filepath.Base(mappingPath), func(t *testing.T) {
			mapping, err := opcm.LoadStaticInputMappingYAMLFile(mappingPath)
			require.NoError(t, err)

			scriptFile := filepath.ToSlash(filepath.Dir(mapping.Script.Artifact))
			contract := strings.TrimSuffix(filepath.Base(mapping.Script.Artifact), ".json")
			require.Equal(t, mapping.Script.Contract, contract)

			artifact, err := artifactsDir.ReadArtifact(scriptFile, contract)
			require.NoError(t, err)
			require.NoError(t, opcm.ValidateStaticInputMapping(artifact.ABI, *mapping))
		})
	}
}
