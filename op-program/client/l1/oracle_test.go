package l1

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	var hints mock.Mock
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

func TestPreimageOracleBlockByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		block, receipts := testutils.RandomBlock(rng, 10)
		t.Run(fmt.Sprintf("block_%d", i), func(t *testing.T) {
			testBlock(t, block, receipts)
		})
	}
}

// TestInitRootsOfUnity validates that the roots of unity are constructed and ordered correctly such that the
// root at index i can be used to compute the field element at index i in a blob
func TestInitRootsOfUnity(t *testing.T) {
	// Check we have the right number of roots
	fieldElementsPerBlob := 4096
	require.Equal(t, fieldElementsPerBlob, len(RootsOfUnity))

	// Create a blob with random data
	var blob kzg4844.Blob
	for i := 0; i < fieldElementsPerBlob; i++ {
		fieldEl := fr.NewElement(uint64(i))
		_, err := fieldEl.SetRandom()
		require.NoError(t, err)
		fieldElBytes := fieldEl.Bytes()
		copy(blob[i*32:(i+1)*32], fieldElBytes[:])
	}

	// Calculate the blob commitment - used for verification later
	blobCommitment, err := kzg4844.BlobToCommitment(&blob)
	require.NoError(t, err)

	// Verify we can generate each field element using the ordered roots of unity
	for i := 0; i < fieldElementsPerBlob; i++ {
		if i%100 == 0 {
			t.Logf("Checking field element %d", i)
		}

		// Extract the target field element
		var targetFieldElement [32]byte
		copy(targetFieldElement[:], blob[i*32:(i+1)*32])

		// We'll use the ith root of unity to evaluate the blob polynomial
		z := RootsOfUnity[i].Bytes()

		// Check that the correct field element is generated by evaluating the polynomial at point z
		kzgProof, evaluation, err := kzg4844.ComputeProof(&blob, z)
		require.NoError(t, err)
		require.Equal(t, targetFieldElement[:], evaluation[:])

		// Verify the proof for good measure
		err = kzg4844.VerifyProof(blobCommitment, z, targetFieldElement, kzgProof)
		require.NoError(t, err)
	}
}
