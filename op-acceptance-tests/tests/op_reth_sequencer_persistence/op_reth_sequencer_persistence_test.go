package op_reth_sequencer_persistence

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// l2BlockTimeSeconds keeps the runs short: the run length is a block count, so halving the
	// block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// memoryBlockBufferTarget is reth's --engine.memory-block-buffer-target. It is passed
	// explicitly, like the persistence threshold, so the persistence schedule does not depend on
	// reth's defaults.
	memoryBlockBufferTarget = 5
	// persistenceRunsCovered is how many of reth's persistence runs the sequencer is held to its
	// schedule across.
	persistenceRunsCovered = 3
	// senderFundsEther pays for one sender's share of the whole run.
	senderFundsEther = 10
	// storageSlotsPerCall is how many storage slots every storage-writing transaction creates.
	storageSlotsPerCall = 16
	// transactionsPerRound is how many transactions the workload submits back to back before it
	// waits for them to be included, so each round lands in one or two blocks.
	transactionsPerRound = 8
)

// load is the kind of transaction the workload keeps the sequencer under.
type load string

const (
	// storageWrites calls a contract that creates storageSlotsPerCall storage slots per call, so
	// every persistence run writes a batch of new state.
	storageWrites load = "storage_writes"
	// transfers moves ether between the workload's accounts, so a block changes two balances.
	transfers load = "transfers"
)

// livenessCase is one cell of the matrix the sequencer is held to its schedule under.
type livenessCase struct {
	load                 load
	persistenceThreshold uint64
}

func (c livenessCase) name() string {
	return fmt.Sprintf("threshold_%d-%s", c.persistenceThreshold, c.load)
}

// engineArgs pins reth's persistence schedule.
func (c livenessCase) engineArgs() []string {
	return []string{
		fmt.Sprintf("--engine.memory-block-buffer-target=%d", memoryBlockBufferTarget),
		fmt.Sprintf("--engine.persistence-threshold=%d", c.persistenceThreshold),
	}
}

// runBlocks is how many blocks the sequencer is held to its schedule for: three times the
// threshold, which puts at least persistenceRunsCovered persistence runs inside the window
// wherever it starts. Reth persists once more than the threshold of blocks sit in memory and
// keeps the last memoryBlockBufferTarget of them, so a run starts every
// threshold+1-memoryBlockBufferTarget blocks: every 46 blocks at threshold 50 (blocks 51, 97,
// 143, ...) and every 3 blocks at threshold 7.
func (c livenessCase) runBlocks() uint64 {
	return persistenceRunsCovered * c.persistenceThreshold
}

func livenessMatrix() []livenessCase {
	var cases []livenessCase
	for _, threshold := range []uint64{7, 50} {
		for _, caseLoad := range []load{transfers, storageWrites} {
			cases = append(cases, livenessCase{load: caseLoad, persistenceThreshold: threshold})
		}
	}
	return cases
}

type sequencerSystem struct {
	EL        *dsl.L2ELNode
	FunderL2  *dsl.FunderEOA
	BlockTime time.Duration
}

// newSequencerSystem starts an op-reth sequencer alone, on the case's persistence schedule and
// without proofs history, so reth's persistence is the only one on it; the mixed runtime attaches
// the batcher to it.
func newSequencerSystem(t devtest.T, c livenessCase) *sequencerSystem {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds)},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{{
			CLKey:       "sequencer",
			CLKind:      sysgo.ResolveMixedL2CLKind(),
			ELKey:       "sequencer",
			ELKind:      sysgo.MixedL2ELOpReth,
			IsSequencer: true,
			OpRethOpts: []sysgo.OpRethOption{
				sysgo.OpRethWithoutProofsHistory(),
				sysgo.OpRethWithExtraArgs(c.engineArgs()...),
			},
		}},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 1, "expected the sequencer alone")
	return &sequencerSystem{
		BlockTime: time.Duration(frontends.L2Network.Escape().RollupConfig().BlockTime) * time.Second,
		EL:        frontends.Nodes[0].EL,
		FunderL2:  frontends.FunderL2,
	}
}

// workload keeps the sequencer under a steady load of the case's transactions from two accounts,
// off the test goroutine, so the test goroutine can hold the chain to its schedule meanwhile.
// Nothing asserts off the test goroutine: the first failure ends the load, and stop reports it.
type workload struct {
	cancel   context.CancelFunc
	err      error
	filler   common.Address
	finished chan struct{}
	load     load
	rounds   atomic.Uint64
	senders  []*dsl.EOA
	t        devtest.T
}

