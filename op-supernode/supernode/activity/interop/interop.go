package interop

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

// Compile-time interface conformance assertions.
var (
	_                  activity.RunnableActivity     = (*Interop)(nil)
	_                  activity.VerificationActivity = (*Interop)(nil)
	backoffPeriod                                    = 1 * time.Second // backoff when chains aren't ready
	errorBackoffPeriod                               = 2 * time.Second // backoff on errors
)

// InteropActivationTimestampFlag is the CLI flag for the interop activation timestamp.
var InteropActivationTimestampFlag = &cli.Uint64Flag{
	Name:  "interop.activation-timestamp",
	Usage: "The timestamp at which interop should start",
	Value: 0,
}

func init() {
	flags.RegisterActivityFlags(InteropActivationTimestampFlag)
}

// chainsReadyResult holds the parallel query results from checkChainsReady.
type chainsReadyResult struct {
	blocks  map[eth.ChainID]eth.BlockID // L2 blocks at the timestamp
	l1Heads map[eth.ChainID]eth.BlockID // per-chain L1 inclusion heads
}

// RoundObservation is a consistent snapshot of the current round's state,
// captured upfront so the decision function operates on immutable data.
type RoundObservation struct {
	LastVerifiedTS *uint64
	LastVerified   *VerifiedResult
	NextTimestamp  uint64
	ChainsReady    bool
	BlocksAtTS     map[eth.ChainID]eth.BlockID
	L1Heads        map[eth.ChainID]eth.BlockID
	L1Consistent   bool
	Paused         bool
}

// Decision represents the outcome of the pure decision function.
type Decision int

const (
	DecisionWait Decision = iota
	DecisionAdvance
	DecisionInvalidate
	DecisionRewind
)

// StepOutput combines a decision with the verification result (if any).
type StepOutput struct {
	Decision Decision
	Result   Result
}

// Interop is a VerificationActivity that can also run background work as a RunnableActivity.
type Interop struct {
	log                 log.Logger
	chains              map[eth.ChainID]cc.ChainContainer
	activationTimestamp uint64
	dataDir             string

	verifiedDB *VerifiedDB
	logsDBs    map[eth.ChainID]LogsDB

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool

	currentL1 eth.BlockID

	verifyFn func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error)

	// cycleVerifyFn handles same-timestamp cycle verification.
	// It is called after verifyFn in progressInterop, and its results are merged.
	// Set to verifyCycleMessages by default in New().
	cycleVerifyFn func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error)

	// pauseAtTimestamp is used for integration test control only.
	// When non-zero, progressInterop will return early without processing
	// if the next timestamp to process is >= this value.
	pauseAtTimestamp atomic.Uint64

	l1Checker    *byNumberConsistencyChecker
	frontierView *frontierVerificationView
}

func (i *Interop) Name() string {
	return "interop"
}

// New constructs a new Interop activity.
func New(
	log log.Logger,
	activationTimestamp uint64,
	chains map[eth.ChainID]cc.ChainContainer,
	dataDir string,
	l1Source l1ByNumberSource,
) *Interop {
	verifiedDB, err := OpenVerifiedDB(dataDir)
	if err != nil {
		log.Error("failed to open verified DB", "err", err)
		return nil
	}

	// Initialize logsDBs for each chain
	logsDBs := make(map[eth.ChainID]LogsDB)
	for chainID := range chains {
		logsDB, err := openLogsDB(log, chainID, dataDir)
		if err != nil {
			log.Error("failed to open logs DB for chain", "chainID", chainID, "err", err)
			// Clean up already created logsDBs
			for _, db := range logsDBs {
				_ = db.Close()
			}
			_ = verifiedDB.Close()
			return nil
		}
		logsDBs[chainID] = logsDB
	}

	i := &Interop{
		log:                 log,
		chains:              chains,
		verifiedDB:          verifiedDB,
		logsDBs:             logsDBs,
		dataDir:             dataDir,
		activationTimestamp: activationTimestamp,
	}
	// default to using the verifyInteropMessages function
	// (can be overridden by tests)
	i.verifyFn = i.verifyInteropMessages
	i.cycleVerifyFn = i.verifyCycleMessages
	i.l1Checker = newByNumberConsistencyChecker(l1Source)
	return i
}

