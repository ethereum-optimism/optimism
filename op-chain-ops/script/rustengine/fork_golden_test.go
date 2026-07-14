package rustengine

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
)

// TestRustEngineForkedGolden goldenizes TestRustEngineForkedParity: it drives opcm.SetDisputeGameImpl
// against the committed OP-Sepolia fork fixture through the Rust engine's fork mode and pins the three
// output axes (version() read-through, pruned fork diff, broadcast bundle) that the parity test
// compared against the live Go forked host. Hermetic (replays the committed RPC fixture; no network).
func TestRustEngineForkedGolden(t *testing.T) {
	bin := buildEngine(t)
	logw := testWriter{t}

	loc, _ := testutil.LocalArtifacts(t)
	artifactsDir := loc.URL.Path

	srv, dump := startForkFixtureServer(t, forkBlock)
	defer srv.Close()
	defer dump()

	input := opcm.SetDisputeGameImplInput{
		Factory:             forkDGF,
		Impl:                common.Address{'I'},
		GameType:            forkGameType,
		AnchorStateRegistry: common.Address{},
		GameArgs:            gameargs.GameArgs{}.PackPermissionless(),
	}

	re, err := Spawn(bin, SpawnOpts{
		ArtifactsDir:    artifactsDir,
		ChainID:         1337,
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
	rustPruned := pruneForkScaffolding(t, rustDiffJSON)

	// Non-vacuity guards (carried over from the parity test).
	require.NotEmpty(t, rustVersion, "version() read must be non-empty")
	require.GreaterOrEqual(t, len(rustBroadcasts), 1, "the script must produce at least one broadcast")
	require.Contains(t, string(rustPruned), strings.ToLower(forkDGF.Hex()),
		"pruned diff must still contain the DGF (the load-bearing storage write)")

	requireTextMatchesGolden(t, "fork.version.txt", rustVersion)
	requireJSONBytesMatchesGolden(t, "fork.diff.json", rustPruned)
	requireJSONMatchesGolden(t, "fork.broadcasts.json", rustBroadcasts)
}

// TestRustEngineForkedIsolatedGolden goldenizes TestRustEngineForkedIsolatedParity: the same forked
// script run with broadcast isolation on (env.DefaultForkedScriptHost / apply.go setting). It pins the
// isolated broadcast bundle and keeps the load-bearing "isolated gasUsed exceeds the non-isolated
// value" assertion that proves the access-list reset actually re-cooled the DGF's gameImpls slot.
func TestRustEngineForkedIsolatedGolden(t *testing.T) {
	bin := buildEngine(t)
	logw := testWriter{t}

	loc, _ := testutil.LocalArtifacts(t)
	artifactsDir := loc.URL.Path

	srv, dump := startForkFixtureServer(t, forkBlock)
	defer srv.Close()
	defer dump()

	input := opcm.SetDisputeGameImplInput{
		Factory:             forkDGF,
		Impl:                common.Address{'I'},
		GameType:            forkGameType,
		AnchorStateRegistry: common.Address{},
		GameArgs:            gameargs.GameArgs{}.PackPermissionless(),
	}

	re, err := Spawn(bin, SpawnOpts{
		ArtifactsDir:       artifactsDir,
		ChainID:            1337,
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

	// Non-vacuity guards (carried over).
	require.GreaterOrEqual(t, len(rustBroadcasts), 1, "the script must produce at least one broadcast")
	require.Greater(t, rustBroadcasts[0].GasUsed, uint64(0), "broadcast gasUsed must be non-zero")

	// Isolation is non-trivial: the isolated gasUsed must exceed the non-isolated value the R2 gate
	// records, proving the access-list reset re-cooled the gameImpls storage slot (a warm-list
	// underestimate would make padded live txs run out of gas).
	require.Greater(t, rustBroadcasts[0].GasUsed, nonIsolatedForkGasUsed,
		"isolated gasUsed must exceed the non-isolated value (proves the access-list reset re-cooled state)")

	requireJSONMatchesGolden(t, "fork.isolated.broadcasts.json", rustBroadcasts)
}

// nonIsolatedForkGasUsed is the SetDisputeGameImpl broadcast gasUsed WITHOUT isolation (recorded from
// the Go host; see fork.broadcasts.json). The isolated run must exceed it.
const nonIsolatedForkGasUsed = uint64(59699)
