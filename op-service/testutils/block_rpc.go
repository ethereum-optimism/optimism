package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// RPCTx is a transaction of a block served by a BlockRPC: a go-ethereum
// *types.Transaction, or an op-core/types *DepositTx or *PostExecTx.
type RPCTx interface {
	json.Marshaler
	MarshalBinary() ([]byte, error)
}

// RPCBlock is a block in the JSON encodings of eth_getBlockByHash, with full
// transactions, and eth_getBlockReceipts.
type RPCBlock struct {
	Ref      eth.L1BlockRef
	Block    json.RawMessage
	Receipts json.RawMessage
}

// NewRPCBlock builds a post-merge block holding txs, with the given receipts,
// that passes the op-service/sources block, transactions and receipts
// verification. Receipts pair with txs by index, and only their consensus
// fields are read: type, status, cumulative gas used, the logs' address,
// topics and data, and a deposit receipt's nonce and version. A block whose
// first transaction is a deposit is shaped like an OP Stack block, whose
// withdrawals root does not commit to its empty withdrawals list.
func NewRPCBlock(t require.TestingT, number uint64, txs []RPCTx, receipts optypes.Receipts) RPCBlock {
	require.Len(t, receipts, len(txs))
	rawTxs := make(rawTxList, len(txs))
	txsJSON := make([]json.RawMessage, len(txs))
	for i, tx := range txs {
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)
		rawTxs[i] = raw
		txsJSON[i], err = tx.MarshalJSON()
		require.NoError(t, err)
	}

	for _, r := range receipts {
		r.Bloom = types.CreateBloom(&r.Receipt)
	}

	withdrawalsRoot := types.EmptyWithdrawalsHash
	if len(rawTxs) > 0 && optypes.IsDepositTxBytes(rawTxs[0]) {
		withdrawalsRoot = common.Hash{0x01}
	}
	header := &types.Header{
		ParentHash:      common.Hash{0xaa},
		UncleHash:       types.EmptyUncleHash,
		Root:            common.Hash{0xbb},
		TxHash:          types.DeriveSha(rawTxs, trie.NewStackTrie(nil)),
		ReceiptHash:     types.DeriveSha(receipts, trie.NewStackTrie(nil)),
		Bloom:           types.MergeBloom(receipts.Geth()),
		Difficulty:      new(big.Int),
		Number:          new(big.Int).SetUint64(number),
		GasLimit:        30_000_000,
		Time:            1_700_000_000 + number*2,
		BaseFee:         big.NewInt(7),
		WithdrawalsHash: &withdrawalsRoot,
	}
	hash := header.Hash()

	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)
	var block map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(headerJSON, &block))
	block["transactions"], err = json.Marshal(txsJSON)
	require.NoError(t, err)
	block["withdrawals"] = json.RawMessage("[]")
	blockJSON, err := json.Marshal(block)
	require.NoError(t, err)

	var logIndex uint
	var prevCumulativeGas uint64
	for i, r := range receipts {
		r.TxHash = crypto.Keccak256Hash(rawTxs[i])
		r.BlockHash = hash
		r.BlockNumber = new(big.Int).SetUint64(number)
		r.TransactionIndex = uint(i)
		r.GasUsed = r.CumulativeGasUsed - prevCumulativeGas
		prevCumulativeGas = r.CumulativeGasUsed
		for _, l := range r.Logs {
			l.TxHash = r.TxHash
			l.TxIndex = uint(i)
			l.BlockHash = hash
			l.BlockNumber = number
			l.Index = logIndex
			logIndex++
		}
	}
	receiptsJSON, err := json.Marshal(receipts)
	require.NoError(t, err)

	return RPCBlock{
		Ref: eth.L1BlockRef{
			Hash:       hash,
			Number:     number,
			ParentHash: header.ParentHash,
			Time:       header.Time,
		},
		Block:    blockJSON,
		Receipts: receiptsJSON,
	}
}

// BlockRPC is an op-service/client.RPC that serves eth_getBlockByHash and
// eth_getBlockReceipts for a set of blocks, decoding their JSON encoding into
// the caller's result as a JSON-RPC client does.
type BlockRPC struct {
	blocks map[common.Hash]RPCBlock
}

func NewBlockRPC(blocks ...RPCBlock) *BlockRPC {
	m := make(map[common.Hash]RPCBlock, len(blocks))
	for _, b := range blocks {
		m[b.Ref.Hash] = b
	}
	return &BlockRPC{blocks: m}
}

func (r *BlockRPC) CallContext(_ context.Context, result any, method string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%s: missing block hash", method)
	}
	hash, ok := args[0].(common.Hash)
	if !ok {
		return fmt.Errorf("%s: unsupported block argument %v", method, args[0])
	}
	block, ok := r.blocks[hash]
	if !ok {
		return json.Unmarshal([]byte("null"), result)
	}
	switch method {
	case "eth_getBlockByHash":
		if len(args) != 2 || args[1] != true {
			return fmt.Errorf("%s: only full-transaction blocks are served", method)
		}
		return json.Unmarshal(block.Block, result)
	case "eth_getBlockReceipts":
		return json.Unmarshal(block.Receipts, result)
	default:
		return fmt.Errorf("unsupported method %s", method)
	}
}

func (r *BlockRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	for i := range b {
		b[i].Error = r.CallContext(ctx, b[i].Result, b[i].Method, b[i].Args...)
	}
	return nil
}

func (r *BlockRPC) Subscribe(context.Context, string, any, ...any) (ethereum.Subscription, error) {
	return nil, errors.New("subscriptions are not supported")
}

func (r *BlockRPC) Close() {}

// rawTxList is a list of canonical transaction encodings that derives the
// transactions root: a typed transaction's trie leaf is its encoding verbatim.
type rawTxList [][]byte

func (l rawTxList) Len() int { return len(l) }

func (l rawTxList) EncodeIndex(i int, w *bytes.Buffer) { w.Write(l[i]) }
