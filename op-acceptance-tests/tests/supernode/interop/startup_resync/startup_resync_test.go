// Package startup_resync contains acceptance tests for the op-supernode
// interop startup rework's cold-start resync path.
package startup_resync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	l2BlockTime         = uint64(1)
	backfillDepth       = 3 * time.Second
	preRestartFinalized = uint64(5)
	preActivationDelay  = uint64(60 * 60)
)

// TestSupernodeResyncResumesAtActivation_PostActivation wipes the supernode
// data dir after the chain has crossed activation and asserts cross-safe
// resumes. The "EL data wiped" subtest additionally wipes the supernode-
// fronted EL so it must execution-layer-sync from a sibling sequencer EL.
func TestSupernodeResyncResumesAtActivation_PostActivation(gt *testing.T) {
	for _, tc := range []struct {
		name   string
		wipeEL bool
	}{
		{"EL data intact", false},
		{"EL data wiped", true},
	} {
		gt.Run(tc.name, func(gt *testing.T) {
			runPostActivationResync(gt, tc.wipeEL)
		})
	}
}

// TestSupernodeResyncSchedulesAtActivation_PreActivation wipes the supernode
// data dir while interop is still scheduled and asserts cold-start parks the
// verifier at activation. The "EL data wiped" subtest additionally wipes the
// supernode-fronted EL.
func TestSupernodeResyncSchedulesAtActivation_PreActivation(gt *testing.T) {
	for _, tc := range []struct {
		name   string
		wipeEL bool
	}{
		{"EL data intact", false},
		{"EL data wiped", true},
	} {
		gt.Run(tc.name, func(gt *testing.T) {
			runPreActivationResync(gt, tc.wipeEL)
		})
	}
}

func runPostActivationResync(gt *testing.T, wipeEL bool) {
	t := devtest.SerialT(gt)
	sys := newResyncSystem(t, 0, wipeEL)
	sys.supernode.AwaitBackfillCompleted()

	dsl.CheckAll(t,
		sys.l2ACL.AdvancedFn(types.Finalized, preRestartFinalized, 180),
		sys.l2BCL.AdvancedFn(types.Finalized, preRestartFinalized, 180),
	)

	activation := sys.supernode.ActivationTimestamp()
	sys.wipeAndRestart()
	if wipeEL {
		// Cold-start hands off at whichever safeDB timestamp first appears
		// post-restart, which sits at or after activation.
		sys.supernode.AwaitVerificationStartsAtOrAfter(activation)
	} else {
		sys.supernode.AwaitBackfillCompleted()
	}

	dsl.CheckAll(t,
		sys.l2ACL.AdvancedFn(types.CrossSafe, 1, 240),
		sys.l2BCL.AdvancedFn(types.CrossSafe, 1, 240),
	)

	if !wipeEL {
		sys.supernode.AssertBackfillCovers(backfillDepth, l2BlockTime,
			sys.l2A.ChainID(), sys.l2B.ChainID())
	}
}

func runPreActivationResync(gt *testing.T, wipeEL bool) {
	t := devtest.SerialT(gt)
	// preActivationDelay keeps the chain well below activation throughout
	// the test, so cold-start always parks at the future activation
	// timestamp regardless of CI scheduling variance.
	sys := newResyncSystem(t, preActivationDelay, wipeEL)
	sys.supernode.AwaitBackfillCompleted()
	activation := sys.supernode.ActivationTimestamp()

	dsl.CheckAll(t,
		sys.l2ACL.AdvancedFn(types.LocalSafe, 2, 60),
		sys.l2BCL.AdvancedFn(types.LocalSafe, 2, 60),
	)

	sys.wipeAndRestart()
	sys.supernode.AwaitVerificationStartsAt(activation)

	dsl.CheckAll(t,
		sys.l2ACL.AdvancedFn(types.CrossSafe, 1, 240),
		sys.l2BCL.AdvancedFn(types.CrossSafe, 1, 240),
	)
}

// resyncSystem hides the EL-intact vs EL-wiped preset choice and exposes a
// single wipeAndRestart action that does the right thing for each variant.
type resyncSystem struct {
	wipeEL    bool
	supernode *dsl.Supernode
	l2A       *dsl.L2Network
	l2B       *dsl.L2Network
	l2ACL     *dsl.L2CLNode
	l2BCL     *dsl.L2CLNode
	l2ELA     *dsl.L2ELNode
	l2ELB     *dsl.L2ELNode
	// Sibling sequencer pair — only populated when wipeEL is true.
	seqELA *dsl.L2ELNode
	seqELB *dsl.L2ELNode
	seqCLA *dsl.L2CLNode
	seqCLB *dsl.L2CLNode
}

func newResyncSystem(t devtest.T, delaySeconds uint64, wipeEL bool) *resyncSystem {
	opts := []presets.Option{
		presets.WithUniformL2BlockTimes(l2BlockTime),
		presets.WithInteropLogBackfillDepth(backfillDepth),
	}
	if !wipeEL {
		base := presets.NewTwoL2SupernodeInterop(t, delaySeconds, opts...)
		return &resyncSystem{
			supernode: base.Supernode,
			l2A:       base.L2A, l2B: base.L2B,
			l2ACL: base.L2ACL, l2BCL: base.L2BCL,
			l2ELA: base.L2ELA, l2ELB: base.L2ELB,
		}
	}
	peer := presets.NewTwoL2SupernodeInteropPeerEL(t, delaySeconds, opts...)
	return &resyncSystem{
		wipeEL:    true,
		supernode: peer.Supernode,
		l2A:       peer.L2A, l2B: peer.L2B,
		l2ACL: peer.L2ACL, l2BCL: peer.L2BCL,
		l2ELA: peer.L2ELA, l2ELB: peer.L2ELB,
		seqELA: peer.SequencerL2AEL, seqELB: peer.SequencerL2BEL,
		seqCLA: peer.SequencerL2ACL, seqCLB: peer.SequencerL2BCL,
	}
}

func (s *resyncSystem) wipeAndRestart() {
	if !s.wipeEL {
		s.supernode.RestartWithFreshDataDir()
		return
	}
	s.supernode.RestartWithFreshDataDirAndELs(s.l2ELA, s.l2ELB)
	// Discovery is off and the wipe resets every peer table — re-add the
	// sibling sequencer pair so the VN learns about new heads (CL pubsub)
	// and the EL can backfill via execution-layer devp2p.
	s.l2ACL.ConnectPeer(s.seqCLA)
	s.l2BCL.ConnectPeer(s.seqCLB)
	s.l2ELA.PeerWith(s.seqELA)
	s.l2ELB.PeerWith(s.seqELB)
}
