package follow_source

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestFollowSourceSafeAndFinalized(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	logger := t.Logger()
	// Takes about 2 minutes for L1 finalization
	attempts := 70
	target := uint64(3)

	// L2CL is the sequencer with follow source, derivation disabled
	// L2CLB is the verifier without follow source, derivation enabled
	// L2CLC is the verifier with follow source, derivation disabled
	// All verifiers must eventually advance unsafe, safe, finalized
	checkMatchedAll := func(lvl types.SafetyLevel) {
		dsl.CheckAll(t,
			sys.L2CL.ReachedFn(lvl, target, attempts),
			sys.L2CLB.ReachedFn(lvl, target, attempts),
			sys.L2CLC.ReachedFn(lvl, target, attempts),
		)
		dsl.CheckAll(t,
			sys.L2CLB.MatchedFn(sys.L2CL, lvl, attempts),
			sys.L2CLB.MatchedFn(sys.L2CLC, lvl, attempts),
		)
	}

	checkMatchedAll(types.LocalUnsafe)
	logger.Info("Unsafe head advanced due to CLP2P", "target", target)

	checkMatchedAll(types.LocalSafe)
	logger.Info("Safe head followed source", "target", target)

	checkMatchedAll(types.Finalized)
	logger.Info("Finalized head followed source", "target", target)
}

func TestFollowSourceWithoutCLP2P(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainTwoVerifiersWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()

	attempts := 20
	target := uint64(3)

	// L2CLB is the verifier without follow source, derivation enabled
	sys.L2CLB.Advanced(types.LocalUnsafe, target, attempts)

	// The test's primary target is the L2CLC, with follow source and derivation disabled
	// Normally there should be delta between safe head between unsafe head
	status := sys.L2CLC.SyncStatus()
	require.NotEqual(status.LocalSafeL2, status.UnsafeL2)

	logger.Info("Disconnect CLP2P")
	// L2CLC is the verifier with follow source, derivation disabled
	// Disconnect CLP2P of verifier which follow source is enabled
	sys.L2CLC.DisconnectPeer(sys.L2CLB)
	sys.L2CLB.DisconnectPeer(sys.L2CLC)
	sys.L2CLC.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLC)

	// Advance few safe blocks
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)
	sys.L2CLC.Matched(sys.L2CLB, types.LocalSafe, attempts)

	// Make sure the safe head reaches non-moving unsafe head
	sys.L2CLC.Reached(types.LocalSafe, sys.L2CLC.UnsafeHead().BlockRef.Number, attempts)
	// The only data source for L2CLC is the safe source.
	// L2CLC unsafe head will only be advancing with safe head together
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)

	// Advance few safe blocks
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)

	// Check once again that the unsafe head is moving together with safe head
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(types.LocalSafe, target, attempts)

	// Recover CLP2P
	logger.Info("Recover CLP2P")
	sys.L2CLC.ConnectPeer(sys.L2CLB)
	sys.L2CLB.ConnectPeer(sys.L2CLC)
	sys.L2CLC.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLC)

	// Sequencer unsafe payload will arrive to the verifier, triggering EL sync and filling in the unsafe gap
	dsl.CheckAll(t,
		// Match with sequencer with derivation disabled
		sys.L2CLC.MatchedFn(sys.L2CL, types.LocalSafe, attempts),
		sys.L2CLC.MatchedFn(sys.L2CL, types.LocalUnsafe, attempts),
		// Match with other verifier with derivation enabled
		sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalSafe, attempts),
		sys.L2CLC.MatchedFn(sys.L2CLB, types.LocalUnsafe, attempts),
	)

	t.Cleanup(func() {
		sys.L2CLC.ConnectPeer(sys.L2CLB)
		sys.L2CLB.ConnectPeer(sys.L2CLC)
		sys.L2CLC.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLC)
	})
}
