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

	require.False(t, builder.HasNonEmptyCandidate())
	require.Equal(t, builder.Batches(), []*derive.BatchData{})
	expResponse := createResponse(testPrevBlockID, testPrevBlockID, nil)
	require.Equal(t, expResponse, builder.Response(nil))
}

// TestBundleBuilderAddCandidate asserts the state of a BundleBuilder after
// progressively adding various BundleCandidates.
func TestBundleBuilderAddCandidate(t *testing.T) {
	builder := node.NewBundleBuilder(testPrevBlockID)

	// Add an empty candidate.
	blockID6 := eth.BlockID{
		Number: 6,
		Hash:   common.HexToHash("0x66"),
	}
	builder.AddCandidate(node.BundleCandidate{
		ID:    blockID6,
		Batch: nil,
	})

	// Should behave the same as completely empty builder except for updated
	// last block ID fields.
	require.False(t, builder.HasNonEmptyCandidate())
	require.Equal(t, builder.Batches(), []*derive.BatchData{})
	expResponse := createResponse(testPrevBlockID, blockID6, nil)
	require.Equal(t, expResponse, builder.Response(nil))

	// Add non-empty candidate.
	blockID7 := eth.BlockID{
		Number: 7,
		Hash:   common.HexToHash("0x77"),
	}
	batchData7 := &derive.BatchData{
		BatchV1: derive.BatchV1{
			Epoch:     7,
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

	// HasNonEmptyCandidate should register that we have data to submit to L1,
	// last block ID fields should also be updated.
	require.True(t, builder.HasNonEmptyCandidate())
	require.Equal(t, builder.Batches(), []*derive.BatchData{batchData7})
	expResponse = createResponse(testPrevBlockID, blockID7, testBundleData)
	require.Equal(t, expResponse, builder.Response(testBundleData))

	// Add another empty block.
	blockID8 := eth.BlockID{
		Number: 8,
		Hash:   common.HexToHash("0x88"),
	}
	builder.AddCandidate(node.BundleCandidate{
		ID:    blockID8,
		Batch: nil,
	})

	// Last block ID fields should be updated.
	require.True(t, builder.HasNonEmptyCandidate())
	require.Equal(t, builder.Batches(), []*derive.BatchData{batchData7})
	expResponse = createResponse(testPrevBlockID, blockID8, testBundleData)
	require.Equal(t, expResponse, builder.Response(testBundleData))
}

var pruneLastNonEmptyTests = []pruneLastNonEmptyTestCase{
	{
		name:        "no candidates",
		candidates:  nil,
		expResponse: createResponse(testPrevBlockID, testPrevBlockID, nil),
	},
	{
		name: "only empty blocks",
		candidates: []node.BundleCandidate{
			{
				ID: eth.BlockID{
					Number: 6,
					Hash:   common.HexToHash("0x66"),
				},
				Batch: nil,
			},
			{
				ID: eth.BlockID{
					Number: 7,
					Hash:   common.HexToHash("0x77"),
				},
				Batch: nil,
			},
		},
		expResponse: createResponse(
			testPrevBlockID,
			eth.BlockID{
				Number: 7,
				Hash:   common.HexToHash("0x77"),
			}, nil,
		),
	},
	{
		name: "last block is non empty",
		candidates: []node.BundleCandidate{
			{
				ID: eth.BlockID{
					Number: 6,
					Hash:   common.HexToHash("0x66"),
				},
				Batch: nil,
			},
			{
				ID: eth.BlockID{
					Number: 7,
					Hash:   common.HexToHash("0x77"),
				},
				Batch: &derive.BatchData{},
			},
		},
		expResponse: createResponse(
			testPrevBlockID,
			eth.BlockID{
				Number: 6,
				Hash:   common.HexToHash("0x66"),
			}, nil,
		),
	},
	{
		name: "non empty block followed by empty block",
		candidates: []node.BundleCandidate{
			{
				ID: eth.BlockID{
					Number: 6,
					Hash:   common.HexToHash("0x66"),
				},
				Batch: nil,
			},
			{
				ID: eth.BlockID{
					Number: 7,
					Hash:   common.HexToHash("0x77"),
				},
				Batch: &derive.BatchData{},
			},
			{
				ID: eth.BlockID{
					Number: 8,
					Hash:   common.HexToHash("0x88"),
				},
				Batch: nil,
			},
		},
		expResponse: createResponse(
			testPrevBlockID,
			eth.BlockID{
				Number: 6,
				Hash:   common.HexToHash("0x66"),
			}, nil,
		),
	},
}

// TestBundleBuilderPruneLastNonEmpty asserts that pruning the BundleBuilder
// always removes the last non-empty block, if one exists, and any subsequent
// empty blocks.
func TestBundleBuilderPruneLastNonEmpty(t *testing.T) {
	for _, test := range pruneLastNonEmptyTests {
		t.Run(test.name, test.run)
	}
}

type pruneLastNonEmptyTestCase struct {
	name        string
	candidates  []node.BundleCandidate
	expResponse *node.BatchBundleResponse
}

func (tc *pruneLastNonEmptyTestCase) run(t *testing.T) {
	builder := node.NewBundleBuilder(testPrevBlockID)
	for _, candidate := range tc.candidates {
		builder.AddCandidate(candidate)
	}

	builder.PruneLastNonEmpty()
	require.Equal(t, tc.expResponse, builder.Response(nil))
}
