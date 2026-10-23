package node

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/sources"

	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

// Every observed unsafe block must become safe without changing its contents.
func TestSyncUnsafeBecomesSafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	checkBlockRangesOnPeers(t, newCommonPreset(t), safety.LocalUnsafe, safety.CrossSafe)
}

// Unsafe blocks agree on this single-sequencer network without reorgs.
func TestSyncUnsafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	checkBlockRangesOnPeers(t, newCommonPreset(t), safety.LocalUnsafe, safety.LocalUnsafe)
}

// Safe blocks agree on this network with a single DA layer.
func TestSyncSafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	checkBlockRangesOnPeers(t, newCommonPreset(t), safety.CrossSafe, safety.CrossSafe)
}

// Every node finalizes multiple new blocks, and their contents agree across peers.
func TestSyncFinalized(gt *testing.T) {
	t := devtest.ParallelT(gt)
	checkBlockRangesOnPeers(t, newCommonPreset(t), safety.Finalized, safety.Finalized)
}

func syncHead(status *eth.SyncStatus, level safety.Level) eth.L2BlockRef {
	switch level {
	case safety.LocalUnsafe:
		return status.UnsafeL2
	case safety.CrossSafe:
		return status.SafeL2
	case safety.Finalized:
		return status.FinalizedL2
	default:
		panic("unsupported sync test safety level")
	}
}

// Capture a consecutive range from EVERY source before waiting on peers. Checking only one
// latest head can miss disagreements in intermediate blocks or on another source node.
func checkBlockRangesOnPeers(t devtest.T, preset *node_utils.MixedOpKonaPreset, observed, target safety.Level) {
	t.Helper()
	// The simple-kona preset has one Kona source. Include its op-node peers in
	// the agreement checks through ordinary RPC; op-node does not expose WS here.
	nodes := preset.L2CLKonaNodes()
	peers := preset.L2CLNodes()
	t.Require().GreaterOrEqual(len(nodes), 1, "sync agreement requires a Kona WebSocket source")
	t.Require().GreaterOrEqual(len(peers), 2, "sync agreement requires multiple nodes")
	const blockCount uint64 = 20
	clients := make([]*sources.RollupClient, len(nodes))
	starts := make([]uint64, len(nodes))
	checks := make([]dsl.CheckFunc, 0, len(nodes))
	for i := range nodes {
		clients[i] = node_utils.RollupWS(t, &nodes[i])
		status, err := clients[i].SyncStatus(t.Ctx())
		t.Require().NoError(err)
		starts[i] = syncHead(status, observed).Number + 1
		checks = append(checks, nodes[i].ReachedFn(observed, starts[i]+blockCount-1, 120))
	}
	dsl.CheckAll(t, checks...)

	var heads []eth.L2BlockRef
	var highest uint64
	for i := range nodes {
		for number := starts[i]; number < starts[i]+blockCount; number++ {
			output, err := clients[i].OutputAtBlock(t.Ctx(), number)
			t.Require().NoError(err)
			t.Require().Equal(number, output.BlockRef.Number)
			heads = append(heads, output.BlockRef)
			highest = max(highest, number)
		}
	}
	t.Require().Len(heads, len(nodes)*int(blockCount), "must check every block in every source range")

	checks = checks[:0]
	for i := range peers {
		checks = append(checks, peers[i].ReachedFn(target, highest, 120))
	}
	dsl.CheckAll(t, checks...)
	for i := range peers {
		for _, head := range heads {
			output, err := peers[i].Escape().RollupAPI().OutputAtBlock(t.Ctx(), head.Number)
			t.Require().NoError(err)
			t.Require().Equal(head, output.BlockRef, "block %d mismatch on %s at %s", head.Number, peers[i].Escape().Name(), target)
		}
	}
}
