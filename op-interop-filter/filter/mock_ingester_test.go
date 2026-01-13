package filter

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// MockChainIngester is an in-memory implementation of ChainIngester for testing.
// It stores logs in a map and provides simple state management.
type MockChainIngester struct {
	mu sync.RWMutex

	// Logs stored by their identifying query
	logs map[logKey]types.BlockSeal

	// Executing messages with their inclusion context
	execMsgs []IncludedMessage

	// State
	ready            bool
	err              *IngesterError
	latestBlock      eth.BlockID
	latestTimestamp  uint64
	earliestBlockNum uint64
}

// logKey uniquely identifies a log entry
type logKey struct {
	Timestamp uint64
	BlockNum  uint64
	LogIdx    uint32
	Checksum  types.MessageChecksum
}

// NewMockChainIngester creates a new in-memory chain ingester.
func NewMockChainIngester() *MockChainIngester {
	return &MockChainIngester{
		logs:     make(map[logKey]types.BlockSeal),
		execMsgs: make([]IncludedMessage, 0),
		ready:    true, // Default to ready for simple tests
	}
}

// Start implements ChainIngester (no-op for in-memory).
func (m *MockChainIngester) Start() error { return nil }

// Stop implements ChainIngester (no-op for in-memory).
func (m *MockChainIngester) Stop() error { return nil }

// AddLog adds a log entry to the ingester.
func (m *MockChainIngester) AddLog(timestamp, blockNum uint64, logIdx uint32, checksum types.MessageChecksum, seal types.BlockSeal) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := logKey{
		Timestamp: timestamp,
		BlockNum:  blockNum,
		LogIdx:    logIdx,
		Checksum:  checksum,
	}
	m.logs[key] = seal

	// Update latest block/timestamp if needed
	if blockNum > m.latestBlock.Number {
		m.latestBlock = eth.BlockID{Number: blockNum}
		m.latestTimestamp = timestamp
	}
	if m.earliestBlockNum == 0 || blockNum < m.earliestBlockNum {
		m.earliestBlockNum = blockNum
	}
}

// AddExecMsg adds an executing message with its inclusion context.
func (m *MockChainIngester) AddExecMsg(msg IncludedMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.execMsgs = append(m.execMsgs, msg)

	// Update latest block/timestamp if needed
	if msg.InclusionBlockNum > m.latestBlock.Number {
		m.latestBlock = eth.BlockID{Number: msg.InclusionBlockNum}
		m.latestTimestamp = msg.InclusionTimestamp
	}
	if m.earliestBlockNum == 0 || msg.InclusionBlockNum < m.earliestBlockNum {
		m.earliestBlockNum = msg.InclusionBlockNum
	}
}

// SetReady sets the ready state.
func (m *MockChainIngester) SetReady(ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = ready
}

// Contains implements ChainIngester.
func (m *MockChainIngester) Contains(query types.ContainsQuery) (types.BlockSeal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := logKey{
		Timestamp: query.Timestamp,
		BlockNum:  query.BlockNum,
		LogIdx:    query.LogIdx,
		Checksum:  query.Checksum,
	}

	seal, ok := m.logs[key]
	if !ok {
		return types.BlockSeal{}, types.ErrConflict
	}
	return seal, nil
}

// LatestBlock implements ChainIngester.
func (m *MockChainIngester) LatestBlock() (eth.BlockID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.latestBlock.Number == 0 {
		return eth.BlockID{}, false
	}
	return m.latestBlock, true
}

// LatestTimestamp implements ChainIngester.
func (m *MockChainIngester) LatestTimestamp() (uint64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.latestTimestamp == 0 {
		return 0, false
	}
	return m.latestTimestamp, true
}

// GetExecMsgsAtTimestamp implements ChainIngester.
func (m *MockChainIngester) GetExecMsgsAtTimestamp(timestamp uint64) ([]IncludedMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []IncludedMessage
	for _, msg := range m.execMsgs {
		if msg.InclusionTimestamp == timestamp {
			result = append(result, msg)
		}
	}
	return result, nil
}

// Ready implements ChainIngester.
func (m *MockChainIngester) Ready() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

// Error implements ChainIngester.
func (m *MockChainIngester) Error() *IngesterError {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.err
}

// SetError implements ChainIngester.
func (m *MockChainIngester) SetError(reason IngesterErrorReason, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = &IngesterError{
		Reason:  reason,
		Message: msg,
	}
}

// ClearError implements ChainIngester.
func (m *MockChainIngester) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = nil
}

// SetLatestTimestamp sets the latest ingested timestamp (for testing).
func (m *MockChainIngester) SetLatestTimestamp(ts uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latestTimestamp = ts
}

// Ensure MockChainIngester implements ChainIngester
var _ ChainIngester = (*MockChainIngester)(nil)
