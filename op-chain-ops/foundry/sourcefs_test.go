package foundry

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:generate ./testdata/srcmaps/generate.sh

func TestSourceMapFS(t *testing.T) {
	artifactFS := OpenArtifactsDir("./testdata/srcmaps/test-artifacts")
	exampleArtifact, err := artifactFS.ReadArtifact("SimpleStorage.sol", "SimpleStorage")
	require.NoError(t, err)
	srcFS := NewSourceMapFS(os.DirFS("./testdata/srcmaps"))
	srcMap, err := srcFS.SourceMap(exampleArtifact, "SimpleStorage")
	require.NoError(t, err)
	seenInfo := make(map[string]struct{})
	for i := range exampleArtifact.DeployedBytecode.Object {
		seenInfo[srcMap.FormattedInfo(uint64(i))] = struct{}{}
	}
	require.Contains(t, seenInfo, "src/SimpleStorage.sol:11:5")
	require.Contains(t, seenInfo, "src/StorageLibrary.sol:8:9")
}
