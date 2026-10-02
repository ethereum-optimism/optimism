package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
	"github.com/ethereum-optimism/optimism/op-service/ipc"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// ELSelectorEnv chooses which execution layer backs an L2Engine in the action tests. It exists for
// the op-geth-decoupling switch: the in-process op-geth EL (the historical default) is being
// replaced by the out-of-process op-reth-test-engine binary, driven over a Unix socket.
const ELSelectorEnv = "OP_E2E_ACTIONS_EL"

const (
	elGeth           = "geth"
	elRethTestEngine = "reth-test-engine"
)

// RethBackendSelected reports whether OP_E2E_ACTIONS_EL selects the out-of-process reth engine.
// Tests use it to gate behavior specific to the in-process op-geth engine before creating one.
//
// The default is geth; an unrecognized value fails loudly rather than silently falling back.
func RethBackendSelected() bool {
	switch v := os.Getenv(ELSelectorEnv); v {
	case "", elGeth:
		return false
	case elRethTestEngine:
		return true
	default:
		panic(fmt.Sprintf("unknown %s=%q (want %q or %q)", ELSelectorEnv, v, elGeth, elRethTestEngine))
	}
}

// rethBackend is the out-of-process op-reth-test-engine backing an L2Engine: the spawned subprocess
// and its dialed IPC client. All engine/eth/optest RPC goes over the socket.
type rethBackend struct {
	proc   *ipc.Proc
	client *rpc.Client
	log    log.Logger

	// id identifies this engine to the in-process peer registry that stands in for devp2p. The
	// ephemeral reth engines have no networking, so peering — and the EL sync it enables — is
	// emulated here rather than run over a real p2p stack.
	id enode.ID

	mu           sync.Mutex
	peers        map[enode.ID]*rethBackend // engines this one is peered with (both directions)
	pumpCancels  []context.CancelFunc      // one per peer sync pump
	pumpWG       sync.WaitGroup
	closed       bool // set by shutdown; no pumps start afterwards
	shutdownOnce sync.Once
	// syncProblems holds, per peer, the last backfill failure logged, so a failure that persists
	// across pump passes is logged once.
	syncProblems map[enode.ID]string
	// syncedTo holds, per peer, the sync target a pass already backfilled from it, so later passes
	// skip it until the engine reports a different target.
	syncedTo map[enode.ID]common.Hash
	// syncSeen maps the hash of every payload op-node has submitted via engine_newPayload to its
	// block number, so a subsequent forkchoice update that reports SYNCING can be logged with the
	// head's number — reproducing the in-process op-geth engine API's "Forkchoice requested sync to
	// new head" line the EL-sync tests assert on.
	syncSeen map[common.Hash]uint64
}

// resolveEngineBinary locates the op-reth-test-engine binary once per process (see
// testengine.ResolveBinary for the environment it reads).
var resolveEngineBinary = sync.OnceValues(func() (string, error) {
	return testengine.ResolveBinary(context.Background())
})

// newRethL2Engine spawns the op-reth-test-engine subprocess over the given genesis and returns an
// L2Engine backed by it. Genesis is marshalled to a temp file in exactly the op-geth core.Genesis
// JSON the binary parses (OpChainSpec::from_genesis).
func newRethL2Engine(t Testing, logger log.Logger, genesis *core.Genesis) *L2Engine {
	binPath, err := resolveEngineBinary()
	require.NoError(t, err, "resolve op-reth-test-engine binary")

	data, err := json.Marshal(genesis)
	require.NoError(t, err, "marshal L2 genesis")
	genesisPath := filepath.Join(t.TempDir(), "l2-genesis.json")
	require.NoError(t, os.WriteFile(genesisPath, data, 0o644))

	proc, err := testengine.Spawn(binPath, genesisPath, &engineLogWriter{log: logger})
	require.NoError(t, err, "spawn op-reth-test-engine")

	reth := &rethBackend{
		proc:   proc,
		client: proc.Client(),
		log:    logger,
		id:     randomEnodeID(),
		peers:  make(map[enode.ID]*rethBackend),
	}
	registerRethPeer(reth)
	t.Cleanup(reth.shutdown)

	return &L2Engine{
		log:      logger,
		reth:     reth,
		l2Signer: types.LatestSigner(genesis.Config),
	}
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

// remainingBlockGas returns the gas still available in the in-flight block (optest_remainingBlockGas).
func (b *rethBackend) remainingBlockGas(t Testing) uint64 {
	var gas uint64
	require.NoError(t, b.client.CallContext(t.Ctx(), &gas, "optest_remainingBlockGas"))
	return gas
}

// forcedEmpty reports whether the in-flight block is force-empty (optest_forcedEmpty).
func (b *rethBackend) forcedEmpty(t Testing) bool {
	var forced bool
	require.NoError(t, b.client.CallContext(t.Ctx(), &forced, "optest_forcedEmpty"))
	return forced
}

// setForceEmpty sets the in-flight block's force-empty flag (optest_setForceEmpty).
func (b *rethBackend) setForceEmpty(t Testing, v bool) {
	var ok bool
	require.NoError(t, b.client.CallContext(t.Ctx(), &ok, "optest_setForceEmpty", v))
}

// restoreForceEmpty is setForceEmpty for a deferred call: it reports a failure without ending the
// action, which may already be ending.
func (b *rethBackend) restoreForceEmpty(t Testing, v bool) {
	var ok bool
	if err := b.client.CallContext(t.Ctx(), &ok, "optest_setForceEmpty", v); err != nil {
		t.Errorf("restore force-empty flag: %v", err)
	}
}

// includeNextTxResult is the optest_includeNextTx reply: exactly one of the fields is set.
type includeNextTxResult struct {
	TxHash  *common.Hash `json:"txHash"`
	GasUsed uint64       `json:"gasUsed"`
	Skipped bool         `json:"skipped"`
	NoTx    bool         `json:"noTx"`
}

// includeTxErr submits a raw transaction directly to optest_includeTx and returns the engine's
// error verbatim (nil on success). The reth engine rejects an unsupported transaction (e.g. a blob
// tx) while decoding it, so the message differs from op-geth's block-build rejection.
func (b *rethBackend) includeTxErr(t Testing, tx *types.Transaction) error {
	raw, err := tx.MarshalBinary()
	require.NoError(t, err, "marshal tx")
	var res includeNextTxResult
	return b.client.CallContext(t.Ctx(), &res, "optest_includeTx", hexutil.Bytes(raw))
}

// includeNextTx drains the next parked transaction from `from` into the block being built, mapping
// engine errors to the same t.InvalidAction outcomes the geth ActL2IncludeTx path produces.
func (b *rethBackend) includeNextTx(t Testing, from common.Address) {
	var res includeNextTxResult
	err := b.client.CallContext(t.Ctx(), &res, "optest_includeNextTx", from)
	if err != nil {
		msg := err.Error()
		// Mirror the engineapi sentinel-error mapping (over RPC we match the messages the engine's
		// Error enum formats, which are copied from the engineapi strings).
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
	// Skipped means force-empty ate the tx; the caller already guards on forcedEmpty for the normal
	// path, so a skip here is a no-op just like the geth engine returning (nil, nil).
}
