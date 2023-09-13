package genesis_test

import (
	"context"
	"testing"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/bindings"
	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/deployer"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/genesis"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/stretchr/testify/require"
)

func TestBuildL2DeveloperGenesis(t *testing.T) {
	config, err := genesis.NewDeployConfig("./testdata/test-deploy-config-devnet-l1.json")
	require.NoError(t, err)

	backend := deployer.NewBackend()
	block, err := backend.BlockByNumber(context.Background(), common.Big0)
	require.NoError(t, err)

	gen, err := genesis.BuildL2DeveloperGenesis(config, block.Header())
	require.NoError(t, err)
	require.NotNil(t, gen)

	depB, err := bindings.GetDeployedBytecode("Proxy")
	require.NoError(t, err)

	for name, address := range predeploys.Predeploys {
		addr := *address

		account, ok := gen.Alloc[addr]
		require.Equal(t, ok, true)
		require.NotEmpty(t, len(account.Code), 0)

		if name == "GovernanceToken" || name == "LegacyERC20ETH" || name == "ProxyAdmin" || name == "WETH9" || name == "BobaL2" {
			continue
		}

		adminSlot, ok := account.Storage[genesis.AdminSlot]
		require.Equal(t, ok, true)
		require.Equal(t, adminSlot, predeploys.ProxyAdminAddr.Hash())
		require.Equal(t, account.Code, depB)
	}
	require.Equal(t, 2067, len(gen.Alloc))
}
