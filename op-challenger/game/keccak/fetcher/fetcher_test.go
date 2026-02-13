package fetcher

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	oracleAddr     = common.Address{0x99, 0x98}
	otherAddr      = common.Address{0x12, 0x34}
	claimantKey, _ = crypto.GenerateKey()
	ident          = keccakTypes.LargePreimageIdent{
		Claimant: crypto.PubkeyToAddress(claimantKey.PublicKey),
		UUID:     big.NewInt(888),
	}
	blockHash = common.Hash{0xdd}
	input1    = keccakTypes.InputData{
		Input:       []byte{0xbb, 0x11},
		Commitments: []common.Hash{{0xcc, 0x11}},
	}
	input2 = keccakTypes.InputData{
		Input:       []byte{0xbb, 0x22},
		Commitments: []common.Hash{{0xcc, 0x22}},
	}
	input3 = keccakTypes.InputData{
		Input:       []byte{0xbb, 0x33},
		Commitments: []common.Hash{{0xcc, 0x33}},
	}
	input4 = keccakTypes.InputData{
		Input:       []byte{0xbb, 0x44},
		Commitments: []common.Hash{{0xcc, 0x44}},
		Finalize:    true,
	}
)

func TestFetchLeaves_NoBlocks(t *testing.T) {
	fetcher, oracle, _ := setupFetcherTest(t)
	oracle.leafBlocks = []uint64{}
	leaves, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Empty(t, leaves)
}

func TestFetchLeaves_ErrorOnUnavailableInputBlocks(t *testing.T) {
	fetcher, oracle, _ := setupFetcherTest(t)
	mockErr := fmt.Errorf("oops")
	oracle.inputDataBlocksError = mockErr

	leaves, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.ErrorContains(t, err, "failed to retrieve leaf block nums")
	require.Empty(t, leaves)
}

func TestFetchLeaves_ErrorOnUnavailableL1Block(t *testing.T) {
	blockNum := uint64(7)
	fetcher, oracle, _ := setupFetcherTest(t)
	oracle.leafBlocks = []uint64{blockNum}

	// No txs means stubL1Source will return an error when we try to fetch the block
	leaves, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.ErrorContains(t, err, fmt.Sprintf("failed getting info for block %v", blockNum))
	require.Empty(t, leaves)
}

func TestFetchLeaves_SingleTxSingleLog(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	proposal := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SingleTxMultipleLogs(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	proposal1 := oracle.createProposal(input1)
	proposal2 := oracle.createProposal(input2)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal1, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1, input2}, inputs)
}

func TestFetchLeaves_MultipleBlocksAndLeaves(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	block1 := uint64(7)
	block2 := uint64(15)
	oracle.leafBlocks = []uint64{block1, block2}

	proposal1 := oracle.createProposal(input1)
	proposal2 := oracle.createProposal(input2)
	proposal3 := oracle.createProposal(input3)
	proposal4 := oracle.createProposal(input4)
	l1Source.createReceipt(block1, types.ReceiptStatusSuccessful, proposal1)
	l1Source.createReceipt(block1, types.ReceiptStatusSuccessful, proposal2)
	l1Source.createReceipt(block2, types.ReceiptStatusSuccessful) // Add tx with no logs
	l1Source.createReceipt(block2, types.ReceiptStatusSuccessful, proposal3, proposal4)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1, input2, input3, input4}, inputs)
}

func TestFetchLeaves_SkipLogFromWrongContract(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Emit log from an irrelevant contract address
	proposal1 := oracle.createProposal(input2)
	rcpt := l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal1)
	rcpt.Logs[0].Address = otherAddr

	// Valid tx
	proposal2 := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipProposalWithWrongUUID(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Valid tx but with a different UUID
	proposal1 := oracle.createProposal(input2)
	proposal1.uuid = big.NewInt(874927294)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal1)

	// Valid tx
	proposal2 := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipProposalWithWrongClaimant(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Valid tx but with a different claimant
	proposal1 := oracle.createProposal(input2)
	proposal1.claimantAddr = otherAddr
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal1)
	// Valid tx
	proposal2 := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipInvalidProposal(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Set up proposal decoding to fail
	proposal1 := oracle.createProposal(input2)
	proposal1.valid = false
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal1)
	// Valid tx
	proposal2 := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipProposalWithInsufficientData(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Log contains insufficient data
	// It should hold a 20 byte address followed by the proposal payload
	proposal1 := oracle.createProposal(input2)
	rcpt1 := l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal1)
	rcpt1.Logs[0].Data = proposal1.claimantAddr[:19]
	// Valid tx
	proposal2 := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipProposalMissingCallData(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Truncate call data from log so that is only contains an address
	proposal1 := oracle.createProposal(input2)
	rcpt1 := l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal1)
	rcpt1.Logs[0].Data = rcpt1.Logs[0].Data[0:20]
	// Valid tx
	proposal2 := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipTxWithReceiptStatusFail(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Valid proposal, but tx reverted
	proposal1 := oracle.createProposal(input2)
	l1Source.createReceipt(blockNum, types.ReceiptStatusFailed, proposal1)
	// Valid tx
	proposal2 := oracle.createProposal(input1)
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful, proposal2)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_ErrorsOnMissingReceipts(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Block exists but receipts return not found
	l1Source.blocks[blockNum] = uint64ToHash(blockNum)

	input, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.ErrorContains(t, err, fmt.Sprintf("failed to retrieve receipts for block %v", blockNum))
	require.Nil(t, input)
}

