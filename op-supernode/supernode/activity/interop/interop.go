package interop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
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
	Name:    "interop.activation-timestamp",
	Usage:   "Override the interop activation timestamp derived from rollup configs",
	EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "INTEROP_ACTIVATION_TIMESTAMP"),
	Value:   0,
}

// InteropLogBackfillDepthFlag extends initiating-message log ingestion backward from the L2 tip by this duration (clamped to activation). Validation still starts only beyond the local safe head.
var InteropLogBackfillDepthFlag = &cli.DurationFlag{
	Name:    "interop.log-backfill-depth",
	Usage:   "Duration to pre-ingest logs behind the tip before interop validation (e.g. 168h). Never loads logs before interop.activation-timestamp. Requires interop.activation-timestamp.",
	EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "INTEROP_LOG_BACKFILL_DEPTH"),
	Value:   0,
}

func init() {
	flags.RegisterActivityFlags(InteropActivationTimestampFlag, InteropLogBackfillDepthFlag)
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
	L1NeedsRewind  bool
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

// Decision is serialized as a self-describing string in the WAL so that the
// on-disk format survives enum re-ordering or the insertion of new variants.
func (d Decision) String() string {
	switch d {
	case DecisionWait:
		return "wait"
	case DecisionAdvance:
		return "advance"
	case DecisionInvalidate:
		return "invalidate"
	case DecisionRewind:
		return "rewind"
	default:
		return fmt.Sprintf("unknown(%d)", int(d))
	}
}

func (d Decision) MarshalJSON() ([]byte, error) {
	switch d {
	case DecisionWait, DecisionAdvance, DecisionInvalidate, DecisionRewind:
		return json.Marshal(d.String())
	default:
		return nil, fmt.Errorf("marshal decision: unknown value %d", int(d))
	}
}

func (d *Decision) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("unmarshal decision: expected string: %w", err)
	}
	switch s {
	case "wait":
		*d = DecisionWait
	case "advance":
		*d = DecisionAdvance
	case "invalidate":
		*d = DecisionInvalidate
	case "rewind":
		*d = DecisionRewind
	default:
		return fmt.Errorf("unmarshal decision: unknown value %q", s)
	}
	return nil
}

// StepOutput combines a decision with the verification result (if any).
type StepOutput struct {
	Decision Decision
	Result   Result
}

// Interop is a VerificationActivity that can also run background work as a RunnableActivity.
type Interop struct {
	log                 log.Logger
	chains              map[eth.ChainID]cc.InteropChain
	activationTimestamp uint64 // immutable protocol activation timestamp

	// backfillEndTimestamp represents the end of the range of timestamps that were sealed by runLogBackfill.
	// this is used for loop handoff from log backfill to main processing.
	// firstVerifiableTimestamp is used to determine the start of the main processing loop, which is backfillEndTimestamp + 1
	// after backfill, or the safe-head-derived startup timestamp when backfill was not used.
	backfillEndTimestamp uint64
	firstVerifiableSet   bool
	firstVerifiable      uint64

	dataDir string

	messageExpiryWindow uint64

	verifiedDB *VerifiedDB
	logsDBs    map[eth.ChainID]LogsDB

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool

	currentL1 eth.BlockID

	// l1Heads is the snapshot captured with blocksAtTimestamp in observeRound; passing
	// it through avoids a TOCTOU race against L2 reorgs.
	verifyFn func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID, l1Heads map[eth.ChainID]eth.BlockID, view *frontierVerificationView) (Result, error)

	// cycleVerifyFn handles same-timestamp cycle verification.
	// It is called after verifyFn in progressInterop, and its results are merged.
	// Set to verifyCycleMessages by default in New().
	cycleVerifyFn func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID, view *frontierVerificationView) (Result, error)

	// pauseAtTimestamp is used for integration test control only.
	// When non-zero, progressInterop will return early without processing
	// if the next timestamp to process is >= this value.
	pauseAtTimestamp atomic.Uint64

	// backfillAttempts counts the number of times runLogBackfill was invoked
	// since Start. Read by integration tests to confirm the retry loop is engaged.
	backfillAttempts atomic.Int32
	// backfillCompleted is set to true once runLogBackfill returns nil (or was skipped
	// because logBackfillDepth <= 0). Read by integration tests to gate on backfill finishing.
	backfillCompleted atomic.Bool

	// l1Checker must be non-nil whenever observeRound runs. Production sets it
	// via New; tests inject noopL1Checker.
	l1Checker l1ConsistencyChecker

	logBackfillDepth time.Duration
	metrics          *resources.SupernodeMetrics
}

