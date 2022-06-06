package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"

	rollupEth "github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

func waitForTransaction(hash common.Hash, client *ethclient.Client, timeout time.Duration) (*types.Receipt, error) {
	timeoutCh := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if receipt != nil && err == nil {
			return receipt, nil
		} else if err != nil && !errors.Is(err, ethereum.NotFound) {
			return nil, err
		}

		select {
		case <-timeoutCh:
			return nil, errors.New("timeout")
		case <-ticker.C:
		}
	}
}

func waitForBlock(number *big.Int, client *ethclient.Client, timeout time.Duration) (*types.Block, error) {
	timeoutCh := time.After(timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	headChan := make(chan *types.Header, 100)
	headSub, err := client.SubscribeNewHead(ctx, headChan)
	if err != nil {
		return nil, err
	}
	defer headSub.Unsubscribe()

	for {
		select {
		case head := <-headChan:
			if head.Number.Cmp(number) >= 0 {
				return client.BlockByNumber(ctx, number)
			}
		case err := <-headSub.Err():
			return nil, fmt.Errorf("Error in head subscription: %w", err)
		case <-timeoutCh:
			return nil, errors.New("timeout")
		}
	}
}

func getGenesisInfo(client *ethclient.Client) (id rollupEth.BlockID, timestamp uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	block, err := client.BlockByNumber(ctx, common.Big0)
	if err != nil {
		panic(err)
	}
	return rollupEth.BlockID{Hash: block.Hash(), Number: block.NumberU64()}, block.Time()
}

func initL1Geth(cfg *SystemConfig, wallet *hdwallet.Wallet, genesis *core.Genesis) (*node.Node, *eth.Ethereum, error) {
	signer := deriveAccount(wallet, cfg.CliqueSignerDerivationPath)
	pk, err := wallet.PrivateKey(signer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to locate private key in wallet: %w", err)
	}

	ethConfig := &ethconfig.Config{
		NetworkId: cfg.L1ChainID.Uint64(),
		Genesis:   genesis,
	}
	nodeConfig := &node.Config{
		Name:        "l1-geth",
		WSHost:      "127.0.0.1",
		WSPort:      0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
	}

	return createGethNode(false, nodeConfig, ethConfig, []*ecdsa.PrivateKey{pk})
}

// init a geth node.
func initL2Geth(name string, l2ChainID *big.Int, genesis *core.Genesis, jwtPath string) (*node.Node, *eth.Ethereum, error) {
	ethConfig := &ethconfig.Config{
		NetworkId: l2ChainID.Uint64(),
		Genesis:   genesis,
	}
	nodeConfig := &node.Config{
		Name:        fmt.Sprintf("l2-geth-%v", name),
		WSHost:      "127.0.0.1",
		WSPort:      0,
		AuthAddr:    "127.0.0.1",
		AuthPort:    0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
		JWTSecret:   jwtPath,
	}
	return createGethNode(true, nodeConfig, ethConfig, nil)
}

// createGethNode creates an in-memory geth node based on the configuration.
// The private keys are added to the keystore and are unlocked.
// If the node is l2, catalyst is enabled.
// The node should be started and then closed when done.
func createGethNode(l2 bool, nodeCfg *node.Config, ethCfg *ethconfig.Config, privateKeys []*ecdsa.PrivateKey) (*node.Node, *eth.Ethereum, error) {
	ethCfg.NoPruning = true // force everything to be an archive node
	n, err := node.New(nodeCfg)
	if err != nil {
		n.Close()
		return nil, nil, err
	}

	keydir := n.KeyStoreDir()
	scryptN := 2
	scryptP := 1
	n.AccountManager().AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	ks := n.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	password := "foobar"
	for _, pk := range privateKeys {
		act, err := ks.ImportECDSA(pk, password)
		if err != nil {
			n.Close()
			return nil, nil, err
		}
		err = ks.Unlock(act, password)
		if err != nil {
			n.Close()
			return nil, nil, err
		}
	}

	backend, err := eth.New(n, ethCfg)
	if err != nil {
		n.Close()
		return nil, nil, err

	}

	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	// Enable catalyst if l2
	if l2 {
		if err := catalyst.Register(n, backend); err != nil {
			n.Close()
			return nil, nil, err
		}
	}
	return n, backend, nil

}
