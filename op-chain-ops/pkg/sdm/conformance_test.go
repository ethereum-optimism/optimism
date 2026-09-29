package sdm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

type conformanceRPC struct {
	block    *RPCBlock
	resolved rpcTransactionDetails
	rawTx    hexutil.Bytes
	receipts map[common.Hash]*RPCReceipt
	replay   *ReplaySDMBlock
}

func (f *conformanceRPC) CallContext(_ context.Context, result any, method string, args ...any) error {
	var value any
	switch method {
	case "eth_getBlockByNumber":
		value = f.block
	case "eth_getTransactionByHash":
		value = f.resolved
	case "eth_getRawTransactionByHash":
		value = f.rawTx
	case "eth_getTransactionReceipt":
		hash, ok := args[0].(common.Hash)
		if !ok {
			return fmt.Errorf("receipt argument is %T", args[0])
		}
		value = f.receipts[hash]
	case "debug_replaySDMBlock":
		value = f.replay
	default:
		return fmt.Errorf("unexpected method %s", method)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, result)
}

func uint64Ptr(value uint64) *hexutil.Uint64 {
	converted := hexutil.Uint64(value)
	return &converted
}

func bigPtr(value int64) *hexutil.Big {
	converted := hexutil.Big(*big.NewInt(value))
	return &converted
}

func addressPtr(value common.Address) *common.Address {
	return &value
}

func conformanceFixture(t *testing.T) *conformanceRPC {
	blockNum := uint64(42)
	depositHash := common.HexToHash("0x01")
	userHash := common.HexToHash("0x02")
	blockHash := common.HexToHash("0xbeef")
	payload := optypes.PostExecPayload{
		Version:          optypes.PostExecPayloadVersion,
		BlockNumber:      blockNum,
		GasRefundEntries: []optypes.SDMGasEntry{{Index: 1, GasRefund: 7}},
	}
	input, err := rlp.EncodeToBytes(payload)
	require.NoError(t, err)
	postTx := types.NewTx(&types.PostExecTx{Data: input})
	postHash := postTx.Hash()
	zero := common.Address{}

	block := &RPCBlock{
		Number:      hexutil.Uint64(blockNum),
		Hash:        blockHash,
		GasUsed:     193,
		BlobGasUsed: uint64Ptr(123),
		Timestamp:   1_000,
		Transactions: []RPCTransaction{
			{Hash: depositHash, Type: types.DepositTxType},
			{Hash: userHash, Type: types.DynamicFeeTxType},
			{Hash: postHash, Type: optypes.PostExecTxType, Input: input},
		},
	}
	blockFields := func(receipt *RPCReceipt) {
		receipt.L1GasPrice = bigPtr(10)
		receipt.L1BaseFeeScalar = bigPtr(1_368)
		receipt.L1BlobBaseFee = bigPtr(57)
		receipt.L1BlobBaseFeeScalar = bigPtr(801_949)
		receipt.OperatorFeeScalar = bigPtr(0)
		receipt.OperatorFeeConstant = bigPtr(0)
		receipt.DAFootprintGasScalar = bigPtr(400)
	}
	depositReceipt := &RPCReceipt{Type: types.DepositTxType, TxHash: depositHash, BlockHash: blockHash, TransactionIndex: 0, GasUsed: 100, BlobGasUsed: uint64Ptr(99)}
	userReceipt := &RPCReceipt{Type: types.DynamicFeeTxType, TxHash: userHash, BlockHash: blockHash, TransactionIndex: 1, GasUsed: 93, BlobGasUsed: uint64Ptr(123), OPGasRefund: uint64Ptr(7), L1Fee: bigPtr(3), L1GasUsed: bigPtr(4)}
	postReceipt := &RPCReceipt{Type: optypes.PostExecTxType, TxHash: postHash, BlockHash: blockHash, TransactionIndex: 2, Status: hexutil.Uint64(types.ReceiptStatusSuccessful), GasUsed: 0, EffectiveGasPrice: bigPtr(0), BlobGasUsed: uint64Ptr(0), L1Fee: bigPtr(0), L1GasUsed: bigPtr(0)}
	blockFields(userReceipt)
	blockFields(postReceipt)

	refund := uint64(7)
	postIndex := uint64(2)
	replay := &ReplaySDMBlock{
		BlockNum: blockNum, BlockHash: blockHash, PostExecTxPresent: true,
		PostExecTxIndex: &postIndex, EmbeddedPayload: &payload,
		Txs: []ReplaySDMTx{
			{TxIndex: 0, ReplayTxIndex: 0, TxHash: depositHash, TxType: types.DepositTxType, IsDepositTx: true, RawGasUsed: 100, CanonicalGasUsed: 100},
			{TxIndex: 1, ReplayTxIndex: 1, TxHash: userHash, TxType: types.DynamicFeeTxType, RawGasUsed: 100, CanonicalGasUsed: 93, OPGasRefundPayload: &refund},
		},
		Summary: ReplaySDMSummary{
			BlockNum: blockNum, BlockHash: blockHash, TxCountTotal: 2, TxCountUser: 1,
			PostExecTxPresent: true, PostExecPayloadEntryCount: 1,
			BlockGasUsed: 193, BlockRawGasUsed: 200, PayloadRefundTotal: 7,
		},
	}
	resolved := rpcTransactionDetails{
		RPCTransaction: block.Transactions[2],
		From:           zero, To: addressPtr(zero), Nonce: uint64Ptr(0), Gas: 0,
		GasPrice: bigPtr(0), Value: bigPtr(0),
	}
	return &conformanceRPC{
		block: block, resolved: resolved, rawTx: append(hexutil.Bytes{byte(optypes.PostExecTxType)}, input...), replay: replay,
		receipts: map[common.Hash]*RPCReceipt{
			depositHash: depositReceipt,
			userHash:    userReceipt,
			postHash:    postReceipt,
		},
	}
}

