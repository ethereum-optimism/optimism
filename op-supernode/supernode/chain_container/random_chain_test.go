package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func newMessage(chainID, blockNum uint64, logIdx uint32, ts uint64) *supervisortypes.Message {
	return &supervisortypes.Message{
		Identifier: supervisortypes.Identifier{
			Origin:      params.InteropCrossL2InboxAddress,
			ChainID:     eth.ChainIDFromUInt64(chainID),
			BlockNumber: blockNum,
			LogIndex:    logIdx,
			Timestamp:   ts,
		},
		PayloadHash: common.BytesToHash([]byte{byte(blockNum), byte(logIdx), byte(ts)}),
	}
}

func TestL2BlockReceiptsDecode(t *testing.T) {
	m := newMessage(10, 100, 0, 5000)
	blk := &L2Block{ExecMsgs: map[uint32]*supervisortypes.Message{0: m}}

	rcpts := blk.Receipts()
	require.Len(t, rcpts, 1)
	require.Len(t, rcpts[0].Logs, 1)

	decoded, err := processors.DecodeExecutingMessageLog(rcpts[0].Logs[0])
	require.NoError(t, err)
	require.Equal(t, m.Identifier.ChainID, decoded.ChainID)
	require.Equal(t, m.Identifier.BlockNumber, decoded.BlockNum)
	require.Equal(t, m.Checksum(), decoded.Checksum)
}

func TestL2BlockOutput(t *testing.T) {
	wRoot := common.HexToHash("0x1234")
	blk := &L2Block{
		Payload: &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				StateRoot:       eth.Bytes32(common.HexToHash("0xdead")),
				WithdrawalsRoot: &wRoot,
				BlockHash:       common.HexToHash("0xabcd"),
			},
		},
	}

	out := blk.Output()
	require.Equal(t, eth.Bytes32(common.HexToHash("0xdead")), out.StateRoot)
	require.Equal(t, eth.Bytes32(wRoot), out.MessagePasserStorageRoot)
	require.Equal(t, common.HexToHash("0xabcd"), out.BlockHash)
}

func TestRandomChainSafeDB(t *testing.T) {
	rc := &RandomChain{
		safeDB: []SafeHeadEntry{
			{L1: eth.BlockID{Number: 100}, L2: eth.BlockID{Number: 5}},
			{L1: eth.BlockID{Number: 110}, L2: eth.BlockID{Number: 8}},
		},
	}

	_, err := rc.L1AtSafeHead(context.Background(), eth.BlockID{Number: 8})
	require.ErrorIs(t, err, virtual_node.ErrVirtualNodeNotRunning)

	require.NoError(t, rc.Start(context.Background()))

	l1, err := rc.L1AtSafeHead(context.Background(), eth.BlockID{Number: 8})
	require.NoError(t, err)
	require.Equal(t, uint64(110), l1.Number)

	_, err = rc.L1AtSafeHead(context.Background(), eth.BlockID{Number: 9})
	require.ErrorIs(t, err, safedb.ErrL1AtSafeHeadNotFound)

	gL1, gL2, err := rc.FirstSafeHeadEntry(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(100), gL1.Number)
	require.Equal(t, uint64(5), gL2.Number)

	aL1, aL2, err := rc.SafeHeadAtL1(context.Background(), 105)
	require.NoError(t, err)
	require.Equal(t, uint64(100), aL1.Number)
	require.Equal(t, uint64(5), aL2.Number)
}

func TestRandomChainL2Provider(t *testing.T) {
	wRoot := common.HexToHash("0x1234")
	blkHash := common.HexToHash("0xfeed")
	rc := &RandomChain{
		safe: 0,
		l2: []L2Block{{
			Ref: eth.L2BlockRef{Hash: blkHash, Number: 0, Time: 1234},
			Payload: &eth.ExecutionPayloadEnvelope{
				ExecutionPayload: &eth.ExecutionPayload{
					StateRoot:       eth.Bytes32(common.HexToHash("0xdead")),
					WithdrawalsRoot: &wRoot,
					BlockHash:       blkHash,
				},
			},
			ExecMsgs: map[uint32]*supervisortypes.Message{0: newMessage(10, 0, 0, 1234)},
		}},
	}

	ref, err := rc.L2BlockRefByNumber(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, blkHash, ref.Hash)

	labelRef, err := rc.L2BlockRefByLabel(context.Background(), eth.Safe)
	require.NoError(t, err)
	require.Equal(t, blkHash, labelRef.Hash)

	_, err = rc.L2BlockRefByNumber(context.Background(), 1)
	require.ErrorIs(t, err, ethereum.NotFound)

	out, err := rc.OutputV0AtBlock(context.Background(), blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, out.BlockHash)

	info, rcpts, err := rc.FetchReceipts(context.Background(), blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, info.Hash())
	require.Len(t, rcpts[0].Logs, 1)

	_, _, err = rc.FetchReceipts(context.Background(), common.HexToHash("0x0"))
	require.ErrorIs(t, err, ethereum.NotFound)
}

func TestRandomChainManagerAccessors(t *testing.T) {
	a := eth.ChainIDFromUInt64(1)
	b := eth.ChainIDFromUInt64(2)
	m := &RandomChainManager{
		chains: map[eth.ChainID]*RandomChain{a: {chainID: a}, b: {chainID: b}},
		order:  []eth.ChainID{a, b},
	}
	m.l1Source = &RandomL1Source{parent: m}

	require.Equal(t, a, m.Chain(a).chainID)
	require.Nil(t, m.Chain(eth.ChainIDFromUInt64(99)))

	chains := m.Chains()
	require.Len(t, chains, 2)
	require.Equal(t, a, chains[0].chainID)
	require.Equal(t, b, chains[1].chainID)

	require.Same(t, m, m.L1Source().parent)
}
