package helpers

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// L2Engine is an implementation of the Engine API backed by an out-of-process
// op-reth-test-engine subprocess, driven over a Unix socket, without support for snap-sync. All
// engine/eth/optest RPC goes over the socket of its client, which owns the subprocess.
type L2Engine struct {
	log    log.Logger
	client *testengine.Client

	FailL2RPC func(call []rpc.BatchElem) error // mock error
}

// resolveEngineBinary locates the op-reth-test-engine binary once per process (see
// testengine.ResolveBinary for the environment it reads).
var resolveEngineBinary = sync.OnceValues(func() (string, error) {
	return testengine.ResolveBinary(context.Background())
})

// NewL2Engine spawns a fresh op-reth-test-engine subprocess over the given genesis and returns the
// L2Engine handle driving it. The engine is ephemeral: it keeps no state across processes. Genesis
// is marshalled to a temp file in exactly the op-geth core.Genesis JSON the binary parses
// (OpChainSpec::from_genesis).
func NewL2Engine(t Testing, logger log.Logger, genesis *core.Genesis) *L2Engine {
	binPath, err := resolveEngineBinary()
	require.NoError(t, err, "resolve op-reth-test-engine binary")

	data, err := json.Marshal(genesis)
	require.NoError(t, err, "marshal L2 genesis")
	genesisPath := filepath.Join(t.TempDir(), "l2-genesis.json")
	require.NoError(t, os.WriteFile(genesisPath, data, 0o644))

	engine, err := testengine.Spawn(binPath, genesisPath, &engineLogWriter{log: logger})
	require.NoError(t, err, "spawn op-reth-test-engine")
	t.Cleanup(engine.Close)

	return &L2Engine{log: logger, client: engine}
}

// engineLogWriter forwards the engine subprocess's stderr into the test logger at Info, so engine
// panics and startup errors show at the usual test log levels.
type engineLogWriter struct {
	log log.Logger
}

func (w *engineLogWriter) Write(p []byte) (int, error) {
	w.log.Info("op-reth-test-engine", "line", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// EthClient returns an eth client for the engine. Every client from EthClient, GethClient and
// RPCClient shares the engine's single connection, so closing one closes them all: release the
// engine with Close instead.
func (e *L2Engine) EthClient() *ethclient.Client {
	return ethclient.NewClient(e.client.RPC())
}

// GethClient returns a geth-extensions client for the engine; see EthClient on closing it.
func (e *L2Engine) GethClient() *gethclient.Client {
	return gethclient.New(e.client.RPC())
}

// RPCClient returns a raw RPC client for the engine, failing calls while FailL2RPC is set; see
// EthClient on closing it.
func (e *L2Engine) RPCClient() client.RPC {
	return testutils.RPCErrFaker{
		RPC: client.NewBaseRPCClient(e.client.RPC()),
		ErrFn: func(call []rpc.BatchElem) error {
			if e.FailL2RPC == nil {
				return nil
			}
			return e.FailL2RPC(call)
		},
	}
}

// RPCEndpoint returns the engine's IPC endpoint, the path of its Unix socket, for processes that
// read the L2 chain from the engine themselves, such as kona-host.
func (e *L2Engine) RPCEndpoint() string {
	return e.client.SocketPath()
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

// HeaderByHash returns the L2 header with the given hash over the eth RPC.
func (e *L2Engine) HeaderByHash(t Testing, hash common.Hash) *types.Header {
	h, err := e.EthClient().HeaderByHash(t.Ctx(), hash)
	require.NoError(t, err, "header %s", hash)
	return h
}

// DumpAccounts returns the state of the given accounts at the L2 block with the given hash
// (optest_dumpAccounts). Accounts that do not exist at that block are absent from the result.
func (e *L2Engine) DumpAccounts(t Testing, blockHash common.Hash, addrs []common.Address) types.GenesisAlloc {
	alloc, err := e.client.DumpAccounts(t.Ctx(), blockHash, addrs)
	require.NoError(t, err, "dump accounts at %s", blockHash)
	return alloc
}

// GenesisBlock returns the L2 genesis block.
func (e *L2Engine) GenesisBlock(t Testing) *types.Block {
	return e.BlockByNumber(t, 0)
}

// RemainingBlockGas returns the gas still available in the block currently being built
// (optest_remainingBlockGas).
func (e *L2Engine) RemainingBlockGas(t Testing) uint64 {
	gas, err := e.client.RemainingBlockGas(t.Ctx())
	require.NoError(t, err)
	return gas
}

// ForcedEmpty reports whether the block currently being built is forced to stay empty
// (optest_forcedEmpty).
func (e *L2Engine) ForcedEmpty(t Testing) bool {
	forced, err := e.client.ForcedEmpty(t.Ctx())
	require.NoError(t, err)
	return forced
}

// IncludeTxErr submits tx directly into the block currently being built (optest_includeTx) and
// returns the engine's error verbatim (nil if it was included). Unlike ActL2IncludeTx it consults
// neither the parking buffer nor the force-empty flag and does not map errors to InvalidAction;
// tests use it to assert that a specific transaction is rejected at inclusion time. The engine
// rejects an unsupported transaction type (e.g. a blob tx) while decoding it.
func (e *L2Engine) IncludeTxErr(t Testing, tx *types.Transaction, from common.Address) error {
	raw, err := tx.MarshalBinary()
	require.NoError(t, err, "marshal tx")
	_, err = e.client.IncludeTx(t.Ctx(), raw)
	return err
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

// ActL2IncludeTx includes the next transaction from the given address in the block that is being
// built, unless the block is forced to stay empty.
func (e *L2Engine) ActL2IncludeTx(from common.Address) Action {
	return func(t Testing) {
		if e.ForcedEmpty(t) {
			e.log.Info("Skipping including a transaction because forced-empty is set")
			return
		}
		e.includeNextTx(t, from)
	}
}

// ActL2IncludeTxIgnoreForcedEmpty includes the next transaction from the given address in the
// block that is being built, even if the block is forced to stay empty. The force-empty flag is
// restored afterwards.
func (e *L2Engine) ActL2IncludeTxIgnoreForcedEmpty(from common.Address) Action {
	return func(t Testing) {
		if e.ForcedEmpty(t) {
			e.log.Info("Ignoring forced-empty to include a transaction")
			require.NoError(t, e.client.SetForceEmpty(t.Ctx(), false))
			defer func() { require.NoError(t, e.client.SetForceEmpty(t.Ctx(), true)) }()
		}
		e.includeNextTx(t, from)
	}
}

// includeNextTx drains the next parked transaction from `from` into the block being built, mapping
// the engine's not-building and out-of-gas errors to t.InvalidAction.
func (e *L2Engine) includeNextTx(t Testing, from common.Address) {
	res, err := e.client.IncludeNextTx(t.Ctx(), from)
	if err != nil {
		msg := err.Error()
		// Over RPC the engine's errors arrive as messages, so match the text its Error enum formats.
		switch {
		case strings.Contains(msg, "not currently building a block"):
			t.InvalidAction("%s", msg)
		case strings.Contains(msg, "action takes too much gas"):
			t.InvalidAction("included tx uses too much gas: %v", err)
		default:
			require.NoError(t, err, "include next tx")
		}
		return
	}
	if res.NoTx {
		require.Fail(t, "no pending tx", "no pending tx from %s to include", from)
		return
	}
	// Skipped means force-empty dropped the tx; the caller already guards on ForcedEmpty, so a skip
	// here is a no-op.
}

// Close stops the engine process.
func (e *L2Engine) Close() error {
	e.client.Close()
	return nil
}
