package engine

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// blockRPC answers every call with its JSON block.
type blockRPC struct {
	block json.RawMessage
}

var _ client.RPC = blockRPC{}

func (b blockRPC) CallContext(_ context.Context, result any, _ string, _ ...any) error {
	return json.Unmarshal(b.block, result)
}

func (blockRPC) BatchCallContext(context.Context, []rpc.BatchElem) error { panic("unused") }

func (blockRPC) Subscribe(context.Context, string, any, ...any) (ethereum.Subscription, error) {
	panic("unused")
}

func (blockRPC) Close() {}

func readBlock(t *testing.T, path string) blockRPC {
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return blockRPC{block: data}
}

// preIsthmusBlock returns a Canyon L2 block holding only a deposit: its header carries the
// empty withdrawals root.
func preIsthmusBlock(t *testing.T) blockRPC {
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
	return blockRPC{block: block}
}

// TestCopiedPayloadWithdrawalsRoot checks that a copied payload keeps the source block's
// withdrawals root, while its withdrawals follow the destination's Engine API version.
func TestCopiedPayloadWithdrawalsRoot(t *testing.T) {
	ctx := context.Background()

	// An Isthmus L2 block carries the L2ToL1MessagePasser storage root; with its root kept and
	// its empty withdrawals, the copy still hashes to the source block.
	isthmus, err := getPayload(ctx, readBlock(t, "../../op-service/sources/testdata/l2/l2_901_8_block.json"), "latest")
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x8ed4baae3a927be3dea54996b4d5899f8c01e7594bf50b17dc1e741388ce3d12"),
		*isthmus.ExecutionPayload.WithdrawalsRoot)
	setWithdrawalsForEngine(isthmus.ExecutionPayload, StaticVersionProvider(3))
	require.NotNil(t, isthmus.ExecutionPayload.Withdrawals)
	require.Empty(t, *isthmus.ExecutionPayload.Withdrawals)
	_, ok := isthmus.CheckBlockHash()
	require.True(t, ok, "copied Isthmus payload must hash to its source block")

	// A pre-Isthmus block's empty withdrawals root is not a payload withdrawals root.
	canyon, err := getPayload(ctx, preIsthmusBlock(t), "latest")
	require.NoError(t, err)
	require.Nil(t, canyon.ExecutionPayload.WithdrawalsRoot)
	setWithdrawalsForEngine(canyon.ExecutionPayload, StaticVersionProvider(2))
	require.NotNil(t, canyon.ExecutionPayload.Withdrawals)
	require.Nil(t, canyon.ExecutionPayload.WithdrawalsRoot)
	_, ok = canyon.CheckBlockHash()
	require.True(t, ok, "copied pre-Isthmus payload must hash to its source block")
	setWithdrawalsForEngine(canyon.ExecutionPayload, StaticVersionProvider(1))
	require.Nil(t, canyon.ExecutionPayload.Withdrawals)
}