func newWorkload(t devtest.T, funder *dsl.FunderEOA, caseLoad load) *workload {
	w := &workload{
		load:    caseLoad,
		senders: []*dsl.EOA{funder.NewFundedEOA(eth.Ether(senderFundsEther)), funder.NewFundedEOA(eth.Ether(senderFundsEther))},
		t:       t,
	}
	if caseLoad == storageWrites {
		w.filler = w.senders[0].DeployStorageFiller(storageSlotsPerCall)
	}
	return w
}

// plan is one transaction of the load from sender: a call to the storage filler, or a transfer
// to the other account.
func (w *workload) plan(sender, other *dsl.EOA) txplan.Option {
	if w.load == storageWrites {
		return txplan.Combine(sender.Plan(), txplan.WithTo(&w.filler))
	}
	return sender.PlanTransfer(other.Address(), eth.OneHundredthEther)
}

// start submits rounds of transactionsPerRound transactions, alternating the senders, and waits
// for every transaction of a round to succeed before the next, until stop is called or the test
// ends.
func (w *workload) start() {
	ctx, cancel := context.WithCancel(w.t.Ctx())
	w.cancel = cancel
	w.finished = make(chan struct{})
	w.t.Logger().Info("Starting workload", "load", w.load, "transactions_per_round", transactionsPerRound)
	go func() {
		defer close(w.finished)
		w.err = w.run(ctx)
	}()
	// Registered after the system's cleanups, so it runs before them: the load ends before the
	// nodes it drives do, also when the test fails before it calls stop.
	w.t.Cleanup(w.halt)
}

func (w *workload) run(ctx context.Context) error {
	for round := uint64(0); ; round++ {
		var txs []*txplan.PlannedTx
		for i := range transactionsPerRound {
			sender, other := w.senders[i%2], w.senders[(i+1)%2]
			tx := txplan.NewPlannedTx(w.plan(sender, other))
			if _, err := tx.Submitted.Eval(ctx); err != nil {
				return w.unlessStopped(ctx, fmt.Errorf("round %d: submitting transaction %d from %s: %w", round, i, sender, err))
			}
			txs = append(txs, tx)
		}
		for i, tx := range txs {
			if _, err := tx.Success.Eval(ctx); err != nil {
				return w.unlessStopped(ctx, fmt.Errorf("round %d: transaction %d did not succeed: %w", round, i, err))
			}
		}
		w.rounds.Add(1)
	}
}

// unlessStopped drops the failures stopping the load causes by canceling its context.
func (w *workload) unlessStopped(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func (w *workload) halt() {
	w.cancel()
	<-w.finished
}

// stop ends the load, fails the test if the load had failed first, and returns how many rounds
// were included.
func (w *workload) stop() uint64 {
	w.halt()
	rounds := w.rounds.Load()
	w.t.Logger().Info("Stopped workload", "load", w.load, "rounds", rounds, "transactions", rounds*transactionsPerRound)
	w.t.Require().NoError(w.err, "the workload failed while the sequencer was held to its schedule")
	return rounds
}

// TestOpRethSequencerKeepsScheduleAcrossPersistenceRuns runs an op-reth sequencer on an explicit
// persistence schedule under a steady transaction load for three times the persistence threshold
// in blocks, which spans at least three of reth's persistence runs, and requires every block to
// extend the previous one with its timestamp plus the block time and to be canonical before its
// slot ends, and op-reth to log no error meanwhile.
func TestOpRethSequencerKeepsScheduleAcrossPersistenceRuns(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range livenessMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			sysgo.SkipOnOpGeth(t, "the schedule is held across op-reth's persistence runs")
			sys := newSequencerSystem(t, c)
			errorLogs := sys.EL.RecordLogs(log.LevelError)
			w := newWorkload(t, sys.FunderL2, c.load)

			w.start()
			sys.EL.AdvancedOnSchedule(c.runBlocks(), sys.BlockTime)
			rounds := w.stop()

			t.Require().NotZero(rounds, "the workload must have landed transactions by the time the sequencer had kept its schedule")
			errorLogs.VerifyEmpty(fmt.Sprintf("sequencing %d blocks under %s across persistence runs", c.runBlocks(), c.load))
		})
	}
}
