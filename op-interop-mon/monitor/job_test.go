package monitor

import (
	"testing"

	"encoding/binary"
	"math/big"

	optimiseth "github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestJob_UpdateStatus(t *testing.T) {
	j := &Job{}
	require.Equal(t, jobStatusUnknown, j.LatestStatus(), "expected initial status to be jobStatusUnknown")
	require.Equal(t, 0, len(j.status), "expected 0 statuses")
	j.UpdateStatus(jobStatusValid)
	require.Equal(t, jobStatusValid, j.LatestStatus(), "expected status to be jobStatusValid")
	j.UpdateStatus(jobStatusValid) // should not append duplicate
	require.Equal(t, 1, len(j.status), "expected 1 status")
	j.UpdateStatus(jobStatusInvalid)
	require.Equal(t, jobStatusInvalid, j.LatestStatus(), "expected status to be jobStatusInvalid")
	require.Equal(t, 2, len(j.status), "expected 2 statuses")
}

func TestJobFromExecutingMessageLog_Error(t *testing.T) {
	log := &types.Log{}
	_, err := JobFromExecutingMessageLog(log)
	require.Error(t, err, "expected error for empty log")
}

func TestJobFromLog(t *testing.T) {
	payloadHash := common.HexToHash("0xabc123")
	address := params.InteropCrossL2InboxAddress
	blockHash := common.HexToHash("0xdeadbeef")
	blockNumber := uint64(42)
	logIndex := uint32(7)
	chainID := big.NewInt(10)
	timestamp := uint64(123456)

	// Build valid data for the event
	var data [32 * 5]byte
	// address padding (12 bytes) + address (20 bytes)
	copy(data[12:32], address.Bytes())
	// block number (8 bytes, big endian)
	binary.BigEndian.PutUint64(data[32+24:32+32], blockNumber)
	// log index (4 bytes, big endian)
	binary.BigEndian.PutUint32(data[64+28:64+32], logIndex)
	// timestamp (8 bytes, big endian)
	binary.BigEndian.PutUint64(data[96+24:96+32], timestamp)
	// chainID (32 bytes, big endian)
	chainIDBytes := make([]byte, 32)
	chainID.FillBytes(chainIDBytes)
	copy(data[128:160], chainIDBytes)

	tests := []struct {
		name       string
		log        *types.Log
		expectsErr bool
		expectsJob *Job
	}{
		{
			name: "not an executing message (wrong address)",
			log: &types.Log{
				Address: common.HexToAddress("0x1234"),
			},
			expectsErr: true,
		},
		{
			name: "not an executing message (wrong topics)",
			log: &types.Log{
				Address: address,
				Topics:  []common.Hash{},
			},
			expectsErr: true,
		},
		{
			name: "valid executing message",
			log: &types.Log{
				Address:     address,
				BlockHash:   blockHash,
				BlockNumber: blockNumber,
				Topics: []common.Hash{
					supervisortypes.ExecutingMessageEventTopic,
					payloadHash,
				},
				Data: data[:],
			},
			expectsErr: false,
			expectsJob: &Job{
				executingAddress: address,
				executingChain:   optimiseth.ChainIDFromBig(chainID),
				executingBlock:   optimiseth.BlockID{Hash: blockHash, Number: blockNumber},
				executingPayload: payloadHash,
				initiating: &supervisortypes.Identifier{
					Origin:      address,
					BlockNumber: blockNumber,
					LogIndex:    logIndex,
					Timestamp:   timestamp,
					ChainID:     optimiseth.ChainIDFromBig(chainID),
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			job, err := JobFromExecutingMessageLog(tc.log)
			if tc.expectsErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tc.expectsJob)
				require.Equal(t, tc.expectsJob.executingAddress, job.executingAddress)
				require.Equal(t, tc.expectsJob.executingChain, job.executingChain)
				require.Equal(t, tc.expectsJob.executingBlock, job.executingBlock)
				require.Equal(t, tc.expectsJob.executingPayload, job.executingPayload)
				require.Equal(t, *tc.expectsJob.initiating, *job.initiating)
			}
		})
	}
}
