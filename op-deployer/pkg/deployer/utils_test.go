package deployer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureDefaultCacheDir(t *testing.T) {
	cacheDir := DefaultCacheDir()
	require.NotNil(t, cacheDir)
}
