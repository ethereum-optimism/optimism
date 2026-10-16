package helpers

import (
	"errors"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers/engineapi"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// L2Engine is an in-memory implementation of the Engine API,
// without support for snap-sync, and no concurrency or background processes.
type L2Engine struct {
	log log.Logger

	node *node.Node
	Eth  *geth.Ethereum

	// L2 evm / chain
	l2Chain  *core.BlockChain
	l2Signer types.Signer

	EngineApi *engineapi.L2EngineAPI

	ethCl    *ethclient.Client
	sourceCl *sources.EthClient

	FailL2RPC func(call []rpc.BatchElem) error // mock error
}

type EngineOption func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error

func NewL2Engine(t Testing, log log.Logger, genesis *core.Genesis, jwtPath string, options ...EngineOption) *L2Engine {
	n, ethBackend, apiBackend := newBackend(t, genesis, jwtPath, options)
	engineApi := engineapi.NewL2EngineAPI(log, apiBackend, ethBackend.Downloader())
	chain := ethBackend.BlockChain()
	eng := &L2Engine{
		log:       log,
		node:      n,
		Eth:       ethBackend,
		l2Chain:   chain,
		l2Signer:  types.LatestSigner(genesis.Config),
		EngineApi: engineApi,
	}
	// register the custom engine API, so we can serve engine requests while having more control
	// over sequencing of individual txs.
	n.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       eng.EngineApi,
			Authenticated: true,
		},
	})
	require.NoError(t, n.Start(), "failed to start L2 op-geth node")

	eng.ethCl = ethclient.NewClient(n.Attach())
	sourceCl, err := sources.NewEthClient(client.NewBaseRPCClient(n.Attach()), log, nil, sources.DefaultEthClientConfig(10))
	require.NoError(t, err)
	eng.sourceCl = sourceCl

	return eng
}

func newBackend(t e2eutils.TestingBase, genesis *core.Genesis, jwtPath string, options []EngineOption) (*node.Node, *geth.Ethereum, *engineApiBackend) {
	ethCfg := &ethconfig.Config{
		NetworkId:   bigs.Uint64Strict(genesis.Config.ChainID),
		Genesis:     genesis,
		StateScheme: rawdb.HashScheme,
		NoPruning:   true,
		// Record trie-key preimages when generating pre-fork state artifacts, so
		// the post-activation state can be enumerated and dumped. Off otherwise to
		// avoid the recording overhead in normal test runs.
		Preimages: os.Getenv("OP_E2E_GEN_PREFORK_STATE") != "",
	}
	nodeCfg := &node.Config{
		Name:        "l2-geth",
		WSHost:      "127.0.0.1",
		WSPort:      0,
		HTTPHost:    "127.0.0.1",
		HTTPPort:    0,
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

func (s *L2Engine) L2Chain() *core.BlockChain {
	return s.l2Chain
}

func (s *L2Engine) Enode() *enode.Node {
	return s.node.Server().LocalNode().Node()
}

func (s *L2Engine) AddPeers(peers ...*enode.Node) {
	for _, en := range peers {
		s.node.Server().AddPeer(en)
	}
}

func (s *L2Engine) PeerCount() int {
	return s.node.Server().PeerCount()
}

func (s *L2Engine) HTTPEndpoint() string {
	return s.node.HTTPEndpoint()
}

// EthClient returns the engine's eth RPC client. The engine owns it and closes it in Close.
func (s *L2Engine) EthClient() *ethclient.Client {
	return s.ethCl
}

func (s *L2Engine) GethClient() *gethclient.Client {
	cl := s.node.Attach()
	return gethclient.New(cl)
}

func (e *L2Engine) RPCClient() client.RPC {
	cl := e.node.Attach()
	return testutils.RPCErrFaker{
		RPC: client.NewBaseRPCClient(cl),
		ErrFn: func(call []rpc.BatchElem) error {
			if e.FailL2RPC == nil {
				return nil
			}
			return e.FailL2RPC(call)
		},
	}
}

// LatestHeader returns the current unsafe (canonical head) L2 header, read over the eth RPC.
func (e *L2Engine) LatestHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.LatestBlockNumber)
}

// SafeHeader returns the current safe L2 header, read over the eth RPC.
func (e *L2Engine) SafeHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.SafeBlockNumber)
}

