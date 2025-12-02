package unsafe_only

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestUnsafeOnly_VerifierUnsafeGapClosed(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	require := t.Require()
	attempts := 10

	sys.L2CL.AdvancedUnsafe(3, attempts)
	sys.L2EL.MatchedUnsafe(sys.L2ELB, attempts)
	sys.L2CL.MatchedUnsafe(sys.L2CLB, attempts)

	// Case 1: Closing the gap starting from genesis
	sys.L2CLB.Stop()
	sys.L2ELB.DisconnectPeerWith(sys.L2EL)
	// Wipe EL to genesis
	sys.L2ELB.Stop()
	sys.L2ELB.Start()
	// Check EL rewinded to genesis. Unsafe gap introduced
	sys.L2ELB.UnsafeHead().IsGenesis()
	// Verifier CL triggers EL Sync to close the gap including genesis
	sys.L2CLB.Start()
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2ELB.PeerWith(sys.L2EL)
	// Gap is closed
	sys.L2CLB.MatchedUnsafe(sys.L2CL, attempts)
	sys.L2ELB.MatchedUnsafe(sys.L2EL, attempts)

	// Case 2: Closing the gap not starting from genesis
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.AdvancedUnsafe(3, attempts)
	sys.L2CLB.NotAdvanced(types.LocalUnsafe, 3)
	// Turn back the CLP2P
	sys.L2CLB.ConnectPeer(sys.L2CL)
	// gap is closed again
	sys.L2CLB.MatchedUnsafe(sys.L2CL, attempts)
	sys.L2ELB.MatchedUnsafe(sys.L2EL, attempts)

	// Derivation did not happen
	sys.L2CL.SafeHead().IsGenesis()

	// Derivation happened at the second verifier
	require.Greater(sys.L2CLC.SafeHead().BlockRef.Number, uint64(0))

	t.Cleanup(func() {
		sys.L2ELB.Start()
		sys.L2ELB.PeerWith(sys.L2EL)
		sys.L2CLB.Start()
		sys.L2CLB.ConnectPeer(sys.L2CL)
	})
}

func TestUnsafeOnly_SequencerRestart(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	require := t.Require()

	attempts := 10

	sys.L2CL.AdvancedUnsafe(3, attempts)
	sys.L2EL.MatchedUnsafe(sys.L2ELB, attempts)
	sys.L2CL.MatchedUnsafe(sys.L2CLB, attempts)

	// Stop the sequencer
	sys.L2CL.Stop()
	sys.L2ELB.NotAdvancedUnsafe(3)

	// Restart the sequencer
	sys.L2CL.Start()
	// Sequencer produces blocks again
	sys.L2CL.AdvancedUnsafe(3, attempts)

	// Derivation did not happen at sequencer
	sys.L2CL.SafeHead().IsGenesis()

	// Stop the sequencer with API
	sys.L2CL.StopSequencer()
	sys.L2ELB.NotAdvancedUnsafe(3)

	// Restart the sequencer with API
	sys.L2CL.StartSequencer()
	// Sequencer produces blocks again
	sys.L2CL.AdvancedUnsafe(3, attempts)

	// Derivation did not happen at sequencer
	sys.L2CL.SafeHead().IsGenesis()

	// Derivation happened at the second verifier
	safeHeadNum := sys.L2CLC.SafeHead().BlockRef.Number
	require.Greater(safeHeadNum, uint64(0))

	t.Cleanup(func() {
		sys.L2CL.Start()
	})
}
