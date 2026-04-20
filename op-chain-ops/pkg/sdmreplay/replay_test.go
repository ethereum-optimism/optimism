package sdmreplay

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

type mockSource struct {
	head          uint64
	chainID       uint64
	clientVersion string
	blocks        map[uint64]*RPCBlock
	receipts      map[common.Hash]*RPCReceipt
	replays       map[uint64]*ReplaySdmBlock
}

func (m *mockSource) GetBlockNumber(context.Context) (uint64, error) {
	return m.head, nil
}

func (m *mockSource) ClientVersion(context.Context) (string, error) {
	return m.clientVersion, nil
}

func (m *mockSource) ChainID(context.Context) (uint64, error) {
	return m.chainID, nil
}

func (m *mockSource) GetBlockByNumber(_ context.Context, blockNum uint64) (*RPCBlock, error) {
	block, ok := m.blocks[blockNum]
	if !ok {
		return nil, errors.New("missing block")
	}
	return block, nil
}

func (m *mockSource) GetTransactionReceipt(_ context.Context, txHash common.Hash) (*RPCReceipt, error) {
	receipt, ok := m.receipts[txHash]
	if !ok {
		return nil, errors.New("missing receipt")
	}
	return receipt, nil
}

func (m *mockSource) ReplaySdmBlock(_ context.Context, blockNum uint64, _ bool, _ bool) (*ReplaySdmBlock, error) {
	replay, ok := m.replays[blockNum]
	if !ok {
		return nil, errors.New("missing replay")
	}
	return replay, nil
}

func TestResolveBlockNum(t *testing.T) {
	src := &mockSource{head: 123}
	ctx := context.Background()

	n, err := ResolveBlockNum(ctx, src, "latest")
	require.NoError(t, err)
	require.Equal(t, uint64(123), n)

	n, err = ResolveBlockNum(ctx, src, "latest-3")
	require.NoError(t, err)
	require.Equal(t, uint64(120), n)

	n, err = ResolveBlockNum(ctx, src, "0x10")
	require.NoError(t, err)
	require.Equal(t, uint64(16), n)
}

func TestReplayRangeCounterfactualRPC(t *testing.T) {
	ctx := context.Background()
	blockNum := uint64(10)
	depositHash := common.HexToHash("0x1")
	userAHash := common.HexToHash("0x3")
	userBHash := common.HexToHash("0x4")

	src := &mockSource{
		head:          15,
		chainID:       10,
		clientVersion: "op-reth/v1.7.0-sdm",
		blocks: map[uint64]*RPCBlock{
			blockNum: {
				Number:     hexutil.Uint64(blockNum),
				Hash:       common.HexToHash("0x10"),
				ParentHash: common.HexToHash("0x09"),
				GasUsed:    hexutil.Uint64(92000),
				Transactions: []RPCTransaction{
					{Hash: depositHash, Type: hexutil.Uint64(126), From: common.HexToAddress("0x100")},
					{Hash: userAHash, Type: hexutil.Uint64(2), From: common.HexToAddress("0x101"), To: addrPtr(common.HexToAddress("0x201"))},
					{Hash: userBHash, Type: hexutil.Uint64(2), From: common.HexToAddress("0x102"), To: addrPtr(common.HexToAddress("0x202"))},
				},
			},
		},
		receipts: map[common.Hash]*RPCReceipt{
			depositHash: {TransactionHash: depositHash, TransactionIndex: hexutil.Uint64(0), GasUsed: hexutil.Uint64(21000), Status: hexutil.Uint64(1)},
			userAHash:   {TransactionHash: userAHash, TransactionIndex: hexutil.Uint64(1), GasUsed: hexutil.Uint64(21000), Status: hexutil.Uint64(1)},
			userBHash:   {TransactionHash: userBHash, TransactionIndex: hexutil.Uint64(2), GasUsed: hexutil.Uint64(50000), Status: hexutil.Uint64(1)},
		},
		replays: map[uint64]*ReplaySdmBlock{
			blockNum: {
				BlockNum:     blockNum,
				BlockHash:    common.HexToHash("0x10"),
				ParentHash:   common.HexToHash("0x09"),
				SDMTxPresent: false,
				Txs: []ReplaySdmTx{
					{TxIndex: 0, ReplayTxIndex: 0, TxHash: depositHash, TxType: 126, IsDepositTx: true, GasUsed: 21000, OPGasRefundReplay: 0, EffectiveGas: 21000},
					{TxIndex: 1, ReplayTxIndex: 1, TxHash: userAHash, TxType: 2, IsDepositTx: false, GasUsed: 21000, OPGasRefundReplay: 0, EffectiveGas: 21000},
					{TxIndex: 2, ReplayTxIndex: 2, TxHash: userBHash, TxType: 2, IsDepositTx: false, GasUsed: 50000, OPGasRefundReplay: 2500, OPGasRefundReceipt: uint64Ptr(2500), EffectiveGas: 47500},
				},
				Summary: ReplaySdmSummary{
					BlockNum:               blockNum,
					BlockHash:              common.HexToHash("0x10"),
					TxCountTotal:           3,
					TxCountUser:            2,
					SDMTxPresent:           false,
					SDMPayloadEntryCount:   1,
					BlockGasUsed:           92000,
					ReplayRefundTotal:      2500,
					PayloadRefundTotal:     0,
					NodeReceiptRefundTotal: 2500,
					BlockEffectiveGas:      89500,
					MismatchCount:          0,
					ReplayMode:             string(ReplayModeCounterfactualEnabled),
				},
			},
		},
	}

	result, err := ReplayRange(ctx, src, Config{
		RPCURL:             "http://example.invalid",
		FromBlockSelector:  "10",
		ToBlockSelector:    "10",
		FromBlock:          blockNum,
		ToBlock:            blockNum,
		CompareRPCReceipts: true,
		Workers:            1,
		Format:             "jsonl",
	})
	require.NoError(t, err)
	require.Equal(t, string(ReplayModeCounterfactualEnabled), result.RunConfig.ReplayMode)
	require.Len(t, result.Blocks, 1)
	require.Equal(t, 0, result.Summary.MismatchCount)
	require.Equal(t, 2, result.Blocks[0].Block.TxCountUser)
	require.Equal(t, uint64(2500), result.Blocks[0].Block.ReplayRefundTotal)
	require.Equal(t, uint64(2500), result.Blocks[0].Block.NodeReceiptRefundTotal)
	require.Len(t, result.Blocks[0].Txs, 3)
	require.Equal(t, 2, result.Blocks[0].Txs[2].ReplayTxIndex)
	require.Equal(t, uint64(2500), result.Blocks[0].Txs[2].OPGasRefundReplay)
	require.Equal(t, "debug_replaySDMBlock", result.Blocks[0].Txs[2].AccountingSource)
}

