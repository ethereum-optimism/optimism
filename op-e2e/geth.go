package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
)

var (
	// errTimeout represents a timeout
	errTimeout = errors.New("timeout")
)

func waitForL1OriginOnL2(l1BlockNum uint64, client *ethclient.Client, timeout time.Duration) (*types.Block, error) {
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
			block, err := client.BlockByNumber(ctx, head.Number)
			if err != nil {
				return nil, err
			}
			l1Info, err := derive.L1InfoDepositTxData(block.Transactions()[0].Data())
			if err != nil {
				return nil, err
			}
			if l1Info.Number >= l1BlockNum {
				return block, nil
			}

		case err := <-headSub.Err():
			return nil, fmt.Errorf("error in head subscription: %w", err)
		case <-timeoutCh:
			return nil, errTimeout
		}
	}
}

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
			tip, err := client.BlockByNumber(context.Background(), nil)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("receipt for transaction %s not found. tip block number is %d: %w", hash.Hex(), tip.NumberU64(), errTimeout)
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
			return nil, fmt.Errorf("error in head subscription: %w", err)
		case <-timeoutCh:
			return nil, errTimeout
		}
	}
}

func initL1Geth(cfg *SystemConfig, genesis *core.Genesis, c clock.Clock, opts ...GethOption) (*node.Node, *eth.Ethereum, error) {
	ethConfig := &ethconfig.Config{
		NetworkId: cfg.DeployConfig.L1ChainID,
		Genesis:   genesis,
	}
	nodeConfig := &node.Config{
		Name:        "l1-geth",
		HTTPHost:    "127.0.0.1",
		HTTPPort:    0,
		WSHost:      "127.0.0.1",
		WSPort:      0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
	}

	l1Node, l1Eth, err := createGethNode(false, nodeConfig, ethConfig, []*ecdsa.PrivateKey{cfg.Secrets.CliqueSigner}, opts...)
	if err != nil {
		return nil, nil, err
	}
	// Activate merge
	l1Eth.Merger().FinalizePoS()

	// Instead of running a whole beacon node, we run this fake-proof-of-stake sidecar that sequences L1 blocks using the Engine API.
	l1Node.RegisterLifecycle(&fakePoS{
		clock:     c,
		eth:       l1Eth,
		log:       log.Root(), // geth logger is global anyway. Would be nice to replace with a local logger though.
		blockTime: cfg.DeployConfig.L1BlockTime,
		// for testing purposes we make it really fast, otherwise we don't see it finalize in short tests
		finalizedDistance: 8,
		safeDistance:      4,
		engineAPI:         catalyst.NewConsensusAPI(l1Eth),
	})

	return l1Node, l1Eth, nil
}

func defaultNodeConfig(name string, jwtPath string) *node.Config {
	return &node.Config{
		Name:        name,
		WSHost:      "127.0.0.1",
		WSPort:      0,
		AuthAddr:    "127.0.0.1",
		AuthPort:    0,
		HTTPHost:    "127.0.0.1",
		HTTPPort:    0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine"},
		JWTSecret:   jwtPath,
	}
}

type GethOption func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error

// init a geth node.
func initL2Geth(name string, l2ChainID *big.Int, genesis *core.Genesis, jwtPath string, opts ...GethOption) (*node.Node, *eth.Ethereum, error) {
	ethConfig := &ethconfig.Config{
		NetworkId: l2ChainID.Uint64(),
		Genesis:   genesis,
		Miner: miner.Config{
			Etherbase:         common.Address{},
			ExtraData:         nil,
			GasFloor:          0,
			GasCeil:           0,
			GasPrice:          nil,
			Recommit:          0,
			NewPayloadTimeout: 0,
		},
	}
	nodeConfig := defaultNodeConfig(fmt.Sprintf("l2-geth-%v", name), jwtPath)
	return createGethNode(true, nodeConfig, ethConfig, nil, opts...)
}

// createGethNode creates an in-memory geth node based on the configuration.
// The private keys are added to the keystore and are unlocked.
// If the node is l2, catalyst is enabled.
// The node should be started and then closed when done.
func createGethNode(l2 bool, nodeCfg *node.Config, ethCfg *ethconfig.Config, privateKeys []*ecdsa.PrivateKey, opts ...GethOption) (*node.Node, *eth.Ethereum, error) {
	for i, opt := range opts {
		if err := opt(ethCfg, nodeCfg); err != nil {
			return nil, nil, fmt.Errorf("failed to apply geth option %d: %w", i, err)
		}
	}
	ethCfg.NoPruning = true // force everything to be an archive node
	n, err := node.New(nodeCfg)
	if err != nil {
		n.Close()
		return nil, nil, err
	}

	if !l2 {
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
	}

	backend, err := eth.New(n, ethCfg)
	if err != nil {
		n.Close()
		return nil, nil, err

	}

	// PR 25459 changed this to only default in CLI, but not in default programmatic RPC selection.
	// PR 25642 fixed it for the mobile version only...
	utils.RegisterFilterAPI(n, backend.APIBackend, ethCfg)

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