// Start begins the Interop activity background loop and blocks until ctx is canceled.
func (i *Interop) Start(ctx context.Context) error {
	i.mu.Lock()
	if i.started {
		i.mu.Unlock()
		<-ctx.Done()
		return ctx.Err()
	}
	i.ctx, i.cancel = context.WithCancel(ctx)
	i.started = true
	i.mu.Unlock()

	for {
		select {
		case <-i.ctx.Done():
			return i.ctx.Err()
		default:
			madeProgress, err := i.progressAndRecord()
			if err != nil {
				// Error: back off before next attempt
				i.log.Error("failed to progress and record interop", "err", err)
				time.Sleep(errorBackoffPeriod)
				continue
			}
			if !madeProgress {
				// Chains not ready, back off before next attempt
				time.Sleep(backoffPeriod)
			}
			// Otherwise: immediately ready for next iteration (aggressive catch-up)
		}
	}
}

// Stop stops the Interop activity.
func (i *Interop) Stop(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.cancel != nil {
		i.cancel()
	}
	// Close all logsDBs
	for chainID, db := range i.logsDBs {
		if err := db.Close(); err != nil {
			i.log.Error("failed to close logs DB", "chainID", chainID, "err", err)
		}
	}
	if i.verifiedDB != nil {
		return i.verifiedDB.Close()
	}
	return nil
}

// PauseAt sets a timestamp at which the interop activity should pause.
// When progressInterop encounters this timestamp or any later timestamp, it returns early without processing.
// Uses >= check so that if the activity is already beyond the pause point, it will still stop.
// This function is for integration test control only.
// Pass 0 to clear the pause (equivalent to calling Resume).
func (i *Interop) PauseAt(ts uint64) {
	i.pauseAtTimestamp.Store(ts)
	i.log.Info("interop pause set", "pauseAtTimestamp", ts)
}

// Resume clears any pause timestamp, allowing normal processing to continue.
// This function is for integration test control only.
func (i *Interop) Resume() {
	i.pauseAtTimestamp.Store(0)
	i.log.Info("interop pause cleared")
}

// checkPreconditions determines whether observation alone already implies an
// action, before running verification. It returns nil when verification should
// proceed.
func checkPreconditions(obs RoundObservation) *StepOutput {
	if obs.Paused {
		output := StepOutput{Decision: DecisionWait}
		return &output
	}
	if !obs.ChainsReady {
		output := StepOutput{Decision: DecisionWait}
		return &output
	}
	if !obs.L1Consistent {
		output := StepOutput{Decision: DecisionRewind}
		return &output
	}
	return nil
}

// decideVerifiedResult determines the next action from a completed verification
// result. No side effects, no I/O.
func decideVerifiedResult(_ RoundObservation, verified Result) StepOutput {
	if verified.IsEmpty() {
		return StepOutput{Decision: DecisionWait}
	}
	if !verified.IsValid() {
		return StepOutput{Decision: DecisionInvalidate, Result: verified}
	}
	return StepOutput{Decision: DecisionAdvance, Result: verified}
}

// progressAndRecord attempts to progress interop and record the result.
// Returns (madeProgress, error) where madeProgress indicates if we advanced the verified timestamp.
func (i *Interop) progressAndRecord() (bool, error) {
	pending, err := i.verifiedDB.GetPendingTransition()
	if err != nil {
		return false, fmt.Errorf("get pending transition: %w", err)
	}
	if pending != nil {
		return i.applyPendingTransition(*pending)
	}

	output, obs, err := i.progressInterop()
	if err != nil {
		return false, err
	}
	if output.Decision == DecisionWait {
		return i.refreshCurrentL1OnWait()
	}
	if output.Decision == DecisionRewind && obs.LastVerifiedTS == nil {
		return false, nil
	}

	pendingTx, err := i.buildPendingTransition(output, obs)
	if err != nil {
		return false, err
	}
	if err := i.verifiedDB.SetPendingTransition(pendingTx); err != nil {
		return false, fmt.Errorf("persist pending transition: %w", err)
	}
	return i.applyPendingTransition(pendingTx)
}

func (i *Interop) refreshCurrentL1OnWait() (bool, error) {
	localL1, err := i.collectCurrentL1()
	if err != nil {
		// Non-fatal: just keep existing currentL1.
		i.log.Debug("failed to collect current L1 on wait", "err", err)
		return false, nil
	}
	i.mu.Lock()
	i.currentL1 = localL1
	i.mu.Unlock()
	return false, nil
}

