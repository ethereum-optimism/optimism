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

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/clock"
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

// DefaultLogBackfillDepth matches the interop message expiry window so backfill
// covers every initiating message that could still be referenced by an
// executing message.
const DefaultLogBackfillDepth = time.Duration(depset.MessageExpiryTimeSecondsInterop) * time.Second

// Interop activity state values exposed via supernode_interop_activity_state gauge.
const (
	InteropStateNotStarted       = 0
	InteropStateColdStartWaiting = 1
	InteropStateRunning          = 2
	InteropStateHalted           = 3
)

// InteropActivationTimestampFlag is the CLI flag for the interop activation timestamp.
var InteropActivationTimestampFlag = &cli.Uint64Flag{
	Name:    "interop.activation-timestamp",
	Usage:   "Override the interop activation timestamp derived from rollup configs",
	EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "INTEROP_ACTIVATION_TIMESTAMP"),
	Value:   0,
}

// InteropLogBackfillDepthFlag extends initiating-message log ingestion backward from the startup boundary by this duration (clamped to activation).
var InteropLogBackfillDepthFlag = &cli.DurationFlag{
	Name:    "interop.log-backfill-depth",
	Usage:   "Duration to pre-ingest logs behind the tip before interop validation. Never loads logs before interop.activation-timestamp. Set to 0 to disable.",
	EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "INTEROP_LOG_BACKFILL_DEPTH"),
	Value:   DefaultLogBackfillDepth,
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

	// verificationStartTimestamp is the first L2 timestamp the main loop
	// attempts to verify. Set exactly once during tryInitFromVerifiedDB
	// (resume path) or by advanceColdStartInit, then immutable.
	verificationStartTimestamp uint64

	// initialized is set true once verificationStartTimestamp has been
	// chosen. RPC accessors return ErrNotStarted while false.
	initialized atomic.Bool

	// waitingForSync is true between tryInitFromVerifiedDB deferring
	// cold-start origin selection and the loop iteration that completes it.
	// Only read/written by the main loop goroutine; no mutex needed.
	waitingForSync bool

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

	// backfillAttempts counts cold-start init iterations since the most
	// recent Start. Read by integration tests to confirm the retry loop has
	// engaged.
	backfillAttempts atomic.Int32
	// backfillCompleted is set true once cold-start init finishes — either
	// backfill ran to completion or resume skipped it. Read by integration
	// tests to gate on cold-start init finishing.
	backfillCompleted atomic.Bool

	// verifyRounds counts verification loop iterations for periodic progress logging.
	verifyRounds atomic.Int32

	// l1Checker must be non-nil whenever observeRound runs. Production sets it
	// via New; tests inject noopL1Checker.
	l1Checker l1ConsistencyChecker

	logBackfillDepth time.Duration
	metrics          *resources.SupernodeMetrics

	// clock is used for all wall-clock reads and sleeps so deterministic
	// tests can inject a fake. Defaults to clock.SystemClock in New.
	clock clock.Clock
}

func (i *Interop) Name() string {
	return "interop"
}

// firstVerifiableTimestamp is the earliest timestamp the verifier covers.
// If commits exist, the verifiedDB's first committed timestamp is the
// authoritative lower bound (it cannot move). Otherwise it is the chosen
// verificationStartTimestamp. Returns ErrNotStarted until initialization
// completes.
func (i *Interop) firstVerifiableTimestamp() (uint64, error) {
	if i.verifiedDB != nil {
		if first, initialized := i.verifiedDB.FirstTimestamp(); initialized {
			return first, nil
		}
	}
	if !i.initialized.Load() {
		return 0, ErrNotStarted
	}
	return i.verificationStartTimestamp, nil
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
		clock:               clock.SystemClock,
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

	i.tryInitFromVerifiedDB()
	return i.runLoop()
}

