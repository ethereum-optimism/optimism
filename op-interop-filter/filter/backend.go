package filter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// Backend coordinates chain ingesters and handles CheckAccessList requests.
// Failsafe is enabled if manually set OR if any chain ingester has an error.
type Backend struct {
	log     log.Logger
	metrics metrics.Metricer

	// Chain ingesters keyed by chain ID.
	chains map[eth.ChainID]ChainIngester

	// Cross-validator handles all cross-chain message validation.
	crossValidator CrossValidator

	// Manual failsafe override
	manualFailsafe atomic.Bool

	// failsafeMu serializes failsafe metric/log refreshes so concurrent error
	// transitions cannot latch a stale gauge value or interleave log lines.
	failsafeMu          sync.Mutex
	lastFailsafeSummary string

	// Passthrough mode: all transactions pass without filtering
	passthrough bool

	// Compatibility mode for legacy clients that omit executing chainID.
	legacyCheckAccessListFormat bool

	ctx    context.Context
	cancel context.CancelFunc

	reorgRecoveryEnabled bool
	reorgRecoveryWg      sync.WaitGroup

	// failsafeLogInterval is how often the active failsafe reason is re-logged
	// while failsafe is enabled. failsafeWg tracks that heartbeat goroutine.
	failsafeLogInterval time.Duration
	failsafeWg          sync.WaitGroup
}

// BackendParams contains parameters for creating a Backend.
type BackendParams struct {
	Logger                      log.Logger
	Metrics                     metrics.Metricer
	Chains                      map[eth.ChainID]ChainIngester
	CrossValidator              CrossValidator
	Passthrough                 bool
	LegacyCheckAccessListFormat bool

	ReorgRecoveryEnabled bool
	FailsafeLogInterval  time.Duration
}

// Failsafe reason labels that are not chain-ingester errors. Chain errors use
// IngesterErrorReason.String() (reorg, conflict, data_corruption, invalid_log).
const (
	failsafeReasonManual          = "manual"
	failsafeReasonCrossValidation = "cross_validation"
	failsafeReasonNone            = "none"
)

// defaultFailsafeLogInterval is the fallback heartbeat interval used when
// BackendParams.FailsafeLogInterval is unset (e.g. in tests). Production wires
// the --failsafe-log-interval flag (also defaulting to 1m).
const defaultFailsafeLogInterval = time.Minute

// allFailsafeReasons is the full set of reason labels the failsafe_reason_active
// gauge can emit. Every refresh sets all of them so a cleared reason drops back
// to 0 instead of holding its last value.
var allFailsafeReasons = []string{
	failsafeReasonManual,
	ErrorReorg.String(),
	ErrorConflict.String(),
	ErrorDataCorruption.String(),
	ErrorInvalidExecutingMessage.String(),
	failsafeReasonCrossValidation,
}

// failsafeObserver is implemented by components whose error state feeds into
// FailsafeEnabled. The backend wires a callback so the failsafe metrics and logs
// are refreshed whenever a component enters or clears its error state, not only
// on the manual SetFailsafeEnabled path.
type failsafeObserver interface {
	SetOnFailsafeChange(func())
}

// NewBackend creates a new Backend instance with the provided components.
func NewBackend(parentCtx context.Context, params BackendParams) *Backend {
	ctx, cancel := context.WithCancel(parentCtx)

	b := &Backend{
		log:                         params.Logger,
		metrics:                     params.Metrics,
		chains:                      params.Chains,
		crossValidator:              params.CrossValidator,
		passthrough:                 params.Passthrough,
		legacyCheckAccessListFormat: params.LegacyCheckAccessListFormat,
		ctx:                         ctx,
		cancel:                      cancel,
		reorgRecoveryEnabled:        params.ReorgRecoveryEnabled,
		failsafeLogInterval:         params.FailsafeLogInterval,
	}
	if b.failsafeLogInterval <= 0 {
		b.failsafeLogInterval = defaultFailsafeLogInterval
	}

	// Initialize so the first benign refresh (state == off) does not log a
	// spurious "cleared" transition.
	b.lastFailsafeSummary = failsafeReasonNone

	// Refresh failsafe metrics/logs whenever a component's error state changes,
	// so auto-triggered failsafe (chain or cross-validator errors) is reflected,
	// not just the manual SetFailsafeEnabled path.
	for _, ingester := range b.chains {
		if o, ok := ingester.(failsafeObserver); ok {
			o.SetOnFailsafeChange(b.refreshFailsafe)
		}
	}
	if o, ok := b.crossValidator.(failsafeObserver); ok {
		o.SetOnFailsafeChange(b.refreshFailsafe)
	}

	return b
}