func TestValidatePostExecConformance(t *testing.T) {
	fixture := conformanceFixture(t)
	result, err := ValidatePostExecConformance(context.Background(), fixture, 42)
	require.NoError(t, err)
	require.Equal(t, uint64(7), result.TotalPayloadRefund)
	require.Len(t, result.Receipts, 3)
	_, err = json.Marshal(result)
	require.NoError(t, err)
}

func TestValidatePostExecConformanceAcceptsGasPrice(t *testing.T) {
	fixture := conformanceFixture(t)
	fixture.resolved.GasPrice = bigPtr(1)
	_, err := ValidatePostExecConformance(context.Background(), fixture, 42)
	require.NoError(t, err)
}

func postExecReceipt(f *conformanceRPC) *RPCReceipt {
	return f.receipts[f.block.Transactions[len(f.block.Transactions)-1].Hash]
}

func userReceipt(f *conformanceRPC) *RPCReceipt {
	return f.receipts[f.block.Transactions[1].Hash]
}

// addUntargetedUserTx inserts a second regular transaction, which has no
// payload entry, just before the PostExec transaction.
func addUntargetedUserTx(f *conformanceRPC) *RPCReceipt {
	hash := common.HexToHash("0x03")
	post := f.block.Transactions[2]
	f.block.Transactions = []RPCTransaction{f.block.Transactions[0], f.block.Transactions[1], {Hash: hash, Type: types.DynamicFeeTxType}, post}
	receipt := *userReceipt(f)
	receipt.TxHash = hash
	receipt.TransactionIndex = 2
	receipt.OPGasRefund = nil
	receipt.BlobGasUsed = uint64Ptr(0)
	f.receipts[hash] = &receipt
	postExecReceipt(f).TransactionIndex = 3
	postIndex := uint64(3)
	f.replay.PostExecTxIndex = &postIndex
	return &receipt
}

func TestValidatePostExecConformanceFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*conformanceRPC)
		want   string
	}{
		{
			name: "noncanonical hash",
			mutate: func(f *conformanceRPC) {
				f.block.Transactions[2].Hash = common.HexToHash("0xbad")
				f.resolved.Hash = common.HexToHash("0xbad")
			},
			want: "canonical hash",
		},
		{name: "replay embedded payload mismatch", mutate: func(f *conformanceRPC) { f.replay.EmbeddedPayload.GasRefundEntries[0].GasRefund = 0 }, want: "replay embedded payload"},
		{name: "resolved post-exec identity", mutate: func(f *conformanceRPC) { f.resolved = rpcTransactionDetails{} }, want: "identity mismatch"},
		{name: "raw bytes", mutate: func(f *conformanceRPC) { f.rawTx = append(f.rawTx, 0) }, want: "non-canonical raw EIP-2718 bytes"},
		{name: "sender", mutate: func(f *conformanceRPC) { f.resolved.From = common.HexToAddress("0x01") }, want: "from is"},
		{name: "recipient", mutate: func(f *conformanceRPC) { f.resolved.To = addressPtr(common.HexToAddress("0x01")) }, want: "to is"},
		{name: "nonce", mutate: func(f *conformanceRPC) { f.resolved.Nonce = uint64Ptr(1) }, want: "nonce is 1"},
		{name: "gas", mutate: func(f *conformanceRPC) { f.resolved.Gas = 1 }, want: "gas is 1"},
		{name: "value", mutate: func(f *conformanceRPC) { f.resolved.Value = bigPtr(1) }, want: "value is 1"},
		{name: "receipt block hash", mutate: func(f *conformanceRPC) { userReceipt(f).BlockHash = common.HexToHash("0xdead") }, want: "block hash"},
		{name: "receipt index", mutate: func(f *conformanceRPC) { userReceipt(f).TransactionIndex = 5 }, want: "transaction index 5, want 1"},
		{name: "post-exec receipt type", mutate: func(f *conformanceRPC) { postExecReceipt(f).Type = types.DynamicFeeTxType }, want: "receipt type"},
		{name: "post-exec status", mutate: func(f *conformanceRPC) { postExecReceipt(f).Status = 0 }, want: "status is 0"},
		{name: "post-exec gas used", mutate: func(f *conformanceRPC) { postExecReceipt(f).GasUsed = 1 }, want: "gasUsed is 1"},
		{name: "post-exec effective gas price omitted", mutate: func(f *conformanceRPC) { postExecReceipt(f).EffectiveGasPrice = nil }, want: "effectiveGasPrice is <absent>"},
		{name: "post-exec blob gas omitted", mutate: func(f *conformanceRPC) { postExecReceipt(f).BlobGasUsed = nil }, want: "blobGasUsed is <absent>"},
		{name: "post-exec refund", mutate: func(f *conformanceRPC) { postExecReceipt(f).OPGasRefund = uint64Ptr(1) }, want: "unexpectedly exposes opGasRefund"},
		{name: "post-exec l1 fee", mutate: func(f *conformanceRPC) { postExecReceipt(f).L1Fee = bigPtr(1) }, want: "l1Fee is 1, want explicit zero"},
		{name: "post-exec l1 gas used", mutate: func(f *conformanceRPC) { postExecReceipt(f).L1GasUsed = nil }, want: "l1GasUsed is <absent>"},
		{name: "post-exec l1 fee scalar", mutate: func(f *conformanceRPC) { postExecReceipt(f).L1FeeScalar = bigPtr(1) }, want: "removed l1FeeScalar"},
		{name: "block-scoped field", mutate: func(f *conformanceRPC) { postExecReceipt(f).L1BlobBaseFee = bigPtr(58) }, want: "l1BlobBaseFee is 58"},
		{name: "block-scoped field omitted", mutate: func(f *conformanceRPC) { postExecReceipt(f).OperatorFeeScalar = nil }, want: "operatorFeeScalar is <absent>"},
		{name: "deposit refund", mutate: func(f *conformanceRPC) { f.receipts[f.block.Transactions[0].Hash].OPGasRefund = uint64Ptr(1) }, want: "deposit receipt"},
		{name: "regular receipt omits DA footprint", mutate: func(f *conformanceRPC) { userReceipt(f).BlobGasUsed = nil }, want: "omits blobGasUsed"},
		{name: "header omits DA footprint", mutate: func(f *conformanceRPC) { f.block.BlobGasUsed = nil }, want: "block 42 omits blobGasUsed"},
		{name: "DA footprint mismatch", mutate: func(f *conformanceRPC) { f.block.BlobGasUsed = uint64Ptr(124) }, want: "receipt blobGasUsed total"},
		{name: "untargeted refund", mutate: func(f *conformanceRPC) { addUntargetedUserTx(f).OPGasRefund = uint64Ptr(1) }, want: "without a payload entry"},
		{name: "replay block identity", mutate: func(f *conformanceRPC) { f.replay.BlockHash = common.HexToHash("0xdead") }, want: "replay block identity"},
		{name: "replay mismatch count", mutate: func(f *conformanceRPC) { f.replay.Summary.MismatchCount = 1 }, want: "reports 1 mismatches"},
		{name: "replay block gas", mutate: func(f *conformanceRPC) { f.replay.Summary.BlockGasUsed++ }, want: "replay block gas used"},
		{name: "replay raw block gas", mutate: func(f *conformanceRPC) { f.replay.Summary.BlockRawGasUsed++ }, want: "raw block gas"},
		{name: "replay row count", mutate: func(f *conformanceRPC) { f.replay.Txs = f.replay.Txs[:1] }, want: "replay contains 1 rows"},
		{name: "replay duplicate row", mutate: func(f *conformanceRPC) { f.replay.Txs[1].TxIndex = 0 }, want: "duplicate transaction index 0"},
		{name: "replay row hash", mutate: func(f *conformanceRPC) { f.replay.Txs[1].TxHash = common.HexToHash("0xdead") }, want: "replay transaction 1 hash"},
		{name: "replay canonical gas", mutate: func(f *conformanceRPC) { f.replay.Txs[1].CanonicalGasUsed = 94 }, want: "canonical gas 94, want receipt gasUsed 93"},
		{name: "replay deposit refund", mutate: func(f *conformanceRPC) { f.replay.Txs[0].RawGasUsed = 101 }, want: "deposit replay row 0"},
		{name: "replay row raw gas", mutate: func(f *conformanceRPC) { f.replay.Txs[1].RawGasUsed = 101 }, want: "replay transaction 1 raw gas 101"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := conformanceFixture(t)
			test.mutate(fixture)
			_, err := ValidatePostExecConformance(context.Background(), fixture, 42)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), test.want), "error %q does not contain %q", err, test.want)
		})
	}
}