func (i *Interop) Name() string {
	return "interop"
}

// firstVerifiableTimestamp is the earliest timestamp the main loop will attempt
// to verify. If verification has already committed results, the first committed
// timestamp is the durable handoff boundary. Otherwise it is backfillEndTimestamp+1
// after log backfill, or the safe-head-derived startup timestamp.
func (i *Interop) firstVerifiableTimestamp(ctx context.Context) (uint64, error) {
	if i.verifiedDB != nil {
		if first, initialized := i.verifiedDB.FirstTimestamp(); initialized {
			return first, nil
		}
	}
	if i.backfillEndTimestamp != 0 {
		next := i.backfillEndTimestamp + 1
		if next < i.activationTimestamp {
			return i.activationTimestamp, nil
		}
		return next, nil
	}
	if i.firstVerifiableSet {
		return i.firstVerifiable, nil
	}
	return i.resolveFirstVerifiableTimestamp(ctx)
}

// New constructs a new Interop activity.
func New(
	log log.Logger,
	activationTimestamp uint64,
	messageExpiryWindow uint64,
	chains map[eth.ChainID]cc.InteropChain,
	dataDir string,
	l1Source l1ByNumberSource,
	logBackfillDepth time.Duration,
	metrics *resources.SupernodeMetrics,
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

	if messageExpiryWindow == 0 {
		messageExpiryWindow = defaultMessageExpiryWindow
	}
	if metrics == nil {
		metrics = resources.NewSupernodeMetrics()
	}
	i := &Interop{
		log:                 log,
		chains:              chains,
		verifiedDB:          verifiedDB,
		logsDBs:             logsDBs,
		dataDir:             dataDir,
		activationTimestamp: activationTimestamp,
		messageExpiryWindow: messageExpiryWindow,
		logBackfillDepth:    logBackfillDepth,
		metrics:             metrics,
	}
	// default to using the verifyInteropMessages function
	// (can be overridden by tests)
	i.verifyFn = i.verifyInteropMessages
	i.cycleVerifyFn = i.verifyCycleMessages
	i.l1Checker = newL1ConsistencyChecker(l1Source)
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

	if i.logBackfillDepth > 0 {
		i.log.Info("interop log backfill depth configured", "duration", i.logBackfillDepth.String())
		for {
			i.backfillAttempts.Add(1)
			end, err := i.runLogBackfill()
			if err == nil {
				i.backfillEndTimestamp = end
				break
			}
			i.log.Warn("log backfill failed, retrying (virtual nodes may not be ready yet)", "err", err)
			for cid := range i.chains {
				i.metrics.LogBackfillRetries.WithLabelValues(cid.String()).Inc()
			}
			select {
			case <-i.ctx.Done():
				return fmt.Errorf("log backfill interrupted: %w", i.ctx.Err())
			case <-time.After(errorBackoffPeriod):
			}
		}
	}
	i.backfillCompleted.Store(true)
	i.log.Info("log backfill complete", "backfillEndTimestamp", i.backfillEndTimestamp)

	firstVerifiableLog := uint64(0)
	if i.backfillEndTimestamp != 0 {
		firstVerifiableLog = i.backfillEndTimestamp + 1
		if firstVerifiableLog < i.activationTimestamp {
			firstVerifiableLog = i.activationTimestamp
		}
	} else if lastTS, initialized := i.verifiedDB.LastTimestamp(); initialized {
		firstVerifiableLog = lastTS + 1
	} else {
		for {
			first, err := i.readyFirstVerifiableTimestamp(i.ctx)
			if err == nil {
				i.firstVerifiable = first
				i.firstVerifiableSet = true
				firstVerifiableLog = first
				break
			}
			// Permanent SafeDB gap must halt normal startup cleanly. Backfill-enabled
			// startup reaches this path only if backfill had no range to seal.
			if errors.Is(err, cc.ErrHistoryUnavailable) {
				i.log.Error("interop activity halted: SafeDB history unavailable on this node", "err", err,
					"remediation", "reseed data dir, advance interop.activation-timestamp past the gap, or rederive from L1")
				return fmt.Errorf("interop halted due to unavailable history: %w", err)
			}
			i.log.Warn("first verifiable timestamp unavailable, retrying (virtual nodes may not be ready yet)", "err", err)
			select {
			case <-i.ctx.Done():
				return fmt.Errorf("first verifiable timestamp interrupted: %w", i.ctx.Err())
			case <-time.After(errorBackoffPeriod):
			}
		}
	}
	i.log.Info("interop first verifiable timestamp resolved",
		"activationTimestamp", i.activationTimestamp,
		"firstVerifiableTimestamp", firstVerifiableLog)

	for {
		select {
		case <-i.ctx.Done():
			return i.ctx.Err()
		default:
			madeProgress, err := i.progressAndRecord()
			if err != nil {
				// Permanent SafeDB gap: log once and halt — retrying cannot fix it.
				if errors.Is(err, cc.ErrHistoryUnavailable) {
					i.metrics.ActivityErrors.WithLabelValues("interop", "history_unavailable").Inc()
					i.log.Error("interop activity halted: SafeDB history unavailable on this node", "err", err,
						"remediation", "reseed data dir, advance interop.activation-timestamp past the gap, or rederive from L1")
					return fmt.Errorf("interop halted due to unavailable history: %w", err)
				}
				i.metrics.ActivityErrors.WithLabelValues("interop", "progress").Inc()
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

// readyFirstVerifiableTimestamp resolves the first timestamp that still needs
// interop verification and proves every chain can serve the optimistic L2/L1
// data needed to verify it.
func (i *Interop) readyFirstVerifiableTimestamp(ctx context.Context) (uint64, error) {
	first, err := i.resolveFirstVerifiableTimestamp(ctx)
	if err != nil {
		return 0, err
	}
	if _, err := i.checkChainsReady(first); err != nil {
		return 0, err
	}
	return first, nil
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
	if obs.L1NeedsRewind {
		output := StepOutput{Decision: DecisionRewind}
		return &output
	}
	if !obs.L1Consistent {
		output := StepOutput{Decision: DecisionWait}
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
		i.metrics.InteropRoundDecisions.WithLabelValues(pending.Decision.String()).Inc()
		return i.applyPendingTransition(*pending)
	}

	verifyStart := time.Now()
	output, obs, err := i.progressInterop()
	if err != nil {
		return false, err
	}
	i.metrics.InteropRoundDecisions.WithLabelValues(output.Decision.String()).Inc()
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
	progress, applyErr := i.applyPendingTransition(pendingTx)
	// Record verification latency for the full round including apply.
	i.metrics.InteropVerificationDuration.Observe(time.Since(verifyStart).Seconds())
	return progress, applyErr
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

	result, err := i.verify(obs.NextTimestamp, obs.BlocksAtTS, obs.L1Heads)
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
		next, err := i.firstVerifiableTimestamp(i.ctx)
		if err != nil {
			return obs, err
		}
		obs.NextTimestamp = next
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

	if obs.LastVerified != nil {
		same, err := i.l1Checker.SameL1Chain(i.ctx, []eth.BlockID{obs.LastVerified.L1Inclusion})
		if err != nil {
			return obs, fmt.Errorf("L1 consistency check: %w", err)
		}
		if !same {
			obs.L1Consistent = false
			obs.L1NeedsRewind = true
			return obs, nil
		}
	}

	// Check the new frontier independently from the accepted L1 head. If the
	// accepted head is still canonical but a frontier L1 head is stale, waiting
	// gives the L2 nodes time to catch up to the L1 reorg.
	heads := make([]eth.BlockID, 0, len(obs.L1Heads))
	for _, l1 := range obs.L1Heads {
		heads = append(heads, l1)
	}
	same, err := i.l1Checker.SameL1Chain(i.ctx, heads)
	if err != nil {
		return obs, fmt.Errorf("L1 consistency check: %w", err)
	}
	obs.L1Consistent = same

	return obs, nil
}

// verify runs the heavy I/O: log loading, message verification, and cycle detection.
// l1Heads must be the snapshot from observeRound — see verifyFn doc comment.
func (i *Interop) verify(ts uint64, blocksAtTS map[eth.ChainID]eth.BlockID, l1Heads map[eth.ChainID]eth.BlockID) (Result, error) {
	view, err := i.resolveFrontierVerificationView(blocksAtTS)
	if err != nil {
		return Result{}, fmt.Errorf("resolve frontier verification view: %w", err)
	}

	result, err := i.verifyFn(ts, blocksAtTS, l1Heads, view)
	if err != nil {
		return Result{}, err
	}

	cycleResult, err := i.cycleVerifyFn(ts, blocksAtTS, view)
	if err != nil {
		return Result{}, fmt.Errorf("cycle verification failed: %w", err)
	}

	if len(cycleResult.InvalidHeads) > 0 {
		if result.InvalidHeads == nil {
			result.InvalidHeads = make(map[eth.ChainID]InvalidHead)
		}
		for chainID, invalidBlock := range cycleResult.InvalidHeads {
			result.InvalidHeads[chainID] = invalidBlock
		}
	}

	return result, nil
}

// newInvalidHead constructs a fully-formed InvalidHead with the output preimage
// fields already attached. Returns an error if the output cannot be computed —
// at verification time the engine should always have the block, so failure
// indicates a transient RPC issue or a serious invariant violation.
func (i *Interop) newInvalidHead(chainID eth.ChainID, blockID eth.BlockID) (InvalidHead, error) {
	head := InvalidHead{BlockID: blockID}
	chain, ok := i.chains[chainID]
	if !ok {
		return head, fmt.Errorf("chain %s not found", chainID)
	}
	outputV0, err := chain.OutputV0AtBlockNumber(i.ctx, blockID.Number)
	if err != nil {
		return head, fmt.Errorf("chain %s: failed to compute OutputV0 for block %d: %w", chainID, blockID.Number, err)
	}
	if outputV0.BlockHash != blockID.Hash {
		return head, fmt.Errorf("chain %s: block %d hash changed (expected %s, got %s): possible reorg",
			chainID, blockID.Number, blockID.Hash, outputV0.BlockHash)
	}
	head.StateRoot = outputV0.StateRoot
	head.MessagePasserStorageRoot = outputV0.MessagePasserStorageRoot
	return head, nil
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
		i.metrics.InteropRewinds.Inc()
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
		for chainID, invalidHead := range pending.Result.InvalidHeads {
			invalidations = append(invalidations, PendingInvalidation{
				ChainID:                  chainID,
				BlockID:                  invalidHead.BlockID,
				Timestamp:                pending.Result.Timestamp,
				StateRoot:                invalidHead.StateRoot,
				MessagePasserStorageRoot: invalidHead.MessagePasserStorageRoot,
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
			if err := i.invalidateBlock(p.ChainID, p.BlockID, p.Timestamp, p.StateRoot, p.MessagePasserStorageRoot); err != nil {
				i.log.Error("invalidation failed, transition preserved for retry on restart",
					"chain", p.ChainID, "block", p.BlockID, "err", err)
				failedAny = true
			} else {
				i.metrics.InteropInvalidations.WithLabelValues(p.ChainID.String()).Inc()
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
		i.metrics.InteropTimestampsVerified.Inc()
		i.metrics.InteropVerifiedTimestamp.Set(float64(pending.Result.Timestamp))
		// L1Inclusion is the max L1 block used for derivation across all chains at this
		// timestamp. It can exceed some chains' actual CurrentL1 — e.g. chain A derived
		// from L1 1000 while chain B derived from L1 990. Chain B may then advance to
		// the next timestamp (finding its batch at L1 995) without ever reaching L1 1000.
		// Cap at the min of all nodes' CurrentL1 so this field is individually safe, not
		// just safe when aggregated with chain CurrentL1s via syncstatus.Aggregate.
		currentL1 := pending.Result.L1Inclusion
		if localL1, err := i.collectCurrentL1(); err != nil {
			i.log.Warn("failed to collect node CurrentL1 on advance, using L1Inclusion", "err", err)
		} else if localL1.Number < currentL1.Number {
			currentL1 = localL1
		}
		i.mu.Lock()
		i.currentL1 = currentL1
		i.mu.Unlock()
		return true, nil
	}

	return false, nil
}

func (i *Interop) buildRewindPlan(lastTS uint64) (RewindPlan, error) {
	plan := RewindPlan{
		RewindAtOrAfter: lastTS,
	}

	resetEngines, err := i.shouldResetEnginesOnRewind(lastTS)
	if err != nil {
		return RewindPlan{}, err
	}
	if resetEngines {
		if lastTS == 0 {
			return RewindPlan{}, fmt.Errorf("cannot reset engines before timestamp 0")
		}
		resetTo := lastTS - 1
		plan.ResetAllChainsTo = &resetTo
	}

	first, err := i.firstVerifiableTimestamp(i.ctx)
	if err != nil {
		return RewindPlan{}, err
	}
	if lastTS <= first {
		return plan, nil
	}

	rewindTargetTS := lastTS - 1
	prevResult, err := i.verifiedDB.Get(rewindTargetTS)
	if err != nil {
		return RewindPlan{}, fmt.Errorf("read previous verified result at %d: %w", rewindTargetTS, err)
	}
	plan.TargetHeads = prevResult.L2Heads
	return plan, nil
}

func (i *Interop) shouldResetEnginesOnRewind(timestamp uint64) (bool, error) {
	for chainID, chain := range i.chains {
		hasDenied, err := chain.HasDeniedAtOrAfterTimestamp(timestamp)
		if err != nil {
			return false, fmt.Errorf("chain %s: inspect deny list for rewind: %w", chainID, err)
		}
		if hasDenied {
			return true, nil
		}
	}
	return false, nil
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
	}

	if plan.TargetHeads == nil {
		for chainID, db := range i.logsDBs {
			if err := db.Clear(&noopInvalidator{}); err != nil {
				i.log.Error("failed to clear logsDB on full rewind", "chain", chainID, "err", err)
				recordErr(fmt.Errorf("chain %s: clear logsDB on full rewind: %w", chainID, err))
			}
		}
		if len(allErrs) == 0 {
			i.resetChainEnginesIfNeeded(plan, sortedChainIDs, recordErr)
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

	if len(allErrs) == 0 {
		i.resetChainEnginesIfNeeded(plan, sortedChainIDs, recordErr)
	}
	return errors.Join(allErrs...)
}

func (i *Interop) resetChainEnginesIfNeeded(plan RewindPlan, sortedChainIDs []eth.ChainID, recordErr func(error)) {
	if plan.ResetAllChainsTo == nil {
		return
	}
	for _, chainID := range sortedChainIDs {
		i.log.Warn("rewinding chain engine after pruning deny-list entries",
			"chain", chainID, "rewindToTimestamp", *plan.ResetAllChainsTo)
		if err := i.chains[chainID].RewindEngine(i.ctx, *plan.ResetAllChainsTo, eth.BlockRef{}); err != nil {
			i.log.Error("failed to reset chain engine after pruning deny-list entries", "chain", chainID, "err", err)
			recordErr(fmt.Errorf("chain %s: reset chain engine after pruning deny-list entries: %w", chainID, err))
		}
	}
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
// Timestamps before the first verifiable timestamp are already covered by
// pre-activation consensus or by the safe-head startup handoff.
func (i *Interop) VerifiedAtTimestamp(ts uint64) (bool, error) {
	if ts < i.activationTimestamp {
		return true, nil
	}
	firstVerifiable, err := i.firstVerifiableTimestamp(i.ctx)
	if err != nil {
		return false, err
	}
	if ts < firstVerifiable {
		return true, nil
	}
	return i.verifiedDB.Has(ts)
}

// IsActiveAt reports whether the interop verifier is responsible for verifying
// L2 content at the given timestamp. Returns false for timestamps strictly
// before the configured activation timestamp.
func (i *Interop) IsActiveAt(ts uint64) bool {
	return ts >= i.activationTimestamp
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
func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32) error {
	chain, ok := i.chains[chainID]
	if !ok {
		return fmt.Errorf("chain %s not found", chainID)
	}
	_, err := chain.InvalidateBlock(i.ctx, blockID.Number, blockID.Hash, decisionTimestamp, stateRoot, messagePasserStorageRoot)
	return err
}
