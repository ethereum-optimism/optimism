package op_reth_sequencer_crash

import (
	"fmt"
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// batcherPollInterval is how often the batcher loads the sequencer's new blocks. Below the
	// block time, so when the sequencer's consensus client stops the batcher has loaded at least
	// the block before the head, and flushing it is known to post that block.
	batcherPollInterval = 250 * time.Millisecond
	// l2BlockTimeSeconds keeps the runs short: the kill point is a block height, so halving the
	// block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so the number of blocks a kill loses does not
	// depend on reth's defaults.
	memoryBlockBufferTarget = 5
	// postCrashTrafficBlocks is how far the chain is driven once the sequencer is back, so the
	// proofs sweep also covers blocks built after the crash.
	postCrashTrafficBlocks = 10
	// proofsSweepLeadIn is how many blocks below the persisted tip the proofs sweep starts, so it
	// covers blocks the crash left in place as well as every block it lost.
	proofsSweepLeadIn = 5

	// The attempt counts below bound waits in 2s polls.

	// batcherLandingAttempts bounds how long the blocks the batcher had loaded when the sequencer
	// died may take to reach L1 and be derived by the followers.
	batcherLandingAttempts = 30
	// followerReorgAttempts bounds how long a follower may take to reorg to the rebuilt chain. A
	// follower running op-node drops a gossiped block at or below its unsafe head, so it only
	// reorgs once the batcher has posted the rebuilt chain and derivation reaches it.
	followerReorgAttempts = 90
	// resumeAttempts bounds how long the restarted sequencer may take to sequence again: to
	// replace the block that held the workload's last transaction, or to build past the head the
	// kill cut short, which includes deriving back from L1 whatever the batcher had posted.
	resumeAttempts = 60
	// resyncAttempts bounds how long the followers may take to catch up with the sequencer.
	resyncAttempts = 60
	// safeAttempts bounds how long the batcher and derivation may take to make a block safe: the
	// posted blocks on the restarted sequencer, and the chain past the head the kill cut short on
	// every node.
	safeAttempts = 90
)

const (
	gethFollowerKey   = "geth-follower"
	proofsFollowerKey = "proofs-follower"
	sequencerKey      = "sequencer"
)

// crashCase is one cell of the matrix the sequencer is killed under.
type crashCase struct {
	// killTargetBlock is the height the workload drives the chain to before the kill.
	killTargetBlock      uint64
	persistenceThreshold uint64
}

func (c crashCase) name() string {
	return fmt.Sprintf("threshold_%d", c.persistenceThreshold)
}

// engineArgs pins reth's persistence schedule, which decides how many blocks an unclean stop
// loses. Both op-reth nodes get them, so the follower absorbs the reorg under the same schedule.
func (c crashCase) engineArgs() []string {
	return []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
	}
}

// maxLostBlocks is the most blocks a kill may lose. Reth persists once more than the threshold of
// blocks sit in memory, so at most the threshold plus one are unpersisted when a save starts, and
// the save runs in the background while more blocks arrive; the memory buffer target, which a
// save leaves in memory, is the slack this bound allows for them.
func (c crashCase) maxLostBlocks() uint64 {
	return c.persistenceThreshold + memoryBlockBufferTarget
}

// crashMatrix sets each kill where reth's datadir trails the head the most under its threshold.
// Reth persists once more than the threshold of blocks sit in memory and keeps the last
// memoryBlockBufferTarget of them: at threshold 7 it persists every three blocks from block 8, so
// the datadir trails the head by 5 to 7 blocks wherever the kill lands and the kill height only
// has to clear the workload's setup; at threshold 50 it persists at blocks 51 and 97, so a kill
// around block 90 loses the blocks above 46.
func crashMatrix() []crashCase {
	return []crashCase{
		{killTargetBlock: 60, persistenceThreshold: 7},
		{killTargetBlock: 90, persistenceThreshold: 50},
	}
}

