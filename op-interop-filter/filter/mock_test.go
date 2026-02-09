package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// mockChainIngester is an in-memory implementation of ChainIngester for testing.
// It stores logs in a map and provides simple state management.
type mockChainIngester struct {
	mu sync.RWMutex

	// Logs stored by their identifying query
	logs map[logKey]types.BlockSeal

	// Executing messages with their inclusion context
	execMsgs []IncludedMessage

	// State
	ready                 bool
	err                   *IngesterError
	latestBlock           eth.BlockID
	latestTimestamp       uint64
	earliestIngestedBlock uint64
}

// logKey uniquely identifies a log entry
type logKey struct {
	Timestamp uint64
	BlockNum  uint64
	LogIdx    uint32
	Checksum  types.MessageChecksum
}

// newMockChainIngester creates a new in-memory chain ingester.
func newMockChainIngester() *mockChainIngester {
	return &mockChainIngester{
		logs:     make(map[logKey]types.BlockSeal),
		execMsgs: make([]IncludedMessage, 0),
		ready:    true, // Default to ready for simple tests
	}
}

// Start implements ChainIngester (no-op for in-memory).
func (m *mockChainIngester) Start() error { return nil }

// Stop implements ChainIngester (no-op for in-memory).
func (m *mockChainIngester) Stop() error { return nil }

// AddLog adds a log entry to the ingester.
func (m *mockChainIngester) AddLog(timestamp, blockNum uint64, logIdx uint32, checksum types.MessageChecksum, seal types.BlockSeal) {
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
	if m.earliestIngestedBlock == 0 || blockNum < m.earliestIngestedBlock {
		m.earliestIngestedBlock = blockNum
	}
}

// AddExecMsg adds an executing message with its inclusion context.
func (m *mockChainIngester) AddExecMsg(msg IncludedMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.execMsgs = append(m.execMsgs, msg)

	// Update latest block/timestamp if needed
	if msg.InclusionBlockNum > m.latestBlock.Number {
		m.latestBlock = eth.BlockID{Number: msg.InclusionBlockNum}
		m.latestTimestamp = msg.InclusionTimestamp
	}
	if m.earliestIngestedBlock == 0 || msg.InclusionBlockNum < m.earliestIngestedBlock {
		m.earliestIngestedBlock = msg.InclusionBlockNum
	}
}

// SetReady sets the ready state.
func (m *mockChainIngester) SetReady(ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = ready
}

// Contains implements ChainIngester.
func (m *mockChainIngester) Contains(query types.ContainsQuery) (types.BlockSeal, error) {
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
func (m *mockChainIngester) LatestBlock() (eth.BlockID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.latestBlock.Number == 0 {
		return eth.BlockID{}, false
	}
	return m.latestBlock, true
}

// LatestTimestamp implements ChainIngester.
func (m *mockChainIngester) LatestTimestamp() (uint64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.latestTimestamp == 0 {
		return 0, false
	}
	return m.latestTimestamp, true
}

// GetExecMsgsAtTimestamp implements ChainIngester.
func (m *mockChainIngester) GetExecMsgsAtTimestamp(timestamp uint64) ([]IncludedMessage, error) {
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
func (m *mockChainIngester) Ready() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

// Error implements ChainIngester.
func (m *mockChainIngester) Error() *IngesterError {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.err
}

// SetError implements ChainIngester.
func (m *mockChainIngester) SetError(reason IngesterErrorReason, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = &IngesterError{
		Reason:  reason,
		Message: msg,
	}
}

// ClearError implements ChainIngester.
func (m *mockChainIngester) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = nil
}

// SetLatestTimestamp sets the latest ingested timestamp (for testing).
func (m *mockChainIngester) SetLatestTimestamp(ts uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latestTimestamp = ts
}

// ClearExecMsgs removes all executing messages (for testing).
func (m *mockChainIngester) ClearExecMsgs() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execMsgs = nil
}

// Ensure mockChainIngester implements ChainIngester
var _ ChainIngester = (*mockChainIngester)(nil)

// =============================================================================
// MockCrossValidator
// =============================================================================

// mockCrossValidator is a minimal mock for backend tests that don't need real validation.
type mockCrossValidator struct {
	validateErr error
	errState    *ValidatorError
}

