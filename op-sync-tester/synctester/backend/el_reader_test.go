package backend

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// blockRPC answers eth_getBlockByHash and eth_getBlockByNumber with its JSON block.
type blockRPC struct {
	block json.RawMessage
}

var _ client.RPC = blockRPC{}

func (b blockRPC) CallContext(_ context.Context, result any, method string, _ ...any) error {
	switch method {
	case "eth_getBlockByHash", "eth_getBlockByNumber":
		return json.Unmarshal(b.block, result)
	default:
		panic("unexpected method " + method)
	}
}

func (blockRPC) BatchCallContext(context.Context, []rpc.BatchElem) error { panic("unused") }

func (blockRPC) Subscribe(context.Context, string, any, ...any) (ethereum.Subscription, error) {
	panic("unused")
}

func (blockRPC) Close() {}

func newTestELReader(t *testing.T, block json.RawMessage) *ELReader {
	r, err := NewELReader(testlog.Logger(t, log.LevelInfo), blockRPC{block: block})
	require.NoError(t, err)
	return r
}

// TestELReaderPayloadOfLagoonBlock reads op-reth's Lagoon block, which holds deposits and a
// trailing post-exec transaction: every transaction comes back as the bytes op-reth hashed.
func TestELReaderPayloadOfLagoonBlock(t *testing.T) {
	data, err := os.ReadFile("../../../op-service/sources/testdata/l2/l2_901_8_block.json")
	require.NoError(t, err)
	var block struct {
		Hash         common.Hash `json:"hash"`
		Transactions []struct {
			Hash common.Hash `json:"hash"`
		} `json:"transactions"`
	}
	require.NoError(t, json.Unmarshal(data, &block))
	r := newTestELReader(t, data)

	byHash, err := r.PayloadByHash(context.Background(), block.Hash)
	require.NoError(t, err)
	byNumber, err := r.PayloadByNumber(context.Background(), 8)
	require.NoError(t, err)
	for name, env := range map[string]*eth.ExecutionPayloadEnvelope{"by hash": byHash, "by number": byNumber} {
		txs := env.ExecutionPayload.Transactions
		require.Len(t, txs, len(block.Transactions), name)
		for i, tx := range txs {
			require.Equal(t, block.Transactions[i].Hash, crypto.Keccak256Hash(tx), "%s: tx %d", name, i)
		}
		require.Equal(t, byte(optypes.DepositTxType), txs[0][0], name)
		require.Equal(t, byte(optypes.PostExecTxType), txs[len(txs)-1][0], name)
		require.NotNil(t, env.ExecutionPayload.WithdrawalsRoot, "%s: an Isthmus payload carries its withdrawals root", name)
	}
}

// TestELReaderPayloadOfPreIsthmusBlock reads a Canyon block, whose header carries the empty
// withdrawals root: its payload has no withdrawals root.
func TestELReaderPayloadOfPreIsthmusBlock(t *testing.T) {
	to := common.Address{0x03}
	deposit := &optypes.DepositTx{SourceHash: common.Hash{0x01}, From: common.Address{0x02}, To: &to,
		Mint: big.NewInt(0), Value: big.NewInt(0), Gas: 1_000_000, Data: []byte{0x01}}
	raw, err := deposit.MarshalBinary()
	require.NoError(t, err)
	header := &types.Header{
		UncleHash:       types.EmptyUncleHash,
		TxHash:          types.DeriveSha(sources.RawTransactions{raw}, trie.NewStackTrie(nil)),
		ReceiptHash:     types.EmptyReceiptsHash,
		Difficulty:      big.NewInt(0),
		Number:          big.NewInt(5),
		GasLimit:        30_000_000,
		Time:            1_000,
		BaseFee:         big.NewInt(1),
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
	}
	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)
	depositJSON, err := json.Marshal(deposit)
	require.NoError(t, err)
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(headerJSON, &obj))
	obj["transactions"] = json.RawMessage("[" + string(depositJSON) + "]")
	obj["withdrawals"] = json.RawMessage("[]")
	block, err := json.Marshal(obj)
	require.NoError(t, err)

	env, err := newTestELReader(t, block).PayloadByHash(context.Background(), header.Hash())
	require.NoError(t, err)
	require.Nil(t, env.ExecutionPayload.WithdrawalsRoot)
	require.Equal(t, []eth.Data{raw}, env.ExecutionPayload.Transactions)
}