// progressInterop prepares the next interop action by observing the world,
// optionally verifying the frontier, and returning the resulting decision.
// It does not apply any side effects itself.
func (i *Interop) progressInterop() (StepOutput, RoundObservation, error) {
	obs, err := i.observeRound()
	if err != nil {
		return StepOutput{}, RoundObservation{}, err
	}

	if early := checkPreconditions(obs); early != nil {
		return *early, obs, nil
	}

	result, err := i.verify(obs.NextTimestamp, obs.BlocksAtTS)
	if err != nil {
		return StepOutput{}, obs, err
	}

	return decideVerifiedResult(obs, result), obs, nil
}

// observeRound captures a consistent snapshot of the current round state.
// All reads happen here; the decision function operates on this snapshot.
func (i *Interop) observeRound() (RoundObservation, error) {
	var obs RoundObservation

	lastTS, initialized := i.verifiedDB.LastTimestamp()
	if initialized {
		ts := lastTS
		obs.LastVerifiedTS = &ts
		result, err := i.verifiedDB.Get(lastTS)
		if err != nil {
			return obs, fmt.Errorf("failed to read last verified result: %w", err)
		}
		obs.LastVerified = &result
		obs.NextTimestamp = lastTS + 1
	} else {
		obs.NextTimestamp = i.activationTimestamp
	}

	if pauseTS := i.pauseAtTimestamp.Load(); pauseTS != 0 && obs.NextTimestamp >= pauseTS {
		obs.Paused = true
		return obs, nil
	}

	ready, err := i.checkChainsReady(obs.NextTimestamp)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			obs.ChainsReady = false
			return obs, nil
		}
		return obs, err
	}
	obs.ChainsReady = true
	obs.BlocksAtTS = ready.blocks
	obs.L1Heads = ready.l1Heads

	// Check that all frontier L1 heads AND the accepted L1 head are on the same canonical fork.
	obs.L1Consistent = true
	if i.l1Checker != nil {
		heads := make([]eth.BlockID, 0, len(obs.L1Heads)+1)
		if obs.LastVerified != nil {
			heads = append(heads, obs.LastVerified.L1Inclusion)
		}
		for _, l1 := range obs.L1Heads {
			heads = append(heads, l1)
		}
		same, err := i.l1Checker.SameL1Chain(i.ctx, heads)
		if err != nil {
			return obs, fmt.Errorf("L1 consistency check: %w", err)
		}
		obs.L1Consistent = same
	}

	return obs, nil
}

// verify runs the heavy I/O: log loading, message verification, and cycle detection.
func (i *Interop) verify(ts uint64, blocksAtTS map[eth.ChainID]eth.BlockID) (Result, error) {
	view, err := i.resolveFrontierVerificationView(blocksAtTS)
	if err != nil {
		return Result{}, fmt.Errorf("resolve frontier verification view: %w", err)
	}
	i.frontierView = view
	defer func() {
		i.frontierView = nil
	}()

	result, err := i.verifyFn(ts, blocksAtTS)
	if err != nil {
		return Result{}, err
	}

	cycleResult, err := i.cycleVerifyFn(ts, blocksAtTS)
	if err != nil {
		return Result{}, fmt.Errorf("cycle verification failed: %w", err)
	}

	if len(cycleResult.InvalidHeads) > 0 {
		if result.InvalidHeads == nil {
			result.InvalidHeads = make(map[eth.ChainID]eth.BlockID)
		}
		for chainID, invalidBlock := range cycleResult.InvalidHeads {
			result.InvalidHeads[chainID] = invalidBlock
		}
	}

	return result, nil
}

func (i *Interop) buildPendingTransition(output StepOutput, obs RoundObservation) (PendingTransition, error) {
	switch output.Decision {
	case DecisionAdvance, DecisionInvalidate:
		result := output.Result
		return PendingTransition{
			Decision: output.Decision,
			Result:   &result,
		}, nil
	case DecisionRewind:
		rewindPlan, err := i.buildRewindPlan(*obs.LastVerifiedTS)
		if err != nil {
			return PendingTransition{}, fmt.Errorf("build rewind plan: %w", err)
		}
		return PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &rewindPlan,
		}, nil
	default:
		return PendingTransition{}, fmt.Errorf("unsupported transition decision: %v", output.Decision)
	}
}

