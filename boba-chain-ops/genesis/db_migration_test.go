package genesis

import (
	"testing"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/ether"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

func TestRetrieveLegacyTuringCredit(t *testing.T) {
	g := &types.Genesis{
		Alloc: types.GenesisAlloc{
			predeploys.BobaLegacyTuringCreditAddr: types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{
					ether.BobaLegacyProxyOwnerSlot:          {1},
					ether.BobaLegacyProxyImplementationSlot: {2},
					{3}:                                     {4},
					{5}:                                     {6},
				},
			},
		},
	}
	expected := map[common.Hash]common.Hash{
		{3}: {4},
		{5}: {6},
	}
	legacyStorge := RetrieveLegacyTuringCredit(g)
	require.Equal(t, expected, legacyStorge)
}

func TestMigrateTuringCredit(t *testing.T) {
	g := &types.Genesis{
		Alloc: types.GenesisAlloc{
			predeploys.BobaLegacyTuringCreditAddr: types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{
					ether.BobaLegacyProxyOwnerSlot:          {1},
					ether.BobaLegacyProxyImplementationSlot: {2},
					{1}:                                     {1},
					{2}:                                     {2},
				},
			},
		},
	}

	expected := map[common.Hash]common.Hash{
		AdminSlot: predeploys.ProxyAdminAddr.Hash(),
		{1}:       {1},
		{2}:       {2},
	}
	legacyTuringCredit := RetrieveLegacyTuringCredit(g)

	err := WipePredeployStorage(g)
	require.NoError(t, err)
	err = SetL2Proxies(g)
	require.NoError(t, err)

	err = ether.MigrateTuringCredit(g, legacyTuringCredit, true)
	require.NoError(t, err)

	require.Equal(t, expected, g.Alloc[predeploys.BobaTuringCreditAddr].Storage)
}

func TestMigrateTuringCreditCheck(t *testing.T) {
	g := &types.Genesis{
		Alloc: types.GenesisAlloc{
			predeploys.BobaLegacyTuringCreditAddr: types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{
					ether.BobaLegacyProxyOwnerSlot:          {1},
					ether.BobaLegacyProxyImplementationSlot: {2},
					AdminSlot:                               predeploys.ProxyAdminAddr.Hash(),
					{1}:                                     {1},
					{2}:                                     {2},
				},
			},
		},
	}

	legacyTuringCredit := RetrieveLegacyTuringCredit(g)
	err := WipePredeployStorage(g)
	require.NoError(t, err)
	err = SetL2Proxies(g)
	require.NoError(t, err)

	err = ether.MigrateTuringCredit(g, legacyTuringCredit, false)
	require.ErrorContains(t, err, "duplicate address")
}
