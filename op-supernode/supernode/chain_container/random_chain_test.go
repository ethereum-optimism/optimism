package chain_container

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
