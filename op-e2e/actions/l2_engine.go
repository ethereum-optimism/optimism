package actions

import (
	"errors"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
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
	l2Chain    *core.BlockChain
	l2Database ethdb.Database
	l2Cfg      *core.Genesis
	l2Signer   types.Signer

	// L2 block building data
	l2BuildingHeader *types.Header             // block header that we add txs to for block building
	l2BuildingState  *state.StateDB            // state used for block building
	l2GasPool        *core.GasPool             // track gas used of ongoing building
	pendingIndices   map[common.Address]uint64 // per account, how many txs from the pool were already included in the block, since the pool is lagging behind block mining.
	l2Transactions   []*types.Transaction      // collects txs that were successfully included into current block build
	l2Receipts       []*types.Receipt          // collect receipts of ongoing building
	l2ForceEmpty     bool                      // when no additional txs may be processed (i.e. when sequencer drift runs out)
	l2TxFailed       []*types.Transaction      // log of failed transactions which could not be included

	payloadID engine.PayloadID // ID of payload that is currently being built

	failL2RPC error // mock error
}

type EngineOption func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error

func NewL2Engine(t Testing, log log.Logger, genesis *core.Genesis, rollupGenesisL1 eth.BlockID, jwtPath string, options ...EngineOption) *L2Engine {
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
	require.NoError(t, n.Start(), "failed to start L2 op-geth node")

	return eng
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
		if e.l2BuildingHeader == nil {
			t.InvalidAction("not currently building a block, cannot include tx from queue")
			return
		}
		if e.l2ForceEmpty {
			e.log.Info("Skipping including a transaction because e.L2ForceEmpty is true")
			// t.InvalidAction("cannot include any sequencer txs")
			return
		}

		i := e.pendingIndices[from]
		txs, q := e.eth.TxPool().ContentFrom(from)
		if uint64(len(txs)) <= i {
			t.Fatalf("no pending txs from %s, and have %d unprocessable queued txs from this account", from, len(q))
		}
		tx := txs[i]
		if tx.Gas() > e.l2BuildingHeader.GasLimit {
			t.Fatalf("tx consumes %d gas, more than available in L2 block %d", tx.Gas(), e.l2BuildingHeader.GasLimit)
		}
		if tx.Gas() > uint64(*e.l2GasPool) {
			t.InvalidAction("action takes too much gas: %d, only have %d", tx.Gas(), uint64(*e.l2GasPool))
			return
		}
		e.pendingIndices[from] = i + 1 // won't retry the tx
		e.l2BuildingState.SetTxContext(tx.Hash(), len(e.l2Transactions))
		receipt, err := core.ApplyTransaction(e.l2Cfg.Config, e.l2Chain, &e.l2BuildingHeader.Coinbase,
			e.l2GasPool, e.l2BuildingState, e.l2BuildingHeader, tx, &e.l2BuildingHeader.GasUsed, *e.l2Chain.GetVMConfig())
		if err != nil {
			e.l2TxFailed = append(e.l2TxFailed, tx)
			t.Fatalf("failed to apply transaction to L2 block (tx %d): %v", len(e.l2Transactions), err)
		}
		e.l2Receipts = append(e.l2Receipts, receipt)
		e.l2Transactions = append(e.l2Transactions, tx)
	}
}

func (e *L2Engine) Close() error {
	return e.node.Close()
}