func TestFetchLeaves_ErrorsWhenNoValidLeavesInBlock(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}

	// Irrelevant tx - reverted
	proposal1 := oracle.createProposal(input2)
	l1Source.createReceipt(blockNum, types.ReceiptStatusFailed, proposal1)
	// Irrelevant tx - no logs are emitted
	l1Source.createReceipt(blockNum, types.ReceiptStatusSuccessful)

	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.ErrorIs(t, err, ErrNoLeavesFound)
	require.Nil(t, inputs)
}

func setupFetcherTest(t *testing.T) (*InputFetcher, *stubOracle, *stubL1Source) {
	oracle := &stubOracle{
		proposals: make(map[byte]*proposalConfig),
	}
	l1Source := &stubL1Source{
		blocks:     make(map[uint64]common.Hash),
		rcpts:      make(map[common.Hash]types.Receipts),
		txs:        make(map[uint64]types.Transactions),
		rcptStatus: make(map[common.Hash]uint64),
		logs:       make(map[common.Hash][]*types.Log),
	}
	fetcher := NewPreimageFetcher(testlog.Logger(t, log.LevelTrace), l1Source)
	return fetcher, oracle, l1Source
}

type proposalConfig struct {
	id           byte
	claimantAddr common.Address
	inputData    keccakTypes.InputData
	uuid         *big.Int
	valid        bool
}

type stubOracle struct {
	leafBlocks     []uint64
	nextProposalId byte
	proposals      map[byte]*proposalConfig
	// Add a field to allow for mocking of errors
	inputDataBlocksError error
}

func (o *stubOracle) Addr() common.Address {
	return oracleAddr
}

func (o *stubOracle) GetInputDataBlocks(_ context.Context, _ rpcblock.Block, _ keccakTypes.LargePreimageIdent) ([]uint64, error) {
	if o.inputDataBlocksError != nil {
		return nil, o.inputDataBlocksError
	}
	return o.leafBlocks, nil
}

func (o *stubOracle) DecodeInputData(data []byte) (*big.Int, keccakTypes.InputData, error) {
	if len(data) == 0 {
		return nil, keccakTypes.InputData{}, contracts.ErrInvalidAddLeavesCall
	}
	proposalId := data[0]
	proposal, ok := o.proposals[proposalId]
	if !ok || !proposal.valid {
		return nil, keccakTypes.InputData{}, contracts.ErrInvalidAddLeavesCall
	}

	return proposal.uuid, proposal.inputData, nil
}

func (o *stubOracle) createProposal(input keccakTypes.InputData) *proposalConfig {
	id := o.nextProposalId
	o.nextProposalId++

	proposal := &proposalConfig{
		id:           id,
		claimantAddr: ident.Claimant,
		inputData:    input,
		uuid:         ident.UUID,
		valid:        true,
	}
	o.proposals[id] = proposal

	return proposal
}

type stubL1Source struct {
	nextTxId uint64

	// Map block number to block hash
	blocks map[uint64]common.Hash
	// Map block hash to receipts
	rcpts map[common.Hash]types.Receipts
	// Map block number to tx
	txs map[uint64]types.Transactions
	// Map txHash to receipt
	rcptStatus map[common.Hash]uint64
	// Map txHash to logs
	logs map[common.Hash][]*types.Log
}

func (s *stubL1Source) BlockRefByNumber(_ context.Context, num uint64) (eth.BlockRef, error) {
	hash, ok := s.blocks[num]
	if !ok {
		return eth.BlockRef{}, errors.New("not found")
	}
	return eth.BlockRef{
		Number: num,
		Hash:   hash,
	}, nil
}

func (s *stubL1Source) FetchReceipts(_ context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	rcpts, ok := s.rcpts[blockHash]
	if !ok {
		return nil, nil, errors.New("not found")
	}
	return nil, rcpts, nil
}

func uint64ToHash(num uint64) common.Hash {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, num)
	return crypto.Keccak256Hash(data)
}

func (s *stubL1Source) createReceipt(blockNum uint64, status uint64, proposals ...*proposalConfig) *types.Receipt {
	// Make the block exist
	s.blocks[blockNum] = uint64ToHash(blockNum)

	txId := s.nextTxId
	s.nextTxId++

	logs := make([]*types.Log, len(proposals))
	for i, proposal := range proposals {
		// Concat the claimant address and the proposal id
		// These will be split back into address and id in fetcher.extractRelevantLeavesFromTx
		data := append(proposal.claimantAddr[:], proposal.id)

		txLog := &types.Log{
			Address: oracleAddr,
			Data:    data,
			Topics:  []common.Hash{},

			// ignored (zeroed):
			BlockNumber: 0,
			TxHash:      common.Hash{},
			TxIndex:     0,
			BlockHash:   common.Hash{},
			Index:       0,
			Removed:     false,
		}
		logs[i] = txLog
	}
	rcpt := &types.Receipt{TxHash: uint64ToHash(txId), Status: status, Logs: logs}
	blockHash := s.blocks[blockNum]
	rcpts := s.rcpts[blockHash]
	s.rcpts[blockHash] = append(rcpts, rcpt)
	return rcpt
}
