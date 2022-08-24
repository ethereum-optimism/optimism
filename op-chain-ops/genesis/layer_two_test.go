package genesis_test

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestBuildOptimismGenesis(t *testing.T) {
	tmpdir := filepath.Join(os.TempDir(), fmt.Sprintf("l2-test-%d", time.Now().Unix()))
	fmt.Println(tmpdir)
	require.NoError(t, untar("testdata/artifacts.tar.gz", tmpdir))

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

	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)

	gen, err := genesis.BuildOptimismDeveloperGenesis(hh, config, backend)
	require.Nil(t, err)
	require.NotNil(t, gen)

	proxyAdmin, err := hh.GetDeployment("ProxyAdmin")
	require.Nil(t, err)
	proxy, err := hh.GetArtifact("Proxy")
	require.Nil(t, err)

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
		_ = ioutil.WriteFile("genesis.json", file, 0644)
	}
}

func untar(tarball, target string) error {
	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
