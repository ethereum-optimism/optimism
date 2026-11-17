package types_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// The vectors in this file are the values op-geth computes for the same inputs.

// zeroBytesHex is the hex encoding, without prefix, of n zero bytes.
func zeroBytesHex(n int) string {
	return strings.Repeat("00", n)
}

// TestDepositTxGoldenVectors pins the binary encoding, the hash and the RPC JSON encoding
// of every deposit in depositTxTestCases, and that the JSON decodes to the same deposit.
func TestDepositTxGoldenVectors(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tc.tx.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, tc.binary, hexutil.Encode(raw))
			require.Equal(t, tc.hash, tc.tx.Hash().Hex())

			data, err := json.Marshal(&tc.tx)
			require.NoError(t, err)
			require.JSONEq(t, tc.json, string(data))

			var decoded optypes.DepositTx
			require.NoError(t, json.Unmarshal([]byte(tc.json), &decoded))
			reencoded, err := decoded.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, tc.binary, hexutil.Encode(reencoded))
		})
	}
}

// depositTxWithNonceJSON is the first of depositTxTestCases as an RPC response serves it
// once the deposit is mined after Regolith: with the deposit nonce.
const depositTxWithNonceJSON = `{"type":"0x7e","nonce":"0x7","to":"0x4242424242424242424242424242424242424242","gas":"0x5208","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x162e","input":"0xdeadbeef","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000001","from":"0x1111111111111111111111111111111111111111","mint":"0x4d2","isSystemTx":false,"hash":"0xa549b7517be66c7890e09cf1f598b8faefbe83f997f59f96b9fc99b60e75bd8a"}`

// depositTxWithoutIsSystemTxJSON returns the RPC JSON encoding of tc without the
// isSystemTx key, which decodes as false.
func depositTxWithoutIsSystemTxJSON(t *testing.T, tc depositTxTestCase) []byte {
	t.Helper()
	obj := toJSONMap(t, []byte(tc.json))
	delete(obj, "isSystemTx")
	data, err := json.Marshal(obj)
	require.NoError(t, err)
	return data
}

// TestDepositTxJSONDecodeGoldenVectors decodes deposit JSON in shapes that the encoder
// does not produce: with the deposit nonce, which is not part of the transaction, and
// without isSystemTx.
func TestDepositTxJSONDecodeGoldenVectors(t *testing.T) {
	cases := depositTxTestCases()
	t.Run("with nonce", func(t *testing.T) {
		var d optypes.DepositTx
		require.NoError(t, json.Unmarshal([]byte(depositTxWithNonceJSON), &d))
		raw, err := d.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, cases[0].binary, hexutil.Encode(raw))
	})
	for _, tc := range cases {
		if tc.tx.IsSystemTransaction {
			continue
		}
		t.Run("without isSystemTx/"+tc.name, func(t *testing.T) {
			var d optypes.DepositTx
			require.NoError(t, json.Unmarshal(depositTxWithoutIsSystemTxJSON(t, tc), &d))
			raw, err := d.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, tc.binary, hexutil.Encode(raw))
		})
	}
}

// postExecPayloadData returns the RLP encoding of p, the Data of a post-exec transaction.
func postExecPayloadData(p optypes.PostExecPayload) []byte {
	data, err := rlp.EncodeToBytes(&p)
	if err != nil {
		panic(err)
	}
	return data
}

// postExecTxJSONVectors are post-exec transactions with their RPC JSON encoding, which
// holds the hash that op-alloy computes too. The first is the payload of
// TestPostExecTxHashGoldenVector.
var postExecTxJSONVectors = []struct {
	data []byte
	json string
}{
	{
		data: postExecPayloadData(optypes.PostExecPayload{Version: optypes.PostExecPayloadVersion, BlockNumber: 42,
			GasRefundEntries: []optypes.SDMGasEntry{{Index: 3, GasRefund: 7}}}),
		json: `{"type":"0x7d","nonce":null,"to":null,"gas":"0x0","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0xc6012ac3c20307","v":null,"r":null,"s":null,"hash":"0xcbc64a9665039744307b562634bf760d96f3bed235fecd9e09a1f1276a1823c7"}`,
	},
	{
		data: postExecPayloadData(optypes.PostExecPayload{Version: optypes.PostExecPayloadVersion, BlockNumber: 1,
			GasRefundEntries: []optypes.SDMGasEntry{{Index: 1, GasRefund: 21_000}, {Index: 2, GasRefund: 1}}}),
		json: `{"type":"0x7d","nonce":null,"to":null,"gas":"0x0","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0xcb0101c8c401825208c20201","v":null,"r":null,"s":null,"hash":"0xad5a96debf4068e1ccd92a170a601f1be244547f7e47fde3244dac73d494482d"}`,
	},
	{
		data: postExecPayloadData(optypes.PostExecPayload{Version: optypes.PostExecPayloadVersion, BlockNumber: 123_456_789,
			GasRefundEntries: []optypes.SDMGasEntry{{Index: 1, GasRefund: 21_000}, {Index: 5, GasRefund: 4800}, {Index: 17, GasRefund: 100_000}}}),
		json: `{"type":"0x7d","nonce":null,"to":null,"gas":"0x0","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0xd70184075bcd15d0c401825208c4058212c0c511830186a0","v":null,"r":null,"s":null,"hash":"0x99f492786fb9a8224176e5d79693f3d61e4ef216320f9d5e3f612972b42f1bb0"}`,
	},
}