// Start starts all chain ingesters and the cross-validator
func (b *Backend) Start(ctx context.Context) error {
	b.log.Info("Starting backend")

	for chainID, ingester := range b.chains {
		if err := ingester.Start(); err != nil {
			return fmt.Errorf("failed to start chain ingester for %v: %w", chainID, err)
		}
	}

	if err := b.crossValidator.Start(); err != nil {
		return fmt.Errorf("failed to start cross-validator: %w", err)
	}

	if b.reorgRecoveryEnabled {
		b.reorgRecoveryWg.Add(1)
		go b.runReorgRecovery(b.ctx)
	}

	b.failsafeWg.Add(1)
	go b.runFailsafeHeartbeat(b.ctx)

	return nil
}

// Stop stops all chain ingesters and the cross-validator
func (b *Backend) Stop(ctx context.Context) error {
	b.log.Info("Stopping backend")
	b.cancel()

	var result error

	b.reorgRecoveryWg.Wait()
	b.failsafeWg.Wait()

	if err := b.crossValidator.Stop(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to stop cross-validator: %w", err))
	}

	for chainID, ingester := range b.chains {
		if err := ingester.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop chain ingester for %v: %w", chainID, err))
		}
	}

	return result
}

// FailsafeEnabled returns true if failsafe is manually enabled OR any chain has an error
// OR the cross-validator has an error.
func (b *Backend) FailsafeEnabled() bool {
	return b.manualFailsafe.Load() || len(b.GetChainErrors()) > 0 || b.crossValidator.Error() != nil
}

// SetFailsafeEnabled sets the manual failsafe override.
func (b *Backend) SetFailsafeEnabled(enabled bool) {
	b.manualFailsafe.Store(enabled)
	b.refreshFailsafe()
}

// refreshFailsafe recomputes the failsafe metrics and logs reason transitions.
// It is the single choke point for failsafe observability: invoked on the manual
// toggle and via the onFailsafeChange callback whenever a chain ingester or the
// cross-validator changes error state.
//
// failsafeMu serializes compute+record+log so concurrent transitions cannot
// latch a stale gauge value (the last writer observes all completed state
// changes) or interleave log lines.
func (b *Backend) refreshFailsafe() {
	b.failsafeMu.Lock()
	defer b.failsafeMu.Unlock()

	manual := b.manualFailsafe.Load()
	chainErrs := b.GetChainErrors()
	cvErr := b.crossValidator.Error()

	enabled := manual || len(chainErrs) > 0 || cvErr != nil

	// Build the active-reason set (deduped across chains: a reason is "active"
	// if at least one source currently holds it).
	active := make(map[string]bool, len(allFailsafeReasons))
	if manual {
		active[failsafeReasonManual] = true
	}
	for _, ie := range chainErrs {
		active[ie.Reason.String()] = true
	}
	if cvErr != nil {
		active[failsafeReasonCrossValidation] = true
	}

	b.metrics.RecordFailsafeEnabled(enabled)
	for _, reason := range allFailsafeReasons {
		b.metrics.RecordFailsafeReason(reason, active[reason])
	}

	// Log only when the reason set changes, to avoid spam from frequent refreshes.
	summary := failsafeSummary(manual, chainErrs, cvErr)
	if summary == b.lastFailsafeSummary {
		return
	}
	b.lastFailsafeSummary = summary
	if enabled {
		b.log.Warn("Failsafe active", "reasons", summary,
			"detail", failsafeReasonDetail(manual, chainErrs, cvErr))
	} else {
		b.log.Info("Failsafe cleared")
	}
}

