package op_reth_withdrawals_root_persistence

import (
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/stretchr/testify/require"
)

const (
	// catchUpAttempts bounds, in 2s polls, how long a node may trail a height the whole system
	// has to reach before a sweep starts.
	catchUpAttempts = 30
	// l2BlockTimeSeconds keeps the runs short: the run length is a block count, so halving the
	// block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so the persistence schedule does not depend on
	// reth's defaults.
	memoryBlockBufferTarget = 5
	// proofWindow lets a node without proofs history answer eth_getProof for every block of the
	// run from reth's own historical state; reth's default window of zero rejects them all.
	proofWindow = 1_000_000
	// senderFundsEther pays for one sender's withdrawals across the whole run.
	senderFundsEther = 10
	// withdrawersPerBlock is how many accounts initiate a withdrawal in every block.
	withdrawersPerBlock = 4
)

// withdrawalAmount is what every withdrawal moves to the L2ToL1MessagePasser. A plain transfer
// to the message passer initiates a withdrawal through its receive function.
var withdrawalAmount = eth.OneHundredthEther

const (
	gethFollowerKey      = "geth-follower"
	proofsFollowerKey    = "proofs-follower"
	referenceFollowerKey = "reference-follower"
	sequencerKey         = "sequencer"
)

// withdrawalsRootCase is one cell of the matrix the chain runs under.
type withdrawalsRootCase struct {
	persistenceThreshold uint64
	stateMaskingBlocks   uint64
}

func (c withdrawalsRootCase) name() string {
	return fmt.Sprintf("threshold_%d-masking_%d", c.persistenceThreshold, c.stateMaskingBlocks)
}

// engineArgs pins reth's persistence schedule. Every op-reth node gets them, so each computes the
// storage roots under the same schedule.
func (c withdrawalsRootCase) engineArgs() []string {
	args := []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
	}
	if c.stateMaskingBlocks != 0 {
		args = append(args, fmt.Sprintf("--engine.num-state-masking-blocks=%d", c.stateMaskingBlocks))
	}
	return args
}

// maxBlocksInMemory is the most blocks reth holds unpersisted. Reth persists once more than the
// threshold of blocks sit in memory and keeps the last memoryBlockBufferTarget of them, and the
// save runs in the background while more blocks arrive; the buffer target is the slack this bound
// allows for them. A block is persisted once this many newer blocks exist.
func (c withdrawalsRootCase) maxBlocksInMemory() uint64 {
	return c.persistenceThreshold + memoryBlockBufferTarget
}

// persistencePeriod is how many blocks apart reth starts persistence runs once it is in its
// steady state, and so how many blocks each run persists: 3 at threshold 7 and 46 at threshold
// 50.
func (c withdrawalsRootCase) persistencePeriod() uint64 {
	return c.persistenceThreshold + 1 - memoryBlockBufferTarget
}

// runBlocks is how many blocks the workload initiates withdrawals in: twice the most blocks reth
// holds in memory, so the run crosses a persistence boundary wherever it starts and a sweep right
// after it covers blocks reth persisted during the run as well as blocks it still holds in memory.
func (c withdrawalsRootCase) runBlocks() uint64 {
	return 2 * c.maxBlocksInMemory()
}

func withdrawalsRootMatrix() []withdrawalsRootCase {
	var cases []withdrawalsRootCase
	for _, persistenceThreshold := range []uint64{7, 50} {
		for _, stateMaskingBlocks := range []uint64{0, 30} {
			cases = append(cases, withdrawalsRootCase{
				persistenceThreshold: persistenceThreshold,
				stateMaskingBlocks:   stateMaskingBlocks,
			})
		}
	}
	return cases
}

type withdrawalsRootSystem struct {
	BlockTime time.Duration
	FunderL2  *dsl.FunderEOA
	// GethFollower serves eth_getProof from its own archive state.
	GethFollower *dsl.L2ELNode
	// ProofsFollower is an op-reth node running proofs history, checked against ReferenceFollower.
	ProofsFollower *dsl.L2ELNode
	// ReferenceFollower is an op-reth node without proofs history that serves eth_getProof from
	// reth's own historical state, so it is independent of the proofs-history store.
	ReferenceFollower *dsl.L2ELNode
	// Sequencer is an op-reth node without proofs history that, like ReferenceFollower, serves
	// eth_getProof from reth's own historical state.
	Sequencer *dsl.L2ELNode
}

func (sys *withdrawalsRootSystem) followers() []*dsl.L2ELNode {
	return []*dsl.L2ELNode{sys.ReferenceFollower, sys.ProofsFollower, sys.GethFollower}
}

func (sys *withdrawalsRootSystem) nodes() []*dsl.L2ELNode {
	return append([]*dsl.L2ELNode{sys.Sequencer}, sys.followers()...)
}

