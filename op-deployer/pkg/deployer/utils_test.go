package deployer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureDefaultCacheDir(t *testing.T) {
	cacheDir := EnsureDefaultCacheDir()
	require.NotNil(t, cacheDir)
}
