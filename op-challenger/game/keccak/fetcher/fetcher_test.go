package fetcher

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	oracleAddr = common.Address{0x99, 0x98}
	privKey, _ = crypto.GenerateKey()
	ident      = keccakTypes.LargePreimageIdent{
		Claimant: crypto.PubkeyToAddress(privKey.PublicKey),
		UUID:     big.NewInt(888),
	}
	chainID   = big.NewInt(123)
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

func TestFetchLeaves_SingleTx(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}
	l1Source.txs[blockNum] = types.Transactions{oracle.txForInput(ValidTx, input1)}
	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_MultipleBlocksAndLeaves(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	block1 := uint64(7)
	block2 := uint64(15)
	block3 := uint64(20)
	oracle.leafBlocks = []uint64{block1, block2, block3}
	l1Source.txs[block1] = types.Transactions{oracle.txForInput(ValidTx, input1)}
	l1Source.txs[block2] = types.Transactions{oracle.txForInput(ValidTx, input2)}
	l1Source.txs[block3] = types.Transactions{oracle.txForInput(ValidTx, input3), oracle.txForInput(ValidTx, input4)}
	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1, input2, input3, input4}, inputs)
}

func TestFetchLeaves_SkipTxToWrongContract(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}
	// Valid tx but to a different contract
	tx1 := oracle.txForInput(WithToAddr(common.Address{0x88, 0x99, 0x11}), input2)
	// Valid tx but without a to addr
	tx2 := oracle.txForInput(WithoutToAddr(), input2)
	// Valid tx to the correct contract
	tx3 := oracle.txForInput(ValidTx, input1)
	l1Source.txs[blockNum] = types.Transactions{tx1, tx2, tx3}
	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipTxWithDifferentUUID(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}
	// Valid tx but with a different UUID
	tx1 := oracle.txForInput(WithUUID(big.NewInt(874927294)), input2)
	// Valid tx
	tx2 := oracle.txForInput(ValidTx, input1)
	l1Source.txs[blockNum] = types.Transactions{tx1, tx2}
	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipTxWithInvalidCall(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}
	// Call to preimage oracle but fails to decode
	tx1 := oracle.txForInput(WithInvalidData(), input2)
	// Valid tx
	tx2 := oracle.txForInput(ValidTx, input1)
	l1Source.txs[blockNum] = types.Transactions{tx1, tx2}
	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipTxWithInvalidSender(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}
	// Call to preimage oracle with different Chain ID
	tx1 := oracle.txForInput(WithChainID(big.NewInt(992)), input3)
	// Call to preimage oracle with wrong sender
	wrongKey, _ := crypto.GenerateKey()
	tx2 := oracle.txForInput(WithPrivKey(wrongKey), input4)
	// Valid tx
	tx3 := oracle.txForInput(ValidTx, input1)
	l1Source.txs[blockNum] = types.Transactions{tx1, tx2, tx3}
	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_SkipTxWithReceiptStatusFail(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}
	// Valid call to the preimage oracle but that reverted
	tx1 := oracle.txForInput(ValidTx, input2)
	l1Source.rcptStatus[tx1.Hash()] = types.ReceiptStatusFailed
	// Valid tx
	tx2 := oracle.txForInput(ValidTx, input1)
	l1Source.txs[blockNum] = types.Transactions{tx1, tx2}
	inputs, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.InputData{input1}, inputs)
}

func TestFetchLeaves_ErrorsWhenNoValidLeavesInBlock(t *testing.T) {
	fetcher, oracle, l1Source := setupFetcherTest(t)
	blockNum := uint64(7)
	oracle.leafBlocks = []uint64{blockNum}
	// Irrelevant call
	tx1 := oracle.txForInput(WithUUID(big.NewInt(492)), input2)
	l1Source.rcptStatus[tx1.Hash()] = types.ReceiptStatusFailed
	l1Source.txs[blockNum] = types.Transactions{tx1}
	_, err := fetcher.FetchInputs(context.Background(), blockHash, oracle, ident)
	require.ErrorIs(t, err, ErrNoLeavesFound)
}