func TestPostExecTxJSONGoldenVectors(t *testing.T) {
	for _, v := range postExecTxJSONVectors {
		t.Run(hexutil.Encode(v.data), func(t *testing.T) {
			data, err := json.Marshal(&optypes.PostExecTx{Data: v.data})
			require.NoError(t, err)
			require.JSONEq(t, v.json, string(data))

			var decoded optypes.PostExecTx
			require.NoError(t, json.Unmarshal([]byte(v.json), &decoded))
			require.Equal(t, v.data, decoded.Data)
		})
	}
}

// zeroBloomRLP is the RLP encoding of an empty logs bloom.
var zeroBloomRLP = "b90100" + zeroBytesHex(256)

// receiptLogs are the logs of the receipt vectors that have logs.
func receiptLogs() []*gethtypes.Log {
	return []*gethtypes.Log{{
		Address: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		Topics:  []common.Hash{common.HexToHash("0x02"), common.HexToHash("0x03")},
		Data:    []byte{0x01, 0x02},
	}}
}

// logsBloomRLP and logsRLP are the RLP encodings of the logs bloom and the logs of
// receiptLogs.
const (
	logsBloomRLP = "b90100" +
		"0400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000" +
		"0000000000000000400000000000000000000000000000000000000000000000000000000000000000010100000000000000000000800000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008000000000000000000000"
	logsRLP = "f85ef85c942222222222222222222222222222222222222222f842a00000000000000000000000000000000000000000000000000000000000000002a00000000000000000000000000000000000000000000000000000000000000003820102"
)

type receiptVector struct {
	name    string
	receipt *optypes.Receipt
	// preimage is the receipt's encoding in the receipts trie (Receipts.EncodeIndex).
	preimage string
	// binary is the receipt's MarshalBinary encoding, which has the deposit nonce whenever
	// it is set.
	binary string
	// rlp is the receipt's RLP encoding: binary as an RLP string.
	rlp string
}

