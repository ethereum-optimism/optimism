package filter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// pendingExecMsg represents an executing message awaiting cross-unsafe validation
type pendingExecMsg struct {
	execChainID eth.ChainID              // Chain where the executing message was found
	timestamp   uint64                   // Timestamp of the block containing the executing message
	msg         *types.ExecutingMessage  // The executing message itself (references source chain)
}

// Backend coordinates chain ingesters, manages failsafe state, and handles CheckAccessList requests.
type Backend struct {
	log     log.Logger
	metrics metrics.Metricer
	cfg     *Config

	// Chain ingesters keyed by chain ID
	chains   map[eth.ChainID]*ChainIngester
	chainsMu sync.RWMutex

	// Failsafe state
	failsafe atomic.Bool

	// Cross-unsafe validation state - flat list of pending executing messages
	pendingExecMsgs []pendingExecMsg
	pendingMu       sync.Mutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewBackend creates a new Backend instance
func NewBackend(parentCtx context.Context, logger log.Logger, m metrics.Metricer, cfg *Config) (*Backend, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	b := &Backend{
		log:     logger,
		metrics: m,
		cfg:     cfg,
		chains:  make(map[eth.ChainID]*ChainIngester),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Helper to cleanup on error
	cleanup := func() {
		cancel()
		for _, ingester := range b.chains {
			_ = ingester.Stop()
		}
	}

	// Create chain ingesters for each L2 RPC
	for _, rpcURL := range cfg.L2RPCs {
		// Query chain ID from the RPC
		chainID, err := b.queryChainID(ctx, rpcURL)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to query chain ID from %s: %w", rpcURL, err)
		}

		// Check for duplicate chain IDs BEFORE creating ingester
		if _, exists := b.chains[chainID]; exists {
			cleanup()
			return nil, fmt.Errorf("duplicate chain ID %s: multiple RPCs return the same chain ID", chainID)
		}

		logger.Info("Creating chain ingester", "chain", chainID, "rpc", rpcURL)

		ingester, err := NewChainIngester(
			ctx,
			logger,
			m,
			chainID,
			rpcURL,
			cfg.DataDir,
			cfg.BackfillDuration,
			b.onExecutingMessages,
			b.onReorg,
		)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to create chain ingester for chain %s: %w", chainID, err)
		}

		b.chains[chainID] = ingester
	}

	logger.Info("Created backend", "chains", len(b.chains))
	return b, nil
}

// queryChainID queries the chain ID from an RPC endpoint
func (b *Backend) queryChainID(ctx context.Context, rpcURL string) (eth.ChainID, error) {
	rpcClient, err := client.NewRPC(ctx, b.log, rpcURL)
	if err != nil {
		return eth.ChainID{}, fmt.Errorf("failed to create RPC client: %w", err)
	}
	defer rpcClient.Close()

	var chainIDHex string
	if err := rpcClient.CallContext(ctx, &chainIDHex, "eth_chainId"); err != nil {
		return eth.ChainID{}, fmt.Errorf("failed to query chain ID: %w", err)
	}

	chainID, err := eth.ChainIDFromString(chainIDHex)
	if err != nil {
		return eth.ChainID{}, fmt.Errorf("invalid chain ID response: %w", err)
	}

	return chainID, nil
}

// Start starts all chain ingesters
func (b *Backend) Start(ctx context.Context) error {
	b.log.Info("Starting backend")

	for chainID, ingester := range b.chains {
		if err := ingester.Start(); err != nil {
			return fmt.Errorf("failed to start chain ingester for chain %s: %w", chainID, err)
		}
	}

	return nil
}

// Stop stops all chain ingesters
func (b *Backend) Stop(ctx context.Context) error {
	b.log.Info("Stopping backend")
	b.cancel()

	var result error
	for chainID, ingester := range b.chains {
		if err := ingester.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop chain ingester for chain %s: %w", chainID, err))
		}
	}

	return result
}

// FailsafeEnabled returns whether failsafe is enabled
func (b *Backend) FailsafeEnabled() bool {
	return b.failsafe.Load()
}

// SetFailsafeEnabled sets the failsafe state
func (b *Backend) SetFailsafeEnabled(enabled bool) {
	b.failsafe.Store(enabled)
	b.metrics.RecordFailsafeEnabled(enabled)
	b.log.Info("Failsafe state changed", "enabled", enabled)
}

// Ready returns true if all chains have completed backfill
func (b *Backend) Ready() bool {
	b.chainsMu.RLock()
	defer b.chainsMu.RUnlock()

	for _, ingester := range b.chains {
		if !ingester.Ready() {
			return false
		}
	}

	return len(b.chains) > 0
}