// failsafeSummary renders the active failsafe reasons as a stable, greppable
// string with per-chain detail, e.g. "manual,chain[901]=reorg,cross_validation".
// Returns failsafeReasonNone when nothing is active. Chain IDs are sorted for
// deterministic output (so change-detection logging does not fire spuriously).
func failsafeSummary(manual bool, chainErrs map[eth.ChainID]*IngesterError, cvErr *ValidatorError) string {
	var parts []string
	if manual {
		parts = append(parts, failsafeReasonManual)
	}
	ids := make([]eth.ChainID, 0, len(chainErrs))
	for id := range chainErrs {
		ids = append(ids, id)
	}
	eth.SortChainID(ids) // numeric (not lexicographic) order for stable, readable summaries
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("chain[%s]=%s", id, chainErrs[id].Reason))
	}
	if cvErr != nil {
		parts = append(parts, failsafeReasonCrossValidation)
	}
	if len(parts) == 0 {
		return failsafeReasonNone
	}
	return strings.Join(parts, ",")
}

// failsafeReasonDetail renders the active failsafe reasons together with each
// source's underlying error message — the "why" behind the failsafe. Returns
// failsafeReasonNone when nothing is active.
func failsafeReasonDetail(manual bool, chainErrs map[eth.ChainID]*IngesterError, cvErr *ValidatorError) string {
	var parts []string
	if manual {
		parts = append(parts, "manual override")
	}
	ids := make([]eth.ChainID, 0, len(chainErrs))
	for id := range chainErrs {
		ids = append(ids, id)
	}
	eth.SortChainID(ids)
	for _, id := range ids {
		ie := chainErrs[id]
		parts = append(parts, fmt.Sprintf("chain[%s] %s: %s", id, ie.Reason, ie.Message))
	}
	if cvErr != nil {
		parts = append(parts, fmt.Sprintf("cross-validation: %s", cvErr.Message))
	}
	if len(parts) == 0 {
		return failsafeReasonNone
	}
	return strings.Join(parts, "; ")
}

// runFailsafeHeartbeat periodically re-logs the active failsafe reason while
// failsafe remains on. The transition log in refreshFailsafe fires only once
// (when the reason set changes), so without this a long-lived failsafe (e.g. a
// reorg awaiting recovery) would stop appearing in recent logs.
func (b *Backend) runFailsafeHeartbeat(ctx context.Context) {
	defer b.failsafeWg.Done()

	ticker := time.NewTicker(b.failsafeLogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.logFailsafeIfActive()
		}
	}
}

// logFailsafeIfActive logs the current failsafe reasons at Warn if failsafe is
// active; no-op otherwise.
func (b *Backend) logFailsafeIfActive() {
	manual := b.manualFailsafe.Load()
	chainErrs := b.GetChainErrors()
	cvErr := b.crossValidator.Error()
	if !manual && len(chainErrs) == 0 && cvErr == nil {
		return
	}
	b.log.Warn("Failsafe still active",
		"reasons", failsafeSummary(manual, chainErrs, cvErr),
		"detail", failsafeReasonDetail(manual, chainErrs, cvErr))
}

// GetChainErrors returns all chains that are in an error state
func (b *Backend) GetChainErrors() map[eth.ChainID]*IngesterError {
	errs := make(map[eth.ChainID]*IngesterError)
	for chainID, ingester := range b.chains {
		if err := ingester.Error(); err != nil {
			errs[chainID] = err
		}
	}
	return errs
}

// Ready returns true if all chains have completed backfill
func (b *Backend) Ready() bool {
	for _, ingester := range b.chains {
		if !ingester.Ready() {
			return false
		}
	}

	return len(b.chains) > 0
}

// supportedSafetyLevel returns true if the safety level is supported for access list checks.
func supportedSafetyLevel(level safety.Level) bool {
	return level == safety.LocalUnsafe || level == safety.CrossUnsafe
}