func (m *mockCrossValidator) Start() error { return nil }
func (m *mockCrossValidator) Stop() error  { return nil }
func (m *mockCrossValidator) ValidateAccessEntry(access types.Access, minSafety types.SafetyLevel, execDescriptor types.ExecutingDescriptor) error {
	return m.validateErr
}
func (m *mockCrossValidator) CrossValidatedTimestamp() (uint64, bool) { return 0, false }
func (m *mockCrossValidator) Error() *ValidatorError                  { return m.errState }

// SetError sets the error state for the mock validator.
func (m *mockCrossValidator) SetError(msg string) {
	m.errState = &ValidatorError{Message: msg}
}

// Ensure mockCrossValidator implements CrossValidator
var _ CrossValidator = (*mockCrossValidator)(nil)

// =============================================================================
// MockEthClient
// =============================================================================

// MockEthClient is an in-memory implementation of EthClient for testing.
// It can be populated manually or loaded from captured JSON data.
type MockEthClient struct {
	mu sync.RWMutex

	// Block info keyed by block number
	blocksByNumber map[uint64]eth.BlockInfo

	// Block info keyed by label (e.g., "unsafe")
	blocksByLabel map[eth.BlockLabel]eth.BlockInfo

	// Receipts keyed by block hash
	receiptsByHash map[common.Hash]gethTypes.Receipts

	// Error injection
	infoByNumberErr  error
	infoByLabelErr   error
	fetchReceiptsErr error
}

// NewMockEthClient creates a new mock eth client.
func NewMockEthClient() *MockEthClient {
	return &MockEthClient{
		blocksByNumber: make(map[uint64]eth.BlockInfo),
		blocksByLabel:  make(map[eth.BlockLabel]eth.BlockInfo),
		receiptsByHash: make(map[common.Hash]gethTypes.Receipts),
	}
}

// InfoByLabel implements EthClient.
func (m *MockEthClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.infoByLabelErr != nil {
		return nil, m.infoByLabelErr
	}

	block, ok := m.blocksByLabel[label]
	if !ok {
		return nil, fmt.Errorf("block not found for label %s", label)
	}
	return block, nil
}

// InfoByNumber implements EthClient.
func (m *MockEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.infoByNumberErr != nil {
		return nil, m.infoByNumberErr
	}

	block, ok := m.blocksByNumber[number]
	if !ok {
		return nil, fmt.Errorf("block %d not found", number)
	}
	return block, nil
}

// FetchReceipts implements EthClient.
func (m *MockEthClient) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, gethTypes.Receipts, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.fetchReceiptsErr != nil {
		return nil, nil, m.fetchReceiptsErr
	}

	receipts, ok := m.receiptsByHash[blockHash]
	if !ok {
		return nil, nil, fmt.Errorf("receipts not found for block %s", blockHash.Hex())
	}

	// Find the block info for this hash
	for _, block := range m.blocksByNumber {
		if block.Hash() == blockHash {
			return block, receipts, nil
		}
	}

	return nil, receipts, nil
}

// Close implements EthClient.
func (m *MockEthClient) Close() {}

// AddBlock adds a block to the mock.
func (m *MockEthClient) AddBlock(block eth.BlockInfo, receipts gethTypes.Receipts) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blocksByNumber[block.NumberU64()] = block
	m.receiptsByHash[block.Hash()] = receipts
}

// SetHeadBlock sets a block as the head (unsafe label).
func (m *MockEthClient) SetHeadBlock(block eth.BlockInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blocksByLabel[eth.Unsafe] = block
}

// SetInfoByNumberErr sets an error to return from InfoByNumber.
func (m *MockEthClient) SetInfoByNumberErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoByNumberErr = err
}

// SetInfoByLabelErr sets an error to return from InfoByLabel.
func (m *MockEthClient) SetInfoByLabelErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoByLabelErr = err
}

// SetFetchReceiptsErr sets an error to return from FetchReceipts.
func (m *MockEthClient) SetFetchReceiptsErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fetchReceiptsErr = err
}

// Ensure MockEthClient implements EthClient
var _ EthClient = (*MockEthClient)(nil)

// =============================================================================
// Captured Data Types (for loading test fixtures)
// =============================================================================

// CapturedData is the format output by spammer --capture
type CapturedData struct {
	ChainID uint64           `json:"chain_id"`
	Blocks  []*CapturedBlock `json:"blocks"`
}