// CheckAccessList validates the given access list entries.
func (b *Backend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, execDescriptor types.ExecutingDescriptor) error {

	// Check failsafe first
	if b.failsafe.Load() {
		b.metrics.RecordCheckAccessList(false)
		return types.ErrFailsafeEnabled
	}

	// Check if all chains are ready
	if !b.Ready() {
		b.metrics.RecordCheckAccessList(false)
		return types.ErrUninitialized
	}

	// We support LocalUnsafe and CrossUnsafe (we don't track derivation for Safe/Finalized)
	// CrossUnsafe is supported because we perform cross-chain validation in tryValidateCrossUnsafe()
	// and enable failsafe if any validation fails
	if minSafety != types.LocalUnsafe && minSafety != types.CrossUnsafe {
		b.metrics.RecordCheckAccessList(false)
		return fmt.Errorf("unsupported safety level %s: only %s and %s are supported",
			minSafety, types.LocalUnsafe, types.CrossUnsafe)
	}

	// Parse and validate each access entry
	remaining := inboxEntries
	for len(remaining) > 0 {
		var access types.Access
		var err error
		remaining, access, err = types.ParseAccess(remaining)
		if err != nil {
			b.metrics.RecordCheckAccessList(false)
			return fmt.Errorf("failed to parse access entry: %w", err)
		}

		// Validate the access entry
		if err := b.validateAccess(ctx, access, execDescriptor); err != nil {
			b.metrics.RecordCheckAccessList(false)
			return err
		}
	}

	b.metrics.RecordCheckAccessList(true)
	return nil
}

// validateAccess validates a single access entry
// This follows the same linking rules as op-supervisor:
// 1. initTimestamp <= execTimestamp (same-timestamp is valid)
// 2. initTimestamp + MessageExpiryWindow >= execTimestamp (message not expired)
// 3. If Timeout > 0: initTimestamp + MessageExpiryWindow >= execTimestamp + Timeout
func (b *Backend) validateAccess(ctx context.Context, access types.Access, execDescriptor types.ExecutingDescriptor) error {
	// Find the chain ingester for this access entry
	b.chainsMu.RLock()
	ingester, ok := b.chains[access.ChainID]
	b.chainsMu.RUnlock()

	if !ok {
		return fmt.Errorf("chain %s: %w", access.ChainID, types.ErrUnknownChain)
	}

	// Initiating message timestamp must not be after execution timestamp
	// (same-timestamp is valid per supervisor spec)
	if access.Timestamp > execDescriptor.Timestamp {
		return fmt.Errorf("initiating message timestamp %d after execution timestamp %d: %w",
			access.Timestamp, execDescriptor.Timestamp, types.ErrConflict)
	}

	// Check expiry: message expires at initTimestamp + MessageExpiryWindow
	expiresAt := saturatingAdd(access.Timestamp, b.cfg.MessageExpiryWindow)
	if expiresAt < execDescriptor.Timestamp {
		return fmt.Errorf("initiating message expired: init %d + expiry window %d = %d < exec %d: %w",
			access.Timestamp, b.cfg.MessageExpiryWindow, expiresAt, execDescriptor.Timestamp, types.ErrConflict)
	}

	// If Timeout is set, also verify the message won't expire at execTimestamp + Timeout
	// This ensures the message remains valid for the preverification window
	if execDescriptor.Timeout > 0 {
		maxExecTimestamp := saturatingAdd(execDescriptor.Timestamp, execDescriptor.Timeout)
		if expiresAt < maxExecTimestamp {
			return fmt.Errorf("initiating message will expire before timeout: init %d + expiry %d = %d < exec %d + timeout %d = %d: %w",
				access.Timestamp, b.cfg.MessageExpiryWindow, expiresAt,
				execDescriptor.Timestamp, execDescriptor.Timeout, maxExecTimestamp, types.ErrConflict)
		}
	}

	// Create query and check if log exists
	query := access.Query()
	_, err := ingester.Contains(query)
	if err != nil {
		return fmt.Errorf("log validation failed for chain %s, block %d, index %d: %w",
			access.ChainID, access.BlockNumber, access.LogIndex, err)
	}

	return nil
}

// saturatingAdd adds two uint64 values, returning max uint64 on overflow
func saturatingAdd(a, b uint64) uint64 {
	result := a + b
	if result < a { // overflow
		return ^uint64(0) // max uint64
	}
	return result
}

// Rewind rewinds a chain to a specific block
func (b *Backend) Rewind(ctx context.Context, chainID eth.ChainID, newHead eth.BlockID) error {
	b.chainsMu.RLock()
	ingester, ok := b.chains[chainID]
	b.chainsMu.RUnlock()

	if !ok {
		return types.ErrUnknownChain
	}

	return ingester.Rewind(newHead)
}