func (i *Interop) applyPendingTransition(pending PendingTransition) (bool, error) {
	switch pending.Decision {
	case DecisionRewind:
		if pending.Rewind == nil {
			return false, fmt.Errorf("invalid pending rewind transition: missing rewind plan")
		}
		i.mu.Lock()
		i.currentL1 = eth.BlockID{}
		i.mu.Unlock()
		if err := i.applyRewindPlan(*pending.Rewind); err != nil {
			return false, fmt.Errorf("apply rewind plan: %w", err)
		}
		if err := i.verifiedDB.ClearPendingTransition(); err != nil {
			return false, fmt.Errorf("clear pending transition: %w", err)
		}
		return false, nil

	case DecisionInvalidate:
		if pending.Result == nil {
			if err := i.verifiedDB.ClearPendingTransition(); err != nil {
				return false, fmt.Errorf("clear empty invalidation transition: %w", err)
			}
			return false, nil
		}
		invalidations := make([]PendingInvalidation, 0, len(pending.Result.InvalidHeads))
		for chainID, blockID := range pending.Result.InvalidHeads {
			invalidations = append(invalidations, PendingInvalidation{
				ChainID:   chainID,
				BlockID:   blockID,
				Timestamp: pending.Result.Timestamp,
			})
		}
		sort.Slice(invalidations, func(i, j int) bool {
			return invalidations[i].ChainID.Cmp(invalidations[j].ChainID) < 0
		})
		// Freeze ALL chains' VNs before rewinding any. Without this, a non-invalidated
		// chain's still-running VN can observe the interop state change from onReset and
		// issue a ForkchoiceUpdate that advances its safe head. If that chain is later
		// invalidated (e.g. transitive invalidation across multiple rounds), its rewind
		// will be rejected because the safe head was already advanced.
		// This is broader than freezing only invalid chains because transitive invalidation
		// requires multiple verification rounds — a chain valid in round N may become
		// invalid in round N+1 after its dependency is replaced.
		for chainID, chain := range i.chains {
			if err := chain.PauseAndStopVN(i.ctx); err != nil {
				i.log.Error("failed to freeze chain before rewind", "chainID", chainID, "err", err)
			}
		}
		var failedAny bool
		for _, p := range invalidations {
			if err := i.invalidateBlock(p.ChainID, p.BlockID, p.Timestamp); err != nil {
				i.log.Error("invalidation failed, transition preserved for retry on restart",
					"chain", p.ChainID, "block", p.BlockID, "err", err)
				failedAny = true
			}
		}
		// Resume non-invalidated chains. Invalidated chains are resumed by RewindEngine.
		for chainID, chain := range i.chains {
			if _, isInvalid := pending.Result.InvalidHeads[chainID]; !isInvalid {
				if err := chain.Resume(i.ctx); err != nil {
					i.log.Error("failed to resume chain after rewind", "chainID", chainID, "err", err)
				}
			}
		}
		if failedAny {
			return false, fmt.Errorf("one or more invalidations failed, transition preserved")
		}
		if err := i.verifiedDB.ClearPendingTransition(); err != nil {
			return false, fmt.Errorf("clear pending transition: %w", err)
		}
		return false, nil

	case DecisionAdvance:
		if pending.Result == nil {
			if err := i.verifiedDB.ClearPendingTransition(); err != nil {
				return false, fmt.Errorf("clear empty advance transition: %w", err)
			}
			return false, nil
		}
		if err := i.persistFrontierLogs(pending.Result.Timestamp, pending.Result.L2Heads); err != nil {
			return false, fmt.Errorf("persist frontier logs: %w", err)
		}

		if err := i.commitVerifiedResult(pending.Result.Timestamp, pending.Result.ToVerifiedResult()); err != nil {
			return false, fmt.Errorf("commit verified result: %w", err)
		}
		if err := i.verifiedDB.ClearPendingTransition(); err != nil {
			return false, fmt.Errorf("clear pending transition: %w", err)
		}
		i.log.Info("committed verified result", "timestamp", pending.Result.Timestamp)
		i.mu.Lock()
		i.currentL1 = pending.Result.L1Inclusion
		i.mu.Unlock()
		return true, nil
	}

	return false, nil
}

func (i *Interop) buildRewindPlan(lastTS uint64) (RewindPlan, error) {
	plan := RewindPlan{
		RewindAtOrAfter: lastTS,
	}

	if lastTS <= i.activationTimestamp {
		return plan, nil
	}

	rewindTargetTS := lastTS - 1
	prevResult, err := i.verifiedDB.Get(rewindTargetTS)
	if err != nil {
		return RewindPlan{}, fmt.Errorf("read previous verified result at %d: %w", rewindTargetTS, err)
	}
	plan.ResetAllChainsTo = &rewindTargetTS
	plan.TargetHeads = prevResult.L2Heads
	return plan, nil
}