// CapturedBlock holds a block's info and receipts
type CapturedBlock struct {
	Number     uint64             `json:"number"`
	Hash       common.Hash        `json:"hash"`
	ParentHash common.Hash        `json:"parent_hash"`
	Timestamp  uint64             `json:"timestamp"`
	Receipts   []*CapturedReceipt `json:"receipts"`
}

// CapturedReceipt holds receipt data with logs
type CapturedReceipt struct {
	TxHash common.Hash    `json:"tx_hash"`
	Logs   []*CapturedLog `json:"logs"`
}

// CapturedLog holds log data needed for testing
type CapturedLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    []byte         `json:"data"`
	Index   uint           `json:"index"`
}

// mockBlockInfo implements eth.BlockInfo for test data
type mockBlockInfo struct {
	number     uint64
	hash       common.Hash
	parentHash common.Hash
	timestamp  uint64
}

func (b *mockBlockInfo) Hash() common.Hash                          { return b.hash }
func (b *mockBlockInfo) ParentHash() common.Hash                    { return b.parentHash }
func (b *mockBlockInfo) NumberU64() uint64                          { return b.number }
func (b *mockBlockInfo) Time() uint64                               { return b.timestamp }
func (b *mockBlockInfo) Coinbase() common.Address                   { return common.Address{} }
func (b *mockBlockInfo) Root() common.Hash                          { return common.Hash{} }
func (b *mockBlockInfo) ReceiptHash() common.Hash                   { return common.Hash{} }
func (b *mockBlockInfo) GasUsed() uint64                            { return 0 }
func (b *mockBlockInfo) GasLimit() uint64                           { return 0 }
func (b *mockBlockInfo) BaseFee() *big.Int                          { return nil }
func (b *mockBlockInfo) BlobBaseFee(_ *params.ChainConfig) *big.Int { return nil }
func (b *mockBlockInfo) ExcessBlobGas() *uint64                     { return nil }
func (b *mockBlockInfo) BlobGasUsed() *uint64                       { return nil }
func (b *mockBlockInfo) ParentBeaconRoot() *common.Hash             { return nil }
func (b *mockBlockInfo) WithdrawalsRoot() *common.Hash              { return nil }
func (b *mockBlockInfo) ID() eth.BlockID                            { return eth.BlockID{Hash: b.hash, Number: b.number} }
func (b *mockBlockInfo) MixDigest() common.Hash                     { return common.Hash{} }
func (b *mockBlockInfo) HeaderRLP() ([]byte, error)                 { return nil, nil }
func (b *mockBlockInfo) Header() *gethTypes.Header                  { return nil }

// LoadCapturedData loads captured test data from a JSON file
func LoadCapturedData(path string) (*CapturedData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open captured data file: %w", err)
	}
	defer file.Close()

	var data CapturedData
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode captured data: %w", err)
	}

	return &data, nil
}

// LoadMockEthClientFromCapture creates a MockEthClient populated with captured data
func LoadMockEthClientFromCapture(path string) (*MockEthClient, error) {
	data, err := LoadCapturedData(path)
	if err != nil {
		return nil, err
	}

	mock := NewMockEthClient()

	var latestBlock eth.BlockInfo
	for _, capturedBlock := range data.Blocks {
		// Create block info
		blockInfo := &mockBlockInfo{
			number:     capturedBlock.Number,
			hash:       capturedBlock.Hash,
			parentHash: capturedBlock.ParentHash,
			timestamp:  capturedBlock.Timestamp,
		}

		// Convert captured receipts to geth receipts
		var receipts gethTypes.Receipts
		for _, capturedReceipt := range capturedBlock.Receipts {
			receipt := &gethTypes.Receipt{
				TxHash: capturedReceipt.TxHash,
				Logs:   make([]*gethTypes.Log, len(capturedReceipt.Logs)),
			}
			for i, capturedLog := range capturedReceipt.Logs {
				receipt.Logs[i] = &gethTypes.Log{
					Address: capturedLog.Address,
					Topics:  capturedLog.Topics,
					Data:    capturedLog.Data,
					Index:   capturedLog.Index,
				}
			}
			receipts = append(receipts, receipt)
		}

		mock.AddBlock(blockInfo, receipts)

		// Track the latest block
		if latestBlock == nil || blockInfo.NumberU64() > latestBlock.NumberU64() {
			latestBlock = blockInfo
		}
	}

	// Set the latest block as head
	if latestBlock != nil {
		mock.SetHeadBlock(latestBlock)
	}

	return mock, nil
}
