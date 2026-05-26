package filter

import (
	"context"
	"errors"
	"fmt"
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

	// Passthrough mode: all transactions pass without filtering
	passthrough bool

	// Compatibility mode for legacy clients that omit executing chainID.
	legacyCheckAccessListFormat bool

	ctx    context.Context
	cancel context.CancelFunc

	reorgRecoveryEnabled bool
	reorgRecoveryWg      sync.WaitGroup
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
}

// NewBackend creates a new Backend instance with the provided components.
func NewBackend(parentCtx context.Context, params BackendParams) *Backend {
	ctx, cancel := context.WithCancel(parentCtx)

	return &Backend{
		log:                         params.Logger,
		metrics:                     params.Metrics,
		chains:                      params.Chains,
		crossValidator:              params.CrossValidator,
		passthrough:                 params.Passthrough,
		legacyCheckAccessListFormat: params.LegacyCheckAccessListFormat,
		ctx:                         ctx,
		cancel:                      cancel,
		reorgRecoveryEnabled:        params.ReorgRecoveryEnabled,
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

	if b.reorgRecoveryEnabled {
		b.reorgRecoveryWg.Add(1)
		go b.runReorgRecovery(b.ctx)
	}

	return nil
}

// Stop stops all chain ingesters and the cross-validator
func (b *Backend) Stop(ctx context.Context) error {
	b.log.Info("Stopping backend")
	b.cancel()

	var result error

	b.reorgRecoveryWg.Wait()

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