// newWithdrawalsRootSystem starts an op-reth sequencer, an op-reth follower without proofs
// history, an op-reth follower with proofs history and an op-geth follower, every op-reth node on
// the case's persistence schedule, on a chain with Isthmus active.
func newWithdrawalsRootSystem(t devtest.T, c withdrawalsRootCase) *withdrawalsRootSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	withoutProofsHistory := []sysgo.OpRethOption{
		sysgo.OpRethWithoutProofsHistory(),
		sysgo.OpRethWithEthProofWindow(proofWindow),
		sysgo.OpRethWithExtraArgs(c.engineArgs()...),
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
				OpRethOpts:  withoutProofsHistory,
			},
			{
				CLKey:      referenceFollowerKey,
				CLKind:     clKind,
				ELKey:      referenceFollowerKey,
				ELKind:     sysgo.MixedL2ELOpReth,
				OpRethOpts: withoutProofsHistory,
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
	t.Require().Len(frontends.Nodes, 4, "expected the sequencer and three followers")
	t.Require().True(frontends.L2Network.IsForkActive(forks.Isthmus),
		"Isthmus must be active: before it the withdrawalsRoot is not the message passer's storage root")

	sys := &withdrawalsRootSystem{
		BlockTime: time.Duration(frontends.L2Network.Escape().RollupConfig().BlockTime) * time.Second,
		FunderL2:  frontends.FunderL2,
	}
	for _, node := range frontends.Nodes {
		switch node.Spec.ELKey {
		case sequencerKey:
			sys.Sequencer = node.EL
		case referenceFollowerKey:
			sys.ReferenceFollower = node.EL
		case proofsFollowerKey:
			sys.ProofsFollower = node.EL
		case gethFollowerKey:
			sys.GethFollower = node.EL
		}
	}
	for _, node := range sys.nodes() {
		t.Require().NotNil(node, "missing a node of the system")
	}
	return sys
}

// withdrawalRun is where a workload's withdrawals landed: the inclusive range of blocks from the
// first withdrawal to the last, how many of those blocks carry one, and the most consecutive
// blocks of the range without one.
type withdrawalRun struct {
	blocksWithWithdrawals uint64
	first                 uint64
	last                  uint64
	longestGap            uint64
}

func (r withdrawalRun) blocks() uint64 {
	return r.last - r.first + 1
}

// initiateWithdrawalsEveryBlock submits one withdrawal from each sender to the sequencer, back to
// back, in each of the next blocks consecutive blocks. A round waits for the next block rather
// than for its receipts, so the next round is in the pool while the block after that is built,
// and the receipts are collected once the run ends.
func initiateWithdrawalsEveryBlock(t devtest.T, sequencer *dsl.L2ELNode, senders []*dsl.EOA, blocks uint64) withdrawalRun {
	t.Logger().Info("Initiating withdrawals in every block", "blocks", blocks, "senders", len(senders), "amount", withdrawalAmount)
	txs := make([]*txplan.PlannedTx, 0, blocks*uint64(len(senders)))
	for range blocks {
		for _, sender := range senders {
			tx := txplan.NewPlannedTx(sender.PlanTransfer(predeploys.L2ToL1MessagePasserAddr, withdrawalAmount))
			_, err := tx.Submitted.Eval(t.Ctx())
			t.Require().NoErrorf(err, "failed to submit a withdrawal from %s", sender)
			txs = append(txs, tx)
		}
		sequencer.WaitForBlock()
	}

	withdrawalsPerBlock := make(map[uint64]int)
	for i, tx := range txs {
		_, err := tx.Success.Eval(t.Ctx())
		t.Require().NoErrorf(err, "withdrawal %d (round %d, sender %s) was not included successfully", i, i/len(senders), senders[i%len(senders)])
		withdrawalsPerBlock[bigs.Uint64Strict(tx.Included.Value().BlockNumber)]++
	}
	run := summarizeRun(withdrawalsPerBlock)
	t.Logger().Info("Withdrawals landed", "first", run.first, "last", run.last, "blocks", run.blocks(),
		"blocks_with_withdrawals", run.blocksWithWithdrawals, "longest_gap", run.longestGap, "withdrawals", len(txs))
	return run
}

func summarizeRun(withdrawalsPerBlock map[uint64]int) withdrawalRun {
	blocks := slices.Sorted(maps.Keys(withdrawalsPerBlock))
	run := withdrawalRun{blocksWithWithdrawals: uint64(len(blocks)), first: blocks[0], last: blocks[len(blocks)-1]}
	for i := 1; i < len(blocks); i++ {
		run.longestGap = max(run.longestGap, blocks[i]-blocks[i-1]-1)
	}
	return run
}

