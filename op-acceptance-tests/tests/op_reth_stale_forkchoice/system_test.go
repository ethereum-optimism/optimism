package op_reth_stale_forkchoice

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// chainBuildUpAttempts bounds, in 2s polls, how long the follower may take to hold the blocks
	// a scenario needs before it starts: at most about a minute of 1s blocks.
	chainBuildUpAttempts = 90
	// defaultPersistenceThreshold is reth's --engine.persistence-threshold for the scenarios that
	// do not vary it: reth 2.5's default, passed explicitly so the persistence schedule does not
	// move with the binary.
	defaultPersistenceThreshold = 7
	// l2BlockTimeSeconds keeps the runs short: the chain lengths the scenarios need are block
	// counts, so halving the block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so how many blocks reth holds in memory does
	// not depend on its defaults.
	memoryBlockBufferTarget = 5
	// resyncAttempts bounds, in 2s polls, how long the follower may take to be in sync with the
	// sequencer again once it has recovered.
	resyncAttempts = 60
)

const (
	followerKey  = "follower"
	sequencerKey = "sequencer"
)

// l2Node is one L2 node of the system: its execution layer and the consensus client driving it.
type l2Node struct {
	CL *dsl.L2CLNode
	EL *dsl.L2ELNode
}

// systemConfig is what varies between the systems the scenarios run on.
type systemConfig struct {
	persistenceThreshold uint64
	// syncMode is how the follower's consensus client fills an unsafe gap: CL sync has op-node
	// fetch the blocks from its peers, EL sync hands the target to the EL. kona-node has no CL
	// sync mode and ignores this.
	syncMode nodeSync.Mode
	// withTestSequencer adds a test sequencer that builds L1 blocks on a chosen parent, which is
	// how a scenario reorgs L1.
	withTestSequencer bool
}

func (c systemConfig) engineArgs() []string {
	return []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
	}
}

type staleForkchoiceSystem struct {
	Batcher       *dsl.L2Batcher
	Follower      l2Node
	L1CL          *dsl.L1CLNode
	L1EL          *dsl.L1ELNode
	L1Network     *dsl.L1Network
	Sequencer     l2Node
	TestSequencer *dsl.TestSequencer
}

// newStaleForkchoiceSystem starts an op-reth sequencer and an op-reth follower, both without
// proofs history and on the configured persistence schedule, with the follower's consensus
// client in the configured sync mode.
func newStaleForkchoiceSystem(t devtest.T, cfg systemConfig) *staleForkchoiceSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	opRethOpts := []sysgo.OpRethOption{
		sysgo.OpRethWithoutProofsHistory(),
		sysgo.OpRethWithExtraArgs(cfg.engineArgs()...),
	}
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				CLKey:       sequencerKey,
				CLKind:      clKind,
				ELKey:       sequencerKey,
				ELKind:      sysgo.MixedL2ELOpReth,
				IsSequencer: true,
				OpRethOpts:  opRethOpts,
			},
			{
				CLKey:      followerKey,
				CLKind:     clKind,
				ELKey:      followerKey,
				ELKind:     sysgo.MixedL2ELOpReth,
				L2CLOpts:   []sysgo.L2CLOption{sysgo.L2CLVerifierSyncMode(cfg.syncMode)},
				OpRethOpts: opRethOpts,
			},
		},
		WithTestSequencer: cfg.withTestSequencer,
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "expected the sequencer and the follower")

	sys := &staleForkchoiceSystem{
		Batcher:       frontends.L2Batcher,
		L1CL:          frontends.L1CL,
		L1EL:          frontends.L1EL,
		L1Network:     frontends.L1Network,
		TestSequencer: frontends.TestSequencer,
	}
	for _, node := range frontends.Nodes {
		switch node.Spec.ELKey {
		case sequencerKey:
			sys.Sequencer = l2Node{CL: node.CL, EL: node.EL}
		case followerKey:
			sys.Follower = l2Node{CL: node.CL, EL: node.EL}
		}
	}
	t.Require().NotNil(sys.Sequencer.EL, "missing the sequencer")
	t.Require().NotNil(sys.Follower.EL, "missing the follower")
	return sys
}

// syncModeCase is one cell of the matrix the follower's consensus client runs under.
type syncModeCase struct {
	syncMode nodeSync.Mode
}

func (c syncModeCase) name() string {
	if c.syncMode == nodeSync.ELSync {
		return "el_sync"
	}
	return "cl_sync"
}

func syncModeMatrix() []syncModeCase {
	return []syncModeCase{{syncMode: nodeSync.CLSync}, {syncMode: nodeSync.ELSync}}
}

// skipUnsupportedCell skips the cells the configured clients cannot run.
func skipUnsupportedCell(t devtest.T, c syncModeCase) {
	skipUnsupported(t)
	if c.syncMode == nodeSync.CLSync {
		sysgo.SkipOnKonaNode(t, "kona-node has no CL sync mode: it always lets the EL sync the gap")
	}
}

func skipUnsupported(t devtest.T) {
	sysgo.SkipOnOpGeth(t, "the forkchoice updates under test are answered by op-reth")
}

// engineLabels are an execution engine's head, safe and finalized blocks, read together so a
// forkchoice update can be required to have changed none of them.
type engineLabels struct {
	finalized eth.L2BlockRef
	head      eth.L2BlockRef
	safe      eth.L2BlockRef
}

func readEngineLabels(el *dsl.L2ELNode) engineLabels {
	return engineLabels{
		finalized: el.BlockRefByLabel(eth.Finalized),
		head:      el.BlockRefByLabel(eth.Unsafe),
		safe:      el.BlockRefByLabel(eth.Safe),
	}
}