// tryInitFromVerifiedDB selects verificationStartTimestamp from verifiedDB if
// any commit exists. Otherwise it defers to the cold-start loop, which waits
// for every chain to record a first SafeDB entry before picking an origin.
// Wall-clock time is not consulted: chain derivation progress is the only
// authoritative signal for "where we are" relative to activation.
func (i *Interop) tryInitFromVerifiedDB() {
	if lastTS, ok := i.verifiedDB.LastTimestamp(); ok {
		i.verificationStartTimestamp = lastTS + 1
		i.initialized.Store(true)
		i.backfillCompleted.Store(true) // resume skips backfill
		i.metrics.InteropActivityState.Set(InteropStateRunning)
		i.log.Info("interop resuming from verifiedDB",
			"verificationStartTimestamp", i.verificationStartTimestamp,
			"activationTimestamp", i.activationTimestamp)
		return
	}
	i.waitingForSync = true
	i.metrics.InteropActivityState.Set(InteropStateColdStartWaiting)
	i.log.Info("interop cold start; waiting for SafeDB entries on every chain",
		"activationTimestamp", i.activationTimestamp)
}

// runLoop drives initialization and verification. Each iteration performs
// exactly one of two actions and then sleeps for the duration the action
// chose: waitForColdStartInit while cold-start initialization is in
// progress, otherwise progress to verify the next round.
func (i *Interop) runLoop() error {
	for {
		var (
			sleep time.Duration
			err   error
		)
		if i.waitingForSync {
			sleep, err = i.waitForColdStartInit()
		} else {
			sleep, err = i.progress()
		}
		if err != nil {
			return err
		}
		if sleep > 0 {
			if err := i.clock.SleepCtx(i.ctx, sleep); err != nil {
				return err
			}
		}
	}
}

// waitForColdStartInit runs one cold-start initialization step. Returns
// (0, nil) if the step advanced (so the loop runs again immediately to either
// finish initialization or start progressing), (backoffPeriod, nil) if no
// progress was made yet, or (errorBackoffPeriod, nil) on any error.
//
// Cold-start init runs concurrently with chain-container startup, so every
// failure mode here (VN not yet attached, transient RPC errors, EL not
// ready) is expected during the startup window and must not kill the
// activity. Cold-start has no path to a permanent failure: none of the calls
// it makes return ErrHistoryUnavailable, and any real corruption surfaces in
// the verification loop once initialization completes.
func (i *Interop) waitForColdStartInit() (time.Duration, error) {
	advanced, err := i.advanceColdStartInit()
	if err != nil {
		i.metrics.ActivityErrors.WithLabelValues("interop", "cold_start_init").Inc()
		attempts := i.backfillAttempts.Load()
		i.log.Warn("interop cold start step failed, will retry",
			"err", err, "attempts", attempts)
		return errorBackoffPeriod, nil
	}
	if !advanced {
		return backoffPeriod, nil
	}
	i.waitingForSync = false
	i.initialized.Store(true)
	i.metrics.InteropActivityState.Set(InteropStateRunning)
	i.log.Info("interop cold start complete",
		"activationTimestamp", i.activationTimestamp,
		"verificationStartTimestamp", i.verificationStartTimestamp)
	return 0, nil
}

