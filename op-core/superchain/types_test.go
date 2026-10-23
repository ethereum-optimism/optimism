package superchain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainConfigDependencies(t *testing.T) {
	t.Run("no interop config depends on itself", func(t *testing.T) {
		cfg := &ChainConfig{ChainID: 42}
		require.Equal(t, map[string]Dependency{"42": {}}, cfg.Dependencies())
		require.Nil(t, cfg.Interop, "Dependencies must not modify the config")
	})

	t.Run("interop config dependencies", func(t *testing.T) {
		deps := map[string]Dependency{"42": {}, "43": {}}
		cfg := &ChainConfig{ChainID: 42, Interop: &Interop{Dependencies: deps}}
		require.Equal(t, deps, cfg.Dependencies())
	})
}
