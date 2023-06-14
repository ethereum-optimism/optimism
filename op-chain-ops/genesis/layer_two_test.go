package genesis_test

import (
	"context"
	"encoding/json"
	"flag"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

var writeFile bool

func init() {
	flag.BoolVar(&writeFile, "write-file", false, "write the genesis file to disk")
}

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

func TestBuildL2DeveloperGenesis(t *testing.T) {
	config, err := genesis.NewDeployConfig("./testdata/test-deploy-config-devnet-l1.json")
	require.Nil(t, err)

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)
	block, err := backend.BlockByNumber(context.Background(), common.Big0)
	require.NoError(t, err)

	gen, err := genesis.BuildL2DeveloperGenesis(config, block)
	require.Nil(t, err)
	require.NotNil(t, gen)

	proxyBytecode, err := bindings.GetDeployedBytecode("Proxy")
	require.NoError(t, err)

	for name, address := range predeploys.Predeploys {
		addr := *address

		account, ok := gen.Alloc[addr]
		require.Equal(t, true, ok)
		require.Greater(t, len(account.Code), 0)

		if name == "GovernanceToken" || name == "LegacyERC20ETH" || name == "ProxyAdmin" || name == "WETH9" {
			continue
		}

		adminSlot, ok := account.Storage[genesis.AdminSlot]
		require.Equal(t, true, ok, name)
		require.Equal(t, predeploys.ProxyAdminAddr.Hash(), adminSlot)
		require.Equal(t, proxyBytecode, account.Code)
	}
	require.Equal(t, 2343, len(gen.Alloc))

	if writeFile {
		file, _ := json.MarshalIndent(gen, "", " ")
		_ = os.WriteFile("genesis.json", file, 0644)
	}
}

func TestBuildL2DeveloperGenesisDevAccountsFunding(t *testing.T) {
	config, err := genesis.NewDeployConfig("./testdata/test-deploy-config-devnet-l1.json")
	require.Nil(t, err)
	config.FundDevAccounts = false

	err = config.InitDeveloperDeployedAddresses()
	require.NoError(t, err)

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)
	block, err := backend.BlockByNumber(context.Background(), common.Big0)
	require.NoError(t, err)

	gen, err := genesis.BuildL2DeveloperGenesis(config, block)
	require.NoError(t, err)
	require.Equal(t, 2321, len(gen.Alloc))
}

// Tests the BuildL2MainnetGenesis factory. enableGovernance is used to override enableGovernance
// config option. When false, the test confirms the governance token predeploy address instead
// holds a proxy contract.
func testBuildL2Genesis(t *testing.T, enableGovernance bool) {
	config, err := genesis.NewDeployConfig("./testdata/test-deploy-config-devnet-l1.json")
	require.Nil(t, err)
	config.EnableGovernance = enableGovernance

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)
	block, err := backend.BlockByNumber(context.Background(), common.Big0)
	require.NoError(t, err)

	gen, err := genesis.BuildL2MainnetGenesis(config, block)
	require.Nil(t, err)
	require.NotNil(t, gen)

	proxyBytecode, err := bindings.GetDeployedBytecode("Proxy")
	require.NoError(t, err)

	for name, predeploy := range predeploys.Predeploys {
		addr := *predeploy

		account, ok := gen.Alloc[addr]
		if predeploys.IsDeprecated(addr) && !predeploys.IsProxied(addr) {
			// deprecated, non-proxied predeploys should have no account
			require.Equal(t, false, ok, name)
			continue
		}
		require.Equal(t, true, ok, name)
		require.Greater(t, len(account.Code), 0)

		adminSlot, ok := account.Storage[genesis.AdminSlot]
		isProxy := predeploys.IsProxied(addr) ||
			(!enableGovernance && addr == predeploys.GovernanceTokenAddr)
		if isProxy {
			require.Equal(t, true, ok, name)
			require.Equal(t, predeploys.ProxyAdminAddr.Hash(), adminSlot)
			require.Equal(t, proxyBytecode, account.Code)
		} else {
			require.Equal(t, false, ok, name)
			require.NotEqual(t, proxyBytecode, account.Code, name)
		}
	}
	require.Equal(t, 2063, len(gen.Alloc))

	if writeFile {
		file, _ := json.MarshalIndent(gen, "", " ")
		_ = os.WriteFile("genesis.json", file, 0644)
	}
}

func TestBuildL2MainnetGenesis(t *testing.T) {
	testBuildL2Genesis(t, true)
}

func TestBuildL2MainnetNoGovernanceGenesis(t *testing.T) {
	testBuildL2Genesis(t, false)
}