// receiptVectors are deposit receipts in each consensus shape and a post-exec receipt:
// before Regolith a deposit receipt has no nonce, from Regolith it has a nonce that the
// receipts trie leaves out, and from Canyon it has a nonce and a receipt version, both in
// the receipts trie.
func receiptVectors() []receiptVector {
	mk := func(typ uint8, status uint64, logs []*gethtypes.Log, nonce, version *uint64) *optypes.Receipt {
		r := &optypes.Receipt{
			Receipt: gethtypes.Receipt{
				Type:              typ,
				Status:            status,
				CumulativeGasUsed: 42_000,
				Logs:              logs,
			},
			DepositNonce:          nonce,
			DepositReceiptVersion: version,
		}
		r.Bloom = gethtypes.CreateBloom(&r.Receipt)
		return r
	}
	noLogs := []*gethtypes.Log{}
	return []receiptVector{
		{
			name:     "deposit pre-Regolith",
			receipt:  mk(optypes.DepositTxType, gethtypes.ReceiptStatusSuccessful, noLogs, nil, nil),
			preimage: "0x7ef901080182a410" + zeroBloomRLP + "c0",
			binary:   "0x7ef901080182a410" + zeroBloomRLP + "c0",
			rlp:      "0xb9010c7ef901080182a410" + zeroBloomRLP + "c0",
		},
		{
			name:     "deposit pre-Canyon",
			receipt:  mk(optypes.DepositTxType, gethtypes.ReceiptStatusSuccessful, noLogs, ptr.New[uint64](7), nil),
			preimage: "0x7ef901080182a410" + zeroBloomRLP + "c0",
			binary:   "0x7ef901090182a410" + zeroBloomRLP + "c007",
			rlp:      "0xb9010d7ef901090182a410" + zeroBloomRLP + "c007",
		},
		{
			name:     "deposit post-Canyon",
			receipt:  mk(optypes.DepositTxType, gethtypes.ReceiptStatusFailed, noLogs, ptr.New[uint64](8), ptr.New[uint64](optypes.CanyonDepositReceiptVersion)),
			preimage: "0x7ef9010a8082a410" + zeroBloomRLP + "c00801",
			binary:   "0x7ef9010a8082a410" + zeroBloomRLP + "c00801",
			rlp:      "0xb9010e7ef9010a8082a410" + zeroBloomRLP + "c00801",
		},
		{
			name:     "deposit post-Canyon with logs",
			receipt:  mk(optypes.DepositTxType, gethtypes.ReceiptStatusSuccessful, receiptLogs(), ptr.New[uint64](9), ptr.New[uint64](optypes.CanyonDepositReceiptVersion)),
			preimage: "0x7ef901690182a410" + logsBloomRLP + logsRLP + "0901",
			binary:   "0x7ef901690182a410" + logsBloomRLP + logsRLP + "0901",
			rlp:      "0xb9016d7ef901690182a410" + logsBloomRLP + logsRLP + "0901",
		},
		{
			name:     "post-exec",
			receipt:  mk(optypes.PostExecTxType, gethtypes.ReceiptStatusSuccessful, receiptLogs(), nil, nil),
			preimage: "0x7df901670182a410" + logsBloomRLP + logsRLP,
			binary:   "0x7df901670182a410" + logsBloomRLP + logsRLP,
			rlp:      "0xb9016b7df901670182a410" + logsBloomRLP + logsRLP,
		},
	}
}

// TestReceiptGoldenVectors pins the consensus encodings of OP Stack receipts, and that the
// binary and RLP encodings decode back to the receipt.
func TestReceiptGoldenVectors(t *testing.T) {
	for _, v := range receiptVectors() {
		t.Run(v.name, func(t *testing.T) {
			var preimage bytes.Buffer
			optypes.Receipts{v.receipt}.EncodeIndex(0, &preimage)
			require.Equal(t, v.preimage, hexutil.Encode(preimage.Bytes()))

			binary, err := v.receipt.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, v.binary, hexutil.Encode(binary))

			encoded, err := rlp.EncodeToBytes(v.receipt)
			require.NoError(t, err)
			require.Equal(t, v.rlp, hexutil.Encode(encoded))

			var fromBinary optypes.Receipt
			require.NoError(t, fromBinary.UnmarshalBinary(hexutil.MustDecode(v.binary)))
			var fromRLP optypes.Receipt
			require.NoError(t, rlp.DecodeBytes(hexutil.MustDecode(v.rlp), &fromRLP))
			for _, got := range []*optypes.Receipt{&fromBinary, &fromRLP} {
				require.Equal(t, v.receipt.Type, got.Type)
				require.Equal(t, v.receipt.Status, got.Status)
				require.Equal(t, v.receipt.CumulativeGasUsed, got.CumulativeGasUsed)
				require.Equal(t, v.receipt.Bloom, got.Bloom)
				require.Equal(t, len(v.receipt.Logs), len(got.Logs))
				require.Equal(t, v.receipt.DepositNonce, got.DepositNonce)
				require.Equal(t, v.receipt.DepositReceiptVersion, got.DepositReceiptVersion)
			}
		})
	}
}

// mixedReceipts are a block's receipts: a user receipt followed by the receipts of
// receiptVectors.
func mixedReceipts() optypes.Receipts {
	user := &optypes.Receipt{Receipt: gethtypes.Receipt{
		Type:              gethtypes.DynamicFeeTxType,
		Status:            gethtypes.ReceiptStatusSuccessful,
		CumulativeGasUsed: 21_000,
		Logs:              receiptLogs(),
	}}
	user.Bloom = gethtypes.CreateBloom(&user.Receipt)
	rs := optypes.Receipts{user}
	for _, v := range receiptVectors() {
		rs = append(rs, v.receipt)
	}
	return rs
}

const mixedReceiptsRoot = "0x6d289c66834e08f16072fcf0064c9cd1256e3e7981b3d87c4e72cc04be575c2a"

