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
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// l2BlockTimeSeconds keeps the runs short: the run length is a block count, so halving the
	// block time halves the time spent producing blocks.
	l2BlockTimeSeconds = 1
	// maxBlocksPerRound is the most blocks a round of the load may take on average before the
	// load counts as stalled. A round's transactions land in the next block or two and the
	// workload sees their receipts within seconds, so a steady load stays well under it.
	maxBlocksPerRound = 7
	// persistenceBackpressureThreshold is reth's --engine.persistence-backpressure-threshold, the
	// value reth derives for every persistence threshold in the matrix, the larger of 100 and twice
	// the threshold. While a save runs, reth stops taking
	// engine API calls once this many blocks beyond the memory buffer await persistence, so a
	// save stalls the sequencer only once it falls this far behind.
	persistenceBackpressureThreshold = 100
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

func (c livenessCase) String() string {
	return fmt.Sprintf("threshold_%d-%s", c.persistenceThreshold, c.load)
}

// persistenceSchedule pins reth's persistence schedule.
func (c livenessCase) persistenceSchedule() sysgo.OpRethPersistenceSchedule {
	return sysgo.OpRethPersistenceSchedule{
		MemoryBlockBufferTarget:          sysgo.OpRethPinnedMemoryBlockBufferTarget,
		PersistenceBackpressureThreshold: persistenceBackpressureThreshold,
		PersistenceThreshold:             c.persistenceThreshold,
	}
}

// runBlocks is how many blocks the sequencer is held to its schedule for: three times the
// threshold, which puts at least persistenceRunsCovered persistence runs inside the window
// wherever it starts. A run starts every persistence interval of the schedule: every 46 blocks at
// threshold 50 (blocks 51, 97, 143, ...) and every 3 blocks at threshold 7.
func (c livenessCase) runBlocks() uint64 {
	return persistenceRunsCovered * c.persistenceThreshold
}

// minRounds is how many rounds the load must land in the blocks the sequencer is held to its
// schedule for, so the persistence runs in the window happen under load.
func (c livenessCase) minRounds() uint64 {
	return c.runBlocks() / maxBlocksPerRound
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
	BlockTime time.Duration
	EL        *dsl.L2ELNode
	FunderL2  *dsl.FunderEOA
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
				sysgo.OpRethWithPersistenceSchedule(c.persistenceSchedule()),
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
	// roundBlocks holds, per landed round, the block its last transaction landed in. Only run
	// writes it, and it is read only once finished is closed.
	roundBlocks []uint64
	senders     []*dsl.EOA
	stopping    atomic.Bool
	t           devtest.T
}

func newWorkload(t devtest.T, sys *sequencerSystem, caseLoad load) *workload {
	w := &workload{
		load:    caseLoad,
		senders: []*dsl.EOA{sys.FunderL2.NewFundedEOA(eth.Ether(senderFundsEther)), sys.FunderL2.NewFundedEOA(eth.Ether(senderFundsEther))},
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
	// A workload is built on a running system, so this cleanup runs before the system's: the load
	// ends before the nodes it drives do, also when the test fails before it calls stop.
	w.t.Cleanup(w.halt)
}

func (w *workload) run(ctx context.Context) error {
	for round := uint64(0); !w.stopping.Load(); round++ {
		var txs []*txplan.PlannedTx
		for i := range transactionsPerRound {
			sender, other := w.senders[i%2], w.senders[(i+1)%2]
			tx := txplan.NewPlannedTx(w.plan(sender, other))
			if _, err := tx.Submitted.Eval(ctx); err != nil {
				return w.unlessStopped(ctx, fmt.Errorf("round %d: submitting transaction %d from %s: %w", round, i, sender, err))
			}
			txs = append(txs, tx)
		}
		var landedIn uint64
		for i, tx := range txs {
			if _, err := tx.Success.Eval(ctx); err != nil {
				return w.unlessStopped(ctx, fmt.Errorf("round %d: transaction %d did not succeed: %w", round, i, err))
			}
			receipt, err := tx.Included.Eval(ctx)
			if err != nil {
				return w.unlessStopped(ctx, fmt.Errorf("round %d: reading the receipt of transaction %d: %w", round, i, err))
			}
			landedIn = max(landedIn, bigs.Uint64Strict(receipt.BlockNumber))
		}
		w.roundBlocks = append(w.roundBlocks, landedIn)
	}
	return nil
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
	if w.err != nil {
		w.t.Logger().Warn("Workload failed before it was halted", "load", w.load, "err", w.err)
	}
}

// stop lets the round in flight finish, so a round stuck waiting for inclusion fails the test
// instead of being canceled, fails the test if the load had failed, and returns how many rounds
// landed in blocks first through last.
func (w *workload) stop(first, last uint64) uint64 {
	w.t.Logger().Info("Stopping workload after the round in flight", "load", w.load)
	w.stopping.Store(true)
	<-w.finished
	w.cancel()
	var roundsInWindow uint64
	for _, landedIn := range w.roundBlocks {
		if landedIn >= first && landedIn <= last {
			roundsInWindow++
		}
	}
	w.t.Logger().Info("Stopped workload", "load", w.load, "rounds", len(w.roundBlocks), "rounds_in_window", roundsInWindow,
		"transactions_in_window", roundsInWindow*transactionsPerRound, "window_first", first, "window_last", last)
	w.t.Require().NoError(w.err, "the workload failed while the sequencer was held to its schedule")
	return roundsInWindow
}

// TestOpRethSequencerKeepsScheduleAcrossPersistenceRuns runs an op-reth sequencer on an explicit
// persistence schedule under a steady transaction load for three times the persistence threshold
// in blocks, which spans at least three of reth's persistence runs, and requires every block to
// extend the previous one with its timestamp plus the block time and to be canonical before its
// slot ends, the load to land a round every maxBlocksPerRound blocks on average, and op-reth to
// log no error meanwhile.
func TestOpRethSequencerKeepsScheduleAcrossPersistenceRuns(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, livenessMatrix(), func(t devtest.T, c livenessCase) {
		sysgo.SkipOnOpGeth(t, "the schedule is held across op-reth's persistence runs")
		sys := newSequencerSystem(t, c)
		errorLogs := sys.EL.RecordLogs(log.LevelError)
		w := newWorkload(t, sys, c.load)

		w.start()
		from := sys.EL.AdvancedOnSchedule(c.runBlocks(), sys.BlockTime)
		rounds := w.stop(from.Number+1, from.Number+c.runBlocks())

		t.Require().GreaterOrEqualf(rounds, c.minRounds(), "the workload must land a round every %d blocks on average across the %d blocks the sequencer was held to its schedule", maxBlocksPerRound, c.runBlocks())
		errorLogs.VerifyEmpty(fmt.Sprintf("sequencing %d blocks under %s across persistence runs", c.runBlocks(), c.load))
	})
}
