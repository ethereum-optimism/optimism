package rustengine

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// The reference fork: OP Sepolia's DisputeGameFactory on L1 Sepolia, pinned to an immutable block.
// version()=1.6.1, owner=deployer, gameImpls(999)=0 at this block (verified when the fixture was
// recorded), so SetDisputeGameImpl runs its full broadcast+assert path.
const (
	forkBlock    = uint64(11250000) // 0xabae10
	forkGameType = uint32(999)
)

var (
	forkDGF      = common.HexToAddress("0x05F9613aDB30026FFd634f38e5C4dFd30a197Fa1")
	forkDeployer = common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2") // DGF owner
)

// TestRustEngineForkedParity is the ForkBackend milestone gate (op-geth decoupling spike 3). It
// drives the smallest real forked op-deployer script (opcm.SetDisputeGameImpl, one broadcast)
// against a FIXED L1 Sepolia block through BOTH the in-process Go forked host and the out-of-process
// Rust engine's fork mode, and requires byte-identical output on three axes:
//
//	(a) version() read-through — proves the lazy RPC-backed base state,
//	(b) forking.ExportDiff vs script_forkDiff — proves the copy-on-write overlay + write-log,
//	(c) the broadcast bundle (from/to/input/value/salt/gasUsed/nonce/type) — the axis that catches
//	    absent-account gas drift.
//
// Both engines fork from the SAME recorded RPC fixture (a Go httptest server), so the base state is
// identical by construction: hermetic, network/secret/anvil-free, runnable in the required
// go-tests-short CI job with 0 skips. Regenerate the fixture with RECORD_FORK_FIXTURE=1.
//
// NOTE (intentional, documented in notes.md): the Go comparison host is built WITHOUT
// WithIsolatedBroadcasts, matching the Rust engine's (pre-existing, fork-independent) lack of
// broadcast isolation. This keeps gasUsed in the comparison a faithful measure of fork-state access
// costs while isolating the fork-state-source as the only variable. Broadcast isolation is an
// orthogonal engine gap tracked for the production-caller routing rounds, not fork mode.
func TestRustEngineForkedParity(t *testing.T) {
	bin := buildEngine(t)
	logw := testWriter{t}

	loc, artifactsFS := testutil.LocalArtifacts(t)
	artifactsDir := loc.URL.Path

	srv, dump := startForkFixtureServer(t, forkBlock)
	defer srv.Close()
	defer dump()

	input := opcm.SetDisputeGameImplInput{
		Factory:             forkDGF,
		Impl:                common.Address{'I'},
		GameType:            forkGameType,
		AnchorStateRegistry: common.Address{}, // not authorized to set the respected game type
		GameArgs:            gameargs.GameArgs{}.PackPermissionless(),
	}

	// ---- Leg A: in-process Go forked host (non-isolated, matching the non-isolated engine) ----
	var goBroadcasts []script.Broadcast
	gh := goForkedHost(t, artifactsFS, srv.URL, forkBlock, false, func(b script.Broadcast) {
		goBroadcasts = append(goBroadcasts, b)
	})
	goVersion := contractVersionAt(t, gh, forkDGF)
	require.NoError(t, opcm.RunScriptVoid[opcm.SetDisputeGameImplInput](
		gh, input, "SetDisputeGameImpl.s.sol", "SetDisputeGameImpl"), "go forked RunScriptVoid")
	goDiff, err := gh.ExportDiff()
	require.NoError(t, err)
	goDiffJSON, err := json.Marshal(goDiff)
	require.NoError(t, err)

	// ---- Leg B: out-of-process Rust engine fork mode ----
	re, err := Spawn(bin, SpawnOpts{
		ArtifactsDir:    artifactsDir,
		ChainID:         1337, // script.DefaultContext.ChainID (fork does not change chain/block env)
		Create2Deployer: true,
	}, logw)
	require.NoError(t, err)
	defer re.Close()

	meta, err := re.CreateSelectFork(srv.URL, ptrU64(forkBlock))
	require.NoError(t, err, "rust createSelectFork")
	require.Equal(t, forkBlock, meta.BlockNumber, "rust pinned block number")

	rustVersion := decodeVersion(t, mustCall(t, re, common.Address{}, forkDGF, versionSelector()))
	require.NoError(t, RunScriptVoid[opcm.SetDisputeGameImplInput](
		re, input, "SetDisputeGameImpl.s.sol", "SetDisputeGameImpl", forkDeployer), "rust forked RunScriptVoid")
	rustBroadcasts, err := re.TakeBroadcasts()
	require.NoError(t, err)
	rustDiffJSON, err := re.ForkDiff()
	require.NoError(t, err)

	// ---- Non-vacuity guards ----
	require.NotEmpty(t, goVersion, "version() read must be non-empty")
	require.GreaterOrEqual(t, len(goBroadcasts), 1, "the script must produce at least one broadcast")
	require.True(t, goDiff.Any(), "the fork overlay diff must be non-empty")

	// forking.ExportDiff is a low-level primitive that records the raw fork overlay, including script
	// scaffolding (ScriptDeployer + its CREATE range: the deploy nonce bumps and the script/precompile
	// created-then-destroyed lifecycle). The Rust engine's forkDiff excludes that scaffolding by
	// design — exactly the set script.Host.StateDump prunes (script.go:828-861). Normalize BOTH diffs
	// by pruning that same scaffolding set, so the comparison is over the REAL L1 state change (the
	// load-bearing fork writes), not the deployment machinery. See notes.md.
	goPruned := pruneForkScaffolding(t, goDiffJSON)
	rustPruned := pruneForkScaffolding(t, rustDiffJSON)
	require.Contains(t, string(goPruned), strings.ToLower(forkDGF.Hex()),
		"pruned diff must still contain the DGF (the load-bearing storage write)")

	// ---- Parity axes ----
	require.Equal(t, goVersion, rustVersion, "(a) version() read-through parity")
	require.JSONEq(t, string(goPruned), string(rustPruned), "(b) ExportDiff vs forkDiff parity")

	gb, err := json.MarshalIndent(goBroadcasts, "", "  ")
	require.NoError(t, err)
	rb, err := json.MarshalIndent(rustBroadcasts, "", "  ")
	require.NoError(t, err)
	require.JSONEq(t, string(gb), string(rb), "(c) broadcast bundle parity (incl. gasUsed)")

	bc := goBroadcasts[0]
	t.Logf("forked A/B parity OK: version=%s, broadcast{from=%s to=%s gasUsed=%d nonce=%d type=%s}, diff=%d bytes",
		goVersion, bc.From, bc.To, bc.GasUsed, bc.Nonce, bc.Type, len(goDiffJSON))
}

