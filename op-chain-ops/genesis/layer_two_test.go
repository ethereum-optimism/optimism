package genesis_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/hardhat"
	"github.com/stretchr/testify/require"
)

var writeFile bool

func init() {
	flag.BoolVar(&writeFile, "write-file", false, "write the genesis file to disk")
}

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

func TestBuildL2DeveloperGenesis(t *testing.T) {
	tmpdir := filepath.Join(os.TempDir(), fmt.Sprintf("l2-test-%d", time.Now().Unix()))
	require.NoError(t, genesis.Untar("testdata/artifacts.tar.gz", tmpdir))

	hh, err := hardhat.New(
		"goerli",
		[]string{
			filepath.Join(tmpdir, "contracts-bedrock"),
			filepath.Join(tmpdir, "contracts-governance"),
		},
		[]string{"../../packages/contracts-bedrock/deployments"},
	)
	require.Nil(t, err)

	config, err := genesis.NewDeployConfig("./testdata/test-deploy-config-devnet-l1.json")
	require.Nil(t, err)

	proxyAdmin, err := hh.GetDeployment("ProxyAdmin")
	require.Nil(t, err)
	proxy, err := hh.GetArtifact("Proxy")
	require.Nil(t, err)

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)
	block, err := backend.BlockByNumber(context.Background(), common.Big0)
	require.NoError(t, err)
	gen, err := genesis.BuildL2DeveloperGenesis(hh, config, block, &genesis.L2Addresses{
		ProxyAdmin: proxyAdmin.Address,
	})
	require.Nil(t, err)
	require.NotNil(t, gen)

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
		require.Equal(t, account.Code, []byte(proxy.DeployedBytecode))
	}

	if writeFile {
		file, _ := json.MarshalIndent(gen, "", " ")
		_ = os.WriteFile("genesis.json", file, 0644)
	}
}
