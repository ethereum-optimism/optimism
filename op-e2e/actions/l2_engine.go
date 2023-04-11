package actions

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"
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
	l2Chain  *core.BlockChain
	l2Signer types.Signer

	engineApi *engineapi.L2EngineAPI

	failL2RPC error // mock error
}

type EngineOption func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error

func NewL2Engine(t Testing, log log.Logger, genesis *core.Genesis, rollupGenesisL1 eth.BlockID, jwtPath string, options ...EngineOption) *L2Engine {
	n, ethBackend, apiBackend := newBackend(t, genesis, jwtPath, options)
	engineApi := engineapi.NewL2EngineAPI(log, apiBackend)
	chain := ethBackend.BlockChain()
	genesisBlock := chain.Genesis()
	eng := &L2Engine{
		log:  log,
		node: n,
		eth:  ethBackend,
		rollupGenesis: &rollup.Genesis{
			L1:     rollupGenesisL1,
			L2:     eth.BlockID{Hash: genesisBlock.Hash(), Number: genesisBlock.NumberU64()},
			L2Time: genesis.Timestamp,
		},
		l2Chain:   chain,
		l2Signer:  types.LatestSigner(genesis.Config),
		engineApi: engineApi,
	}
	// register the custom engine API, so we can serve engine requests while having more control
	// over sequencing of individual txs.
	n.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       eng.engineApi,
			Authenticated: true,
		},
	})
	require.NoError(t, n.Start(), "failed to start L2 op-geth node")

	return eng
}

func newBackend(t e2eutils.TestingBase, genesis *core.Genesis, jwtPath string, options []EngineOption) (*node.Node, *geth.Ethereum, *engineApiBackend) {
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
	for i, opt := range options {
		require.NoError(t, opt(ethCfg, nodeCfg), "engine option %d failed", i)
	}
	n, err := node.New(nodeCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = n.Close()
	})
	backend, err := geth.New(n, ethCfg)
	require.NoError(t, err)
	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	chain := backend.BlockChain()
	db := backend.ChainDb()
	apiBackend := &engineApiBackend{
		BlockChain: chain,
		db:         db,
		genesis:    genesis,
	}
	return n, backend, apiBackend
}

type engineApiBackend struct {
	*core.BlockChain
	db      ethdb.Database
	genesis *core.Genesis
}

func (e *engineApiBackend) Database() ethdb.Database {
	return e.db
}

func (e *engineApiBackend) Genesis() *core.Genesis {
	return e.genesis
}

func (s *L2Engine) EthClient() *ethclient.Client {
	cl, _ := s.node.Attach() // never errors
	return ethclient.NewClient(cl)
}

func (s *L2Engine) GethClient() *gethclient.Client {
	cl, _ := s.node.Attach() // never errors
	return gethclient.New(cl)
}

func (e *L2Engine) RPCClient() client.RPC {
	cl, _ := e.node.Attach() // never errors
	return testutils.RPCErrFaker{
		RPC: client.NewBaseRPCClient(cl),
		ErrFn: func() error {
			err := e.failL2RPC
			e.failL2RPC = nil // reset back, only error once.
			return err
		},
	}
}

func (e *L2Engine) EngineClient(t Testing, cfg *rollup.Config) *sources.EngineClient {
	l2Cl, err := sources.NewEngineClient(e.RPCClient(), e.log, nil, sources.EngineClientDefaultConfig(cfg))
	require.NoError(t, err)
	return l2Cl
}

// ActL2RPCFail makes the next L2 RPC request fail
func (e *L2Engine) ActL2RPCFail(t Testing) {
	if e.failL2RPC != nil { // already set to fail?
		t.InvalidAction("already set a mock L2 rpc error")
		return
	}
	e.failL2RPC = errors.New("mock L2 RPC error")
}

// ActL2IncludeTx includes the next transaction from the given address in the block that is being built
func (e *L2Engine) ActL2IncludeTx(from common.Address) Action {
	return func(t Testing) {
		if e.engineApi.ForcedEmpty() {
			e.log.Info("Skipping including a transaction because e.L2ForceEmpty is true")
			return
		}

		i := e.engineApi.PendingIndices(from)
		txs, q := e.eth.TxPool().ContentFrom(from)
		if uint64(len(txs)) <= i {
			t.Fatalf("no pending txs from %s, and have %d unprocessable queued txs from this account", from, len(q))
		}
		tx := txs[i]
		err := e.engineApi.IncludeTx(tx, from)
		if errors.Is(err, engineapi.ErrNotBuildingBlock) {
			t.InvalidAction(err.Error())
		} else if errors.Is(err, engineapi.ErrUsesTooMuchGas) {
			t.InvalidAction("included tx uses too much gas: %v", err)
		} else if err != nil {
			t.Fatalf("include tx: %v", err)
		}
	}
}

func (e *L2Engine) Close() error {
	return e.node.Close()
}
