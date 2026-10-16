package helpers

import (
	"math/big"
	"reflect"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// l2EngineBackend is an L2 execution layer behind an L2Engine: the in-process L2GethEngine or the
// out-of-process L2RethEngine.
type l2EngineBackend interface {
	EthClient() *ethclient.Client
	GethClient() *gethclient.Client
	RPCClient() client.RPC
	RemainingBlockGas(t Testing) uint64
	ForcedEmpty(t Testing) bool
	IncludeTxErr(t Testing, tx *types.Transaction, from common.Address) error
	ActL2IncludeTx(from common.Address) Action
	ActL2IncludeTxIgnoreForcedEmpty(from common.Address) Action
	Close() error
}

var (
	_ l2EngineBackend = (*L2GethEngine)(nil)
	_ l2EngineBackend = (*L2RethEngine)(nil)
)

// L2Engine is an implementation of the Engine API backed by either an in-process op-geth node or an
// out-of-process op-reth-test-engine subprocess (selected by OP_E2E_ACTIONS_EL), without support
// for snap-sync.
type L2Engine struct {
	log     log.Logger
	backend l2EngineBackend

	FailL2RPC func(call []rpc.BatchElem) error // mock error
}

func NewL2Engine(t Testing, log log.Logger, genesis *core.Genesis, jwtPath string, options ...EngineOption) *L2Engine {
	var backend l2EngineBackend
	if RethBackendSelected() {
		// The reth backend serves the engine over its own socket without a JWT.
		requireRethEngineOptions(t, options)
		backend = NewL2RethEngine(t, log, genesis)
	} else {
		backend = NewL2GethEngine(t, log, genesis, jwtPath, options...)
	}
	return &L2Engine{log: log, backend: backend}
}

// requireRethEngineOptions fails the test if any option sets op-geth config: the reth backend has
// no op-geth node or eth config to apply it to.
func requireRethEngineOptions(t Testing, options []EngineOption) {
	for i, opt := range options {
		var ethCfg ethconfig.Config
		var nodeCfg node.Config
		require.NoError(t, opt(&ethCfg, &nodeCfg), "engine option %d failed", i)
		if !reflect.DeepEqual(ethCfg, ethconfig.Config{}) || !reflect.DeepEqual(nodeCfg, node.Config{}) {
			t.Fatalf("engine option %d sets op-geth config, which the op-reth-test-engine backend cannot apply", i)
		}
	}
}

// EthClient returns an eth client for the engine. On the reth backend every client from EthClient,
// GethClient and RPCClient shares the engine's single connection, so closing one closes them all:
// release the engine with Close instead.
func (e *L2Engine) EthClient() *ethclient.Client {
	return e.backend.EthClient()
}

// GethClient returns a geth-extensions client for the engine; see EthClient on closing it.
func (e *L2Engine) GethClient() *gethclient.Client {
	return e.backend.GethClient()
}

// RPCClient returns a raw RPC client for the engine, failing calls while FailL2RPC is set; see
// EthClient on closing it.
func (e *L2Engine) RPCClient() client.RPC {
	return testutils.RPCErrFaker{
		RPC: e.backend.RPCClient(),
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

// FinalizedHeader returns the current finalized L2 header, read over the eth RPC.
func (e *L2Engine) FinalizedHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.FinalizedBlockNumber)
}

func (e *L2Engine) headerByLabel(t Testing, label rpc.BlockNumber) *types.Header {
	h, err := e.EthClient().HeaderByNumber(t.Ctx(), big.NewInt(int64(label)))
	require.NoError(t, err, "header at %s", label)
	return h
}

// BlockByNumber returns the canonical L2 block at height n, read over the eth RPC.
func (e *L2Engine) BlockByNumber(t Testing, n uint64) *types.Block {
	b, err := e.EthClient().BlockByNumber(t.Ctx(), new(big.Int).SetUint64(n))
	require.NoError(t, err, "block %d", n)
	return b
}

// GenesisBlock returns the L2 genesis block, read over the eth RPC.
func (e *L2Engine) GenesisBlock(t Testing) *types.Block {
	return e.BlockByNumber(t, 0)
}

// RemainingBlockGas returns the gas still available in the block currently being built.
func (e *L2Engine) RemainingBlockGas(t Testing) uint64 {
	return e.backend.RemainingBlockGas(t)
}

// ForcedEmpty reports whether the block currently being built is forced to stay empty.
func (e *L2Engine) ForcedEmpty(t Testing) bool {
	return e.backend.ForcedEmpty(t)
}

// IsReth reports whether this engine is backed by the out-of-process op-reth-test-engine rather
// than the in-process op-geth EL. Tests use it only where the two ELs reject an input with
// genuinely different, irreducible messages.
func (e *L2Engine) IsReth() bool {
	_, ok := e.backend.(*L2RethEngine)
	return ok
}

// IncludeTxErr submits tx directly into the block currently being built and returns the resulting
// error (nil if it was included). Unlike ActL2IncludeTx it consults neither the tx-pool nor the
// parking buffer and does not map errors to InvalidAction; tests use it to assert that a specific
// transaction is rejected at inclusion time.
func (e *L2Engine) IncludeTxErr(t Testing, tx *types.Transaction, from common.Address) error {
	return e.backend.IncludeTxErr(t, tx, from)
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

// ActL2IncludeTxIgnoreForcedEmpty includes the next transaction from the given address in the block
// that is being built, ignoring the block's force-empty flag.
func (e *L2Engine) ActL2IncludeTxIgnoreForcedEmpty(from common.Address) Action {
	return e.backend.ActL2IncludeTxIgnoreForcedEmpty(from)
}

// ActL2IncludeTx includes the next transaction from the given address in the block that is being built
func (e *L2Engine) ActL2IncludeTx(from common.Address) Action {
	return e.backend.ActL2IncludeTx(from)
}

func (e *L2Engine) Close() error {
	return e.backend.Close()
}