// crashNode is one L2 node of the system: its execution layer and the consensus client driving it.
type crashNode struct {
	CL *dsl.L2CLNode
	EL *dsl.L2ELNode
}

type crashSystem struct {
	Batcher        *dsl.L2Batcher
	FunderL2       *dsl.FunderEOA
	GethFollower   crashNode
	ProofsFollower crashNode
	Sequencer      crashNode
}

func (sys *crashSystem) followers() []crashNode {
	return []crashNode{sys.ProofsFollower, sys.GethFollower}
}

func (sys *crashSystem) nodes() []crashNode {
	return append([]crashNode{sys.Sequencer}, sys.followers()...)
}

// newCrashSystem starts an op-reth sequencer without proofs history, an op-reth follower running
// proofs history and an op-geth follower, which serves eth_getProof from its own archive state and
// is the independent reference the proofs follower is checked against. Both op-reth nodes run the
// case's persistence schedule.
func newCrashSystem(t devtest.T, c crashCase) *crashSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		BatcherOptions: []sysgo.BatcherOption{func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			cfg.PollInterval = batcherPollInterval
		}},
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				CLKey:       sequencerKey,
				CLKind:      clKind,
				ELKey:       sequencerKey,
				ELKind:      sysgo.MixedL2ELOpReth,
				IsSequencer: true,
				OpRethOpts: []sysgo.OpRethOption{
					sysgo.OpRethWithoutProofsHistory(),
					sysgo.OpRethWithExtraArgs(c.engineArgs()...),
				},
			},
			{
				CLKey:      proofsFollowerKey,
				CLKind:     clKind,
				ELKey:      proofsFollowerKey,
				ELKind:     sysgo.MixedL2ELOpReth,
				OpRethOpts: []sysgo.OpRethOption{sysgo.OpRethWithExtraArgs(c.engineArgs()...)},
			},
			{
				CLKey:  gethFollowerKey,
				CLKind: clKind,
				ELKey:  gethFollowerKey,
				ELKind: sysgo.MixedL2ELOpGeth,
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 3, "expected the sequencer and both followers")

	sys := &crashSystem{Batcher: frontends.L2Batcher, FunderL2: frontends.FunderL2}
	for _, node := range frontends.Nodes {
		switch node.Spec.ELKey {
		case sequencerKey:
			sys.Sequencer = crashNode{CL: node.CL, EL: node.EL}
		case proofsFollowerKey:
			sys.ProofsFollower = crashNode{CL: node.CL, EL: node.EL}
		case gethFollowerKey:
			sys.GethFollower = crashNode{CL: node.CL, EL: node.EL}
		}
	}
	for _, node := range sys.nodes() {
		t.Require().NotNil(node.EL, "missing a node of the system")
	}
	return sys
}

func skipUnsupported(t devtest.T) {
	sysgo.SkipOnOpGeth(t, "the crash exercises op-reth's persistence threshold")
}

// workload changes state in nearly every block: alice and bob transfer back and forth and bob
// writes to a contract. A chain rebuilt after a kill cannot repeat a block that held one of these
// transactions, since the kill empties the sequencer's transaction pool.
type workload struct {
	alice    *dsl.EOA
	bob      *dsl.EOA
	recorder common.Address
	// setupBlock is the block that completed the funding and the contract deployment; a rebuilt
	// chain must start at or above it or the workload's accounts no longer exist.
	setupBlock uint64
	t          devtest.T
}

func newWorkload(t devtest.T, funder *dsl.FunderEOA) *workload {
	w := &workload{
		alice: funder.NewFundedEOA(eth.OneEther),
		bob:   funder.NewFundedEOA(eth.OneEther),
		t:     t,
	}
	w.recorder = w.alice.DeployBlockRecorder()
	w.setupBlock = includedBlock(t, w.alice.Transact(w.alice.Plan(), txplan.WithTo(&w.recorder)))
	return w
}

// accounts are the accounts the proofs sweep proves at every block.
func (w *workload) accounts() []dsl.ProofAccount {
	return []dsl.ProofAccount{
		{Address: w.alice.Address()},
		{Address: w.bob.Address()},
		{Address: w.recorder, StorageKeys: []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}},
	}
}

