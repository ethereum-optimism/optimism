package release

import (
	"testing"

	"github.com/ethereum-optimism/optimism/oprm/components"
	"github.com/stretchr/testify/require"
)

func TestMatchChangedFiles(t *testing.T) {
	matched, err := MatchChangedFiles([]string{"op-node/**", "go.mod", "op-service/**"}, []string{
		"docs/readme.md",
		"op-node/node/server.go",
		"op-service/cli/flags.go",
		"go.mod",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"go.mod", "op-node/node/server.go", "op-service/cli/flags.go"}, matched)
}

func TestDetectComponentChanges(t *testing.T) {
	spec := components.NewRegistry().MustGet("op-node")
	detection, err := DetectComponentChanges(spec, "develop", []string{"docs/readme.md", "op-service/foo.go"})
	require.NoError(t, err)
	require.True(t, detection.Changed)
	require.Equal(t, []string{"op-service/foo.go"}, detection.MatchedFiles)
	require.Equal(t, "develop", detection.ComparedRef)
}