func setupFetcherTest(t *testing.T) (*InputFetcher, *stubOracle, *stubL1Source) {
	oracle := &stubOracle{
		txInputs: make(map[byte]keccakTypes.InputData),
	}
	l1Source := &stubL1Source{
		txs:        make(map[uint64]types.Transactions),
		rcptStatus: make(map[common.Hash]uint64),
	}
	fetcher := NewPreimageFetcher(testlog.Logger(t, log.LevelTrace), l1Source)
	return fetcher, oracle, l1Source
}

type stubOracle struct {
	nextTxId   byte
	leafBlocks []uint64
	txInputs   map[byte]keccakTypes.InputData
}

func (o *stubOracle) Addr() common.Address {
	return oracleAddr
}

func (o *stubOracle) GetInputDataBlocks(_ context.Context, _ rpcblock.Block, _ keccakTypes.LargePreimageIdent) ([]uint64, error) {
	return o.leafBlocks, nil
}

func (o *stubOracle) DecodeInputData(data []byte) (*big.Int, keccakTypes.InputData, error) {
	if len(data) == 0 {
		return nil, keccakTypes.InputData{}, contracts.ErrInvalidAddLeavesCall
	}
	input, ok := o.txInputs[data[0]]
	if !ok {
		return nil, keccakTypes.InputData{}, contracts.ErrInvalidAddLeavesCall
	}
	uuid := ident.UUID
	// WithUUID appends custom UUIDs to the tx data
	if len(data) > 1 {
		uuid = new(big.Int).SetBytes(data[1:])
	}
	return uuid, input, nil
}

type TxModifier func(tx *types.DynamicFeeTx) *ecdsa.PrivateKey

var ValidTx TxModifier = func(_ *types.DynamicFeeTx) *ecdsa.PrivateKey {
	return privKey
}

func WithToAddr(addr common.Address) TxModifier {
	return func(tx *types.DynamicFeeTx) *ecdsa.PrivateKey {
		tx.To = &addr
		return privKey
	}
}

func WithoutToAddr() TxModifier {
	return func(tx *types.DynamicFeeTx) *ecdsa.PrivateKey {
		tx.To = nil
		return privKey
	}
}

func WithUUID(uuid *big.Int) TxModifier {
	return func(tx *types.DynamicFeeTx) *ecdsa.PrivateKey {
		tx.Data = append(tx.Data, uuid.Bytes()...)
		return privKey
	}
}

func WithInvalidData() TxModifier {
	return func(tx *types.DynamicFeeTx) *ecdsa.PrivateKey {
		tx.Data = []byte{}
		return privKey
	}
}

func WithChainID(id *big.Int) TxModifier {
	return func(tx *types.DynamicFeeTx) *ecdsa.PrivateKey {
		tx.ChainID = id
		return privKey
	}
}

func WithPrivKey(key *ecdsa.PrivateKey) TxModifier {
	return func(tx *types.DynamicFeeTx) *ecdsa.PrivateKey {
		return key
	}
}

func (o *stubOracle) txForInput(txMod TxModifier, input keccakTypes.InputData) *types.Transaction {
	id := o.nextTxId
	o.nextTxId++
	o.txInputs[id] = input
	inner := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     1,
		To:        &oracleAddr,
		Value:     big.NewInt(0),
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Gas:       3,
		Data:      []byte{id},
	}
	key := txMod(inner)
	tx := types.MustSignNewTx(key, types.LatestSignerForChainID(inner.ChainID), inner)
	return tx
}

type stubL1Source struct {
	txs        map[uint64]types.Transactions
	rcptStatus map[common.Hash]uint64
}

func (s *stubL1Source) ChainID(_ context.Context) (*big.Int, error) {
	return chainID, nil
}

func (s *stubL1Source) BlockByNumber(_ context.Context, number *big.Int) (*types.Block, error) {
	txs, ok := s.txs[number.Uint64()]
	if !ok {
		return nil, errors.New("not found")
	}
	return (&types.Block{}).WithBody(txs, nil), nil
}

func (s *stubL1Source) TransactionReceipt(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
	rcptStatus, ok := s.rcptStatus[txHash]
	if !ok {
		rcptStatus = types.ReceiptStatusSuccessful
	}
	return &types.Receipt{Status: rcptStatus}, nil
}
