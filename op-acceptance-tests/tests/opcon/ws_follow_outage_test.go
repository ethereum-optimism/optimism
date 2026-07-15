package opcon

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// TestOpConWSFollowVerifierHealsOutageViaRingReplay is the direct (no-P2P,
// no-sidecar) outage-recovery test for the Direct Sync protocol suite: an
// op-con-node verifier consuming the op-con-node sequencer's signed-payload
// websocket via --follow is stopped, the sequencer keeps producing, and on
// restart the verifier's follow engine resubscribes with its cursor
// ({fromBlock: last ingested + 1}) — the sequencer's signed replay ring
// replays the outage window and the verifier converges on the tip, with NO L1
// help (the batcher is stopped, so derivation cannot heal anything: ring
// replay is the only route).
//
// The topology is the WS-follow preset (TestOpConSequencerVerifiedViaSignedPayloadWS)
// with the batcher STOPPED via the standard batcher option — the same knob the
// depreqres suite and TestOpConVerifierFillsUnsafeGaps use — so both safe
// heads stay at genesis for the whole test. The outage window (a handful of
// blocks) is far inside the default 1024-block ring.
//
//	DEVSTACK_L2CL_KIND=op-con-node \
//	DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConWSFollowVerifierHealsOutageViaRingReplay ./op-acceptance-tests/tests/opcon/
//
// Both roles are opql-specific (the feed and the follower), so the test skips
// on other CL kinds.
func TestOpConWSFollowVerifierHealsOutageViaRingReplay(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node signed-payload websocket feed + follower")
	sys := presets.NewSingleChainOpConSequencerWSFollowWithoutCheck(t,
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			// Stop the batcher so the safe head never advances: the verifier's
			// unsafe chain — including the outage window — can only come from the
			// sequencer's websocket ring replay, never from L1 derivation.
			cfg.Stopped = true
		}),
	)

	// Live baseline: the verifier follows the sequencer's signed feed.
	dsl.CheckAll(t,
		sys.L2CL.ReachedFn(types.LocalUnsafe, 3, 60),
		sys.L2CLB.ReachedFn(types.LocalUnsafe, 3, 60),
	)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)
	preOutageHead := sys.L2CLB.HeadBlockRef(types.LocalUnsafe)

	// Outage: stop the verifier while the sequencer keeps producing, opening a
	// gap of several blocks (well within the default 1024-block replay ring).
	sys.L2CLB.Stop()
	sys.L2CL.Advanced(types.LocalUnsafe, 5, 60)

	// Restart: the verifier's follow engine resubscribes from its restored
	// cursor and the ring replays the outage window; the verifier converges on
	// the sequencer's tip. Pure ring replay — the batcher is stopped.
	sys.L2CLB.Start()
	sys.L2CLB.Reached(types.LocalUnsafe, preOutageHead.Number+5, 90)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 90)

	// The replayed window EXTENDED the verifier's chain: its pre-outage head is
	// still canonical on its EL (no reorg, no re-derivation from scratch).
	sys.L2ELB.VerifyNotReorged(preOutageHead)

	// No L1 assist happened: with the batcher stopped, both safe heads are
	// still at genesis.
	require.Zero(t, sys.L2CL.HeadBlockRef(types.LocalSafe).Number,
		"batcher stopped: the sequencer's safe head must stay at genesis")
	require.Zero(t, sys.L2CLB.HeadBlockRef(types.LocalSafe).Number,
		"batcher stopped: the verifier's safe head must stay at genesis")
}
