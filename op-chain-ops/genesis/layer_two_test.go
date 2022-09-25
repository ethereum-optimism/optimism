package genesis_test

import (
	"context"
	"encoding/json"
	"flag"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/stretchr/testify/require"
)

var writeFile bool

func init() {
	flag.BoolVar(&writeFile, "write-file", false, "write the genesis file to disk")
}

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

func TestBuildL2DeveloperGenesis(t *testing.T) {
	hh, err := hardhat.New(
		"alpha-1",
		nil,
		[]string{"../../packages/contracts-bedrock/deployments"},
	)
	require.Nil(t, err)

	config, err := genesis.NewDeployConfig("./testdata/test-deploy-config-devnet-l1.json")
	require.Nil(t, err)

	proxyAdmin, err := hh.GetDeployment("ProxyAdmin")
	require.Nil(t, err)

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)
	block, err := backend.BlockByNumber(context.Background(), common.Big0)
	require.NoError(t, err)
	gen, err := genesis.BuildL2DeveloperGenesis(config, block, &genesis.L2Addresses{
		ProxyAdmin: proxyAdmin.Address,
	})
	require.Nil(t, err)
	require.NotNil(t, gen)

	depB, err := bindings.GetDeployedBytecode("Proxy")
	require.NoError(t, err)

	for name, address := range predeploys.Predeploys {
		addr := *address

		account, ok := gen.Alloc[addr]
		require.Equal(t, ok, true)
		require.Greater(t, len(account.Code), 0)

		if name == "GovernanceToken" || name == "LegacyERC20ETH" {
			continue
		}

		adminSlot, ok := account.Storage[genesis.AdminSlot]
		require.Equal(t, ok, true)
		require.Equal(t, adminSlot, proxyAdmin.Address.Hash())
		require.Equal(t, account.Code, depB)
	}
	require.Equal(t, 2340, len(gen.Alloc))

	if writeFile {
		file, _ := json.MarshalIndent(gen, "", " ")
		_ = os.WriteFile("genesis.json", file, 0644)
	}
}

func TestBuildL2DeveloperGenesisDevAccountsFunding(t *testing.T) {
	config, err := genesis.NewDeployConfig("./testdata/test-deploy-config-devnet-l1.json")
	require.Nil(t, err)
	config.FundDevAccounts = false

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)
	block, err := backend.BlockByNumber(context.Background(), common.Big0)
	require.NoError(t, err)
	gen, err := genesis.BuildL2DeveloperGenesis(config, block, &genesis.L2Addresses{
		ProxyAdmin: common.Address{},
	})
	require.NoError(t, err)
	require.Equal(t, 2319, len(gen.Alloc))
}
