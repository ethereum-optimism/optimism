package params_test

import (
	"encoding/json"
	"testing"

	gethparams "github.com/ethereum/go-ethereum/params"
	gethsuperchain "github.com/ethereum/go-ethereum/superchain"
	"github.com/stretchr/testify/require"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	opsuperchain "github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

func TestLoadChainConfigFromChainID(t *testing.T) {
	t.Run("mainnet", func(t *testing.T) {
		cfg, err := opparams.LoadChainConfigFromChainID(opparams.OPMainnetChainID)
		require.NoError(t, err)
		require.Equal(t, uint64(opparams.OPMainnetChainID), bigs.Uint64Strict(cfg.ChainID))
		require.NotNil(t, cfg.Optimism)
		require.NotNil(t, cfg.RegolithTime)
		require.Equal(t, uint64(0), *cfg.RegolithTime)
		require.NotNil(t, cfg.BedrockBlock)
		require.Equal(t, int64(opparams.OPMainnetGenesisBlockNum), cfg.BedrockBlock.Int64())
	})

	t.Run("nonexistent chain", func(t *testing.T) {
		cfg, err := opparams.LoadChainConfigFromChainID(23409527340)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}

// TestGethChainConfigMatchesOpGeth is a differential test: for every chain in the
// embedded superchain registry, the GethChainConfig converter must produce the same
// go-ethereum config as op-geth's params.LoadOPStackChainConfig. This pins the
// OP→go-ethereum mapping (including the OptimismConfig wire format and the OP-Mainnet
// pre-Bedrock overrides) to op-geth byte-for-byte.
func TestGethChainConfigMatchesOpGeth(t *testing.T) {
	require.NotEmpty(t, opsuperchain.Chains, "registry has no chains to compare")

	for chainID := range opsuperchain.Chains {
		chainID := chainID
		t.Run(chainName(t, chainID), func(t *testing.T) {
			opChain, err := opsuperchain.GetChain(chainID)
			require.NoError(t, err)
			opSC, err := opChain.Config()
			require.NoError(t, err)
			opCfg := opparams.FromSuperchainConfig(opSC)

			gethChain, err := gethsuperchain.GetChain(chainID)
			require.NoError(t, err)
			gethSC, err := gethChain.Config()
			require.NoError(t, err)
			gethCfg, err := gethparams.LoadOPStackChainConfig(gethSC)
			require.NoError(t, err)
			// op-geth's LoadOPStackChainConfig predates the Karst→Osaka wiring and
			// leaves OsakaTime unset; GethChainConfig sets it (Osaka activates with
			// Karst, as op-reth does). Patch the op-geth side so the rest of the
			// config is still compared.
			gethCfg.OsakaTime = gethCfg.KarstTime

			opJSON, err := json.Marshal(opCfg.GethChainConfig())
			require.NoError(t, err)
			gethJSON, err := json.Marshal(gethCfg)
			require.NoError(t, err)
			require.JSONEq(t, string(gethJSON), string(opJSON))
		})
	}
}

func chainName(t *testing.T, chainID uint64) string {
	t.Helper()
	ch, ok := opsuperchain.Chains[chainID]
	if !ok {
		return "unknown"
	}
	return ch.Name + "-" + ch.Network
}
