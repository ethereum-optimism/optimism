package actions

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

// L2Engine is an in-memory implementation of the Engine API,
// without support for snap-sync, and no concurrency or background processes.
type L2Engine struct {
	log log.Logger

	node *node.Node
	eth  *geth.Ethereum

	rollupGenesis *rollup.Genesis

	// L2 evm / chain
	l2Chain    *core.BlockChain
	l2Database ethdb.Database
	l2Cfg      *core.Genesis
	l2Signer   types.Signer

	// L2 block building data
	// TODO proto - block building PR

	payloadID beacon.PayloadID // ID of payload that is currently being built

	failL2RPC error // mock error
}

func NewL2Engine(log log.Logger, genesis *core.Genesis, rollupGenesisL1 eth.BlockID, jwtPath string) *L2Engine {
	ethCfg := &ethconfig.Config{
		NetworkId: genesis.Config.ChainID.Uint64(),
		Genesis:   genesis,
	}
	nodeCfg := &node.Config{
		Name:        "l2-geth",
		WSHost:      "127.0.0.1",
		WSPort:      0,
		AuthAddr:    "127.0.0.1",
		AuthPort:    0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		JWTSecret:   jwtPath,
	}
	n, err := node.New(nodeCfg)
	if err != nil {
		panic(err)
	}
	backend, err := geth.New(n, ethCfg)
	if err != nil {
		panic(err)
	}
	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	chain := backend.BlockChain()
	db := backend.ChainDb()
	genesisBlock := chain.Genesis()
	eng := &L2Engine{
		log:  log,
		node: n,
		eth:  backend,
		rollupGenesis: &rollup.Genesis{
			L1:     rollupGenesisL1,
			L2:     eth.BlockID{Hash: genesisBlock.Hash(), Number: genesisBlock.NumberU64()},
			L2Time: genesis.Timestamp,
		},
		l2Chain:    chain,
		l2Database: db,
		l2Cfg:      genesis,
		l2Signer:   types.LatestSigner(genesis.Config),
	}
	// register the custom engine API, so we can serve engine requests while having more control
	// over sequencing of individual txs.
	n.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       (*L2EngineAPI)(eng),
			Authenticated: true,
		},
	})
	if err := n.Start(); err != nil {
		panic(fmt.Errorf("failed to start L2 op-geth node: %w", err))
	}

	return eng
}

func (e *L2Engine) RPCClient() client.RPC {
	cl, _ := e.node.Attach() // never errors
	return testutils.RPCErrFaker{
		RPC: cl,
		ErrFn: func() error {
			err := e.failL2RPC
			e.failL2RPC = nil // reset back, only error once.
			return err
		},
	}
}

// ActL2RPCFail makes the next L2 RPC request fail
func (e *L2Engine) ActL2RPCFail(t Testing) {
	if e.failL2RPC != nil { // already set to fail?
		t.InvalidAction("already set a mock L2 rpc error")
		return
	}
	e.failL2RPC = errors.New("mock L2 RPC error")
}
