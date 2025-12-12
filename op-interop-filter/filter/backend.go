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

	// Cross-unsafe validation state
	// Tracks minimum timestamp across all chains for validation
	pendingExecMsgs map[eth.ChainID]map[uint64][]*types.ExecutingMessage // chainID -> timestamp -> execMsgs
	pendingMu       sync.Mutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewBackend creates a new Backend instance
func NewBackend(parentCtx context.Context, logger log.Logger, m metrics.Metricer, cfg *Config) (*Backend, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	b := &Backend{
		log:             logger,
		metrics:         m,
		cfg:             cfg,
		chains:          make(map[eth.ChainID]*ChainIngester),
		pendingExecMsgs: make(map[eth.ChainID]map[uint64][]*types.ExecutingMessage),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Create chain ingesters for each L2 RPC
	for _, rpcURL := range cfg.L2RPCs {
		// Query chain ID from the RPC
		chainID, err := b.queryChainID(ctx, rpcURL)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to query chain ID from %s: %w", rpcURL, err)
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
			cancel()
			return nil, fmt.Errorf("failed to create chain ingester for chain %s: %w", chainID, err)
		}

		b.chains[chainID] = ingester
		b.pendingExecMsgs[chainID] = make(map[uint64][]*types.ExecutingMessage)
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

	// Only LocalUnsafe is supported for now (we don't track derivation)
	if minSafety != types.LocalUnsafe {
		b.metrics.RecordCheckAccessList(false)
		return fmt.Errorf("unsupported safety level %s: only %s is supported", minSafety, types.LocalUnsafe)
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
func (b *Backend) validateAccess(ctx context.Context, access types.Access, execDescriptor types.ExecutingDescriptor) error {
	// Find the chain ingester for this access entry
	b.chainsMu.RLock()
	ingester, ok := b.chains[access.ChainID]
	b.chainsMu.RUnlock()

	if !ok {
		return fmt.Errorf("chain %s: %w", access.ChainID, types.ErrUnknownChain)
	}

	// Important invariant: Messages initiated and executed in the same timestamp are invalid
	if access.Timestamp == execDescriptor.Timestamp {
		return fmt.Errorf("message initiated at same timestamp as execution (%d): %w",
			access.Timestamp, types.ErrConflict)
	}

	// Check timeout validity
	if execDescriptor.Timeout > 0 {
		// The message must be valid from execDescriptor.Timestamp to execDescriptor.Timestamp + Timeout
		// This means the initiating message timestamp must be strictly less than execDescriptor.Timestamp
		if access.Timestamp >= execDescriptor.Timestamp {
			return fmt.Errorf("initiating message timestamp %d not before execution timestamp %d: %w",
				access.Timestamp, execDescriptor.Timestamp, types.ErrConflict)
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

	// Store pending executing messages for cross-unsafe validation
	if _, ok := b.pendingExecMsgs[chainID]; !ok {
		b.pendingExecMsgs[chainID] = make(map[uint64][]*types.ExecutingMessage)
	}
	b.pendingExecMsgs[chainID][timestamp] = append(b.pendingExecMsgs[chainID][timestamp], execMsgs...)

	// Try to validate cross-unsafe messages
	b.tryValidateCrossUnsafe()
}

// tryValidateCrossUnsafe validates executing messages when all chains have caught up
func (b *Backend) tryValidateCrossUnsafe() {
	// Find minimum timestamp across all chains
	minTimestamp := uint64(0)
	first := true

	b.chainsMu.RLock()
	for _, ingester := range b.chains {
		ts, ok := ingester.LatestTimestamp()
		if !ok {
			// Chain not ready yet
			b.chainsMu.RUnlock()
			return
		}
		if first || ts < minTimestamp {
			minTimestamp = ts
			first = false
		}
	}
	b.chainsMu.RUnlock()

	if first {
		return // No chains
	}

	// Validate all pending executing messages with timestamp <= minTimestamp
	for chainID, timestampMsgs := range b.pendingExecMsgs {
		for ts, execMsgs := range timestampMsgs {
			if ts > minTimestamp {
				continue // Not ready to validate yet
			}

			for _, execMsg := range execMsgs {
				if err := b.validateExecutingMessage(execMsg); err != nil {
					b.log.Error("Cross-unsafe validation failed, enabling failsafe",
						"chain", chainID,
						"timestamp", ts,
						"execMsg", execMsg,
						"err", err)
					b.SetFailsafeEnabled(true)
					return
				}
			}

			// Clean up validated messages
			delete(b.pendingExecMsgs[chainID], ts)
		}

		// Record pending messages metric for this chain
		var pendingCount int64
		for _, msgs := range b.pendingExecMsgs[chainID] {
			pendingCount += int64(len(msgs))
		}
		chainIDUint64, _ := chainID.Uint64()
		b.metrics.RecordPendingExecMsgs(chainIDUint64, pendingCount)
	}

	// Record the validated timestamp
	b.metrics.RecordCrossUnsafeValidatedTimestamp(minTimestamp)
}

// validateExecutingMessage validates a single executing message against the source chain
func (b *Backend) validateExecutingMessage(execMsg *types.ExecutingMessage) error {
	b.chainsMu.RLock()
	ingester, ok := b.chains[execMsg.ChainID]
	b.chainsMu.RUnlock()

	if !ok {
		return fmt.Errorf("source chain %s: %w", execMsg.ChainID, types.ErrUnknownChain)
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