func TestReceiptsRootGoldenVector(t *testing.T) {
	require.Equal(t, mixedReceiptsRoot, gethtypes.DeriveSha(mixedReceipts(), trie.NewStackTrie(nil)).Hex())
}

// receiptAllOpFields is gethReceipt with every OP Stack extension field set. A real
// receipt never has all of them at once, but the wire format does not depend on that.
func receiptAllOpFields() *optypes.Receipt {
	return &optypes.Receipt{
		Receipt:               *gethReceipt(),
		DepositNonce:          ptr.New[uint64](12),
		DepositReceiptVersion: ptr.New[uint64](1),
		L1GasPrice:            big.NewInt(42_000_000_000),
		L1BlobBaseFee:         big.NewInt(123_456),
		L1GasUsed:             big.NewInt(2048),
		L1Fee:                 big.NewInt(77_000),
		FeeScalar:             big.NewFloat(0.5),
		L1BaseFeeScalar:       ptr.New[uint64](5227),
		L1BlobBaseFeeScalar:   ptr.New[uint64](1014213),
		OperatorFeeScalar:     ptr.New[uint64](7),
		OperatorFeeConstant:   ptr.New[uint64](9000),
		DAFootprintGasScalar:  ptr.New[uint64](160),
	}
}

// receiptAllOpFieldsJSON is the eth_getTransactionReceipt encoding of receiptAllOpFields.
var receiptAllOpFieldsJSON = `{"type":"0x2","root":"0x","status":"0x1","cumulativeGasUsed":"0x186a0","logsBloom":"0x` + zeroBytesHex(256) + `",` +
	`"logs":[{"address":"0x1111111111111111111111111111111111111111","topics":["0x0000000000000000000000000000000000000000000000000000000000000001"],"data":"0x42","blockNumber":"0x0","transactionHash":"0x0000000000000000000000000000000000000000000000000000000000000000","transactionIndex":"0x0","blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","blockTimestamp":"0x0","logIndex":"0x0","removed":false}],` +
	`"transactionHash":"0x000000000000000000000000000000000000000000000000000000000000aaaa","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x5208","effectiveGasPrice":"0x3b9aca00",` +
	`"depositNonce":"0xc","depositReceiptVersion":"0x1","blockHash":"0x000000000000000000000000000000000000000000000000000000000000bbbb","blockNumber":"0x4d2","transactionIndex":"0x3",` +
	`"l1GasPrice":"0x9c7652400","l1BlobBaseFee":"0x1e240","l1GasUsed":"0x800","l1Fee":"0x12cc8","l1FeeScalar":"0.5","l1BaseFeeScalar":"0x146b","l1BlobBaseFeeScalar":"0xf79c5",` +
	`"operatorFeeScalar":"0x7","operatorFeeConstant":"0x2328","daFootprintGasScalar":"0xa0"}`

// TestReceiptJSONGoldenVector pins the JSON encoding of a receipt with every OP Stack
// extension field, and that it decodes to the same fields.
func TestReceiptJSONGoldenVector(t *testing.T) {
	want := receiptAllOpFields()
	data, err := json.Marshal(want)
	require.NoError(t, err)
	require.JSONEq(t, receiptAllOpFieldsJSON, string(data))

	var got optypes.Receipt
	require.NoError(t, json.Unmarshal([]byte(receiptAllOpFieldsJSON), &got))
	require.Equal(t, want.DepositNonce, got.DepositNonce)
	require.Equal(t, want.DepositReceiptVersion, got.DepositReceiptVersion)
	require.Equal(t, want.L1GasPrice, got.L1GasPrice)
	require.Equal(t, want.L1BlobBaseFee, got.L1BlobBaseFee)
	require.Equal(t, want.L1GasUsed, got.L1GasUsed)
	require.Equal(t, want.L1Fee, got.L1Fee)
	require.NotNil(t, got.FeeScalar)
	require.Zero(t, want.FeeScalar.Cmp(got.FeeScalar))
	require.Equal(t, want.L1BaseFeeScalar, got.L1BaseFeeScalar)
	require.Equal(t, want.L1BlobBaseFeeScalar, got.L1BlobBaseFeeScalar)
	require.Equal(t, want.OperatorFeeScalar, got.OperatorFeeScalar)
	require.Equal(t, want.OperatorFeeConstant, got.OperatorFeeConstant)
	require.Equal(t, want.DAFootprintGasScalar, got.DAFootprintGasScalar)
	require.Equal(t, want.TxHash, got.TxHash)
	require.Equal(t, want.GasUsed, got.GasUsed)
}
