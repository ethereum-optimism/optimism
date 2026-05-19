package filter

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
)

// LogsDB is the subset of an op-supervisor logs DB that
// LogsDBChainIngester depends on. Capturing it as an interface lets tests
// substitute a fake when exercising dispatch paths the real DB cannot
// produce under correct ingester control flow.
type LogsDB interface {
	Close() error
	Contains(query messages.ContainsQuery) (messages.BlockSeal, error)
	LatestSealedBlock() (eth.BlockID, bool)
	FindSealedBlock(number uint64) (messages.BlockSeal, error)
	FirstSealedBlock() (messages.BlockSeal, error)
	OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*messages.ExecutingMessage, error)
	AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *messages.ExecutingMessage) error
	SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error
	Rewind(newHead eth.BlockID) error
}

// IncludedMessage wraps an executing message with its inclusion context.
// The ExecutingMessage contains the initiating message's data (source chain),
// while InclusionBlockNum/Timestamp indicate when it was executed (this chain).
type IncludedMessage struct {
	*messages.ExecutingMessage
	InclusionBlockNum  uint64
	InclusionTimestamp uint64
}

// ChainIngester provides access to chain logs and state.
// Implementations include:
//   - mockChainIngester: in-memory for testing
//   - LogsDBChainIngester: RPC-backed with logsdb for production
type ChainIngester interface {
	// Start begins the ingester's background processing.
	Start() error

	// Stop halts the ingester's background processing.
	Stop() error

	// Contains checks if a log exists in the chain's database.
	Contains(query messages.ContainsQuery) (messages.BlockSeal, error)

	// LatestBlock returns the latest ingested block.
	LatestBlock() (eth.BlockID, bool)

	// BlockHashByNumber returns the hash of the ingested block at the given height.
	BlockHashByNumber(number uint64) (common.Hash, bool)

	// LatestTimestamp returns the timestamp of the latest ingested block.
	LatestTimestamp() (uint64, bool)

	// GetExecMsgsAtTimestamp returns executing messages with the given inclusion timestamp.
	GetExecMsgsAtTimestamp(timestamp uint64) ([]IncludedMessage, error)

	// Ready returns true if the ingester has completed initial sync.
	Ready() bool

	// Error returns the current error state, if any.
	Error() *IngesterError

	// SetError sets an error state on the ingester.
	SetError(reason IngesterErrorReason, msg string)

	// ClearError clears the error state.
	ClearError()

	// RewindToFinalized rewinds durable log state to finalized.
	RewindToFinalized(ctx context.Context) (eth.BlockID, uint64, error)
}

// CrossValidator validates cross-chain messages.
// Implementations:
//   - LockstepCrossValidator: waits for all chains to align before advancing
type CrossValidator interface {
	// Start begins the validator's background processing.
	Start() error

	// Stop halts the validator's background processing.
	Stop() error

	// ValidateAccessEntry validates a single access list entry.
	ValidateAccessEntry(access messages.Access, minSafety types.SafetyLevel, execDescriptor messages.ExecutingDescriptor) error

	// CrossValidatedTimestamp returns the global cross-validated timestamp.
	CrossValidatedTimestamp() (uint64, bool)

	// Error returns the current error state, if any.
	// Validation errors (invalid executing messages) are tracked here.
	Error() *ValidatorError

	// ResetCrossValidatedTimestamp rewinds in-memory validation progress after
	// the underlying log DB has been rewound.
	ResetCrossValidatedTimestamp(timestamp uint64)
}
