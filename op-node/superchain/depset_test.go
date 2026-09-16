package superchain

import (
	"testing"

	"github.com/stretchr/testify/require"

	registry "github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestLoadDependencySet(t *testing.T) {
	chainID, err := registry.ChainIDByName("op-mainnet")
	require.NoError(t, err)
	depSet, err := LoadDependencySet(eth.ChainIDFromUInt64(chainID))
	require.NoError(t, err)
	require.True(t, depSet.HasChain(eth.ChainIDFromUInt64(chainID)))
}
