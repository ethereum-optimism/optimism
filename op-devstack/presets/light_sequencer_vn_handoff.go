package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// BootstrapLightSequencersViaVNHandoff brings up a two-L2 light-sequencer supernode
// interop system that was constructed with WithSupernodeVNSequencerForBootstrap.
//
// A follow-mode ELSync sequencer cannot bootstrap a chain from genesis as the sole
// producer (it deadlocks in willStartEL with no peer payload, #21164). This helper
// uses the supernode VN as the bootstrap producer: the VN actively sequences and
// gossips unsafe blocks so the light ELSync sequencers leave willStartEL via EL
// sync, then sequencing is handed off from the VN to the light sequencers.
//
// On return: the light CLs are the active sequencers, the supernode routes are
// stopped, and the chain is live on the light ELSync sequencers.
func BootstrapLightSequencersViaVNHandoff(t devtest.T, sys *TwoL2SupernodeInterop) {
	// Re-peer the light sequencers to their supernode routes across any restarts.
	sys.L2ACL.ManagePeer(sys.L2ASupernodeCL)
	sys.L2BCL.ManagePeer(sys.L2BSupernodeCL)

	// Bootstrap producer is the VN; the light ELSync sequencers start stopped.
	vnAActive, err := sys.L2ASupernodeCL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain A VN sequencer status")
	t.Require().True(vnAActive, "chain A supernode VN should be the active bootstrap sequencer")
	vnBActive, err := sys.L2BSupernodeCL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain B VN sequencer status")
	t.Require().True(vnBActive, "chain B supernode VN should be the active bootstrap sequencer")

	// Wait for the light ELSync sequencers to EL-sync from the VN's gossip and
	// leave willStartEL.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.LocalUnsafe, 3, 60),
		sys.L2BCL.AdvancedFn(safety.LocalUnsafe, 3, 60),
	)

	// Hand off sequencing from the VN to the light sequencers.
	sys.L2ASupernodeCL.StopSequencer()
	sys.L2BSupernodeCL.StopSequencer()
	sys.L2ACL.StartSequencer()
	sys.L2BCL.StartSequencer()

	lightAActive, err := sys.L2ACL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain A light sequencer status after handoff")
	t.Require().True(lightAActive, "chain A light sequencer should be active after handoff")
	lightBActive, err := sys.L2BCL.Escape().RollupAPI().SequencerActive(t.Ctx())
	t.Require().NoError(err, "chain B light sequencer status after handoff")
	t.Require().True(lightBActive, "chain B light sequencer should be active after handoff")
}