func TestValidatePostExecConformanceAcceptsUntargetedTransaction(t *testing.T) {
	fixture := conformanceFixture(t)
	addUntargetedUserTx(fixture)
	fixture.replay.Txs = append(fixture.replay.Txs, ReplaySDMTx{TxIndex: 2, ReplayTxIndex: 2, TxHash: fixture.block.Transactions[2].Hash, RawGasUsed: 93, CanonicalGasUsed: 93})
	fixture.replay.Summary.TxCountTotal = 3
	_, err := ValidatePostExecConformance(context.Background(), fixture, 42)
	require.NoError(t, err)
}

func TestValidatePayloadEntries(t *testing.T) {
	validation := &ValidationResult{Payload: &optypes.PostExecPayload{
		GasRefundEntries: []optypes.SDMGasEntry{{Index: 1, GasRefund: 3}, {Index: 2, GasRefund: 4}},
	}}
	require.NoError(t, validatePayloadEntries(validation))

	validation.Payload.GasRefundEntries = []optypes.SDMGasEntry{{Index: 2, GasRefund: 3}, {Index: 2, GasRefund: 4}}
	require.ErrorContains(t, validatePayloadEntries(validation), "not strictly increasing")

	validation.Payload.GasRefundEntries = []optypes.SDMGasEntry{{Index: 2, GasRefund: 3}, {Index: 1, GasRefund: 4}}
	require.ErrorContains(t, validatePayloadEntries(validation), "not strictly increasing")

	validation.Payload.GasRefundEntries = []optypes.SDMGasEntry{{Index: 1, GasRefund: 0}}
	require.ErrorContains(t, validatePayloadEntries(validation), "zero refund")
}

func TestValidateVerifierAgreement(t *testing.T) {
	producerRPC := conformanceFixture(t)
	producer, err := ValidatePostExecConformance(context.Background(), producerRPC, 42)
	require.NoError(t, err)

	verifier := conformanceFixture(t)
	require.NoError(t, ValidateVerifierAgreement(context.Background(), producer, verifier))

	verifier.block.Hash = common.HexToHash("0xdead")
	err = ValidateVerifierAgreement(context.Background(), producer, verifier)
	require.ErrorContains(t, err, "verifier block 42 hash")

	verifier = conformanceFixture(t)
	userHash := verifier.block.Transactions[1].Hash
	verifier.receipts[userHash].OPGasRefund = uint64Ptr(8)
	err = ValidateVerifierAgreement(context.Background(), producer, verifier)
	require.ErrorContains(t, err, "opGasRefund 8, want 7")
}
