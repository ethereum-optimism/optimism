package helpers

import (
	"fmt"
	"math/big"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// L2Engine is an implementation of the Engine API backed by an out-of-process
// op-reth-test-engine subprocess, driven over a Unix socket, without support for snap-sync, and
// no concurrency or background processes.
type L2Engine struct {
	log log.Logger

	reth *rethBackend

	l2Signer types.Signer

	FailL2RPC func(call []rpc.BatchElem) error // mock error
}

// NewL2Engine spawns a fresh op-reth-test-engine subprocess over the given genesis and returns the
// L2Engine handle driving it. The engine is ephemeral: it keeps no state across processes.
func NewL2Engine(t Testing, log log.Logger, genesis *core.Genesis) *L2Engine {
	return newRethL2Engine(t, log, genesis)
}

// Enode returns a handle that identifies this engine to a peer's AddPeers. The engines have no
// devp2p, so this is a synthetic record carrying only an id the peer registry resolves back to
// this engine.
func (s *L2Engine) Enode() *enode.Node {
	return s.reth.node()
}

// AddPeers peers this engine with the given nodes. It resolves each node to the engine it
// identifies and starts an in-process sync pump that stands in for devp2p EL sync (see
// l2_engine_reth_sync.go).
func (s *L2Engine) AddPeers(peers ...*enode.Node) {
	for _, en := range peers {
		peer := lookupRethPeer(en.ID())
		if peer == nil {
			panic(fmt.Sprintf("peer enode %s is not a reth engine in this process", en.ID()))
		}
		s.reth.addPeer(peer)
	}
}

func (s *L2Engine) PeerCount() int {
	return s.reth.peerCount()
}

func (s *L2Engine) HTTPEndpoint() string {
	panic("reth backend: HTTP endpoint (kona-host) not yet served")
}

func (s *L2Engine) EthClient() *ethclient.Client {
	return ethclient.NewClient(s.reth.client)
}

func (s *L2Engine) GethClient() *gethclient.Client {
	return gethclient.New(s.reth.client)
}

func (e *L2Engine) RPCClient() client.RPC {
	// Wrap the engine RPC so a forkchoice update that reports SYNCING reproduces the
	// "Forkchoice requested sync to new head" log the EL-sync tests assert on.
	var base client.RPC = elSyncLogRPC{RPC: client.NewBaseRPCClient(e.reth.client), b: e.reth}
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

// LatestHeader returns the current unsafe (canonical head) L2 header, read over the eth RPC.
func (e *L2Engine) LatestHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.LatestBlockNumber)
}

// SafeHeader returns the current safe L2 header.
func (e *L2Engine) SafeHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.SafeBlockNumber)
}

// FinalizedHeader returns the current finalized L2 header.
func (e *L2Engine) FinalizedHeader(t Testing) *types.Header {
	return e.headerByLabel(t, rpc.FinalizedBlockNumber)
}

func (e *L2Engine) headerByLabel(t Testing, label rpc.BlockNumber) *types.Header {
	h, err := e.EthClient().HeaderByNumber(t.Ctx(), big.NewInt(int64(label)))
	require.NoError(t, err, "header at %s", label)
	return h
}

// BlockByNumber returns the canonical L2 block at height n over the eth RPC.
func (e *L2Engine) BlockByNumber(t Testing, n uint64) *types.Block {
	b, err := e.EthClient().BlockByNumber(t.Ctx(), new(big.Int).SetUint64(n))
	require.NoError(t, err, "block %d", n)
	return b
}

// GenesisBlock returns the L2 genesis block.
func (e *L2Engine) GenesisBlock(t Testing) *types.Block {
	return e.BlockByNumber(t, 0)
}

// RemainingBlockGas returns the gas still available in the block currently being built
// (optest_remainingBlockGas).
func (e *L2Engine) RemainingBlockGas(t Testing) uint64 {
	return e.reth.remainingBlockGas(t)
}

// ForcedEmpty reports whether the block currently being built is forced to stay empty
// (optest_forcedEmpty).
func (e *L2Engine) ForcedEmpty(t Testing) bool {
	return e.reth.forcedEmpty(t)
}

// IncludeTxErr submits tx directly into the block currently being built and returns the resulting
// error (nil if it was included). Unlike ActL2IncludeTx it consults neither the parking buffer nor
// the force-empty flag and does not map errors to InvalidAction; tests use it to assert that a
// specific transaction is rejected at inclusion time.
func (e *L2Engine) IncludeTxErr(t Testing, tx *types.Transaction, from common.Address) error {
	return e.reth.includeTxErr(t, tx)
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

// ActL2IncludeTxIgnoreForcedEmpty includes the next transaction from the given address in the
// block that is being built, skipping the usual force-empty check.
func (e *L2Engine) ActL2IncludeTxIgnoreForcedEmpty(from common.Address) Action {
	return func(t Testing) {
		prev := e.reth.forcedEmpty(t)
		e.reth.setForceEmpty(t, false) // ensure the engine can include it
		e.reth.includeNextTx(t, from)
		e.reth.setForceEmpty(t, prev)
	}
}

// ActL2IncludeTx includes the next transaction from the given address in the block that is being built
func (e *L2Engine) ActL2IncludeTx(from common.Address) Action {
	return func(t Testing) {
		if e.reth.forcedEmpty(t) {
			e.log.Info("Skipping including a transaction because forced-empty is set")
			return
		}
		e.reth.includeNextTx(t, from)
	}
}

func (e *L2Engine) Close() error {
	e.reth.shutdown()
	return nil
}
