package op_reth_withdrawals_root_persistence

import (
	"context"
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

const (
	// catchUpAttempts bounds, in 2s polls, how long a node may trail a height the whole system
	// has to reach before a sweep starts.
	catchUpAttempts = 30
	// l2BlockTimeSeconds keeps the runs short: the run length is a block count, so halving the
	// block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// persistenceSlack is how much longer than the block schedule implies a node may take to
	// persist a block, for followers that trail and saves that fall behind.
	persistenceSlack = time.Minute
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
}

func (c withdrawalsRootCase) String() string {
	return fmt.Sprintf("threshold_%d", c.persistenceThreshold)
}

// persistenceSchedule pins reth's persistence schedule. Every op-reth node gets it, so each
// computes the storage roots under the same schedule.
func (c withdrawalsRootCase) persistenceSchedule() sysgo.OpRethPersistenceSchedule {
	return sysgo.OpRethPersistenceSchedule{
		MemoryBlockBufferTarget: sysgo.OpRethPinnedMemoryBlockBufferTarget,
		PersistenceThreshold:    c.persistenceThreshold,
	}
}

// maxBlocksInMemory is the most blocks reth holds unpersisted while its saves keep up with the
// chain. Reth persists once more than the threshold of blocks sit in memory and keeps the newest
// of them up to the memory block buffer target, and the save runs in the background while more
// blocks arrive; the buffer target is the slack this bound allows for them. A save that falls
// behind lets reth hold more.
func (c withdrawalsRootCase) maxBlocksInMemory() uint64 {
	schedule := c.persistenceSchedule()
	return schedule.PersistenceThreshold + schedule.MemoryBlockBufferTarget
}

// runBlocks is how many blocks the workload initiates withdrawals in: twice the most blocks reth
// holds in memory, so the withdrawals included before sequencing stops span more blocks than reth
// holds in memory even when the last rounds are still in the pool.
func (c withdrawalsRootCase) runBlocks() uint64 {
	return 2 * c.maxBlocksInMemory()
}

// withdrawalsRootMatrix has no --engine.num-state-masking-blocks dimension: op-reth rejects a
// nonzero value until ethereum-optimism/protocol-team#457.
func withdrawalsRootMatrix() []withdrawalsRootCase {
	return []withdrawalsRootCase{{persistenceThreshold: 7}, {persistenceThreshold: 50}}
}

type withdrawalsRootSystem struct {
	BlockTime time.Duration
	FunderL2  *dsl.FunderEOA
	// GethFollower serves eth_getProof from its own archive state.
	GethFollower *dsl.L2ELNode
	// PersistedBlocks follows the persisted tip of every op-reth node.
	PersistedBlocks []*dsl.PersistedBlockTracker
	// ProofsFollower is an op-reth node running proofs history, checked against ReferenceFollower.
	ProofsFollower *dsl.L2ELNode
	// ReferenceFollower is an op-reth node without proofs history that serves eth_getProof from
	// reth's own historical state, so it is independent of the proofs-history store.
	ReferenceFollower *dsl.L2ELNode
	// Sequencer is an op-reth node without proofs history that, like ReferenceFollower, serves
	// eth_getProof from reth's own historical state.
	Sequencer   *dsl.L2ELNode
	SequencerCL *dsl.L2CLNode
}

func (sys *withdrawalsRootSystem) followers() []*dsl.L2ELNode {
	return []*dsl.L2ELNode{sys.ReferenceFollower, sys.ProofsFollower, sys.GethFollower}
}

func (sys *withdrawalsRootSystem) nodes() []*dsl.L2ELNode {
	return append([]*dsl.L2ELNode{sys.Sequencer}, sys.followers()...)
}

func (sys *withdrawalsRootSystem) opRethNodes() []*dsl.L2ELNode {
	return []*dsl.L2ELNode{sys.Sequencer, sys.ReferenceFollower, sys.ProofsFollower}
}

// newWithdrawalsRootSystem starts an op-reth sequencer, an op-reth follower without proofs
// history, an op-reth follower with proofs history and an op-geth follower, every op-reth node on
// the case's persistence schedule, on a chain with Isthmus active, and follows the persisted tip
// of every op-reth node.
func newWithdrawalsRootSystem(t devtest.T, c withdrawalsRootCase) *withdrawalsRootSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	withoutProofsHistory := []sysgo.OpRethOption{
		sysgo.OpRethWithoutProofsHistory(),
		sysgo.OpRethWithEthProofWindow(proofWindow),
		sysgo.OpRethWithPersistenceSchedule(c.persistenceSchedule()),
	}
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		// op-geth refuses Karst blocks, which the default dev genesis activates.
		DeployerOptions: []sysgo.DeployerOption{
			sysgo.WithJovianAtGenesis,
			sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds),
		},
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
				OpRethOpts: []sysgo.OpRethOption{sysgo.OpRethWithPersistenceSchedule(c.persistenceSchedule())},
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
			sys.SequencerCL = node.CL
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
	t.Require().NotNil(sys.SequencerCL, "missing the sequencer's consensus client")
	for _, node := range sys.opRethNodes() {
		sys.PersistedBlocks = append(sys.PersistedBlocks, node.TrackPersistedBlock())
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

// initiateWithdrawalsEveryBlock submits one withdrawal from each sender to the sequencer in each of
// the next blocks consecutive blocks, and returns as soon as the sequencer built the block after
// the last round was submitted, without waiting for any withdrawal to be included. Each round is
// submitted as one concurrent batch, so a round reaches the pool one submission round trip after
// the block it follows, however many senders it has. A round waits for the next block rather than
// for its receipts, and the next round is signed during that wait, so signing costs the workload
// no block while it takes less than a block time.
func initiateWithdrawalsEveryBlock(t devtest.T, sequencer *dsl.L2ELNode, senders []*dsl.EOA, blocks uint64) []*txplan.PlannedTx {
	t.Logger().Info("Initiating withdrawals in every block", "blocks", blocks, "senders", len(senders), "amount", withdrawalAmount)
	txs := make([]*txplan.PlannedTx, 0, blocks*uint64(len(senders)))
	round, err := signWithdrawalRound(t.Ctx(), senders)
	t.Require().NoError(err)
	for roundIndex := range blocks {
		submitWithdrawalRound(t, round, senders)
		txs = append(txs, round...)
		var nextRound []*txplan.PlannedTx
		var signing errgroup.Group
		if roundIndex+1 < blocks {
			signing.Go(func() (signErr error) {
				nextRound, signErr = signWithdrawalRound(t.Ctx(), senders)
				return signErr
			})
		}
		sequencer.WaitForBlock()
		t.Require().NoError(signing.Wait())
		round = nextRound
	}
	t.Logger().Info("Submitted every round of withdrawals", "rounds", blocks, "withdrawals", len(txs))
	return txs
}

// signWithdrawalRound signs one withdrawal from each sender, in the order of senders. Each takes
// the sender's pending nonce, so a round signed after the previous one was submitted follows it.
func signWithdrawalRound(ctx context.Context, senders []*dsl.EOA) ([]*txplan.PlannedTx, error) {
	round := make([]*txplan.PlannedTx, len(senders))
	for i, sender := range senders {
		round[i] = txplan.NewPlannedTx(sender.PlanTransfer(predeploys.L2ToL1MessagePasserAddr, withdrawalAmount))
	}
	err := evalWithdrawalRound(ctx, round, senders, "sign", func(ctx context.Context, tx *txplan.PlannedTx) error {
		_, err := tx.Signed.Eval(ctx)
		return err
	})
	return round, err
}

func submitWithdrawalRound(t devtest.T, round []*txplan.PlannedTx, senders []*dsl.EOA) {
	t.Require().NoError(evalWithdrawalRound(t.Ctx(), round, senders, "submit", func(ctx context.Context, tx *txplan.PlannedTx) error {
		_, err := tx.Submitted.Eval(ctx)
		return err
	}))
}

// evalWithdrawalRound runs step on every withdrawal of a round concurrently. The senders differ,
// so no withdrawal of a round depends on another.
func evalWithdrawalRound(ctx context.Context, round []*txplan.PlannedTx, senders []*dsl.EOA, stepName string,
	step func(context.Context, *txplan.PlannedTx) error) error {
	group, groupCtx := errgroup.WithContext(ctx)
	for i, tx := range round {
		sender := senders[i]
		group.Go(func() error {
			if err := step(groupCtx, tx); err != nil {
				return fmt.Errorf("failed to %s a withdrawal from %s: %w", stepName, sender, err)
			}
			return nil
		})
	}
	return group.Wait()
}

// noncesAt returns each sender's nonce in the sequencer's state at block, which counts the
// sender's transactions included up to and including that block.
func noncesAt(t devtest.T, sequencer *dsl.L2ELNode, senders []*dsl.EOA, block eth.L2BlockRef) map[common.Address]uint64 {
	t.Logger().Info("Reading sender nonces", "block", block, "senders", len(senders))
	nonces := make(map[common.Address]uint64, len(senders))
	for _, sender := range senders {
		nonces[sender.Address()] = sequencer.NonceAt(sender.Address(), block.Number)
	}
	t.Logger().Info("Read sender nonces", "block", block, "nonces", nonces)
	return nonces
}

// collectWithdrawalRun requires every withdrawal the sequencer included up to stoppedHead to have
// succeeded and returns where they landed. The withdrawals still in the pool when sequencing
// stopped are not part of the run.
func collectWithdrawalRun(t devtest.T, sequencer *dsl.L2ELNode, stoppedHead eth.L2BlockRef, txs []*txplan.PlannedTx, senders []*dsl.EOA) withdrawalRun {
	includedNonces := noncesAt(t, sequencer, senders, stoppedHead)
	withdrawalsPerBlock := make(map[uint64]int)
	var pending int
	for i, tx := range txs {
		sender := senders[i%len(senders)]
		if tx.Nonce.Value() >= includedNonces[sender.Address()] {
			pending++
			continue
		}
		_, err := tx.Success.Eval(t.Ctx())
		t.Require().NoErrorf(err, "withdrawal %d (round %d, sender %s) was not included successfully", i, i/len(senders), sender)
		withdrawalsPerBlock[bigs.Uint64Strict(tx.Included.Value().BlockNumber)]++
	}
	t.Require().NotEmptyf(withdrawalsPerBlock, "the sequencer included none of the %d withdrawals by block %s", len(txs), stoppedHead)
	run := summarizeRun(withdrawalsPerBlock)
	t.Logger().Info("Withdrawals landed", "first", run.first, "last", run.last, "blocks", run.blocks(),
		"blocks_with_withdrawals", run.blocksWithWithdrawals, "longest_gap", run.longestGap,
		"included", len(txs)-pending, "pending", pending, "stopped_head", stoppedHead.Number)
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

// verifyWithdrawalsRootsAgree requires, at every block of the run, the reference follower and the
// op-geth follower to hold the sequencer's header, the sequencer and the op-geth follower to serve
// its withdrawalsRoot as the L2ToL1MessagePasser storage root through eth_getProof, and the proofs
// follower to hold the reference follower's chain and to serve the same message passer proofs as
// it, whose storage root is the withdrawalsRoot. The newest blocks are checked first: reth persists
// the oldest first, so the newest stay in its memory longest.
func verifyWithdrawalsRootsAgree(t devtest.T, sys *withdrawalsRootSystem, run withdrawalRun, phase string) {
	t.Logger().Info("Verifying withdrawals roots across nodes", "phase", phase, "first", run.first, "last", run.last)
	for i := range run.blocks() {
		block := run.last - i
		expected := sys.Sequencer.VerifyWithdrawalsRootAt(block)
		geth := sys.GethFollower.VerifyWithdrawalsRootAt(block)
		t.Require().Equalf(*expected.WithdrawalsRoot(), *geth.WithdrawalsRoot(),
			"block %d: %s holds withdrawalsRoot %s, the sequencer %s", block, sys.GethFollower, *geth.WithdrawalsRoot(), *expected.WithdrawalsRoot())
		t.Require().Equalf(expected.Hash(), geth.Hash(), "block %d: %s holds %s, the sequencer %s", block, sys.GethFollower, geth.Hash(), expected.Hash())
		reference := sys.ReferenceFollower.BlockRefByNumber(block)
		t.Require().Equalf(expected.Hash(), reference.Hash, "block %d: %s holds %s, the sequencer %s", block, sys.ReferenceFollower, reference.Hash, expected.Hash())
	}
	sys.ProofsFollower.VerifyHistoricalProofsMatch(sys.ReferenceFollower, func(opts *dsl.HistoricalProofsOpts) {
		opts.FirstBlock = run.first
		opts.LastBlock = run.last
	})
	t.Logger().Info("Withdrawals roots agree across nodes", "phase", phase, "first", run.first, "last", run.last)
}

// requireRunEndsInMemoryBuffer requires the run's last withdrawal to be among the newest blocks up
// to the stopped head, as many as the memory block buffer target. Reth never persists past its head minus
// the buffer target, and no node's head passes the stopped head while sequencing is stopped, so
// every op-reth node holds the run's last block in memory until sequencing resumes.
func requireRunEndsInMemoryBuffer(t devtest.T, run withdrawalRun, stoppedHead eth.L2BlockRef) {
	t.Require().Lessf(stoppedHead.Number-run.last, uint64(sysgo.OpRethPinnedMemoryBlockBufferTarget),
		"sequencing stopped at block %d, %d blocks after the run's last withdrawal at %d, so reth may persist the whole run "+
			"while it keeps only the newest %d blocks in memory; the workload must reach the blocks right before the stop",
		stoppedHead.Number, stoppedHead.Number-run.last, run.last, sysgo.OpRethPinnedMemoryBlockBufferTarget)
}

// awaitOldestBlocksPersisted waits, with sequencing stopped, for every op-reth node to report
// the run's first block persisted, so a sweep started afterwards reads the oldest blocks of the
// run from disk. A node whose head is the stopped head persists at least up to that head minus
// the persistence threshold, which lies inside any run longer than reth holds in memory.
func awaitOldestBlocksPersisted(t devtest.T, sys *withdrawalsRootSystem, run withdrawalRun) {
	for _, persisted := range sys.PersistedBlocks {
		tip := persisted.AwaitPersisted(run.first, persistenceSlack)
		t.Logger().Info("Sweep will read the oldest blocks of the run from disk",
			"node", persisted, "persisted_blocks", min(tip.Number, run.last)-run.first+1, "persisted_tip", tip.Number)
	}
}

// requireRunEndNotReportedPersisted cross-checks, after a sweep made with sequencing stopped, that
// no op-reth node has reported the run's last block persisted. It only catches a reth that breaks
// its own buffer rule: notifications arrive asynchronously and reth offers no synchronous read of
// its persisted tip, so a save that finished during the sweep may not be reported yet.
func requireRunEndNotReportedPersisted(t devtest.T, sys *withdrawalsRootSystem, run withdrawalRun) {
	for _, persisted := range sys.PersistedBlocks {
		tip := persisted.Latest()
		t.Require().Lessf(tip.Number, run.last,
			"%s reported block %d persisted by the end of the sweep, past the run's last block %d, while sequencing was stopped "+
				"within reth's memory buffer of it", persisted, tip.Number, run.last)
		t.Logger().Info("Run's last block not reported persisted after the sweep",
			"node", persisted, "reported_persisted_tip", tip.Number, "run_last", run.last)
	}
}

// persistenceTimeout bounds how long, once sequencing resumes, every op-reth node may take to
// persist the run's last block: reth persists a block at the latest once it holds the most blocks
// it keeps in memory plus a persistence interval of newer ones.
func persistenceTimeout(sys *withdrawalsRootSystem, c withdrawalsRootCase) time.Duration {
	return time.Duration(c.maxBlocksInMemory()+c.persistenceSchedule().PersistenceInterval())*sys.BlockTime + persistenceSlack
}

// TestWithdrawalsRootMatchesMessagePasserStorageRootAcrossPersistenceRuns initiates a withdrawal
// from several accounts in every block for twice as many blocks as reth holds in memory, on an
// explicit persistence schedule, and requires every block of the run to carry as its
// withdrawalsRoot the L2ToL1MessagePasser storage root that eth_getProof serves on the sequencer,
// on an op-reth follower reading its own historical state, on an op-geth follower and on a
// proofs-history follower, with every node on the sequencer's chain: once with sequencing stopped
// right after the run, while every op-reth node holds the oldest blocks of it on disk and the
// newest in memory, and again once every op-reth node reports every block of the run persisted.
func TestWithdrawalsRootMatchesMessagePasserStorageRootAcrossPersistenceRuns(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, withdrawalsRootMatrix(), func(t devtest.T, c withdrawalsRootCase) {
		sysgo.SkipOnOpGeth(t, "the withdrawals root is checked across op-reth's persistence runs")
		sys := newWithdrawalsRootSystem(t, c)
		senders := sys.FunderL2.NewFundedEOAs(withdrawersPerBlock, eth.Ether(senderFundsEther))
		sys.Sequencer.AdvancedOnSchedule(0, sys.BlockTime)

		txs := initiateWithdrawalsEveryBlock(t, sys.Sequencer, senders, c.runBlocks())
		stoppedHead := sys.Sequencer.BlockRefByHash(sys.SequencerCL.StopSequencer())
		run := collectWithdrawalRun(t, sys.Sequencer, stoppedHead, txs, senders)
		requireRunEndsInMemoryBuffer(t, run, stoppedHead)
		t.Require().Greaterf(run.blocks(), c.maxBlocksInMemory(),
			"the withdrawals span %d blocks, not more than the %d reth may hold in memory, so the run crossed no persistence boundary",
			run.blocks(), c.maxBlocksInMemory())
		persistenceInterval := c.persistenceSchedule().PersistenceInterval()
		t.Require().Lessf(run.longestGap, persistenceInterval,
			"%d consecutive blocks of the run carry no withdrawal, a whole persistence run of %d blocks", run.longestGap, persistenceInterval)
		t.Require().GreaterOrEqualf(run.blocksWithWithdrawals*5, run.blocks()*4,
			"only %d of the run's %d blocks carry a withdrawal; the workload must reach nearly every block", run.blocksWithWithdrawals, run.blocks())

		for _, node := range sys.nodes() {
			node.Reached(eth.Unsafe, stoppedHead.Number, catchUpAttempts)
		}
		awaitOldestBlocksPersisted(t, sys, run)
		verifyWithdrawalsRootsAgree(t, sys, run, "run just ended")
		requireRunEndNotReportedPersisted(t, sys, run)

		sys.SequencerCL.StartSequencer()
		for _, persisted := range sys.PersistedBlocks {
			persisted.AwaitPersisted(run.last, persistenceTimeout(sys, c))
		}
		verifyWithdrawalsRootsAgree(t, sys, run, "run persisted")
	})
}

func TestWithdrawalsRootMatrixPersistsInRuns(t *testing.T) {
	t.Parallel()
	for _, c := range withdrawalsRootMatrix() {
		schedule := c.persistenceSchedule()
		require.GreaterOrEqualf(t, schedule.PersistenceThreshold, schedule.MemoryBlockBufferTarget,
			"%s: a threshold below the memory block buffer target %d leaves reth no persistence interval", c, schedule.MemoryBlockBufferTarget)
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