// verifyWithdrawalsRootsAgree waits for every node to reach everyNodeAt, then requires, at every
// block of the run, the sequencer, the reference follower and the op-geth follower to hold the
// same header and each to serve the header's withdrawalsRoot as the L2ToL1MessagePasser storage
// root through eth_getProof, and the proofs follower to hold the reference follower's chain and
// to serve the same message passer proofs. The newest blocks are checked first, since they leave
// reth's memory soonest.
func verifyWithdrawalsRootsAgree(t devtest.T, sys *withdrawalsRootSystem, run withdrawalRun, everyNodeAt uint64, attempts int, phase string) {
	t.Logger().Info("Verifying withdrawals roots across nodes", "phase", phase, "first", run.first, "last", run.last, "every_node_at", everyNodeAt)
	for _, node := range sys.nodes() {
		node.Reached(eth.Unsafe, everyNodeAt, attempts)
	}
	for i := range run.blocks() {
		block := run.last - i
		expected := sys.Sequencer.VerifyWithdrawalsRootAt(block)
		for _, node := range []*dsl.L2ELNode{sys.ReferenceFollower, sys.GethFollower} {
			header := node.VerifyWithdrawalsRootAt(block)
			t.Require().Equalf(*expected.WithdrawalsRoot(), *header.WithdrawalsRoot(),
				"block %d: %s holds withdrawalsRoot %s, the sequencer %s", block, node, *header.WithdrawalsRoot(), *expected.WithdrawalsRoot())
			t.Require().Equalf(expected.Hash(), header.Hash(), "block %d: %s holds %s, the sequencer %s", block, node, header.Hash(), expected.Hash())
		}
	}
	sys.ProofsFollower.VerifyHistoricalProofsMatch(sys.ReferenceFollower, func(opts *dsl.HistoricalProofsOpts) {
		opts.FirstBlock = run.first
		opts.LastBlock = run.last
	})
	t.Logger().Info("Withdrawals roots agree across nodes", "phase", phase, "first", run.first, "last", run.last)
}

// TestWithdrawalsRootMatchesMessagePasserStorageRootAcrossPersistenceRuns initiates a withdrawal
// from several accounts in every block for twice as many blocks as reth holds in memory, on an
// explicit persistence schedule, and requires every block of the run to carry as its
// withdrawalsRoot the L2ToL1MessagePasser storage root that eth_getProof serves on the sequencer,
// on an op-reth follower reading its own historical state, on an op-geth follower and on a
// proofs-history follower, with every node on the sequencer's chain: once right after the run,
// while reth still holds the newest blocks of it in memory, and again once every node has as many
// newer blocks as reth holds in memory, so every block of the run has been persisted.
func TestWithdrawalsRootMatchesMessagePasserStorageRootAcrossPersistenceRuns(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range withdrawalsRootMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			sysgo.SkipOnOpGeth(t, "the withdrawals root is checked across op-reth's persistence runs")
			if c.stateMaskingBlocks != 0 {
				t.Skip("state masking is compiled out of reth 2.5.2 on develop and #23046 rejects a nonzero --engine.num-state-masking-blocks; re-enable with ethereum-optimism/protocol-team#457")
			}
			sys := newWithdrawalsRootSystem(t, c)
			senders := sys.FunderL2.NewFundedEOAs(withdrawersPerBlock, eth.Ether(senderFundsEther))
			sys.Sequencer.CaughtUpToSchedule(sys.BlockTime)

			run := initiateWithdrawalsEveryBlock(t, sys.Sequencer, senders, c.runBlocks())
			t.Require().Greaterf(run.blocks(), c.maxBlocksInMemory(),
				"the withdrawals span %d blocks, not more than the %d reth may hold in memory, so the run crossed no persistence boundary",
				run.blocks(), c.maxBlocksInMemory())
			t.Require().Lessf(run.longestGap, c.persistencePeriod(),
				"%d consecutive blocks of the run carry no withdrawal, a whole persistence run of %d blocks", run.longestGap, c.persistencePeriod())
			t.Require().GreaterOrEqualf(run.blocksWithWithdrawals*5, run.blocks()*4,
				"only %d of the run's %d blocks carry a withdrawal; the workload must reach nearly every block", run.blocksWithWithdrawals, run.blocks())

			verifyWithdrawalsRootsAgree(t, sys, run, run.last, catchUpAttempts, "run just ended")
			verifyWithdrawalsRootsAgree(t, sys, run, run.last+c.maxBlocksInMemory(), int(c.maxBlocksInMemory())+catchUpAttempts, "run persisted")
		})
	}
}

func TestSummarizeRunFindsRangeAndLongestGap(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		withdrawalsPerBlock map[uint64]int
		want                withdrawalRun
	}{
		{name: "single block", withdrawalsPerBlock: map[uint64]int{12: 4}, want: withdrawalRun{blocksWithWithdrawals: 1, first: 12, last: 12}},
		{name: "contiguous", withdrawalsPerBlock: map[uint64]int{10: 4, 11: 8, 12: 4}, want: withdrawalRun{blocksWithWithdrawals: 3, first: 10, last: 12}},
		{name: "single gaps", withdrawalsPerBlock: map[uint64]int{10: 4, 12: 4, 14: 4}, want: withdrawalRun{blocksWithWithdrawals: 3, first: 10, last: 14, longestGap: 1}},
		{name: "gap of three", withdrawalsPerBlock: map[uint64]int{10: 4, 11: 4, 15: 4, 17: 4}, want: withdrawalRun{blocksWithWithdrawals: 4, first: 10, last: 17, longestGap: 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, summarizeRun(tc.withdrawalsPerBlock))
		})
	}
}