func (i *Interop) applyRewindPlan(plan RewindPlan) error {
	i.log.Warn("rewinding accepted state due to drift", "timestamp", plan.RewindAtOrAfter)

	if _, err := i.verifiedDB.Rewind(plan.RewindAtOrAfter); err != nil {
		return fmt.Errorf("rewind verifiedDB: %w", err)
	}

	var allErrs []error
	recordErr := func(err error) {
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	sortedChainIDs := make([]eth.ChainID, 0, len(i.chains))
	for chainID := range i.chains {
		sortedChainIDs = append(sortedChainIDs, chainID)
	}
	sort.Slice(sortedChainIDs, func(a, b int) bool {
		return sortedChainIDs[a].Cmp(sortedChainIDs[b]) < 0
	})

	for _, chainID := range sortedChainIDs {
		chain := i.chains[chainID]
		if _, err := chain.PruneDeniedAtOrAfterTimestamp(plan.RewindAtOrAfter); err != nil {
			i.log.Error("failed to prune deny list on rewind", "chain", chainID, "err", err)
			recordErr(fmt.Errorf("chain %s: prune deny list on rewind: %w", chainID, err))
		}
		if plan.ResetAllChainsTo != nil {
			if err := chain.RewindEngine(i.ctx, *plan.ResetAllChainsTo, eth.BlockRef{}); err != nil {
				i.log.Error("failed to reset chain engine on rewind", "chain", chainID, "err", err)
				recordErr(fmt.Errorf("chain %s: reset chain engine on rewind: %w", chainID, err))
			}
		}
	}

	if plan.TargetHeads == nil {
		for chainID, db := range i.logsDBs {
			if err := db.Clear(&noopInvalidator{}); err != nil {
				i.log.Error("failed to clear logsDB on full rewind", "chain", chainID, "err", err)
				recordErr(fmt.Errorf("chain %s: clear logsDB on full rewind: %w", chainID, err))
			}
		}
		return errors.Join(allErrs...)
	}

	for chainID, db := range i.logsDBs {
		expectedHead, ok := plan.TargetHeads[chainID]
		if !ok {
			continue
		}
		latestBlock, hasBlocks := db.LatestSealedBlock()
		if !hasBlocks || latestBlock == expectedHead || latestBlock.Number < expectedHead.Number {
			continue
		}
		i.log.Info("rewinding logsDB to previous verified head",
			"chain", chainID, "from", latestBlock, "to", expectedHead)
		if err := db.Rewind(&noopInvalidator{}, expectedHead); err != nil {
			i.log.Error("failed to rewind logsDB, transition preserved for retry",
				"chain", chainID, "err", err)
			recordErr(fmt.Errorf("chain %s: rewind logsDB to previous verified head: %w", chainID, err))
		}
	}

	return errors.Join(allErrs...)
}

// collectCurrentL1 collects the current L1 head of all chains,
// which is the minimum L1 head of all the derivation pipelines in Chain Containers
func (i *Interop) collectCurrentL1() (eth.BlockID, error) {
	var currentL1 eth.BlockID
	first := true
	for _, chain := range i.chains {
		status, err := chain.SyncStatus(i.ctx)
		if err != nil {
			return eth.BlockID{}, fmt.Errorf("chain %s not ready: %w", chain.ID(), err)
		}
		block := status.CurrentL1
		if first || block.Number < currentL1.Number {
			currentL1 = block.ID()
			first = false
		}
	}
	return currentL1, nil
}

// checkChainsReady checks if all chains are ready to process the next timestamp.
// Queries all chains in parallel for better performance.
// Returns both the L2 blocks at the timestamp and the L1 inclusion heads.
func (i *Interop) checkChainsReady(ts uint64) (chainsReadyResult, error) {
	type result struct {
		chainID eth.ChainID
		blockID eth.BlockID
		l1Head  eth.BlockID
		err     error
	}

	results := make(chan result, len(i.chains))

	// Query all chains in parallel
	for _, chain := range i.chains {
		go func(c cc.ChainContainer) {
			// Use OptimisticAt as the single atomic source for both L2 block and L1 head.
			// This avoids a TOCTOU race between separate LocalSafeBlockAtTimestamp and OptimisticAt calls.
			l2Block, l1Block, err := c.OptimisticAt(i.ctx, ts)
			if err != nil {
				results <- result{chainID: c.ID(), err: fmt.Errorf("chain %s not ready for timestamp %d: %w", c.ID(), ts, err)}
				return
			}
			results <- result{chainID: c.ID(), blockID: l2Block, l1Head: l1Block}
		}(chain)
	}

	// Collect all results before returning so every goroutine completes before the
	// next call spawns a new batch, preventing accumulation of in-flight RPC calls.
	ready := chainsReadyResult{
		blocks:  make(map[eth.ChainID]eth.BlockID),
		l1Heads: make(map[eth.ChainID]eth.BlockID),
	}
	var firstErr error
	for range i.chains {
		r := <-results
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
		} else {
			ready.blocks[r.chainID] = r.blockID
			ready.l1Heads[r.chainID] = r.l1Head
		}
	}
	if firstErr != nil {
		return chainsReadyResult{}, firstErr
	}

	return ready, nil
}

