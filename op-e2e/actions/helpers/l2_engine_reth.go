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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// ELSelectorEnv chooses which execution layer backs an L2Engine in the action tests: ELGeth (the
// default) for the in-process op-geth EL, or ELRethTestEngine for the out-of-process
// op-reth-test-engine binary, driven over a Unix socket.
const ELSelectorEnv = "OP_E2E_ACTIONS_EL"

// Values of ELSelectorEnv.
const (
	ELGeth           = "geth"
	ELRethTestEngine = "reth-test-engine"
)

// RethBackendSelected reports whether OP_E2E_ACTIONS_EL selects the out-of-process reth engine.
// Tests use it to gate behavior specific to the in-process op-geth engine before creating one.
//
// The default is geth; an unrecognized value fails loudly rather than silently falling back.
func RethBackendSelected() bool {
	switch v := os.Getenv(ELSelectorEnv); v {
	case "", ELGeth:
		return false
	case ELRethTestEngine:
		return true
	default:
		panic(fmt.Sprintf("unknown %s=%q (want %q or %q)", ELSelectorEnv, v, ELGeth, ELRethTestEngine))
	}
}

// L2RethEngine is the out-of-process op-reth-test-engine L2 execution engine. All engine/eth/optest
// RPC goes over the socket of its client, which owns the subprocess.
type L2RethEngine struct {
	log    log.Logger
	client *testengine.Client
}

// resolveEngineBinary locates the op-reth-test-engine binary once per process (see
// testengine.ResolveBinary for the environment it reads).
var resolveEngineBinary = sync.OnceValues(func() (string, error) {
	return testengine.ResolveBinary(context.Background())
})

// NewL2RethEngine spawns the op-reth-test-engine subprocess over the given genesis. Genesis is
// marshalled to a temp file in exactly the op-geth core.Genesis JSON the binary parses
// (OpChainSpec::from_genesis).
func NewL2RethEngine(t Testing, logger log.Logger, genesis *core.Genesis) *L2RethEngine {
	binPath, err := resolveEngineBinary()
	require.NoError(t, err, "resolve op-reth-test-engine binary")

	data, err := json.Marshal(genesis)
	require.NoError(t, err, "marshal L2 genesis")
	genesisPath := filepath.Join(t.TempDir(), "l2-genesis.json")
	require.NoError(t, os.WriteFile(genesisPath, data, 0o644))

	engine, err := testengine.Spawn(binPath, genesisPath, &engineLogWriter{log: logger})
	require.NoError(t, err, "spawn op-reth-test-engine")
	t.Cleanup(engine.Close)

	return &L2RethEngine{log: logger, client: engine}
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
func (b *L2RethEngine) EthClient() *ethclient.Client {
	return ethclient.NewClient(b.client.RPC())
}

// GethClient returns a geth-extensions client for the engine; see EthClient on closing it.
func (b *L2RethEngine) GethClient() *gethclient.Client {
	return gethclient.New(b.client.RPC())
}

// RPCClient returns a raw RPC client for the engine; see EthClient on closing it.
func (b *L2RethEngine) RPCClient() client.RPC {
	return client.NewBaseRPCClient(b.client.RPC())
}

// RemainingBlockGas returns the gas still available in the in-flight block (optest_remainingBlockGas).
func (b *L2RethEngine) RemainingBlockGas(t Testing) uint64 {
	gas, err := b.client.RemainingBlockGas(t.Ctx())
	require.NoError(t, err)
	return gas
}

// ForcedEmpty reports whether the in-flight block is force-empty (optest_forcedEmpty).
func (b *L2RethEngine) ForcedEmpty(t Testing) bool {
	forced, err := b.client.ForcedEmpty(t.Ctx())
	require.NoError(t, err)
	return forced
}

// setForceEmpty sets the in-flight block's force-empty flag (optest_setForceEmpty).
func (b *L2RethEngine) setForceEmpty(t Testing, v bool) {
	require.NoError(t, b.client.SetForceEmpty(t.Ctx(), v))
}

// restoreForceEmpty is setForceEmpty for a deferred call: it reports a failure without ending the
// action, which may already be ending.
func (b *L2RethEngine) restoreForceEmpty(t Testing, v bool) {
	if err := b.client.SetForceEmpty(t.Ctx(), v); err != nil {
		t.Errorf("restore force-empty flag: %v", err)
	}
}

// IncludeTxErr submits a raw transaction directly to optest_includeTx and returns the engine's
// error verbatim (nil on success). The reth engine rejects an unsupported transaction (e.g. a blob
// tx) while decoding it, so the message differs from op-geth's block-build rejection.
func (b *L2RethEngine) IncludeTxErr(t Testing, tx *types.Transaction, _ common.Address) error {
	raw, err := tx.MarshalBinary()
	require.NoError(t, err, "marshal tx")
	_, err = b.client.IncludeTx(t.Ctx(), raw)
	return err
}

// ActL2IncludeTxIgnoreForcedEmpty includes the next transaction from the given address in the block
// that is being built, ignoring and then restoring the block's force-empty flag.
func (b *L2RethEngine) ActL2IncludeTxIgnoreForcedEmpty(from common.Address) Action {
	return func(t Testing) {
		prev := b.ForcedEmpty(t)
		b.setForceEmpty(t, false) // ensure the engine can include it
		// Restore even when a failed include ends the action.
		defer b.restoreForceEmpty(t, prev)
		b.includeNextTx(t, from)
	}
}

// ActL2IncludeTx includes the next transaction from the given address in the block that is being
// built, unless the block is forced to stay empty.
func (b *L2RethEngine) ActL2IncludeTx(from common.Address) Action {
	return func(t Testing) {
		if b.ForcedEmpty(t) {
			b.log.Info("Skipping including a transaction because forced-empty is set")
			return
		}
		b.includeNextTx(t, from)
	}
}

// includeNextTx drains the next parked transaction from `from` into the block being built, mapping
// engine errors to the same t.InvalidAction outcomes the geth ActL2IncludeTx path produces.
func (b *L2RethEngine) includeNextTx(t Testing, from common.Address) {
	res, err := b.client.IncludeNextTx(t.Ctx(), from)
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

// Close stops the engine process.
func (b *L2RethEngine) Close() error {
	b.client.Close()
	return nil
}
