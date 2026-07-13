package rustengine

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
)

// TestRustEngineForkedIsolatedParity validates the engine's broadcast isolation (op-geth's
// script.WithIsolatedBroadcasts) against the Go host, byte-for-byte. It drives the same forked
// op-deployer script as TestRustEngineForkedParity (opcm.SetDisputeGameImpl against a pinned OP
// Sepolia block from the committed RPC fixture), but with BOTH the Go host and the Rust engine in
// ISOLATED mode — the production env.DefaultForkedScriptHost / apply.go forked-engine setting.
//
// Isolation resets the access list before each broadcast so its recorded gasUsed measures an
// equivalent standalone tx (cold access list) rather than the script's already-warm state. This is
// load-bearing: op-deployer pads the broadcast gas limit off gasUsed, so a warm-list underestimate
// makes live txs run out of gas (surfaced by TestEndToEndApply before isolation was implemented).
//
// The gate asserts (a) the full isolated broadcast bundle is byte-identical across engines
// (including gasUsed), and (b) the isolated gasUsed exceeds the NON-isolated value the R2 gate
// records (proving isolation actually re-cooled the DGF's gameImpls storage slot). Hermetic: same
// committed fixture, no network/secret/anvil, 0 skips beyond the binary/cargo guard.
func TestRustEngineForkedIsolatedParity(t *testing.T) {
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

	// ---- Leg A: in-process Go forked host, ISOLATED (production env.DefaultForkedScriptHost) ----
	var goBroadcasts []script.Broadcast
	gh := goForkedHost(t, artifactsFS, srv.URL, forkBlock, true, func(b script.Broadcast) {
		goBroadcasts = append(goBroadcasts, b)
	})
	require.NoError(t, opcm.RunScriptVoid[opcm.SetDisputeGameImplInput](
		gh, input, "SetDisputeGameImpl.s.sol", "SetDisputeGameImpl"), "go isolated forked RunScriptVoid")

	// ---- Leg B: out-of-process Rust engine fork mode, ISOLATED ----
	re, err := Spawn(bin, SpawnOpts{
		ArtifactsDir:       artifactsDir,
		ChainID:            1337, // fork does not change chain/block env
		Create2Deployer:    true,
		IsolatedBroadcasts: true,
	}, logw)
	require.NoError(t, err)
	defer re.Close()

	_, err = re.CreateSelectFork(srv.URL, ptrU64(forkBlock))
	require.NoError(t, err, "rust createSelectFork")
	require.NoError(t, RunScriptVoid[opcm.SetDisputeGameImplInput](
		re, input, "SetDisputeGameImpl.s.sol", "SetDisputeGameImpl", forkDeployer), "rust isolated forked RunScriptVoid")
	rustBroadcasts, err := re.TakeBroadcasts()
	require.NoError(t, err)

	// ---- Non-vacuity ----
	require.GreaterOrEqual(t, len(goBroadcasts), 1, "the script must produce at least one broadcast")
	require.Greater(t, goBroadcasts[0].GasUsed, uint64(0), "broadcast gasUsed must be non-zero")

	// ---- Isolated broadcast bundle parity, byte-for-byte including gasUsed ----
	gb, err := json.MarshalIndent(goBroadcasts, "", "  ")
	require.NoError(t, err)
	rb, err := json.MarshalIndent(rustBroadcasts, "", "  ")
	require.NoError(t, err)
	require.JSONEq(t, string(gb), string(rb), "isolated broadcast bundle parity (incl. gasUsed)")

	// ---- Isolation is non-trivial: the isolated gasUsed exceeds the non-isolated R2 value ----
	// SetDisputeGameImpl reads gameImpls(gameType) (warming the slot) and then writes it. Non-isolated,
	// the write hits a warm slot (recorded 59699 in the R2 gate); isolated, the access list is reset so
	// the write pays the +2100 cold-SLOAD surcharge. A gasUsed strictly above the non-isolated value
	// proves the reset actually re-cooled the slot rather than being a no-op.
	const nonIsolatedGasUsed = uint64(59699)
	require.Greater(t, goBroadcasts[0].GasUsed, nonIsolatedGasUsed,
		"isolated gasUsed must exceed the non-isolated value (proves the access-list reset re-cooled state)")

	bc := goBroadcasts[0]
	t.Logf("isolated forked A/B parity OK: broadcast{from=%s to=%s gasUsed=%d (non-isolated %d) nonce=%d type=%s}",
		bc.From, bc.To, bc.GasUsed, nonIsolatedGasUsed, bc.Nonce, bc.Type)
}
