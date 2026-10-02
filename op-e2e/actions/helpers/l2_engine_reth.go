package helpers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// rethBackend is the out-of-process op-reth-test-engine backing an L2Engine. All engine/eth/optest
// RPC goes over the socket of the embedded client, which owns the subprocess.
type rethBackend struct {
	*testengine.Client
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

	engine, err := testengine.Spawn(binPath, genesisPath, &engineLogWriter{log: logger})
	require.NoError(t, err, "spawn op-reth-test-engine")

	reth := &rethBackend{Client: engine}
	t.Cleanup(reth.Close)

	return &L2Engine{
		log:  logger,
		reth: reth,
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
	gas, err := b.RemainingBlockGas(t.Ctx())
	require.NoError(t, err)
	return gas
}

// forcedEmpty reports whether the in-flight block is force-empty (optest_forcedEmpty).
func (b *rethBackend) forcedEmpty(t Testing) bool {
	forced, err := b.ForcedEmpty(t.Ctx())
	require.NoError(t, err)
	return forced
}

// includeTxErr submits a raw transaction directly to optest_includeTx and returns the engine's
// error verbatim (nil on success). The engine rejects an unsupported transaction type (e.g. a blob
// tx) while decoding it.
func (b *rethBackend) includeTxErr(t Testing, tx *types.Transaction) error {
	raw, err := tx.MarshalBinary()
	require.NoError(t, err, "marshal tx")
	_, err = b.IncludeTx(t.Ctx(), raw)
	return err
}

// includeNextTx drains the next parked transaction from `from` into the block being built, mapping
// the engine's not-building and out-of-gas errors to t.InvalidAction.
func (b *rethBackend) includeNextTx(t Testing, from common.Address) {
	res, err := b.IncludeNextTx(t.Ctx(), from)
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
	// Skipped means force-empty dropped the tx; the caller already guards on forcedEmpty, so a skip
	// here is a no-op.
}