// classifyRejectionReason categorizes an error from CheckAccessList into a rejection reason label.
func classifyRejectionReason(err error) string {
	switch {
	case errors.Is(err, interop.ErrFailsafeEnabled):
		return "failsafe"
	case errors.Is(err, interop.ErrUnknownChain):
		return "unknown_chain"
	case errors.Is(err, interop.ErrConflict):
		return "expired_message"
	default:
		return "invalid_executing_message"
	}
}

// CheckAccessList validates the given access list entries.
func (b *Backend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety safety.Level, execDescriptor messages.ExecutingDescriptor) error {

	start := time.Now()
	defer func() {
		b.metrics.RecordCheckAccessListDuration(time.Since(start).Seconds())
	}()

	if b.passthrough {
		b.metrics.RecordCheckAccessList(true)
		return nil
	}

	if b.FailsafeEnabled() {
		b.metrics.RecordCheckAccessList(false)
		b.metrics.RecordCheckAccessListRejection("failsafe")
		return interop.ErrFailsafeEnabled
	}

	if !b.Ready() {
		b.metrics.RecordCheckAccessList(false)
		b.metrics.RecordCheckAccessListRejection("failsafe")
		b.log.Debug("Backend not ready; rejecting access list check")
		return interop.ErrUninitialized
	}

	if !supportedSafetyLevel(minSafety) {
		b.metrics.RecordCheckAccessList(false)
		b.metrics.RecordCheckAccessListRejection("invalid_executing_message")
		return fmt.Errorf("unsupported safety level %s: only %s and %s are supported",
			minSafety, safety.LocalUnsafe, safety.CrossUnsafe)
	}

	if _, ok := b.chains[execDescriptor.ChainID]; !ok {
		if !b.legacyCheckAccessListFormat {
			b.metrics.RecordCheckAccessList(false)
			b.metrics.RecordCheckAccessListRejection("unknown_chain")
			return fmt.Errorf("executing chain %s: %w", execDescriptor.ChainID, interop.ErrUnknownChain)
		}
		b.log.Debug("Supporting legacy check access list format", "executing_chain", execDescriptor.ChainID)
	}

	remaining := inboxEntries
	for len(remaining) > 0 {
		var access messages.Access
		var err error
		remaining, access, err = messages.ParseAccess(remaining)
		if err != nil {
			b.metrics.RecordCheckAccessList(false)
			b.metrics.RecordCheckAccessListRejection("parse_error")
			return fmt.Errorf("failed to parse access entry: %w", err)
		}

		if err := b.crossValidator.ValidateAccessEntry(access, minSafety, execDescriptor); err != nil {
			b.metrics.RecordCheckAccessList(false)
			b.metrics.RecordCheckAccessListRejection(classifyRejectionReason(err))
			return err
		}
	}

	b.metrics.RecordCheckAccessList(true)
	return nil
}

// GetBlockHashByNumber returns the latest block hash or the block hash at a specific height for the given chain.
// Accepts rpc.BlockNumber: "latest" or a numeric block number. Other named tags are not supported.
func (b *Backend) GetBlockHashByNumber(chainID eth.ChainID, blockNum rpc.BlockNumber) (common.Hash, error) {
	ingester, ok := b.chains[chainID]
	if !ok {
		return common.Hash{}, fmt.Errorf("chain %s: %w", chainID, interop.ErrUnknownChain)
	}

	if blockNum == rpc.LatestBlockNumber {
		block, ok := ingester.LatestBlock()
		if !ok {
			return common.Hash{}, fmt.Errorf("latest block for chain %s: %w", chainID, ethereum.NotFound)
		}
		return block.Hash, nil
	}
	if blockNum < 0 {
		return common.Hash{}, fmt.Errorf("unsupported block tag %q: only \"latest\" and block numbers are supported", blockNum)
	}

	blockHash, ok := ingester.BlockHashByNumber(uint64(blockNum))
	if !ok {
		return common.Hash{}, fmt.Errorf("block %d for chain %s: %w", blockNum, chainID, ethereum.NotFound)
	}
	return blockHash, nil
}