// goForkedHost builds a fork-backed Go script.Host at a fixed block, dialing forkURL. It mirrors
// env.ForkedScriptHost and captures broadcasts via the passed hook. isolate toggles
// WithIsolatedBroadcasts (the production env.DefaultForkedScriptHost setting): the non-isolated form
// is used by TestRustEngineForkedParity to isolate the fork-state variable; the isolated form by
// TestRustEngineForkedIsolatedParity to validate the engine's broadcast isolation.
func goForkedHost(t *testing.T, artifactsFS foundry.StatDirFs, forkURL string, blockNumber uint64,
	isolate bool, onBroadcast script.BroadcastHook) *script.Host {
	t.Helper()
	forkRPC, err := rpc.Dial(forkURL)
	require.NoError(t, err)
	t.Cleanup(forkRPC.Close)

	scriptCtx := script.DefaultContext
	scriptCtx.Sender = forkDeployer
	scriptCtx.Origin = forkDeployer
	opts := []script.HostOption{
		script.WithBroadcastHook(onBroadcast),
		script.WithCreate2Deployer(),
		script.WithForkHook(func(cfg *script.ForkConfig) (forking.ForkSource, error) {
			src, err := forking.RPCSourceByNumber(cfg.URLOrAlias, forkRPC, *cfg.BlockNumber)
			if err != nil {
				return nil, err
			}
			return forking.Cache(src), nil
		}),
	}
	if isolate {
		opts = append(opts, script.WithIsolatedBroadcasts())
	}
	h := script.NewHost(
		testlog.Logger(t, log.LevelError),
		&foundry.ArtifactsFS{FS: artifactsFS},
		nil,
		scriptCtx,
		opts...,
	)
	require.NoError(t, h.EnableCheats())
	_, err = h.CreateSelectFork(
		script.ForkWithURLOrAlias("main"),
		script.ForkWithBlockNumberU256(new(big.Int).SetUint64(blockNumber)),
	)
	require.NoError(t, err)
	return h
}

func versionSelector() []byte { return crypto.Keccak256([]byte("version()"))[:4] }

func contractVersionAt(t *testing.T, host *script.Host, addr common.Address) string {
	t.Helper()
	data, _, err := host.Call(common.Address{}, addr, versionSelector(), 1_000_000, uint256.NewInt(0))
	require.NoError(t, err)
	return decodeVersion(t, data)
}

func decodeVersion(t *testing.T, data []byte) string {
	t.Helper()
	decoded, err := (abi.Arguments{{Type: abi.Type{T: abi.StringTy}}}).Unpack(data)
	require.NoError(t, err)
	return decoded[0].(string)
}

func mustCall(t *testing.T, e *Engine, from, to common.Address, input []byte) []byte {
	t.Helper()
	out, err := e.Call(from, to, input)
	require.NoError(t, err)
	return out
}

func ptrU64(v uint64) *uint64 { return &v }

// pruneForkScaffolding removes the script-scaffolding accounts from a forking.ExportDiff-shaped JSON
// blob, so an A/B diff compares the real L1 fork writes rather than the deterministic deploy
// machinery. The set matches both script.Host.StateDump's pruning and the Rust engine's built-in
// fork-diff exclusion: DefaultSender, VMAddr, Console, Script/ForgeDeployer, and the whole
// ScriptDeployer CREATE range (the input/output precompiles + the script contract). Applied to BOTH
// sides symmetrically; a no-op on the already-pruned Rust diff.
func pruneForkScaffolding(t *testing.T, raw []byte) []byte {
	t.Helper()
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &top))
	var acct map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(top["account"], &acct))
	for _, a := range forkScaffoldingAddrs() {
		delete(acct, strings.ToLower(a.Hex()))
	}
	reAcct, err := json.Marshal(acct)
	require.NoError(t, err)
	top["account"] = reAcct
	out, err := json.Marshal(top)
	require.NoError(t, err)
	return out
}

func forkScaffoldingAddrs() []common.Address {
	out := []common.Address{
		addresses.DefaultSenderAddr, addresses.VMAddr, addresses.ConsoleAddr,
		addresses.ScriptDeployer, addresses.ForgeDeployer,
	}
	// A RunScriptVoid mints ≤3 script addresses; 8 is a safe upper bound on the CREATE range.
	for i := uint64(0); i < 8; i++ {
		out = append(out, crypto.CreateAddress(addresses.ScriptDeployer, i))
	}
	return out
}
