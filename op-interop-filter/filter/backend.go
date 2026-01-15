package filter

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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

	cancel context.CancelFunc
}

// BackendParams contains parameters for creating a Backend.
type BackendParams struct {
	Logger         log.Logger
	Metrics        metrics.Metricer
	Chains         map[eth.ChainID]ChainIngester
	CrossValidator CrossValidator
}

// NewBackend creates a new Backend instance with the provided components.
func NewBackend(parentCtx context.Context, params BackendParams) *Backend {
	_, cancel := context.WithCancel(parentCtx)

	return &Backend{
		log:            params.Logger,
		metrics:        params.Metrics,
		chains:         params.Chains,
		crossValidator: params.CrossValidator,
		cancel:         cancel,
	}
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

	return nil
}

// Stop stops all chain ingesters and the cross-validator
func (b *Backend) Stop(ctx context.Context) error {
	b.log.Info("Stopping backend")
	b.cancel()

	var result error

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
	b.metrics.RecordFailsafeEnabled(b.FailsafeEnabled())
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
func supportedSafetyLevel(level types.SafetyLevel) bool {
	return level == types.LocalUnsafe || level == types.CrossUnsafe
}

// CheckAccessList validates the given access list entries.
func (b *Backend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, execDescriptor types.ExecutingDescriptor) error {

	if b.FailsafeEnabled() {
		b.metrics.RecordCheckAccessList(false)
		return types.ErrFailsafeEnabled
	}

	if !b.Ready() {
		b.metrics.RecordCheckAccessList(false)
		return types.ErrUninitialized
	}

	if !supportedSafetyLevel(minSafety) {
		b.metrics.RecordCheckAccessList(false)
		return fmt.Errorf("unsupported safety level %s: only %s and %s are supported",
			minSafety, types.LocalUnsafe, types.CrossUnsafe)
	}

	if _, ok := b.chains[execDescriptor.ChainID]; !ok {
		b.metrics.RecordCheckAccessList(false)
		return fmt.Errorf("executing chain %s: %w", execDescriptor.ChainID, types.ErrUnknownChain)
	}

	remaining := inboxEntries
	for len(remaining) > 0 {
		var access types.Access
		var err error
		remaining, access, err = types.ParseAccess(remaining)
		if err != nil {
			b.metrics.RecordCheckAccessList(false)
			return fmt.Errorf("failed to parse access entry: %w", err)
		}

		if err := b.crossValidator.ValidateAccessEntry(access, minSafety, execDescriptor); err != nil {
			b.metrics.RecordCheckAccessList(false)
			return err
		}
	}

	b.metrics.RecordCheckAccessList(true)
	return nil
}
