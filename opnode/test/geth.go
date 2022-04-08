package test

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	rollupEth "github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

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
		Name:   "l1-geth",
		WSHost: cfg.L1WsAddr,
		WSPort: cfg.L1WsPort,
	}

	return createGethNode(false, nodeConfig, ethConfig, []*ecdsa.PrivateKey{pk})
}

func initL2Geth(name, addr string, l2ChainID *big.Int, genesis *core.Genesis) (*node.Node, *eth.Ethereum, error) {
	ethConfig := &ethconfig.Config{
		NetworkId: l2ChainID.Uint64(),
		Genesis:   genesis,
	}
	// Parsing ws://127.0.0.1:9091 for "127.0.0.1" and "9091"
	s := strings.Split(addr, ":")
	_, host, ok := strings.Cut(s[1], "//")
	if !ok {
		return nil, nil, fmt.Errorf("could not find ws host in %s", addr)
	}
	port, err := strconv.ParseInt(s[2], 10, 32)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse port from address: %w", err)
	}
	nodeConfig := &node.Config{
		Name:   fmt.Sprintf("l2-geth-%v", name),
		WSHost: host,
		WSPort: int(port),
	}
	return createGethNode(true, nodeConfig, ethConfig, nil)
}

// createGethNode creates an in-memory geth node based on the configuration.
// The private keys are added to the keystore and are unlocked.
// If the node is l2, catalyst is enabled.
// The node should be started and then closed when done.
func createGethNode(l2 bool, nodeCfg *node.Config, ethCfg *ethconfig.Config, privateKeys []*ecdsa.PrivateKey) (*node.Node, *eth.Ethereum, error) {
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
	// Enable catalyst if l2
	if l2 {
		if err := catalyst.Register(n, backend); err != nil {
			n.Close()
			return nil, nil, err
		}
	}
	return n, backend, nil

}