// driveUntilBlock sends transactions one after another until one lands in a block at or past
// target, and returns that block.
func (w *workload) driveUntilBlock(target uint64) uint64 {
	for {
		w.alice.Transfer(w.bob.Address(), eth.OneHundredthEther)
		w.bob.Transfer(w.alice.Address(), eth.OneHundredthEther)
		block := includedBlock(w.t, w.bob.Transact(w.bob.Plan(), txplan.WithTo(&w.recorder)))
		if block >= target {
			return block
		}
	}
}

func includedBlock(t devtest.T, tx *txplan.PlannedTx) uint64 {
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "transaction must be included")
	return bigs.Uint64Strict(receipt.BlockNumber)
}

// crashOutcome is what the kill cost the sequencer: the head it had when it was killed, the tip
// its datadir held when it came back, the blocks in between as the followers still hold them, and
// among those the block that held the workload's last transaction, which no rebuilt chain can
// repeat.
type crashOutcome struct {
	headAtKill        eth.L2BlockRef
	lastWorkloadBlock eth.L2BlockRef
	lostBlocks        []eth.L2BlockRef
	persistedTip      eth.L2BlockRef
}

// killSequencer stops the sequencer's consensus client, reads the head it leaves behind, waits
// for the followers to hold it too, and kills the sequencer's op-reth.
func killSequencer(t devtest.T, sys *crashSystem) eth.L2BlockRef {
	sys.Sequencer.CL.Stop()
	headAtKill := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe)
	for _, follower := range sys.followers() {
		follower.EL.Matched(sys.Sequencer.EL, safety.LocalUnsafe, resyncAttempts)
	}
	t.Logger().Info("Killing the sequencer's op-reth", "head", headAtKill)
	sys.Sequencer.EL.StopUngracefully()
	return headAtKill
}

// reopenSequencerDatadir brings the sequencer's op-reth back on its datadir and reads the tip it
// persisted, which nothing moves until its consensus client is back. The tip must trail the head
// by at most the persistence threshold plus the memory buffer, must lie at or above the workload's
// setup or a chain rebuilt from it lacks the workload's accounts, and must lie below the block
// that held the workload's last transaction or nothing tells a rebuilt chain from the lost one.
func reopenSequencerDatadir(t devtest.T, sys *crashSystem, c crashCase, w *workload, headAtKill eth.L2BlockRef, lastWorkloadBlock uint64) crashOutcome {
	sys.Sequencer.EL.Start()
	sys.Sequencer.EL.WaitForOnline()
	persistedTip := sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Lessf(persistedTip.Number, headAtKill.Number,
		"the sequencer's op-reth came back at %s, not below the head %s it was killed at, so the kill lost nothing", persistedTip, headAtKill)
	lostBlockCount := headAtKill.Number - persistedTip.Number
	t.Logger().Info("Sequencer op-reth is back on its datadir", "persisted_tip", persistedTip, "head_at_kill", headAtKill,
		"lost_blocks", lostBlockCount, "max_lost_blocks", c.maxLostBlocks(), "last_workload_block", lastWorkloadBlock)
	t.Require().LessOrEqualf(lostBlockCount, c.maxLostBlocks(),
		"the kill lost %d blocks, more than the persistence threshold %d plus the memory buffer %d",
		lostBlockCount, c.persistenceThreshold, memoryBlockBufferTarget)
	t.Require().LessOrEqualf(w.setupBlock, persistedTip.Number,
		"the workload's funding and contract (block %d) must survive the crash, but the persisted tip is %d", w.setupBlock, persistedTip.Number)
	t.Require().Greaterf(lastWorkloadBlock, persistedTip.Number,
		"the block %d that held the workload's last transaction must be among the lost blocks, but the persisted tip is %d", lastWorkloadBlock, persistedTip.Number)
	t.Require().Truef(sys.GethFollower.EL.IsCanonical(persistedTip.ID()),
		"the persisted tip %s is not on the chain the followers hold", persistedTip)
	lostBlocks := blockRefsInRange(sys.GethFollower.EL, persistedTip.Number+1, headAtKill.Number)
	return crashOutcome{
		headAtKill:        headAtKill,
		lastWorkloadBlock: lostBlocks[lastWorkloadBlock-persistedTip.Number-1],
		lostBlocks:        lostBlocks,
		persistedTip:      persistedTip,
	}
}

