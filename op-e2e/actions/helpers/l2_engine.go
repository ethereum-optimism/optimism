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
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// L2Engine is an implementation of the Engine API backed by either an in-process op-geth node or an
// out-of-process op-reth-test-engine subprocess (selected by OP_E2E_ACTIONS_EL), without support
// for snap-sync, and no concurrency or background processes.
type L2Engine struct {
	log log.Logger

	// geth backend — the in-process op-geth EL. Nil when the reth backend is active.
	node *node.Node
	Eth  *geth.Ethereum
	// L2 evm / chain
	l2Chain   *core.BlockChain
	EngineApi *engineapi.L2EngineAPI

	// reth backend — the out-of-process op-reth-test-engine subprocess. Nil when the geth backend
	// is active.
	reth *rethBackend

	l2Signer types.Signer

	FailL2RPC func(call []rpc.BatchElem) error // mock error
}

type EngineOption func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error

func NewL2Engine(t Testing, log log.Logger, genesis *core.Genesis, jwtPath string, options ...EngineOption) *L2Engine {
	if rethBackendSelected() {
		// The reth backend serves the engine over its own socket; the geth-only EngineOptions
		// (P2P, ethconfig tweaks) and the JWT do not apply to it.
		return newRethL2Engine(t, log, genesis)
	}
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
	if s.reth != nil {
		panic("reth backend: EL-sync p2p is emulated by the Go sync pump, not yet wired")
	}
	return s.node.Server().LocalNode().Node()
}

func (s *L2Engine) AddPeers(peers ...*enode.Node) {
	if s.reth != nil {
		panic("reth backend: EL-sync p2p is emulated by the Go sync pump, not yet wired")
	}
	for _, en := range peers {
		s.node.Server().AddPeer(en)
	}
}

func (s *L2Engine) PeerCount() int {
	if s.reth != nil {
		return 0
	}
	return s.node.Server().PeerCount()
}

func (s *L2Engine) HTTPEndpoint() string {
	if s.reth != nil {
		panic("reth backend: HTTP endpoint (kona-host) not yet served")
	}
	return s.node.HTTPEndpoint()
}

func (s *L2Engine) EthClient() *ethclient.Client {
	if s.reth != nil {
		return ethclient.NewClient(s.reth.client)
	}
	cl := s.node.Attach()
	return ethclient.NewClient(cl)
}

func (s *L2Engine) GethClient() *gethclient.Client {
	if s.reth != nil {
		return gethclient.New(s.reth.client)
	}
	cl := s.node.Attach()
	return gethclient.New(cl)
}

func (e *L2Engine) RPCClient() client.RPC {
	var base client.RPC
	if e.reth != nil {
		base = client.NewBaseRPCClient(e.reth.client)
	} else {
		base = client.NewBaseRPCClient(e.node.Attach())
	}
	return testutils.RPCErrFaker{
		RPC: base,
		ErrFn: func(call []rpc.BatchElem) error {
			if e.FailL2RPC == nil {
				return nil
			}
			return e.FailL2RPC(call)
		},
	}
}

// LatestHeader returns the current unsafe (canonical head) L2 header. It is backend-agnostic (it
// reads over the eth RPC), so it replaces the geth-only e.L2Chain().CurrentBlock()/CurrentHeader().
func (e *L2Engine) LatestHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.LatestBlockNumber)
}

// SafeHeader returns the current safe L2 header, replacing e.L2Chain().CurrentSafeBlock().
func (e *L2Engine) SafeHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.SafeBlockNumber)
}

// FinalizedHeader returns the current finalized L2 header, replacing e.L2Chain().CurrentFinalBlock().
func (e *L2Engine) FinalizedHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.FinalizedBlockNumber)
}

func (e *L2Engine) headerByLabel(t Testing, label rpc.BlockNumber) *types.Header {
	h, err := e.EthClient().HeaderByNumber(t.Ctx(), big.NewInt(int64(label)))
	require.NoError(t, err, "header at %s", label)
	return h
}

func (e *L2Engine) EngineClient(t Testing, cfg *rollup.Config) *sources.EngineClient {
	l2Cl, err := sources.NewEngineClient(e.RPCClient(), e.log, nil, sources.EngineClientDefaultConfig(cfg))
	require.NoError(t, err)
	return l2Cl
}

func (e *L2Engine) SourceClient(t Testing, cacheSize int) *sources.EthClient {
	l2sc, err := sources.NewEthClient(e.RPCClient(), e.log, nil, sources.DefaultEthClientConfig(cacheSize))
	require.NoError(t, err)
	return l2sc
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
		if e.reth != nil {
			prev := e.reth.forcedEmpty(t)
			e.reth.setForceEmpty(t, false) // ensure the engine can include it
			e.reth.includeNextTx(t, from)
			e.reth.setForceEmpty(t, prev)
			return
		}
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
		if e.reth != nil {
			if e.reth.forcedEmpty(t) {
				e.log.Info("Skipping including a transaction because forced-empty is set")
				return
			}
			e.reth.includeNextTx(t, from)
			return
		}

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
	if e.reth != nil {
		e.reth.proc.Close()
		return nil
	}
	return e.node.Close()
}
