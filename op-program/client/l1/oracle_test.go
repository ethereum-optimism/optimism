package l1

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/kzg"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// testBlock tests that the given block with receipts can be passed through the preimage oracle.
func testBlock(t *testing.T, block *types.Block, receipts []*types.Receipt) {
	// Prepare the pre-images
	preimages := make(map[common.Hash][]byte)

	hdrBytes, err := rlp.EncodeToBytes(block.Header())
	require.NoError(t, err)
	preimages[preimage.Keccak256Key(block.Hash()).PreimageKey()] = hdrBytes

	opaqueTxs, err := eth.EncodeTransactions(block.Transactions())
	require.NoError(t, err)
	_, txsNodes := mpt.WriteTrie(opaqueTxs)
	for _, p := range txsNodes {
		preimages[preimage.Keccak256Key(crypto.Keccak256Hash(p)).PreimageKey()] = p
	}

	opaqueReceipts, err := eth.EncodeReceipts(receipts)
	require.NoError(t, err)
	_, receiptNodes := mpt.WriteTrie(opaqueReceipts)
	for _, p := range receiptNodes {
		preimages[preimage.Keccak256Key(crypto.Keccak256Hash(p)).PreimageKey()] = p
	}

	// Prepare a raw mock pre-image oracle that will serve the pre-image data and handle hints
	po, hints := createTestPreimageOracle(t, preimages)

	// Check if block-headers work
	hints.On("hint", BlockHeaderHint(block.Hash()).Hint()).Once().Return()
	gotHeader := po.HeaderByBlockHash(block.Hash())
	hints.AssertExpectations(t)

	got, err := json.MarshalIndent(gotHeader, "  ", "  ")
	require.NoError(t, err)
	expected, err := json.MarshalIndent(block.Header(), "  ", "  ")
	require.NoError(t, err)
	require.Equal(t, expected, got, "expecting matching headers")

	// Check if blocks with txs work
	hints.On("hint", BlockHeaderHint(block.Hash()).Hint()).Once().Return()
	hints.On("hint", TransactionsHint(block.Hash()).Hint()).Once().Return()
	inf, gotTxs := po.TransactionsByBlockHash(block.Hash())
	hints.AssertExpectations(t)

	require.Equal(t, inf.Hash(), block.Hash())
	expectedTxs := block.Transactions()
	require.Equal(t, len(expectedTxs), len(gotTxs), "expecting equal tx list length")
	for i, tx := range gotTxs {
		require.Equalf(t, tx.Hash(), expectedTxs[i].Hash(), "expecting tx %d to match", i)
	}

	// Check if blocks with receipts work
	hints.On("hint", BlockHeaderHint(block.Hash()).Hint()).Once().Return()
	hints.On("hint", TransactionsHint(block.Hash()).Hint()).Once().Return()
	hints.On("hint", ReceiptsHint(block.Hash()).Hint()).Once().Return()
	inf, gotReceipts := po.ReceiptsByBlockHash(block.Hash())
	hints.AssertExpectations(t)

	require.Equal(t, inf.Hash(), block.Hash())
	require.Equal(t, len(receipts), len(gotReceipts), "expecting equal tx list length")
	for i, r := range gotReceipts {
		require.Equalf(t, r.TxHash, expectedTxs[i].Hash(), "expecting receipt to match tx %d", i)
	}
}

func TestGetBlob(t *testing.T) {
	testCases := []struct {
		name      string
		blobIndex uint64
	}{
		{"blob index 0", 0},
		{"blob index 1", 1},
		{"blob index 2", 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get random blob
			rng := rand.New(rand.NewSource(567 + int64(tc.blobIndex)))
			blob, blobCommitment, err := testutils.RandomBlob(rng)
			require.NoError(t, err)
			// And random block ref
			blockRef := testutils.RandomBlockRef(rng)

			indexedBlobHash := eth.IndexedBlobHash{
				Index: tc.blobIndex,
				Hash:  eth.KZGToVersionedHash(blobCommitment),
			}

			// Setup preimages
			preimages := make(map[common.Hash][]byte)
			// Store blob commitment
			preimages[preimage.Sha256Key(indexedBlobHash.Hash).PreimageKey()] = blobCommitment[:]
			// Store field elements
			fieldElemKey := make([]byte, 80)
			copy(fieldElemKey[:48], blobCommitment[:])
			for i := 0; i < params.BlobTxFieldElementsPerBlob; i++ {
				rootOfUnity := kzg.RootsOfUnity[i].Bytes()
				copy(fieldElemKey[48:], rootOfUnity[:])

				key := preimage.BlobKey(crypto.Keccak256(fieldElemKey)).PreimageKey()
				value := blob[i*32 : (i+1)*32]
				preimages[key] = value
			}

			// Setup expected hint
			blobReqMeta := make([]byte, 8)
			binary.BigEndian.PutUint64(blobReqMeta[0:8], blockRef.Time)
			expectedBlobHint := BlobHint(append(indexedBlobHash.Hash[:], blobReqMeta...)).Hint()

			po, hints := createTestPreimageOracle(t, preimages)

			// Get Blob and verify expectations
			hints.On("hint", expectedBlobHint).Once().Return()
			actualBlob := po.GetBlob(blockRef, indexedBlobHash.Hash)
			hints.AssertExpectations(t)
			require.Equal(t, blob[:], actualBlob[:])
		})
	}
}

func createTestPreimageOracle(t *testing.T, preimages map[common.Hash][]byte) (*PreimageOracle, *mock.Mock) {
	var hints = new(mock.Mock)
	po := &PreimageOracle{
		oracle: preimage.OracleFn(func(key preimage.Key) []byte {
			v, ok := preimages[key.PreimageKey()]
			require.True(t, ok, "preimage must exist")
			return v
		}),
		hint: preimage.HinterFn(func(v preimage.Hint) {
			hints.MethodCalled("hint", v.Hint())
		}),
	}
	return po, hints
}

func TestPreimageOracleBlockByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		block, receipts := testutils.RandomBlock(rng, 10)
		t.Run(fmt.Sprintf("block_%d", i), func(t *testing.T) {
			testBlock(t, block, receipts)
		})
	}
}