func TestReplayRangeCounterfactualPayloadMismatch(t *testing.T) {
	ctx := context.Background()
	blockNum := uint64(11)
	userHash := common.HexToHash("0x13")

	src := &mockSource{
		head:          15,
		chainID:       10,
		clientVersion: "op-reth/v1.7.0-sdm",
		blocks: map[uint64]*RPCBlock{
			blockNum: {
				Number:     hexutil.Uint64(blockNum),
				Hash:       common.HexToHash("0x20"),
				ParentHash: common.HexToHash("0x19"),
				GasUsed:    hexutil.Uint64(50000),
				Transactions: []RPCTransaction{
					{Hash: userHash, Type: hexutil.Uint64(2), From: common.HexToAddress("0x101"), To: addrPtr(common.HexToAddress("0x201"))},
				},
			},
		},
		receipts: map[common.Hash]*RPCReceipt{
			userHash: {TransactionHash: userHash, TransactionIndex: hexutil.Uint64(0), GasUsed: hexutil.Uint64(50000), Status: hexutil.Uint64(1)},
		},
		replays: map[uint64]*ReplaySdmBlock{
			blockNum: {
				BlockNum:     blockNum,
				BlockHash:    common.HexToHash("0x20"),
				ParentHash:   common.HexToHash("0x19"),
				SDMTxPresent: false,
				Txs: []ReplaySdmTx{
					{TxIndex: 0, ReplayTxIndex: 0, TxHash: userHash, TxType: 2, GasUsed: 50000, OPGasRefundReplay: 2600, EffectiveGas: 47400, Mismatch: true},
				},
				Mismatches: []ReplaySdmMismatch{
					{
						Category: "payload_refund_mismatch",
						BlockNum: blockNum,
						TxIndex:  uint64Ptr(0),
						Actual:   uint64Ptr(2600),
						Message:  "payload refund mismatch for tx index 0",
					},
				},
				Summary: ReplaySdmSummary{
					BlockNum:               blockNum,
					BlockHash:              common.HexToHash("0x20"),
					TxCountTotal:           1,
					TxCountUser:            1,
					SDMTxPresent:           false,
					SDMPayloadEntryCount:   1,
					BlockGasUsed:           50000,
					ReplayRefundTotal:      2600,
					PayloadRefundTotal:     0,
					NodeReceiptRefundTotal: 0,
					BlockEffectiveGas:      47400,
					MismatchCount:          1,
					ReplayMode:             string(ReplayModeCounterfactualEnabled),
				},
			},
		},
	}

	result, err := ReplayRange(ctx, src, Config{
		RPCURL:            "http://example.invalid",
		FromBlockSelector: "11",
		ToBlockSelector:   "11",
		FromBlock:         blockNum,
		ToBlock:           blockNum,
		ComparePayload:    true,
		Workers:           1,
		Format:            "jsonl",
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Summary.MismatchCount)
	require.Len(t, result.Blocks[0].Mismatches, 1)
	require.Equal(t, "payload_refund_mismatch", result.Blocks[0].Mismatches[0].Category)
	require.Equal(t, userHash.Hex(), result.Blocks[0].Mismatches[0].TxHash)
}

func addrPtr(addr common.Address) *common.Address {
	return &addr
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}
