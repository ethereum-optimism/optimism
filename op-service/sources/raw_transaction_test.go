package sources

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
)

// l2BlockTxs builds the transaction list shape of a Lagoon-era L2 block via
// op-geth typed transactions: the L1-info deposit first, user transactions of
// several standard types, and a trailing post-exec transaction.
func l2BlockTxs(t *testing.T) types.Transactions {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	chainID := big.NewInt(10)
	signer := types.LatestSignerForChainID(chainID)
	to := common.HexToAddress("0x4242424242424242424242424242424242424242")

	deposit := types.NewTx(&types.DepositTx{
		SourceHash: common.HexToHash("0x01"),
		From:       common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Mint:       big.NewInt(0),
		Value:      big.NewInt(0),
		Gas:        1_000_000,
		Data:       []byte{0xde, 0xad},
	})
	legacy := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce: 0, GasPrice: big.NewInt(2), Gas: 21000, To: &to, Value: big.NewInt(1),
	})
	dynFee := types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
		ChainID: chainID, Nonce: 1, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(5),
		Gas: 21000, To: &to, Value: big.NewInt(2),
	})
	blob := types.MustSignNewTx(key, signer, &types.BlobTx{
		ChainID: uint256.NewInt(10), Nonce: 2, GasTipCap: uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(5), Gas: 21000, To: to, BlobFeeCap: uint256.NewInt(1),
		BlobHashes: []common.Hash{{0x01}},
	})
	setCode := types.MustSignNewTx(key, signer, &types.SetCodeTx{
		ChainID: uint256.NewInt(10), Nonce: 3, GasTipCap: uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(5), Gas: 50000, To: to,
		AuthList: []types.SetCodeAuthorization{{ChainID: *uint256.NewInt(10), Address: to, Nonce: 7, V: 1, R: *uint256.NewInt(2), S: *uint256.NewInt(3)}},
	})
	postExec := types.NewTx(&types.PostExecTx{Data: []byte{0xc2, 0x80, 0x80}})

	return types.Transactions{deposit, legacy, dynFee, blob, setCode, postExec}
}

// l2RPCBlockJSON serves an op-geth-marshaled RPC block JSON object for the
// given transactions, with a consistent transactions root and block hash.
func l2RPCBlockJSON(t *testing.T, txs types.Transactions) []byte {
	t.Helper()
	hdr := RPCHeader{
		UncleHash: types.EmptyUncleHash,
		Number:    hexutil.Uint64(123),
		Time:      hexutil.Uint64(1_700_000_000),
		GasUsed:   hexutil.Uint64(21_000),
		BaseFee:   (*hexutil.Big)(big.NewInt(7)),
		TxHash:    types.DeriveSha(txs, trie.NewStackTrie(nil)),
	}
	hdr.Hash = hdr.computeBlockHash()

	obj := map[string]json.RawMessage{}
	hdrJSON, err := json.Marshal(&hdr)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(hdrJSON, &obj))

	txsJSON := make([]json.RawMessage, len(txs))
	for i, tx := range txs {
		txsJSON[i], err = tx.MarshalJSON() // op-geth's marshaling: the RPC wire format
		require.NoError(t, err)
	}
	allTxs, err := json.Marshal(txsJSON)
	require.NoError(t, err)
	obj["transactions"] = allTxs

	blockJSON, err := json.Marshal(obj)
	require.NoError(t, err)
	return blockJSON
}

