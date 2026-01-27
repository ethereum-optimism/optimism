package filter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// LockstepCrossValidator validates cross-chain executing messages and tracks
// the cross-validated timestamp.
//
// "Lockstep" refers to its synchronization model: all chains must reach the same
// timestamp before validation can advance. This is simpler but means a slow chain
// holds back validation for all chains.
//
// Simplifications in this implementation:
//   - No cycle detection: same-block executing messages are not supported
//   - Lockstep advancement: waits for ALL chains to reach timestamp T before
//     validating T, rather than validating each chain independently
//
// Future improvement: per-chain validation that tracks cross-validated timestamp
// independently for each chain, allowing faster chains to advance without waiting.
type LockstepCrossValidator struct {
	log     log.Logger
	metrics metrics.Metricer

	messageExpiryWindow uint64
	startTimestamp      uint64 // Initial timestamp to start validation from
	validationInterval  time.Duration

	// Chain ingesters keyed by chain ID (read-only after construction)
	chains map[eth.ChainID]ChainIngester

	// Single global cross-validated timestamp
	crossValidatedTs atomic.Uint64

	// Error state for validation failures
	errMu sync.RWMutex
	err   *ValidatorError

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewLockstepCrossValidator creates a new LockstepCrossValidator.
// The startTimestamp is used to initialize the cross-validated timestamp on first run.
func NewLockstepCrossValidator(
	parentCtx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	messageExpiryWindow uint64,
	startTimestamp uint64,
	validationInterval time.Duration,
	chains map[eth.ChainID]ChainIngester,
) *LockstepCrossValidator {
	ctx, cancel := context.WithCancel(parentCtx)

	return &LockstepCrossValidator{
		log:                 logger.New("component", "cross-validator"),
		metrics:             m,
		messageExpiryWindow: messageExpiryWindow,
		startTimestamp:      startTimestamp,
		validationInterval:  validationInterval,
		chains:              chains,
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// Start starts the validation loop
func (v *LockstepCrossValidator) Start() error {
	v.log.Info("Starting cross-validator", "chains", len(v.chains))

	v.wg.Add(1)
	go v.runValidationLoop()

	return nil
}

// Stop stops the validation loop
func (v *LockstepCrossValidator) Stop() error {
	v.log.Info("Stopping cross-validator")
	v.cancel()
	v.wg.Wait()
	return nil
}

// Error returns the current validation error state, if any.
func (v *LockstepCrossValidator) Error() *ValidatorError {
	v.errMu.RLock()
	defer v.errMu.RUnlock()
	return v.err
}

// setError sets the validation error state.
func (v *LockstepCrossValidator) setError(msg string) {
	v.errMu.Lock()
	defer v.errMu.Unlock()
	v.err = &ValidatorError{
		Message: msg,
	}
}

// CrossValidatedTimestamp returns the global cross-validated timestamp.
func (v *LockstepCrossValidator) CrossValidatedTimestamp() (uint64, bool) {
	ts := v.crossValidatedTs.Load()
	if ts == 0 {
		return 0, false
	}
	return ts, true
}

// validateMessageTiming is a pure function that validates temporal constraints for cross-chain messages.
// Parameters:
//   - initTimestamp: when the initiating message was created
//   - inclusionTimestamp: when the executing message is included
//   - messageExpiryWindow: how long messages remain valid
//   - timeout: optional max execution delay (0 = disabled)
//   - execTimestamp: execution timestamp (only used if timeout > 0)
func validateMessageTiming(
	initTimestamp, inclusionTimestamp, messageExpiryWindow, timeout, execTimestamp uint64,
) error {
	// Rule 1: init must be strictly before inclusion
	if initTimestamp >= inclusionTimestamp {
		return fmt.Errorf("initiating message timestamp %d not before inclusion timestamp %d: %w",
			initTimestamp, inclusionTimestamp, types.ErrConflict)
	}

	// Rule 2: compute expiry with overflow check
	expiresAt := initTimestamp + messageExpiryWindow
	if expiresAt < initTimestamp {
		return fmt.Errorf("overflow in expiry calculation: timestamp %d + window %d: %w",
			initTimestamp, messageExpiryWindow, types.ErrConflict)
	}

	// Rule 3: message must not be expired at inclusion
	if expiresAt < inclusionTimestamp {
		return fmt.Errorf("initiating message expired: init %d + expiry window %d = %d < inclusion %d: %w",
			initTimestamp, messageExpiryWindow, expiresAt, inclusionTimestamp, types.ErrConflict)
	}

	// Rule 4: if timeout set, message must not expire before timeout deadline
	if timeout > 0 {
		maxExecTimestamp := execTimestamp + timeout
		if maxExecTimestamp < execTimestamp {
			return fmt.Errorf("overflow in max exec timestamp calculation: timestamp %d + timeout %d: %w",
				execTimestamp, timeout, types.ErrConflict)
		}
		if expiresAt < maxExecTimestamp {
			return fmt.Errorf("initiating message will expire before timeout: "+
				"init %d + expiry %d = %d < exec %d + timeout %d = %d: %w",
				initTimestamp, messageExpiryWindow, expiresAt,
				execTimestamp, timeout, maxExecTimestamp,
				types.ErrConflict)
		}
	}

	return nil
}

// ValidateAccessEntry validates a single access list entry against all message validity rules.
func (v *LockstepCrossValidator) ValidateAccessEntry(
	access types.Access,
	minSafety types.SafetyLevel,
	execDescriptor types.ExecutingDescriptor,
) error {
	// Check that we have ingested data for the requested timestamp
	minIngestedTs, ok := v.getMinIngestedTimestamp()
	if !ok || access.Timestamp > minIngestedTs {
		return fmt.Errorf("timestamp %d not yet ingested (min ingested: %d): %w",
			access.Timestamp, minIngestedTs, types.ErrOutOfScope)
	}

	// Check cross-unsafe timestamp
	if minSafety == types.CrossUnsafe {
		crossValidatedTs, ok := v.CrossValidatedTimestamp()
		if !ok {
			return fmt.Errorf("cross-validated timestamp not available: %w", types.ErrOutOfScope)
		}
		if access.Timestamp > crossValidatedTs {
			return fmt.Errorf("message at timestamp %d not yet cross-unsafe validated "+
				"(current cross-validated timestamp: %d): %w",
				access.Timestamp, crossValidatedTs, types.ErrOutOfScope)
		}
	}

	// Validate timing constraints (including timeout if set)
	if err := validateMessageTiming(
		access.Timestamp,
		execDescriptor.Timestamp,
		v.messageExpiryWindow,
		execDescriptor.Timeout,
		execDescriptor.Timestamp,
	); err != nil {
		return err
	}

	// Check that the log exists on the source chain
	ingester, ok := v.chains[access.ChainID]
	if !ok {
		return fmt.Errorf("source chain %s: %w", access.ChainID, types.ErrUnknownChain)
	}

	query := types.ContainsQuery{
		Timestamp: access.Timestamp,
		BlockNum:  access.BlockNumber,
		LogIdx:    access.LogIndex,
		Checksum:  access.Checksum,
	}
	_, err := ingester.Contains(query)
	return err
}

func (v *LockstepCrossValidator) validateExecutingMessage(
	execMsg *types.ExecutingMessage,
	inclusionTimestamp uint64,
) error {
	ingester, ok := v.chains[execMsg.ChainID]
	if !ok {
		return fmt.Errorf("source chain %s: %w", execMsg.ChainID, types.ErrUnknownChain)
	}

	// Validate timing constraints (no timeout for background validation)
	if err := validateMessageTiming(
		execMsg.Timestamp,
		inclusionTimestamp,
		v.messageExpiryWindow,
		0, 0, // no timeout
	); err != nil {
		return err
	}

	query := types.ContainsQuery{
		Timestamp: execMsg.Timestamp,
		BlockNum:  execMsg.BlockNum,
		LogIdx:    execMsg.LogIdx,
		Checksum:  execMsg.Checksum,
	}
	_, err := ingester.Contains(query)
	return err
}

func (v *LockstepCrossValidator) runValidationLoop() {
	defer v.wg.Done()

	ticker := time.NewTicker(v.validationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			v.advanceValidation()
		}
	}
}

// advanceValidation tries to advance the cross-validated timestamp one step at a time.
func (v *LockstepCrossValidator) advanceValidation() {
	// Stop if we've already hit a validation error
	if v.Error() != nil {
		return
	}

	// All chains must be ready and error-free
	for _, ingester := range v.chains {
		if ingester.Error() != nil {
			return
		}
		if !ingester.Ready() {
			return
		}
	}

	minIngestedTs, ok := v.getMinIngestedTimestamp()
	if !ok {
		return
	}

	currentTs := v.crossValidatedTs.Load()

	// Lazy initialization: start from the configured start timestamp
	if currentTs == 0 {
		v.crossValidatedTs.Store(v.startTimestamp)
		v.log.Info("Cross-validator initialized", "startTimestamp", v.startTimestamp)
		return
	}

	// Try to advance one timestamp at a time until we catch up or hit an error
	for {
		nextTs := currentTs + 1

		// Don't go past what all chains have ingested
		if nextTs > minIngestedTs {
			return
		}

		// Validate all messages at this timestamp across all chains
		if err := v.validateTimestamp(nextTs); err != nil {
			v.log.Error("Cross-validation failed", "timestamp", nextTs, "err", err)
			v.setError(err.Error())
			return
		}

		// Advance
		v.crossValidatedTs.Store(nextTs)
		currentTs = nextTs

		v.log.Debug("Advanced cross-validated timestamp", "timestamp", nextTs)
	}
}

// validateTimestamp validates all executing messages with the given inclusion timestamp
// across all chains.
func (v *LockstepCrossValidator) validateTimestamp(timestamp uint64) error {
	for chainID, ingester := range v.chains {
		msgs, err := ingester.GetExecMsgsAtTimestamp(timestamp)
		if err != nil {
			return fmt.Errorf("failed to get messages at timestamp %d from chain %s: %w",
				timestamp, chainID, err)
		}

		for _, msg := range msgs {
			if err := v.validateExecutingMessage(msg.ExecutingMessage, msg.InclusionTimestamp); err != nil {
				return fmt.Errorf("validation failed on chain %s at timestamp %d, log %d: %w",
					chainID, timestamp, msg.LogIdx, err)
			}
		}
	}

	return nil
}

func (v *LockstepCrossValidator) getMinIngestedTimestamp() (uint64, bool) {
	if len(v.chains) == 0 {
		return 0, false
	}

	var minTs uint64
	first := true
	for _, ingester := range v.chains {
		if !ingester.Ready() {
			return 0, false
		}
		ts, ok := ingester.LatestTimestamp()
		if !ok {
			return 0, false
		}
		if first || ts < minTs {
			minTs = ts
			first = false
		}
	}
	return minTs, true
}

// Ensure LockstepCrossValidator implements CrossValidator
var _ CrossValidator = (*LockstepCrossValidator)(nil)
