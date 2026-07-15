package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/engineipc"
)

// ELSelectorEnv chooses which execution layer backs an L2Engine in the action tests. It exists for
// the op-geth-decoupling switch: the in-process op-geth EL (the historical default) is being
// replaced by the out-of-process op-reth-test-engine binary, driven over a Unix socket.
const ELSelectorEnv = "OP_E2E_ACTIONS_EL"

const (
	elGeth           = "geth"
	elRethTestEngine = "reth-test-engine"
)

// rethBackendSelected reports whether OP_E2E_ACTIONS_EL selects the out-of-process reth engine.
//
// The default is still geth: the switch lands incrementally, so until every L2Chain()/EngineApi
// site is migrated the suite runs unchanged on op-geth, and individual tests opt into the reth
// backend. An unrecognized value fails loudly rather than silently falling back.
func rethBackendSelected() bool {
	switch v := os.Getenv(ELSelectorEnv); v {
	case "", elGeth:
		return false
	case elRethTestEngine:
		return true
	default:
		panic(fmt.Sprintf("unknown %s=%q (want %q or %q)", ELSelectorEnv, v, elGeth, elRethTestEngine))
	}
}

// RethBackendSelected reports whether the out-of-process reth engine backs the L2 engine. Tests use
// it to gate behavior that is specific to the in-process op-geth engine (e.g. persistent-DataDir
// restart-and-recover) and cannot be reproduced against the ephemeral reth subprocess.
func RethBackendSelected() bool {
	return rethBackendSelected()
}

// rethBackend is the out-of-process op-reth-test-engine backing an L2Engine: the spawned subprocess
// and its dialed IPC client. All engine/eth/optest RPC goes over the socket.
type rethBackend struct {
	proc   *engineipc.Proc
	client *rpc.Client
}

var (
	engineBinOnce sync.Once
	engineBinPath string
	engineBinErr  error
)

// resolveEngineBinary locates the op-reth-test-engine binary once per process. It honours a
// prebuilt-binary path from the environment (as CI supplies) and otherwise builds it with cargo.
// It never falls back to a skip: a missing binary or a failed build is a hard error, so the switch
// gate cannot silently pass with no engine.
func resolveEngineBinary() (string, error) {
	engineBinOnce.Do(func() {
		if override := os.Getenv("RUST_BINARY_PATH_OP_RETH_TEST_ENGINE"); override != "" {
			if _, err := os.Stat(override); err != nil {
				engineBinErr = fmt.Errorf("RUST_BINARY_PATH_OP_RETH_TEST_ENGINE=%s: %w", override, err)
				return
			}
			engineBinPath = override
			return
		}
		// REQUIRE_RUST_ENGINE arms CI: with it set, a missing prebuilt-binary path is a hard error
		// rather than a slow cargo fallback, so a misconfigured job (binary not built/persisted, path
		// unexported) fails loudly instead of masking the config bug behind a 20-minute rebuild.
		if os.Getenv("REQUIRE_RUST_ENGINE") != "" {
			engineBinErr = fmt.Errorf("REQUIRE_RUST_ENGINE is set but RUST_BINARY_PATH_OP_RETH_TEST_ENGINE is empty: refusing to fall back to a cargo build")
			return
		}
		cwd, err := os.Getwd()
		if err != nil {
			engineBinErr = err
			return
		}
		root, err := opservice.FindMonorepoRoot(cwd)
		if err != nil {
			engineBinErr = err
			return
		}
		rustDir := filepath.Join(root, "rust")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "cargo", "build", "-p", "op-reth-test-engine", "--bin", "op-reth-test-engine")
		cmd.Dir = rustDir
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			engineBinErr = fmt.Errorf("cargo build op-reth-test-engine (in %s): %w", rustDir, err)
			return
		}
		engineBinPath = filepath.Join(rustDir, "target", "debug", "op-reth-test-engine")
	})
	return engineBinPath, engineBinErr
}

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

	proc, err := engineipc.Spawn(binPath, []string{"--genesis", genesisPath}, &engineLogWriter{log: logger}, nil)
	require.NoError(t, err, "spawn op-reth-test-engine")
	t.Cleanup(proc.Close)

	return &L2Engine{
		log:      logger,
		reth:     &rethBackend{proc: proc, client: proc.Client()},
		l2Signer: types.LatestSigner(genesis.Config),
	}
}

// engineLogWriter forwards the engine subprocess's stderr into the test logger.
type engineLogWriter struct {
	log log.Logger
}

func (w *engineLogWriter) Write(p []byte) (int, error) {
	w.log.Debug("op-reth-test-engine", "line", strings.TrimRight(string(p), "\n"))
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
