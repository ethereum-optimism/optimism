package sources

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestL1ClientOPStackBlock reads an L1 block that is an OP Stack block, as the L1 of a chain whose
// L1 is an OP Stack chain sees it, from its JSON-RPC encoding: its transactions in their canonical
// encoding and block order, and its receipts, which hold deposit receipts.
func TestL1ClientOPStackBlock(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	chainID := big.NewInt(901)
	to := common.Address{0x42}
	l1Info := &optypes.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       common.Address{0x11},
		To:         &to,
		Value:      big.NewInt(0),
		Gas:        1_000_000,
		Data:       []byte{0xde, 0xad},
	}
	user := types.MustSignNewTx(key, types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
		ChainID: chainID, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(5), Gas: 21_000, To: &to, Value: big.NewInt(1),
	})
	postExec := &optypes.PostExecTx{Data: []byte{0xc2, 0x80, 0x80}}
	txs := []testutils.RPCTx{l1Info, user, postExec}
	receipts := optypes.Receipts{
		{Receipt: types.Receipt{Type: optypes.DepositTxType, Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 50_000, Logs: []*types.Log{}},
			DepositNonce: ptr.New(uint64(7)), DepositReceiptVersion: ptr.New(uint64(1))},
		{Receipt: types.Receipt{Type: types.DynamicFeeTxType, Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 71_000,
			Logs: []*types.Log{{Address: to, Topics: []common.Hash{{0x99}}, Data: []byte{0x01}}}}},
		{Receipt: types.Receipt{Type: optypes.PostExecTxType, Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: 71_000, Logs: []*types.Log{}}},
	}
	block := testutils.NewRPCBlock(t, 100, txs, receipts)

	var want []hexutil.Bytes
	for _, tx := range txs {
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)
		want = append(want, raw)
	}

	s, err := NewL1Client(testutils.NewBlockRPC(block), testlog.Logger(t, log.LevelInfo), nil, L1ClientSimpleConfig(false, RPCKindStandard, 10))
	require.NoError(t, err)

	info, rawTxs, err := s.InfoAndRawTxsByHash(t.Context(), block.Ref.Hash)
	require.NoError(t, err)
	require.Equal(t, block.Ref.Hash, info.Hash())
	require.Equal(t, want, rawTxs)

	_, gotReceipts, err := s.FetchReceipts(t.Context(), block.Ref.Hash)
	require.NoError(t, err)
	require.Len(t, gotReceipts, len(txs))
	require.Equal(t, uint8(optypes.DepositTxType), gotReceipts[0].Type)
	require.Equal(t, uint64(7), *gotReceipts[0].DepositNonce)
	require.Equal(t, user.Hash(), gotReceipts[1].TxHash)
	require.Len(t, gotReceipts[1].Logs, 1)
}
