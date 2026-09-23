package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestPrivateInteropPairComesUp is the topology's own smoke test: the pair stands up, both halves
// exist and are the two chains they are supposed to be, and the rendering starts advancing.
//
// It is separate from the message round trip on purpose. Almost everything that can go wrong with a
// pair goes wrong before any message is sent -- a genesis mismatch, a follow source that never
// answers, a builder that cannot resolve its parent check -- and a failure here says which, where a
// failure in a cross-chain test would only say "the message never arrived".
func TestPrivateInteropPairComesUp(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0, presets.WithPrivateInteropChain())
	require := sys.T.Require()

	require.NotNil(sys.PrivateInterop, "chain B should be a private interop pair")

	// One chain ID, two chains. This is the design's item 6 and the reason nothing here may be
	// peered across the boundary.
	private := sys.L2B.Escape().RollupConfig()
	rendering := sys.PrivateInterop.RenderingRollupConfig
	require.Zero(rendering.L2ChainID.Cmp(private.L2ChainID), "the halves share a chain ID")
	require.NotEqual(private.Genesis.L2.Hash, rendering.Genesis.L2.Hash, "the halves are different chains")
	require.Equal(private.Genesis.L2Time, rendering.Genesis.L2Time, "the halves start at the same instant")
	require.Equal(private.BlockTime, rendering.BlockTime, "the halves share a block time")

	// The private chain sequences on its own, from a claim-driven safe head it has not been given
	// yet. That it builds at all is the first thing the follow source has to not break.
	sys.L2ELB.WaitForBlock()
	sys.L2ELB.WaitForBlock()

	// And the rendering advances, which can only happen if the builder rendered a range, committed
	// to its private input, assembled a claim, posted a batch, and the rendering derived it.
	require.NotNil(sys.PrivateInterop.Invariant, "the standing invariant checker should be on by default")
	sys.PrivateInterop.Invariant.RequireRenderingReached(3, 4*time.Minute)

	checked, ok := sys.PrivateInterop.Invariant.CheckedThrough()
	require.True(ok, "the invariant checker should have verified at least one block of correspondence")
	t.Logger().Info("private interop pair is live",
		"rendering_safe", sys.PrivateInterop.Invariant.RenderingSafeHead(),
		"private_safe", sys.PrivateInterop.Invariant.PrivateSafeHead(),
		"correspondence_checked_through", checked)

	// The sequencer is following the supernode's sibling route, not a sidecar and not the rendering's
	// own route. The distinction is the recorded sharp edge: `<base>/<chainID>` answers with the
	// RENDERING's refs, and a private LightCL pointed there force-resets toward a hash no private peer
	// holds.
	require.Equal(sys.L2BSupernodeCL.Escape().UserRPC()+"/claimed", sys.PrivateInterop.FollowSource(),
		"the private sequencer must follow the supernode's claimed sibling route")

	// The private chain's own safe head is claim-driven: it only moves because a claim landed, the
	// follower found it on the rendering, and the private chain hashed the same at the claimed
	// terminal block. That is strictly later than the rendering advancing, so it gets its own wait.
	sys.PrivateInterop.Invariant.RequirePrivateSafeReached(1, 3*time.Minute)

	privateSafe := sys.L2BCL.Escape().RollupAPI()
	status, err := privateSafe.SyncStatus(t.Ctx())
	require.NoError(err, "reading the private chain's sync status")
	require.Positive(status.SafeL2.Number, "the private chain's claim-driven safe head should have advanced")
	require.LessOrEqual(status.SafeL2.Number, status.UnsafeL2.Number, "safe never exceeds unsafe")

	// Severance, observed: the two halves disagree about the content of a block number they both
	// have, and neither has adopted the other's.
	renderedHead := sys.L2BSupernodeEL.BlockRefByLabel(eth.Safe)
	privateAtSame := sys.L2ELB.BlockRefByNumber(renderedHead.Number)
	require.Equal(renderedHead.Time, privateAtSame.Time, "block-for-block correspondence")
	require.NotEqual(renderedHead.Hash, privateAtSame.Hash, "the two halves are different chains at the same height")
}