// TestRPCBlockDecodesL2Block asserts that an L2 block's op-geth-marshaled JSON
// — deposit, standard user txs, and post-exec — decodes into the canonical
// per-tx binary encodings, verifies against the transactions root, and passes
// through to the execution payload byte-identically.
func TestRPCBlockDecodesL2Block(t *testing.T) {
	txs := l2BlockTxs(t)
	blockJSON := l2RPCBlockJSON(t, txs)

	var block RPCBlock
	require.NoError(t, json.Unmarshal(blockJSON, &block))
	require.Len(t, block.Transactions, len(txs))
	for i, tx := range txs {
		theirs, err := tx.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, theirs, []byte(block.Transactions[i]), "tx %d binary encoding", i)
		require.Equal(t, tx.Hash(), block.Transactions[i].Hash(), "tx %d hash", i)
	}

	require.NoError(t, block.Verify(), "tx root and block hash must verify")

	envelope, err := block.ExecutionPayloadEnvelope(false)
	require.NoError(t, err)
	require.Len(t, envelope.ExecutionPayload.Transactions, len(txs))
	for i := range txs {
		require.Equal(t, []byte(block.Transactions[i]), []byte(envelope.ExecutionPayload.Transactions[i]), "payload tx %d passthrough", i)
	}
}

// TestRPCBlockJSONRoundTrip asserts marshal(unmarshal(J)) is a fixed point on
// the transaction objects.
func TestRPCBlockJSONRoundTrip(t *testing.T) {
	blockJSON := l2RPCBlockJSON(t, l2BlockTxs(t))

	var block RPCBlock
	require.NoError(t, json.Unmarshal(blockJSON, &block))
	remarshaled, err := json.Marshal(&block)
	require.NoError(t, err)
	var block2 RPCBlock
	require.NoError(t, json.Unmarshal(remarshaled, &block2))
	require.Equal(t, block.Transactions, block2.Transactions)
	require.Equal(t, block.RPCHeader, block2.RPCHeader)
}

func TestRawTransactionsPartition(t *testing.T) {
	txs := l2BlockTxs(t)
	blockJSON := l2RPCBlockJSON(t, txs)
	var block RPCBlock
	require.NoError(t, json.Unmarshal(blockJSON, &block))

	userTxs, err := block.Transactions.UserTxs()
	require.NoError(t, err)
	require.Len(t, userTxs, 4, "deposit and post-exec txs are excluded")
	for i, tx := range userTxs {
		require.Equal(t, txs[i+1].Hash(), tx.Hash(), "user tx %d", i)
	}

	deposits, err := block.Transactions.Deposits()
	require.NoError(t, err)
	require.Len(t, deposits, 1)
	require.Equal(t, common.HexToHash("0x01"), deposits[0].SourceHash)

	first, err := block.Transactions.FirstDeposit()
	require.NoError(t, err)
	require.Equal(t, deposits[0], first)

	// A list not starting with a deposit has no first deposit.
	_, err = RawTransactions{block.Transactions[1]}.FirstDeposit()
	require.ErrorIs(t, err, ErrMissingFirstDeposit)
	_, err = RawTransactions{}.FirstDeposit()
	require.ErrorIs(t, err, ErrMissingFirstDeposit)

	// Zero-length entries error rather than classifying as user txs.
	_, err = RawTransactions{{}}.UserTxs()
	require.ErrorContains(t, err, "empty")
	_, err = RawTransactions{{}}.Deposits()
	require.ErrorContains(t, err, "empty")
}

// TestRawTransactionJSONEmptyPostExecRejected pins the canonical-encoding
// contract on decode: a post-exec tx with empty input must fail JSON decode
// rather than produce the one-byte value 0x7d, which the binary decoders
// (UnmarshalPostExecTx, op-geth) reject.
func TestRawTransactionJSONEmptyPostExecRejected(t *testing.T) {
	var tx RawTransaction
	err := json.Unmarshal([]byte(`{"type":"0x7d","input":"0x"}`), &tx)
	require.ErrorContains(t, err, "invalid post-exec tx")
}

func TestRawTransactionsGeth(t *testing.T) {
	txs := l2BlockTxs(t)
	blockJSON := l2RPCBlockJSON(t, txs)
	var block RPCBlock
	require.NoError(t, json.Unmarshal(blockJSON, &block))

	decoded, err := block.Transactions.Geth()
	require.NoError(t, err)
	require.Len(t, decoded, len(txs))
	for i, tx := range decoded {
		require.Equal(t, txs[i].Hash(), tx.Hash())
	}

	_, err = RawTransactions{{0xff}}.Geth()
	require.Error(t, err)
}
