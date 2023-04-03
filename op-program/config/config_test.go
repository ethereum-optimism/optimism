package config

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfigIsValid(t *testing.T) {
	err := NewConfig(&chaincfg.Goerli).Check()
	require.NoError(t, err)
}

func TestRollupConfig(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		err := NewConfig(nil).Check()
		require.ErrorIs(t, err, ErrMissingRollupConfig)
	})

	t.Run("Valid", func(t *testing.T) {
		err := NewConfig(&rollup.Config{}).Check()
		require.ErrorIs(t, err, rollup.ErrBlockTimeZero)
	})
}
