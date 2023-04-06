package config

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var validRollupConfig = &chaincfg.Goerli
var validL2GenesisPath = "genesis.json"
var validL2Head = common.HexToHash("0x6303578b1fa9480389c51bbcef6fe045bb877da39740819e9eb5f36f94949bd0")

func TestDefaultConfigIsValid(t *testing.T) {
	err := NewConfig(validRollupConfig, validL2GenesisPath, validL2Head).Check()
	require.NoError(t, err)
}

func TestRollupConfig(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		err := NewConfig(nil, validL2GenesisPath, validL2Head).Check()
		require.ErrorIs(t, err, ErrMissingRollupConfig)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := NewConfig(&rollup.Config{}, validL2GenesisPath, validL2Head).Check()
		require.ErrorIs(t, err, rollup.ErrBlockTimeZero)
	})
}

func TestL2Genesis(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		err := NewConfig(validRollupConfig, "", validL2Head).Check()
		require.ErrorIs(t, err, ErrMissingL2Genesis)
	})

	t.Run("Valid", func(t *testing.T) {
		err := NewConfig(validRollupConfig, validL2GenesisPath, validL2Head).Check()
		require.NoError(t, err)
	})
}

func TestL2Head(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		err := NewConfig(validRollupConfig, validL2GenesisPath, common.Hash{}).Check()
		require.ErrorIs(t, err, ErrInvalidL2Head)
	})

	t.Run("Valid", func(t *testing.T) {
		err := NewConfig(validRollupConfig, validL2GenesisPath, validL2Head).Check()
		require.NoError(t, err)
	})
}

func TestFetchingEnabled(t *testing.T) {
	t.Run("FetchingNotEnabledWhenNoFetcherUrlsSpecified", func(t *testing.T) {
		cfg := NewConfig(&chaincfg.Beta1, validL2GenesisPath, validL2Head)
		require.False(t, cfg.FetchingEnabled(), "Should not enable fetching when node URL not supplied")
	})

	t.Run("FetchingEnabledWhenFetcherUrlsSpecified", func(t *testing.T) {
		cfg := NewConfig(&chaincfg.Beta1, validL2GenesisPath, validL2Head)
		cfg.L2URL = "https://example.com:1234"
		require.True(t, cfg.FetchingEnabled(), "Should enable fetching when node URL supplied")
	})
}
