package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// proofsHistoryReferenceEthProofWindow lets the sequencer answer eth_getProof for every block of a
// run from reth's own historical state; reth's default window of zero rejects them all.
const proofsHistoryReferenceEthProofWindow = 1_000_000

// ProofsHistoryFollowerConfig is what varies between the systems NewProofsHistoryFollowerSystem
// starts.
type ProofsHistoryFollowerConfig struct {
	DeployerOptions    []sysgo.DeployerOption
	FollowerL2CLOpts   []sysgo.L2CLOption
	FollowerOpRethOpts []sysgo.OpRethOption
	// SequencerOpRethOpts apply after the options that turn the sequencer's proofs history off and
	// open its eth_getProof window.
	SequencerOpRethOpts []sysgo.OpRethOption
}

// ProofsHistoryFollowerSystem is an op-reth sequencer without proofs history, which serves
// eth_getProof from reth's own historical state and so is an independent reference, and an op-reth
// follower running proofs history.
type ProofsHistoryFollowerSystem struct {
	*MixedSingleChainFrontends

	Follower  MixedSingleChainNodeFrontends
	Sequencer MixedSingleChainNodeFrontends
}

// NewProofsHistoryFollowerSystem starts an op-reth sequencer without proofs history and an op-reth
// follower running proofs history.
func NewProofsHistoryFollowerSystem(t devtest.T, configure ...func(*ProofsHistoryFollowerConfig)) *ProofsHistoryFollowerSystem {
	var cfg ProofsHistoryFollowerConfig
	for _, apply := range configure {
		apply(&cfg)
	}
	clKind := sysgo.ResolveMixedL2CLKind()
	sequencerOpRethOpts := append([]sysgo.OpRethOption{
		sysgo.OpRethWithoutProofsHistory(),
		sysgo.OpRethWithEthProofWindow(proofsHistoryReferenceEthProofWindow),
	}, cfg.SequencerOpRethOpts...)
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		DeployerOptions: cfg.DeployerOptions,
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				CLKey:       "sequencer",
				CLKind:      clKind,
				ELKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				IsSequencer: true,
				OpRethOpts:  sequencerOpRethOpts,
			},
			{
				CLKey:      "follower",
				CLKind:     clKind,
				ELKey:      "proofs-follower",
				ELKind:     sysgo.MixedL2ELOpReth,
				L2CLOpts:   cfg.FollowerL2CLOpts,
				OpRethOpts: cfg.FollowerOpRethOpts,
			},
		},
	})
	frontends := NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "expected the sequencer and the proofs follower")

	sys := &ProofsHistoryFollowerSystem{MixedSingleChainFrontends: frontends}
	for _, node := range frontends.Nodes {
		if node.Spec.IsSequencer {
			sys.Sequencer = node
		} else {
			sys.Follower = node
		}
	}
	t.Require().NotNil(sys.Follower.EL, "missing the proofs follower")
	return sys
}
