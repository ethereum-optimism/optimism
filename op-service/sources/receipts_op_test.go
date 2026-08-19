package sources

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// opL2BlockAndReceipts builds a minimal L2 block — one deposit tx and one user
// tx — with receipts carrying the OP Stack extension fields, all consistent
// with the block's transactions root, receipts root, and block hash so that
// untrusted-RPC validation passes.
func opL2BlockAndReceipts(t *testing.T) (*RPCBlock, []*types.Receipt) {
	t.Helper()
	txs := l2BlockTxs(t)[:2] // deposit + legacy user tx
	rawTxs := mustRawTransactions(txs)

	logIdx := uint(0)
	mkLog := func(addr byte) []*types.Log {
		l := &types.Log{Address: common.Address{addr}, Topics: []common.Hash{{addr}}, Index: logIdx}
		logIdx++
		return []*types.Log{l}
	}
	depositReceipt := &types.Receipt{
		Type:                  optypes.DepositTxType,
		Status:                types.ReceiptStatusSuccessful,
		CumulativeGasUsed:     21_000,
		Logs:                  mkLog(1),
		DepositNonce:          ptr.New(uint64(7)),
		DepositReceiptVersion: ptr.New(types.CanyonDepositReceiptVersion),
	}
	depositReceipt.Bloom = types.CreateBloom(depositReceipt)
	userReceipt := &types.Receipt{
		Type:                types.LegacyTxType,
		Status:              types.ReceiptStatusSuccessful,
		CumulativeGasUsed:   42_000,
		Logs:                mkLog(2),
		L1GasPrice:          big.NewInt(42_000_000_000),
		L1BlobBaseFee:       big.NewInt(123_456),
		L1Fee:               big.NewInt(77_000),
		L1BaseFeeScalar:     ptr.New(uint64(5227)),
		L1BlobBaseFeeScalar: ptr.New(uint64(1_014_213)),
		OperatorFeeScalar:   ptr.New(uint64(9)),
		OperatorFeeConstant: ptr.New(uint64(9000)),
	}
	userReceipt.Bloom = types.CreateBloom(userReceipt)
	receipts := []*types.Receipt{depositReceipt, userReceipt}

	block := &RPCBlock{
		RPCHeader: RPCHeader{
			UncleHash:   types.EmptyUncleHash,
			Number:      hexutil.Uint64(101),
			Time:        hexutil.Uint64(1_700_000_000),
			GasUsed:     hexutil.Uint64(42_000),
			BaseFee:     (*hexutil.Big)(big.NewInt(7)),
			TxHash:      types.DeriveSha(rawTxs, trie.NewStackTrie(nil)),
			ReceiptHash: types.DeriveSha(types.Receipts(receipts), trie.NewStackTrie(nil)),
		},
		Transactions: rawTxs,
	}
	block.Hash = block.computeBlockHash()

	// Backfill the derived receipt metadata that validation checks.
	prevGas := uint64(0)
	for i, r := range receipts {
		r.TxHash = rawTxs[i].Hash()
		r.BlockHash = block.Hash
		r.BlockNumber = new(big.Int).SetUint64(uint64(block.Number))
		r.TransactionIndex = uint(i)
		r.GasUsed = r.CumulativeGasUsed - prevGas
		prevGas = r.CumulativeGasUsed
		for _, l := range r.Logs {
			l.BlockHash = block.Hash
			l.BlockNumber = uint64(block.Number)
			l.TxHash = r.TxHash
			l.TxIndex = uint(i)
		}
	}
	return block, receipts
}

func opReceiptsTestClient(t *testing.T, m *mock.Mock, kind RPCProviderKind) *EthClient {
	t.Helper()
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	require.NoError(t, srv.RegisterName("eth", &ethBackend{Mock: m}))
	require.NoError(t, srv.RegisterName("debug", &debugBackend{Mock: m}))

	cfg := DefaultEthClientConfig(10)
	cfg.RPCProviderKind = kind
	cfg.MustBePostMerge = false
	cfg.MethodResetDuration = time.Minute
	ethCl, err := NewEthClient(client.NewBaseRPCClient(rpc.DialInProc(srv)), testlog.Logger(t, log.LevelError), nil, cfg)
	require.NoError(t, err)
	t.Cleanup(ethCl.Close)
	return ethCl
}