func (i *Interop) commitVerifiedResult(timestamp uint64, verifiedResult VerifiedResult) error {
	return i.verifiedDB.Commit(verifiedResult)
}

// CurrentL1 returns the L1 block which has been fully considered for interop,
// whether or not it advanced the verified timestamp.
func (i *Interop) CurrentL1() eth.BlockID {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.currentL1
}

// VerifiedAtTimestamp returns whether the data is verified at the given timestamp.
// For timestamps before the activation timestamp, this returns true since interop
// wasn't active yet and verification proceeds automatically.
// For timestamps at or after the activation timestamp, this checks the verifiedDB.
func (i *Interop) VerifiedAtTimestamp(ts uint64) (bool, error) {
	// Timestamps before the activation timestamp are considered verified
	// because interop wasn't active yet
	if ts < i.activationTimestamp {
		return true, nil
	}
	return i.verifiedDB.Has(ts)
}

// LatestVerifiedL2Block returns the latest L2 block which has been verified,
// along with the timestamp at which it was verified.
func (i *Interop) LatestVerifiedL2Block(chainID eth.ChainID) (eth.BlockID, uint64) {
	emptyBlock := eth.BlockID{}
	ts, ok := i.verifiedDB.LastTimestamp()
	if !ok {
		return emptyBlock, 0
	}
	res, err := i.verifiedDB.Get(ts)
	if err != nil {
		return emptyBlock, 0
	}
	head, ok := res.L2Heads[chainID]
	if !ok {
		return emptyBlock, 0
	}
	return head, ts
}

// VerifiedBlockAtL1 returns the verified L2 block and timestamp
// which guarantees that the verified data at that timestamp
// originates from or before the supplied L1 block.
func (i *Interop) VerifiedBlockAtL1(chainID eth.ChainID, l1Block eth.L1BlockRef) (eth.BlockID, uint64) {
	// If L1 block is empty/zero (e.g. during startup before FinalizedL1 is set),
	// no verified result can match, so return early.
	if l1Block == (eth.L1BlockRef{}) {
		return eth.BlockID{}, 0
	}

	// Get the last verified timestamp
	lastTs, ok := i.verifiedDB.LastTimestamp()
	if !ok {
		return eth.BlockID{}, 0
	}

	// Search backwards from the last timestamp to find the latest result
	// where the L1 inclusion block is at or below the supplied L1 block number.
	// Stop at activationTimestamp — no verified results exist before that.
	lowerBound := i.activationTimestamp
	for ts := lastTs; ts >= lowerBound && ts <= lastTs; ts-- {
		result, err := i.verifiedDB.Get(ts)
		if err != nil {
			// Timestamp might not exist (due to gaps or rewinds), continue searching
			continue
		}

		// Check if this result's L1 inclusion is at or below the supplied L1 block number
		if result.L1Inclusion.Number <= l1Block.Number {
			// Found a finalized result, return the L2 head for this chain
			head, ok := result.L2Heads[chainID]
			if !ok {
				return eth.BlockID{}, 0
			}
			return head, ts
		}
	}

	// No verified block found
	return eth.BlockID{}, 0
}

// Reset is intentionally a no-op for interop.
// Interop-owned invalidation and rewind handling is driven synchronously through
// PendingTransition application, so callback-driven resets are not part of the
// correctness path anymore.
func (i *Interop) Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef) {
}

// invalidateBlock notifies the chain container to add the block to the denylist
// and potentially rewind if the chain is currently using that block.
func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID, decisionTimestamp uint64) error {
	chain, ok := i.chains[chainID]
	if !ok {
		return fmt.Errorf("chain %s not found", chainID)
	}
	_, err := chain.InvalidateBlock(i.ctx, blockID.Number, blockID.Hash, decisionTimestamp)
	return err
}