// rejoinSequencer re-peers the sequencer's op-reth with the followers, starts its consensus
// client and re-peers that too, since both peerings died with the processes.
func rejoinSequencer(sys *crashSystem) {
	for _, follower := range sys.followers() {
		sys.Sequencer.EL.PeerWith(follower.EL)
	}
	sys.Sequencer.CL.Start()
	for _, follower := range sys.followers() {
		sys.Sequencer.CL.ConnectPeer(follower.CL)
	}
}

// blockRefsInRange returns the blocks from first to last inclusive as node holds them.
func blockRefsInRange(node *dsl.L2ELNode, first, last uint64) []eth.L2BlockRef {
	refs := make([]eth.L2BlockRef, 0, last-first+1)
	for number := first; number <= last; number++ {
		refs = append(refs, node.BlockRefByNumber(number))
	}
	return refs
}

// keptThrough returns the highest lost block node still holds, or the persisted tip when it holds
// none. The lost blocks a node holds form an unbroken run above the tip, since a child of a
// replaced block cannot stay canonical.
func keptThrough(node *dsl.L2ELNode, outcome crashOutcome) uint64 {
	kept := outcome.persistedTip.Number
	for _, lost := range outcome.lostBlocks {
		if !node.IsCanonical(lost.ID()) {
			break
		}
		kept = lost.Number
	}
	return kept
}

// verifyChainSettled drives the workload past the head the kill cut short and requires every
// node to settle on one chain that is safe past that head, that keeps the lost blocks through a
// height between keptAtLeast and keptAtMost and replaces the rest, and whose proofs the proofs
// follower serves like the op-geth follower from below the persisted tip to the last block driven.
func verifyChainSettled(t devtest.T, sys *crashSystem, w *workload, outcome crashOutcome, keptAtLeast, keptAtMost uint64) {
	lastBlock := w.driveUntilBlock(outcome.headAtKill.Number + postCrashTrafficBlocks)
	for _, follower := range sys.followers() {
		follower.EL.InSync(sys.Sequencer.EL, safety.LocalUnsafe, resyncAttempts)
	}
	for _, node := range sys.nodes() {
		node.EL.Reached(eth.Safe, outcome.headAtKill.Number+1, safeAttempts)
		kept := keptThrough(node.EL, outcome)
		t.Logger().Info("Lost blocks settled", "node", node.EL, "persisted_tip", outcome.persistedTip.Number,
			"kept_through", kept, "head_at_kill", outcome.headAtKill.Number)
		t.Require().GreaterOrEqualf(kept, keptAtLeast,
			"%s keeps the lost blocks only through %d, but the batcher had posted them through %d before the sequencer came back", node.EL, kept, keptAtLeast)
		t.Require().LessOrEqualf(kept, keptAtMost,
			"%s keeps the lost blocks through %d, but at most those through %d could come back", node.EL, kept, keptAtMost)
	}
	sweepStart := uint64(0)
	if outcome.persistedTip.Number > proofsSweepLeadIn {
		sweepStart = outcome.persistedTip.Number - proofsSweepLeadIn
	}
	sys.ProofsFollower.EL.VerifyHistoricalProofsMatch(sys.GethFollower.EL, func(opts *dsl.HistoricalProofsOpts) {
		opts.Accounts = w.accounts()
		opts.FirstBlock = sweepStart
		opts.LastBlock = lastBlock
	})
}