// TestFetchReceiptsOPFields asserts that L2 receipts fetched over the JSON
// method carry the OP Stack extension fields, and that the deposit-aware
// receipts-root validation passes with untrusted RPC.
func TestFetchReceiptsOPFields(t *testing.T) {
	block, receipts := opL2BlockAndReceipts(t)
	m := new(mock.Mock)
	ethCl := opReceiptsTestClient(t, m, RPCKindStandard)

	m.On("eth_getBlockByHash", block.Hash, true).Once().Return(block)
	m.On("eth_getBlockReceipts", block.Hash.String()).Once().Return(receipts, new(error))

	_, got, err := ethCl.FetchReceipts(context.Background(), block.Hash)
	require.NoError(t, err)
	require.Len(t, got, 2)

	require.Equal(t, receipts[0].DepositNonce, got[0].DepositNonce)
	require.Equal(t, receipts[0].DepositReceiptVersion, got[0].DepositReceiptVersion)
	require.Equal(t, receipts[1].L1GasPrice, got[1].L1GasPrice)
	require.Equal(t, receipts[1].L1BlobBaseFee, got[1].L1BlobBaseFee)
	require.Equal(t, receipts[1].L1Fee, got[1].L1Fee)
	require.Equal(t, receipts[1].L1BaseFeeScalar, got[1].L1BaseFeeScalar)
	require.Equal(t, receipts[1].L1BlobBaseFeeScalar, got[1].L1BlobBaseFeeScalar)
	require.Equal(t, receipts[1].OperatorFeeScalar, got[1].OperatorFeeScalar)
	require.Equal(t, receipts[1].OperatorFeeConstant, got[1].OperatorFeeConstant)
	m.AssertExpectations(t)
}

// TestFetchReceiptsRawPath asserts the consensus-encoded receipts path
// (debug_getRawReceipts): a deposit receipt decodes with its consensus fields
// (nonce+version) and the deposit-aware root validation passes; the JSON-only
// fee fields are structurally absent and stay nil.
func TestFetchReceiptsRawPath(t *testing.T) {
	block, receipts := opL2BlockAndReceipts(t)
	m := new(mock.Mock)
	ethCl := opReceiptsTestClient(t, m, RPCKindQuickNode)

	var raw []hexutil.Bytes
	for _, r := range receipts {
		data, err := r.MarshalBinary()
		require.NoError(t, err)
		raw = append(raw, data)
	}
	m.On("eth_getBlockByHash", block.Hash, true).Once().Return(block)
	m.On("debug_getRawReceipts", block.Hash.String()).Once().Return(raw, new(error))

	_, got, err := ethCl.FetchReceipts(context.Background(), block.Hash)
	require.NoError(t, err)
	require.Len(t, got, 2)

	require.Equal(t, receipts[0].DepositNonce, got[0].DepositNonce)
	require.Equal(t, receipts[0].DepositReceiptVersion, got[0].DepositReceiptVersion)
	require.Nil(t, got[1].L1GasPrice, "JSON-only fee fields are not in the consensus encoding")
	require.Nil(t, got[1].OperatorFeeScalar)
	m.AssertExpectations(t)
}

// TestTransactionReceiptOPFields asserts the single-receipt fetch decodes the
// OP Stack extension fields.
func TestTransactionReceiptOPFields(t *testing.T) {
	_, receipts := opL2BlockAndReceipts(t)
	m := new(mock.Mock)
	ethCl := opReceiptsTestClient(t, m, RPCKindStandard)

	userReceipt := receipts[1]
	m.On("eth_getTransactionReceipt", userReceipt.TxHash).Once().Return(userReceipt, new(error))

	got, err := ethCl.TransactionReceipt(context.Background(), userReceipt.TxHash)
	require.NoError(t, err)
	require.Equal(t, userReceipt.L1GasPrice, got.L1GasPrice)
	require.Equal(t, userReceipt.L1Fee, got.L1Fee)
	require.Equal(t, userReceipt.OperatorFeeScalar, got.OperatorFeeScalar)
	require.Equal(t, userReceipt.OperatorFeeConstant, got.OperatorFeeConstant)
	require.Equal(t, userReceipt.GasUsed, got.GasUsed)
	m.AssertExpectations(t)
}