func (e *L2Engine) headerByLabel(t Testing, label rpc.BlockNumber) *types.Header {
	h, err := e.ethCl.HeaderByNumber(t.Ctx(), big.NewInt(int64(label)))
	require.NoError(t, err, "header at %s", label)
	return h
}

// HeaderByNumber returns the canonical L2 header at height n, read over the eth RPC.
func (e *L2Engine) HeaderByNumber(t Testing, n uint64) *types.Header {
	h, err := e.ethCl.HeaderByNumber(t.Ctx(), new(big.Int).SetUint64(n))
	require.NoError(t, err, "header %d", n)
	return h
}

// HeaderByHash returns the L2 header with the given hash, read over the eth RPC.
func (e *L2Engine) HeaderByHash(t Testing, hash common.Hash) *types.Header {
	h, err := e.ethCl.HeaderByHash(t.Ctx(), hash)
	require.NoError(t, err, "header %s", hash)
	return h
}

func (e *L2Engine) EngineClient(t Testing, cfg *rollup.Config) *sources.EngineClient {
	l2Cl, err := sources.NewEngineClient(e.RPCClient(), e.log, nil, sources.EngineClientDefaultConfig(cfg))
	require.NoError(t, err)
	return l2Cl
}

// SourceClient returns the engine's OP-aware eth RPC client, for reading L2 block bodies and
// receipts. The engine owns it and closes it in Close.
func (e *L2Engine) SourceClient() *sources.EthClient {
	return e.sourceCl
}

// ActL2RPCFail makes the next L2 RPC request fail with given error
func (e *L2Engine) ActL2RPCFail(t Testing, err error) {
	if e.FailL2RPC != nil { // already set to fail?
		t.InvalidAction("already set a mock L2 rpc error")
		return
	}
	e.FailL2RPC = func(call []rpc.BatchElem) error {
		e.FailL2RPC = nil
		return err
	}
}

// ActL2IncludeTxIgnoreForcedEmpty includes the next transaction from the given address in the block that is being built,
// skipping the usual check for e.EngineApi.ForcedEmpty()
func (e *L2Engine) ActL2IncludeTxIgnoreForcedEmpty(from common.Address) Action {
	return func(t Testing) {
		if e.EngineApi.ForcedEmpty() {
			e.log.Info("Ignoring e.L2ForceEmpty=true")
		}

		require.NoError(t, e.Eth.TxPool().Sync(), "must sync tx-pool to get accurate pending txs")
		tx := firstValidTx(t, from, e.EngineApi.PendingIndices, e.Eth.TxPool().ContentFrom, e.EthClient().NonceAt)
		prevState := e.EngineApi.ForcedEmpty()
		e.EngineApi.SetForceEmpty(false) // ensure the engine API can include it
		_, err := e.EngineApi.IncludeTx(tx, from)
		e.EngineApi.SetForceEmpty(prevState)
		if errors.Is(err, engineapi.ErrNotBuildingBlock) {
			t.InvalidAction(err.Error())
		} else if errors.Is(err, engineapi.ErrUsesTooMuchGas) {
			t.InvalidAction("included tx uses too much gas: %v", err)
		} else if err != nil {
			require.NoError(t, err, "include tx")
		}
	}
}

// ActL2IncludeTx includes the next transaction from the given address in the block that is being built
func (e *L2Engine) ActL2IncludeTx(from common.Address) Action {
	return func(t Testing) {

		if e.EngineApi.ForcedEmpty() {
			e.log.Info("Skipping including a transaction because e.L2ForceEmpty is true")
			return
		}

		require.NoError(t, e.Eth.TxPool().Sync(), "must sync tx-pool to get accurate pending txs")
		tx := firstValidTx(t, from, e.EngineApi.PendingIndices, e.Eth.TxPool().ContentFrom, e.EthClient().NonceAt)
		_, err := e.EngineApi.IncludeTx(tx, from)
		if errors.Is(err, engineapi.ErrNotBuildingBlock) {
			t.InvalidAction(err.Error())
		} else if errors.Is(err, engineapi.ErrUsesTooMuchGas) {
			t.InvalidAction("included tx uses too much gas: %v", err)
		} else if err != nil {
			require.NoError(t, err, "include tx")
		}
	}
}

func (e *L2Engine) Close() error {
	e.sourceCl.Close()
	e.ethCl.Close()
	return e.node.Close()
}
