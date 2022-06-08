package node_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

var (
	testPrevBlockID = eth.BlockID{
		Number: 5,
		Hash:   common.HexToHash("0x55"),
	}
	testBundleData = []byte{0xbb, 0xbb}
)

func createResponse(
	prevBlock, lastBlock eth.BlockID,
	bundle []byte,
) *node.BatchBundleResponse {

	return &node.BatchBundleResponse{
		PrevL2BlockHash: prevBlock.Hash,
		PrevL2BlockNum:  hexutil.Uint64(prevBlock.Number),
		LastL2BlockHash: lastBlock.Hash,
		LastL2BlockNum:  hexutil.Uint64(lastBlock.Number),
		Bundle:          hexutil.Bytes(bundle),
	}
}

// TestNewBundleBuilder asserts the state of a BundleBuilder after
// initialization.
func TestNewBundleBuilder(t *testing.T) {
	builder := node.NewBundleBuilder(testPrevBlockID)

	require.False(t, builder.HasCandidate())
	require.Equal(t, builder.Batches(), []*derive.BatchData{})
	expResponse := createResponse(testPrevBlockID, testPrevBlockID, nil)
	require.Equal(t, expResponse, builder.Response(nil))
}

// TestBundleBuilderAddCandidate asserts the state of a BundleBuilder after
// progressively adding various BundleCandidates.
func TestBundleBuilderAddCandidate(t *testing.T) {
	builder := node.NewBundleBuilder(testPrevBlockID)

	// Add candidate.
	blockID7 := eth.BlockID{
		Number: 7,
		Hash:   common.HexToHash("0x77"),
	}
	batchData7 := &derive.BatchData{
		BatchV1: derive.BatchV1{
			Epoch:     3,
			Timestamp: 42,
			Transactions: []hexutil.Bytes{
				hexutil.Bytes([]byte{0x42, 0x07}),
			},
		},
	}
	builder.AddCandidate(node.BundleCandidate{
		ID:    blockID7,
		Batch: batchData7,
	})

	// HasCandidate should register that we have data to submit to L1,
	// last block ID fields should also be updated.
	require.True(t, builder.HasCandidate())
	require.Equal(t, builder.Batches(), []*derive.BatchData{batchData7})
	expResponse := createResponse(testPrevBlockID, blockID7, testBundleData)
	require.Equal(t, expResponse, builder.Response(testBundleData))

	// Add another block.
	blockID8 := eth.BlockID{
		Number: 8,
		Hash:   common.HexToHash("0x88"),
	}
	batchData8 := &derive.BatchData{
		BatchV1: derive.BatchV1{
			Epoch:     5,
			Timestamp: 44,
			Transactions: []hexutil.Bytes{
				hexutil.Bytes([]byte{0x13, 0x37}),
			},
		},
	}
	builder.AddCandidate(node.BundleCandidate{
		ID:    blockID8,
		Batch: batchData8,
	})

	// Last block ID fields should be updated.
	require.True(t, builder.HasCandidate())
	require.Equal(t, builder.Batches(), []*derive.BatchData{batchData7, batchData8})
	expResponse = createResponse(testPrevBlockID, blockID8, testBundleData)
	require.Equal(t, expResponse, builder.Response(testBundleData))
}
