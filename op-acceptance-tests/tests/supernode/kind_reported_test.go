package supernode

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/stretchr/testify/require"
)

// TestSupernodeKindReported records which supernode implementation the run selected, as a
// test name rather than as log output.
//
// The name is the point. gotestsum's JUnit output keeps per-test names for passing tests but
// not their stdout, so a t.Log here would be readable in a failed run and invisible in a green
// one -- and a green run is exactly the case where the question matters, because a green run is
// what gets reported as evidence that an implementation works. Running the assertion inside a
// subtest named for the kind puts the answer in the record itself: the CircleCI test output for
// a green run contains "TestSupernodeKindReported/selected=op-supernode" or
// ".../selected=lokahi", and a reader can tell the two apart without opening the job's exported
// environment.
//
// This is a correction for a specific way the suite has misled readers. The job named
// memory-all-kona-lokahi-op-reth-<fork> ran DEVSTACK_SUPERNODE_KIND=op-supernode for as long as
// the name said lokahi, so three separate green runs were read as lokahi passing the acceptance
// tests when lokahi had not started. The job name is being corrected alongside this test, but a
// name can drift from its config again; a record that names the implementation the run actually
// asked for cannot, because it is produced by the run.
//
// Scope: this reports the *requested* kind. The dispatch honours it directly -- see
// startTwoL2SharedSupernode and startSingleChainSharedSupernode in
// op-devstack/sysgo/multichain_supernode_runtime.go, which branch straight off
// ResolveSupernodeKind -- so requested and started agree unless that dispatch is bypassed.
// Reporting the kind of the supernode object a preset actually built would be strictly stronger
// and needs plumbing the TwoL2 preset does not have today, since it exposes no supernode handle.
// That is left as a follow-up rather than done half-way here.
func TestSupernodeKindReported(gt *testing.T) {
	t := devtest.SerialT(gt)

	// Resolving is itself the first assertion: an unrecognized DEVSTACK_SUPERNODE_KIND fails
	// here rather than falling back to op-supernode, which would report a green run for an
	// implementation that never started.
	kind := sysgo.ResolveSupernodeKind(t)

	// A kind that is resolvable but not one of the two known values would otherwise be reported
	// under a name no reader recognizes. Adding an implementation should mean updating this
	// list, deliberately, rather than having it appear as an unfamiliar record name.
	require.Contains(gt, []sysgo.SupernodeKind{sysgo.SupernodeOpSupernode, sysgo.SupernodeLokahi}, kind,
		"supernode kind %q resolved but is not one this test knows how to report", kind)

	gt.Run("selected="+string(kind), func(gt *testing.T) {
		gt.Logf("multi-chain/interop presets in this run use supernode implementation %q", kind)
	})
}