// progress runs one verification step. Returns (0, nil) when forward progress
// was made (so the loop runs again immediately), (backoffPeriod, nil) when
// the round was a no-op, (errorBackoffPeriod, nil) on a recoverable error,
// or a non-nil error to terminate the loop.
func (i *Interop) progress() (time.Duration, error) {
	round := i.verifyRounds.Add(1)
	madeProgress, err := i.progressAndRecord()
	if err != nil {
		if errors.Is(err, cc.ErrHistoryUnavailable) {
			i.metrics.ActivityErrors.WithLabelValues("interop", "history_unavailable").Inc()
			i.metrics.InteropActivityState.Set(InteropStateHalted)
			i.log.Error("interop activity halted: SafeDB history unavailable on this node", "err", err,
				"remediation", "reseed data dir, advance interop.activation-timestamp past the gap, or rederive from L1")
			return 0, fmt.Errorf("interop halted due to unavailable history: %w", err)
		}
		i.metrics.ActivityErrors.WithLabelValues("interop", "progress").Inc()
		i.log.Error("failed to progress and record interop", "err", err)
		return errorBackoffPeriod, nil
	}
	if round%30 == 0 {
		lastTS, _ := i.verifiedDB.LastTimestamp()
		var tipTS uint64
		fields := []any{
			"round", round, "madeProgress", madeProgress,
		}
		for _, chain := range i.chains {
			status, err := chain.SyncStatus(i.ctx)
			if err != nil {
				continue
			}
			if status.UnsafeL2.Time > tipTS {
				tipTS = status.UnsafeL2.Time
			}
			fields = append(fields,
				fmt.Sprintf("chain_%s", chain.ID()),
				fmt.Sprintf("safe=%d pending_safe=%d unsafe=%d",
					status.SafeL2.Number, status.PendingSafeL2.Number, status.UnsafeL2.Number))
		}
		var behind uint64
		if tipTS > lastTS {
			behind = tipTS - lastTS
		}
		fields = append(fields,
			"lastVerifiedTimestamp", lastTS, "tipTimestamp", tipTS, "behindSeconds", behind)
		i.log.Info("interop verification progress", fields...)
	}
	if !madeProgress {
		return backoffPeriod, nil
	}
	return 0, nil
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

	verifyStart := i.clock.Now()
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
	i.metrics.InteropVerificationDuration.Observe(i.clock.Since(verifyStart).Seconds())
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
		next, err := i.firstVerifiableTimestamp()
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
	case DecisionAdvance:
		result := output.Result
		return PendingTransition{
			Decision: output.Decision,
			Result:   &result,
		}, nil
	case DecisionInvalidate:
		result := output.Result
		parentPayloads, err := i.captureInvalidationParentPayloads(result.InvalidHeads)
		if err != nil {
			return PendingTransition{}, fmt.Errorf("capture invalidation parent payloads: %w", err)
		}
		return PendingTransition{
			Decision:                   output.Decision,
			Result:                     &result,
			InvalidationParentPayloads: parentPayloads,
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

// captureInvalidationParentPayloads fetches, for every invalidated chain, the canonical
// parent payload (height-1) the rewind will restore as the new unsafe head. The payloads
// are persisted in the WAL so apply does not depend on the live EL still having them.
// Any fetch failure aborts the build; the decision will be re-evaluated next round.
func (i *Interop) captureInvalidationParentPayloads(invalidHeads map[eth.ChainID]InvalidHead) (map[eth.ChainID]*eth.ExecutionPayloadEnvelope, error) {
	if len(invalidHeads) == 0 {
		return nil, nil
	}
	parents := make(map[eth.ChainID]*eth.ExecutionPayloadEnvelope, len(invalidHeads))
	for chainID, head := range invalidHeads {
		if head.BlockID.Number == 0 {
			return nil, fmt.Errorf("chain %s: cannot invalidate genesis block (height=0)", chainID)
		}
		chain, ok := i.chains[chainID]
		if !ok {
			return nil, fmt.Errorf("chain %s: not configured", chainID)
		}
		invalidatedRef, err := chain.PayloadByHash(i.ctx, head.BlockID.Hash)
		if err != nil {
			return nil, fmt.Errorf("chain %s: fetch invalidated block %s: %w", chainID, head.BlockID.Hash, err)
		}
		parentEnvelope, err := chain.PayloadByHash(i.ctx, invalidatedRef.ExecutionPayload.ParentHash)
		if err != nil {
			return nil, fmt.Errorf("chain %s: fetch parent payload %s: %w",
				chainID, invalidatedRef.ExecutionPayload.ParentHash, err)
		}
		parents[chainID] = parentEnvelope
	}
	return parents, nil
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
			parentPayload, ok := pending.InvalidationParentPayloads[p.ChainID]
			if !ok || parentPayload == nil {
				// Build path guarantees a parent payload for every invalidated chain.
				// Missing here means a malformed (older-format / corrupted) WAL record —
				// surface and preserve the transition for operator intervention.
				i.log.Error("invalidation parent payload missing from WAL — invalidation cannot proceed",
					"chain", p.ChainID, "block", p.BlockID)
				failedAny = true
				continue
			}
			if err := i.invalidateBlock(p.ChainID, p.BlockID, p.Timestamp, p.StateRoot, p.MessagePasserStorageRoot, parentPayload); err != nil {
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

	first, err := i.firstVerifiableTimestamp()
	if err != nil {
		return RewindPlan{}, err
	}
	if lastTS <= first {
		if plan.ResetAllChainsTo != nil {
			payloads, err := i.captureRewindPayloadsAtTimestamp(*plan.ResetAllChainsTo)
			if err != nil {
				return RewindPlan{}, fmt.Errorf("capture reset target payloads at %d: %w", *plan.ResetAllChainsTo, err)
			}
			plan.TargetPayloads = payloads
		}
		return plan, nil
	}

	rewindTargetTS := lastTS - 1
	prevResult, err := i.verifiedDB.Get(rewindTargetTS)
	if err != nil {
		return RewindPlan{}, fmt.Errorf("read previous verified result at %d: %w", rewindTargetTS, err)
	}
	plan.TargetHeads = prevResult.L2Heads

	// Capture each chain's target payload while it is still canonical. Any failure aborts
	// the build; the decision will be re-evaluated next round.
	if plan.ResetAllChainsTo != nil && len(plan.TargetHeads) > 0 {
		payloads, err := i.captureRewindPayloadsForHeads(plan.TargetHeads, rewindTargetTS)
		if err != nil {
			return RewindPlan{}, err
		}
		plan.TargetPayloads = payloads
	}

	return plan, nil
}

func (i *Interop) captureRewindPayloadsForHeads(heads map[eth.ChainID]eth.BlockID, timestamp uint64) (map[eth.ChainID]*eth.ExecutionPayloadEnvelope, error) {
	if len(heads) == 0 {
		return nil, nil
	}
	payloads := make(map[eth.ChainID]*eth.ExecutionPayloadEnvelope, len(heads))
	for chainID, head := range heads {
		chain, ok := i.chains[chainID]
		if !ok {
			return nil, fmt.Errorf("chain %s referenced in target heads but not configured", chainID)
		}
		envelope, err := chain.PayloadByHash(i.ctx, head.Hash)
		if err != nil {
			return nil, fmt.Errorf("chain %s: fetch target payload %s for rewind to ts=%d: %w",
				chainID, head.Hash, timestamp, err)
		}
		payloads[chainID] = envelope
	}
	return payloads, nil
}

func (i *Interop) captureRewindPayloadsAtTimestamp(timestamp uint64) (map[eth.ChainID]*eth.ExecutionPayloadEnvelope, error) {
	payloads := make(map[eth.ChainID]*eth.ExecutionPayloadEnvelope, len(i.chains))
	for chainID, chain := range i.chains {
		number, err := chain.TimestampToBlockNumber(i.ctx, timestamp)
		if err != nil {
			return nil, fmt.Errorf("chain %s: compute rewind target number for timestamp %d: %w", chainID, timestamp, err)
		}
		envelope, err := chain.PayloadByNumber(i.ctx, number)
		if err != nil {
			return nil, fmt.Errorf("chain %s: fetch reset target payload number %d for timestamp %d: %w",
				chainID, number, timestamp, err)
		}
		payloads[chainID] = envelope
	}
	return payloads, nil
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
			if err := db.Clear(); err != nil {
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
		if err := db.Rewind(expectedHead); err != nil {
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
		target, ok := plan.TargetPayloads[chainID]
		if !ok {
			// The build path guarantees a TargetPayloads entry for every chain in TargetHeads.
			// If we get here, the WAL record is malformed (older format or corruption) — surface
			// it rather than re-deriving from the live EL, which could pick up a stale synthetic
			// block from a prior crashed attempt.
			recordErr(fmt.Errorf("chain %s: missing target payload in WAL'd rewind plan (rewindToTimestamp=%d)",
				chainID, *plan.ResetAllChainsTo))
			continue
		}
		i.log.Warn("rewinding chain engine after pruning deny-list entries",
			"chain", chainID, "rewindToTimestamp", *plan.ResetAllChainsTo,
			"targetHash", target.ExecutionPayload.BlockHash)
		if err := i.chains[chainID].RewindEngine(i.ctx, target, eth.BlockRef{}); err != nil {
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

// CurrentL1 returns the L1 block currently being processed by the interop
// verifier. Every L1 block strictly below CurrentL1.Number has been fully
// considered for interop (i.e. used to verify every L2 timestamp whose source
// is at or below it); data at CurrentL1 itself may still be unverified, since
// L1Inclusion is monotonic in L2 timestamp and the next unverified timestamp
// can share the same L1 source. Consumers must require CurrentL1.Number > X
// to treat L1[≤X] as fully verified.
func (i *Interop) CurrentL1() eth.BlockID {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.currentL1
}

// VerifiedAtTimestamp returns whether the data is verified at the given timestamp.
// Timestamps before the first verifiable timestamp are already covered by
// pre-activation consensus or by the startup handoff.
func (i *Interop) VerifiedAtTimestamp(ts uint64) (bool, error) {
	if ts < i.activationTimestamp {
		return true, nil
	}
	firstVerifiable, err := i.firstVerifiableTimestamp()
	if err != nil {
		return false, err
	}
	if ts < firstVerifiable {
		return true, nil
	}
	return i.verifiedDB.Has(ts)
}

// VerifiedResultAtTimestamp returns the committed VerifiedResult for ts plus
// the verifier's CurrentL1 captured atomically with the verifiedDB read.
//   - ts < activationTimestamp           → ErrNotActive
//   - ts < firstVerifiableTimestamp      → ErrBeforeVerifiedDB
//   - verifiedDB.Get returns ErrNotFound → ethereum.NotFound
//   - else                               → the stored VerifiedResult
//
// The local ErrNotFound is translated to the standard ethereum.NotFound at the
// public boundary so consumers can errors.Is against the standard sentinel
// without taking a dependency on this package's private error.
//
// The atomic (verifiedDB, currentL1) snapshot lets callers report a
// CurrentL1 that cannot overstate verifier progress relative to the
// verifiedDB observation. The verifier holds i.mu when mutating currentL1
// (commit advances currentL1 after writing the entry; rewind zeros
// currentL1 before deleting entries), so a snapshot taken under RLock is
// consistent with one side or the other of those transitions.
func (i *Interop) VerifiedResultAtTimestamp(ts uint64) (VerifiedResult, eth.BlockID, error) {
	if ts < i.activationTimestamp {
		return VerifiedResult{}, eth.BlockID{}, ErrNotActive
	}
	// RPC is registered before Start runs; guard against a nil i.ctx.
	if i.ctx == nil {
		return VerifiedResult{}, eth.BlockID{}, ErrNotStarted
	}
	firstVerifiable, err := i.firstVerifiableTimestamp()
	if err != nil {
		return VerifiedResult{}, eth.BlockID{}, fmt.Errorf("resolve first verifiable: %w", err)
	}
	if ts < firstVerifiable {
		return VerifiedResult{}, eth.BlockID{}, ErrBeforeVerifiedDB
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	currentL1 := i.currentL1
	result, err := i.verifiedDB.Get(ts)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return VerifiedResult{}, currentL1, ethereum.NotFound
		}
		return VerifiedResult{}, currentL1, err
	}
	return result, currentL1, nil
}

// IsActiveAt reports whether the interop verifier is responsible for verifying
// L2 content at the given timestamp. Returns false for timestamps strictly
// before the configured activation timestamp.
func (i *Interop) IsActiveAt(ts uint64) bool {
	return ts >= i.activationTimestamp
}

// ActivationTimestamp returns the immutable protocol-defined interop activation
// timestamp for this verifier. Used by the SuperAuthority to compute the
// per-(chain, verifier) activation-anchor block and by RPC surfaces that expose
// the configured activation point.
func (i *Interop) ActivationTimestamp() uint64 {
	return i.activationTimestamp
}

// activationCap is the L2 timestamp the verifier reports as a cap when it has
// no verified entry for the caller's chain. The caller resolves the canonical
// L2 block at this timestamp (the pre-activation anchor). Returns 0 when no
// activation timestamp is configured — caller treats that as "no contribution".
func (i *Interop) activationCap() uint64 {
	if i.activationTimestamp == 0 {
		return 0
	}
	return i.activationTimestamp - 1
}

// LatestVerifiedL2Block returns the latest verified L2 block for chainID.
// (empty, capTimestamp, nil) means nothing verified — capTimestamp is the
// pre-activation anchor (`activationTimestamp - 1`) for the caller to resolve.
// A non-nil error means verifiedDB could not be read.
func (i *Interop) LatestVerifiedL2Block(chainID eth.ChainID) (eth.BlockID, uint64, error) {
	emptyBlock := eth.BlockID{}
	ts, ok := i.verifiedDB.LastTimestamp()
	if !ok {
		return emptyBlock, i.activationCap(), nil
	}
	res, err := i.verifiedDB.Get(ts)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return emptyBlock, i.activationCap(), nil
		}
		return emptyBlock, 0, fmt.Errorf("LatestVerifiedL2Block: read verifiedDB at %d: %w", ts, err)
	}
	head, ok := res.L2Heads[chainID]
	if !ok {
		return emptyBlock, i.activationCap(), nil
	}
	return head, ts, nil
}

// VerifiedBlockAtL1 returns the latest verified L2 block for chainID whose
// L1 inclusion is at or below l1Block. (empty, capTimestamp, nil) means no
// match — capTimestamp is the pre-activation anchor for the caller to resolve.
// A non-nil error means verifiedDB could not be read.
func (i *Interop) VerifiedBlockAtL1(chainID eth.ChainID, l1Block eth.L1BlockRef) (eth.BlockID, uint64, error) {
	if l1Block == (eth.L1BlockRef{}) {
		return eth.BlockID{}, i.activationCap(), nil
	}

	lastTs, ok := i.verifiedDB.LastTimestamp()
	if !ok {
		return eth.BlockID{}, i.activationCap(), nil
	}

	// activationTimestamp is the floor: no verified results exist before activation.
	lowerBound := i.activationTimestamp
	for ts := lastTs; ts >= lowerBound && ts <= lastTs; ts-- {
		result, err := i.verifiedDB.Get(ts)
		if err != nil {
			// Gaps and rewinds produce ErrNotFound; any other error is a
			// real read failure and must not be treated as no-match.
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return eth.BlockID{}, 0, fmt.Errorf("VerifiedBlockAtL1: read verifiedDB at %d: %w", ts, err)
		}

		if result.L1Inclusion.Number <= l1Block.Number {
			head, ok := result.L2Heads[chainID]
			if !ok {
				return eth.BlockID{}, i.activationCap(), nil
			}
			return head, ts, nil
		}
	}

	return eth.BlockID{}, i.activationCap(), nil
}

// Reset is intentionally a no-op for interop.
// Interop-owned invalidation and rewind handling is driven synchronously through
// PendingTransition application, so callback-driven resets are not part of the
// correctness path anymore.
func (i *Interop) Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef) {
}

// invalidateBlock notifies the chain container to add the block to the denylist
// and potentially rewind if the chain is currently using that block. parentPayload
// is the canonical payload at the rewind destination (height-1), captured from the WAL.
func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32, parentPayload *eth.ExecutionPayloadEnvelope) error {
	chain, ok := i.chains[chainID]
	if !ok {
		return fmt.Errorf("chain %s not found", chainID)
	}
	_, err := chain.InvalidateBlock(i.ctx, blockID.Number, blockID.Hash, decisionTimestamp, stateRoot, messagePasserStorageRoot, parentPayload)
	return err
}
