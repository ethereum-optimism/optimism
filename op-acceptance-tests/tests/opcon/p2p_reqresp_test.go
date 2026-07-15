package opcon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// These tests cover the op-conp2p sidecar's P2P req-resp SERVING path — the
// opcon adaptation of the depreqres suite (op-acceptance-tests/tests/depreqres):
// the sidecar implements p2p.L2Chain over the sequencing op-con-node's Direct
// Sync pull endpoint (opql_getUnsafePayload, served from the signed replay
// ring, with older heights reconstructed from the sequencer's EL), so a stock
// op-node verifier can backfill unsafe blocks it never received on gossip via
// standard OP-stack req-resp (payloads-by-number), served ultimately by
// op-con-node. There is no batcher in either topology: req-resp through the
// sidecar is the verifier's ONLY route to a missed span.
//
//	DEVSTACK_L2CL_KIND=op-con-node \
//	DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	OP_CONP2P_BIN=<optimism>/op-conp2p-bin \
//	go test -run 'TestOpConVerifierBackfillsViaP2PReqResp|TestOpConSidecarRecoversPublishGap' ./op-acceptance-tests/tests/opcon/
//
// These topologies only exist for op-con-node (a stock op-node serves req-resp
// natively), so the tests skip on other CL kinds.

// TestOpConVerifierBackfillsViaP2PReqResp: a stock op-node verifier that joins
// the network LATE — after the op-con-node sequencer has already produced a
// span of signed blocks that were never gossiped — must recover that span over
// P2P req-resp against the sidecar and converge on the sequencer's unsafe tip.
//
// Mechanically (this is op-node's reverse CL sync, the same path the depreqres
// clsync tests exercise against a stock sequencer): the sequencer pre-produces
// the span with no gossip route; then the verifier and the publish sidecar come
// up and the sidecar joins the sequencer's Direct Sync feed live, so only NEW
// blocks are gossiped. The first live gossiped block arrives with an unknown
// parent, which triggers the verifier's req-resp range request to its only
// peer — the sidecar — whose payloads-by-number server pulls each height from
// the sequencer's signed replay ring via opql_getUnsafePayload. With no batcher,
// nothing else can supply the span: convergence proves the sidecar's req-resp
// serving path end to end.
func TestOpConVerifierBackfillsViaP2PReqResp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node is the sequencer under test")

	// The sequencer pre-produces this many blocks before the verifier exists.
	// Well within the default 1024-block signed replay ring, and small enough to
	// keep the test fast (2s block time).
	const preproduced = 8
	sys := presets.NewSingleChainOpConSequencerLateJoinReqRespWithoutCheck(t, preproduced)

	// The sequencer keeps producing after the verifier joins (it was never
	// stopped), so a live gossiped block triggers the reverse sync.
	sys.L2CL.Advanced(types.LocalUnsafe, 1, 60)

	// The verifier must backfill the entire pre-produced span (blocks it never
	// saw on gossip) and converge on the sequencer's tip.
	sys.L2CLB.Reached(types.LocalUnsafe, preproduced, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)

	// No L1 assist: there is no batcher, so both safe heads stay at genesis —
	// the backfilled span can only have come through req-resp.
	require.Zero(t, sys.L2CL.HeadBlockRef(types.LocalSafe).Number,
		"no batcher: the sequencer's safe head must stay at genesis")
	require.Zero(t, sys.L2CLB.HeadBlockRef(types.LocalSafe).Number,
		"no batcher: the verifier's safe head must stay at genesis")
}

// TestOpConSidecarRecoversPublishGap: a sidecar OUTAGE while the sequencer
// keeps producing must not permanently strand the verifier — after the sidecar
// is restarted, the verifier recovers the outage window and converges on the
// sequencer's unsafe tip, with no L1 help (no batcher).
//
// A restarted op-conp2p process holds no in-memory publish cursor (publish.go
// starts with lastBlock=0, which subscribes to the sequencer's Direct Sync
// feed LIVE — an absent fromBlock replays nothing), so the outage window is
// never re-gossiped. Recovery is instead the req-resp path: gossip resumes
// with the next live block, whose unknown parent triggers the verifier's
// reverse sync through the restarted sidecar's payloads-by-number server
// (backed by the sequencer's signed replay ring). The gap is kept small — the
// honest claim for what a fresh sidecar can heal promptly; the deep-gap case
// is TestOpConVerifierBackfillsViaP2PReqResp.
func TestOpConSidecarRecoversPublishGap(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipUnlessOpConNode(t, "op-con-node is the sequencer under test")
	sys := presets.NewSingleChainOpConSequencerP2PRestartableSidecarWithoutCheck(t)

	// Live baseline: the verifier follows the sequencer over gossip.
	sys.L2CL.Advanced(types.LocalUnsafe, 1, 60)
	sys.L2CLB.Advanced(types.LocalUnsafe, 1, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)

	// Outage: kill the sidecar (the verifier's only gossip peer) and let the
	// sequencer open a small gap.
	sys.Sidecar.Stop()
	verifierHeadAtOutage := sys.L2CLB.HeadBlockRef(types.LocalUnsafe)
	sys.L2CL.Advanced(types.LocalUnsafe, 3, 60)

	// Restart: a fresh sidecar process with the same flags re-dials the verifier
	// and rejoins the sequencer's feed live (no cursor to replay from).
	sys.Sidecar.Restart()

	// The verifier heals the outage window (req-resp through the restarted
	// sidecar) and converges on the sequencer's tip.
	sys.L2CLB.Reached(types.LocalUnsafe, verifierHeadAtOutage.Number+3, 60)
	sys.L2CLB.InSync(sys.L2CL, types.LocalUnsafe, 60)

	// No L1 assist: there is no batcher in this topology.
	require.Zero(t, sys.L2CLB.HeadBlockRef(types.LocalSafe).Number,
		"no batcher: the verifier's safe head must stay at genesis")
}