// TestSequencerOpRethCrashKeepsPostedBlocksWithBatcherRunning kills the sequencer's op-reth with
// blocks it has not persisted, while the batcher keeps running, and requires the chain to settle
// on the blocks the batcher had posted: the restarted sequencer derives them back from L1 and
// replaces at most the ones the batcher had not posted, both followers keep up, the batcher keeps
// submitting until the chain past the crash is safe, and the proofs follower's proofs match the
// op-geth follower's across the crash. The batcher's data lands on L1 before the sequencer is
// back, as it would behind any real restart.
func TestSequencerOpRethCrashKeepsPostedBlocksWithBatcherRunning(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range crashMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			sysgo.SkipOnKonaNode(t, "fails on develop under kona-node (#23386): at threshold 50 the restarted sequencer's "+
				"build loop fails every forkchoice update with `error code -38002: Invalid forkchoice state` and reth's safe "+
				"label stays unresolvable, so the wait for the posted blocks to be safe fails with `failed to determine "+
				"L2BlockRef of safe, could not get payload: not found`; at threshold 7 the sequencer resumes, but the "+
				"op-geth follower stops inside the blocks the sequencer rebuilt before derivation brought the posted ones "+
				"back and never catches up: `nodes not in sync: slower head at 78, must reach 106: unsafe`")
			sys := newCrashSystem(t, c)
			w := newWorkload(t, sys.FunderL2)
			lastWorkloadBlock := w.driveUntilBlock(c.killTargetBlock)

			headAtKill := killSequencer(t, sys)
			outcome := reopenSequencerDatadir(t, sys, c, w, headAtKill, lastWorkloadBlock)
			// The batcher checks its open channel's timeout only when its rollup node answers, so
			// with the sequencer's consensus client down only a flush posts the blocks it loaded.
			sys.Batcher.Flush()
			sys.GethFollower.EL.Reached(eth.Safe, outcome.headAtKill.Number-1, batcherLandingAttempts)
			// The flushed data may still be landing, so this is a floor on what comes back.
			postedAtLeast := sys.GethFollower.EL.BlockRefByLabel(eth.Safe).Number
			rejoinSequencer(sys)

			sys.Sequencer.EL.Reached(eth.Safe, postedAtLeast, safeAttempts)
			sys.Sequencer.EL.Reached(eth.Unsafe, outcome.headAtKill.Number+1, resumeAttempts)
			verifyChainSettled(t, sys, w, outcome, postedAtLeast, outcome.headAtKill.Number)
		})
	}
}

// TestSequencerOpRethCrashRebuildsLostBlocksWithBatcherStopped kills the sequencer's op-reth with
// blocks it has not persisted and none of them on L1, since the batcher was stopped before the
// workload began, and requires the restarted sequencer to rebuild the chain from the tip its
// datadir held, both followers to reorg to the rebuilt chain once the restarted batcher posts it,
// the chain past the crash to become safe, and the proofs follower's proofs to match the op-geth
// follower's across the reorg.
func TestSequencerOpRethCrashRebuildsLostBlocksWithBatcherStopped(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range crashMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			sys := newCrashSystem(t, c)
			sys.Batcher.Stop()
			w := newWorkload(t, sys.FunderL2)
			lastWorkloadBlock := w.driveUntilBlock(c.killTargetBlock)

			headAtKill := killSequencer(t, sys)
			outcome := reopenSequencerDatadir(t, sys, c, w, headAtKill, lastWorkloadBlock)
			rejoinSequencer(sys)

			// A lost block without a workload transaction can come back identical, so the chain
			// is known to have diverged by the block that held the last one.
			sys.Sequencer.EL.ReorgTriggered(outcome.lastWorkloadBlock, resumeAttempts)
			sys.Batcher.Start()
			for _, follower := range sys.followers() {
				follower.EL.ReorgTriggered(outcome.lastWorkloadBlock, followerReorgAttempts)
			}
			verifyChainSettled(t, sys, w, outcome, outcome.persistedTip.Number, outcome.lastWorkloadBlock.Number-1)
		})
	}
}