// onExecutingMessages is called when executing messages are detected during ingestion
func (b *Backend) onExecutingMessages(chainID eth.ChainID, timestamp uint64, execMsgs []*types.ExecutingMessage) {
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()

	// Append executing messages to pending list
	for _, msg := range execMsgs {
		b.pendingExecMsgs = append(b.pendingExecMsgs, pendingExecMsg{
			execChainID: chainID,
			timestamp:   timestamp,
			msg:         msg,
		})
	}

	// Try to validate cross-unsafe messages
	b.tryValidateCrossUnsafe()
}

// tryValidateCrossUnsafe validates executing messages when all chains have caught up
func (b *Backend) tryValidateCrossUnsafe() {
	// Find minimum timestamp across all chains
	minTimestamp, ok := b.getMinChainTimestamp()
	if !ok {
		return // Not all chains ready
	}

	// Validate and filter in one pass
	remaining := b.pendingExecMsgs[:0] // reuse backing array
	for _, pending := range b.pendingExecMsgs {
		if pending.timestamp > minTimestamp {
			// Not ready yet - keep for later
			remaining = append(remaining, pending)
			continue
		}

		// Validate this message
		if err := b.validateExecutingMessage(pending.msg, pending.timestamp); err != nil {
			b.log.Error("Cross-unsafe validation failed, enabling failsafe",
				"execChain", pending.execChainID,
				"sourceChain", pending.msg.ChainID,
				"timestamp", pending.timestamp,
				"err", err)
			b.SetFailsafeEnabled(true)
			return
		}
	}
	b.pendingExecMsgs = remaining

	// Record metrics
	b.metrics.RecordPendingExecMsgs(0, int64(len(b.pendingExecMsgs)))
	b.metrics.RecordCrossUnsafeValidatedTimestamp(minTimestamp)
}

// getMinChainTimestamp returns the minimum timestamp across all chains.
// Returns false if any chain is not ready yet.
func (b *Backend) getMinChainTimestamp() (uint64, bool) {
	b.chainsMu.RLock()
	defer b.chainsMu.RUnlock()

	if len(b.chains) == 0 {
		return 0, false
	}

	var minTimestamp uint64
	first := true
	for _, ingester := range b.chains {
		ts, ok := ingester.LatestTimestamp()
		if !ok {
			return 0, false // Chain not ready
		}
		if first || ts < minTimestamp {
			minTimestamp = ts
			first = false
		}
	}
	return minTimestamp, true
}

// validateExecutingMessage validates a single executing message against the source chain
// execTimestamp is the timestamp when the executing message was processed
// Uses the same linking rules as validateAccess (matching op-supervisor)
func (b *Backend) validateExecutingMessage(execMsg *types.ExecutingMessage, execTimestamp uint64) error {
	b.chainsMu.RLock()
	ingester, ok := b.chains[execMsg.ChainID]
	b.chainsMu.RUnlock()

	if !ok {
		return fmt.Errorf("source chain %s: %w", execMsg.ChainID, types.ErrUnknownChain)
	}

	// Initiating message timestamp must not be after execution timestamp
	// (same-timestamp is valid per supervisor spec)
	if execMsg.Timestamp > execTimestamp {
		return fmt.Errorf("initiating message timestamp %d after execution timestamp %d: %w",
			execMsg.Timestamp, execTimestamp, types.ErrConflict)
	}

	// Check expiry: message expires at initTimestamp + MessageExpiryWindow
	expiresAt := saturatingAdd(execMsg.Timestamp, b.cfg.MessageExpiryWindow)
	if expiresAt < execTimestamp {
		return fmt.Errorf("initiating message expired: init %d + expiry window %d = %d < exec %d: %w",
			execMsg.Timestamp, b.cfg.MessageExpiryWindow, expiresAt, execTimestamp, types.ErrConflict)
	}

	// Check log exists in source chain
	query := types.ContainsQuery{
		Timestamp: execMsg.Timestamp,
		BlockNum:  execMsg.BlockNum,
		LogIdx:    execMsg.LogIdx,
		Checksum:  execMsg.Checksum,
	}

	_, err := ingester.Contains(query)
	return err
}

// onReorg is called when a reorg is detected on a chain
func (b *Backend) onReorg(chainID eth.ChainID) {
	b.log.Warn("Reorg detected, enabling failsafe", "chain", chainID)
	b.SetFailsafeEnabled(true)
}

// GetChainIDs returns the chain IDs of all configured chains
func (b *Backend) GetChainIDs() []eth.ChainID {
	b.chainsMu.RLock()
	defer b.chainsMu.RUnlock()

	chainIDs := make([]eth.ChainID, 0, len(b.chains))
	for chainID := range b.chains {
		chainIDs = append(chainIDs, chainID)
	}
	return chainIDs
}
