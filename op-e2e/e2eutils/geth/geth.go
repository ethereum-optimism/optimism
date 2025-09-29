package geth

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"

	"github.com/ethereum-optimism/optimism/op-service/clock"
)

func InitL1(blockTime uint64, finalizedDistance uint64, genesis *core.Genesis, c clock.Clock, blobPoolDir string, beaconSrv Beacon, opts ...GethOption) (*GethInstance, *FakePoS, error) {
	ethConfig := &ethconfig.Config{
		NetworkId: genesis.Config.ChainID.Uint64(),
		Genesis:   genesis,
		BlobPool: blobpool.Config{
			Datadir:   blobPoolDir,
			Datacap:   blobpool.DefaultConfig.Datacap,
			PriceBump: blobpool.DefaultConfig.PriceBump,
		},
		StateScheme: rawdb.HashScheme,
		Miner: miner.Config{
			PendingFeeRecipient: common.Address{},
			ExtraData:           nil,
			GasCeil:             0,
			GasPrice:            nil,
			// enough to build blocks within 1 second, but high enough to avoid unnecessary test CPU cycles.
			Recommit: time.Millisecond * 400,
		},
	}
	nodeConfig := &node.Config{
		Name:        "l1-geth",
		HTTPHost:    "127.0.0.1",
		HTTPPort:    0,
		WSHost:      "127.0.0.1",
		WSPort:      0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine", "miner"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine", "miner"},
	}

	gethInstance, err := createGethNode(false, nodeConfig, ethConfig, opts...)
	if err != nil {
		return nil, nil, err
	}

	fakepos := NewFakePoS(&gethBackend{
		chain: gethInstance.Backend.BlockChain(),
	}, catalyst.NewConsensusAPI(gethInstance.Backend), c, log.Root(), blockTime, finalizedDistance, beaconSrv, gethInstance.Backend.BlockChain().Config())

	// Instead of running a whole beacon node, we run this fake-proof-of-stake sidecar that sequences L1 blocks using the Engine API.
	gethInstance.Node.RegisterLifecycle(fakepos)

	return gethInstance, fakepos, nil
}

func WithAuth(jwtPath string) GethOption {
	return func(_ *ethconfig.Config, nodeCfg *node.Config) error {
		nodeCfg.AuthAddr = "127.0.0.1"
		nodeCfg.AuthPort = 0
		nodeCfg.JWTSecret = jwtPath
		return nil
	}
}

type gethBackend struct {
	chain *core.BlockChain
}

func (b *gethBackend) HeaderByNumber(_ context.Context, num *big.Int) (*types.Header, error) {
	if num == nil {
		return b.chain.CurrentBlock(), nil
	}
	var h *types.Header
	if num.IsInt64() && num.Int64() < 0 {
		switch num.Int64() {
		case int64(rpc.LatestBlockNumber):
			h = b.chain.CurrentBlock()
		case int64(rpc.SafeBlockNumber):
			h = b.chain.CurrentSafeBlock()
		case int64(rpc.FinalizedBlockNumber):
			h = b.chain.CurrentFinalBlock()
		}
	} else {
		h = b.chain.GetHeaderByNumber(num.Uint64())
	}
	if h == nil {
		return nil, ethereum.NotFound
	}
	return h, nil
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
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine", "miner"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal", "engine", "miner"},
		JWTSecret:   jwtPath,
	}
}

type GethOption func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error

// InitL2 inits a L2 geth node.
func InitL2(name string, genesis *core.Genesis, jwtPath string, opts ...GethOption) (*GethInstance, error) {
	ethConfig := &ethconfig.Config{
		NetworkId:   genesis.Config.ChainID.Uint64(),
		Genesis:     genesis,
		StateScheme: rawdb.HashScheme,
		Miner: miner.Config{
			PendingFeeRecipient: common.Address{},
			ExtraData:           nil,
			GasCeil:             0,
			GasPrice:            nil,
			// enough to build blocks within 1 second, but high enough to avoid unnecessary test CPU cycles.
			Recommit: time.Millisecond * 400,
		},
		TxPool: legacypool.Config{
			NoLocals: true,
		},
	}
	nodeConfig := defaultNodeConfig(fmt.Sprintf("l2-geth-%v", name), jwtPath)
	return createGethNode(true, nodeConfig, ethConfig, opts...)
}

// createGethNode creates an in-memory geth node based on the configuration.
// The private keys are added to the keystore and are unlocked.
// Catalyst is always enabled. If the node is an L1, the catalyst API can be used by alternative
// sequencers (e.g., op-test-sequencer) if the default FakePoS is stopped.
// The node should be started and then closed when done.
func createGethNode(l2 bool, nodeCfg *node.Config, ethCfg *ethconfig.Config, opts ...GethOption) (*GethInstance, error) {
	for i, opt := range opts {
		if err := opt(ethCfg, nodeCfg); err != nil {
			return nil, fmt.Errorf("failed to apply geth option %d: %w", i, err)
		}
	}
	ethCfg.StateScheme = rawdb.HashScheme
	ethCfg.NoPruning = true // force everything to be an archive node
	n, err := node.New(nodeCfg)
	if err != nil {
		n.Close()
		return nil, err
	}

	backend, err := eth.New(n, ethCfg)
	if err != nil {
		n.Close()
		return nil, err

	}

	// PR 25459 changed this to only default in CLI, but not in default programmatic RPC selection.
	// PR 25642 fixed it for the mobile version only...
	utils.RegisterFilterAPI(n, backend.APIBackend, ethCfg)

	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	if err := catalyst.Register(n, backend); err != nil {
		n.Close()
		return nil, err
	}
	return &GethInstance{
		Backend: backend,
		Node:    n,
	}, nil
}
