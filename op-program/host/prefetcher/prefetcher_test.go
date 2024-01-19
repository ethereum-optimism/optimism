package prefetcher

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func TestNoHint(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		prefetcher, _, _, _, _ := createPrefetcher(t)
		res, err := prefetcher.GetPreimage(context.Background(), common.Hash{0xab})
		require.ErrorIs(t, err, kvstore.ErrNotFound)
		require.Nil(t, res)
	})

	t.Run("Exists", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		data := []byte{1, 2, 3}
		hash := crypto.Keccak256Hash(data)
		require.NoError(t, kv.Put(hash, data))

		res, err := prefetcher.GetPreimage(context.Background(), hash)
		require.NoError(t, err)
		require.Equal(t, res, data)
	})
}

func TestFetchL1BlockHeader(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	block, rcpts := testutils.RandomBlock(rng, 2)
	hash := block.Hash()
	key := preimage.Keccak256Key(hash).PreimageKey()
	pre, err := rlp.EncodeToBytes(block.Header())
	require.NoError(t, err)

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, rcpts)

		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.HeaderByBlockHash(hash)
		require.Equal(t, eth.HeaderBlockInfo(block.Header()), result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, l1Cl, _, _, _ := createPrefetcher(t)
		l1Cl.ExpectInfoByHash(hash, eth.HeaderBlockInfo(block.Header()), nil)
		defer l1Cl.AssertExpectations(t)

		require.NoError(t, prefetcher.Hint(l1.BlockHeaderHint(hash).Hint()))
		result, err := prefetcher.GetPreimage(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, pre, result)
	})
}

func TestFetchL1Transactions(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	block, rcpts := testutils.RandomBlock(rng, 10)
	hash := block.Hash()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)

		storeBlock(t, kv, block, rcpts)

		// Check the data is available (note the oracle does not know about the block, only the kvstore does)
		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		header, txs := oracle.TransactionsByBlockHash(hash)
		require.EqualValues(t, hash, header.Hash())
		assertTransactionsEqual(t, block.Transactions(), txs)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, l1Cl, _, _, _ := createPrefetcher(t)
		l1Cl.ExpectInfoByHash(hash, eth.BlockToInfo(block), nil)
		l1Cl.ExpectInfoAndTxsByHash(hash, eth.BlockToInfo(block), block.Transactions(), nil)
		defer l1Cl.AssertExpectations(t)

		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		header, txs := oracle.TransactionsByBlockHash(hash)
		require.EqualValues(t, hash, header.Hash())
		assertTransactionsEqual(t, block.Transactions(), txs)
	})
}

func TestFetchL1Receipts(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	block, receipts := testutils.RandomBlock(rng, 10)
	hash := block.Hash()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, receipts)

		// Check the data is available (note the oracle does not know about the block, only the kvstore does)
		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		header, actualReceipts := oracle.ReceiptsByBlockHash(hash)
		require.EqualValues(t, hash, header.Hash())
		assertReceiptsEqual(t, receipts, actualReceipts)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, l1Cl, _, _, _ := createPrefetcher(t)
		l1Cl.ExpectInfoByHash(hash, eth.BlockToInfo(block), nil)
		l1Cl.ExpectInfoAndTxsByHash(hash, eth.BlockToInfo(block), block.Transactions(), nil)
		l1Cl.ExpectFetchReceipts(hash, eth.BlockToInfo(block), receipts, nil)
		defer l1Cl.AssertExpectations(t)

		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		header, actualReceipts := oracle.ReceiptsByBlockHash(hash)
		require.EqualValues(t, hash, header.Hash())
		assertReceiptsEqual(t, receipts, actualReceipts)
	})

	// Blocks may have identical RLP receipts for different transactions.
	// Check that the node already existing is handled
	t.Run("CommonTrieNodes", func(t *testing.T) {
		prefetcher, l1Cl, _, _, kv := createPrefetcher(t)
		l1Cl.ExpectInfoByHash(hash, eth.BlockToInfo(block), nil)
		l1Cl.ExpectInfoAndTxsByHash(hash, eth.BlockToInfo(block), block.Transactions(), nil)
		l1Cl.ExpectFetchReceipts(hash, eth.BlockToInfo(block), receipts, nil)
		defer l1Cl.AssertExpectations(t)

		// Pre-store one receipt node (but not the whole trie leading to it)
		// This would happen if an identical receipt was in an earlier block
		opaqueRcpts, err := eth.EncodeReceipts(receipts)
		require.NoError(t, err)
		_, nodes := mpt.WriteTrie(opaqueRcpts)
		require.NoError(t, kv.Put(preimage.Keccak256Key(crypto.Keccak256Hash(nodes[0])).PreimageKey(), nodes[0]))

		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		header, actualReceipts := oracle.ReceiptsByBlockHash(hash)
		require.EqualValues(t, hash, header.Hash())
		assertReceiptsEqual(t, receipts, actualReceipts)
	})
}

// Globally initialize a kzgCtx for blob tests.
var kzgCtx, _ = gokzg4844.NewContext4096Secure()

// Returns a serialized random field element in big-endian
func GetRandFieldElement(seed int64) [32]byte {
	var r fr.Element
	_, _ = r.SetRandom()
	return gokzg4844.SerializeScalar(r)
}

func GetRandBlob(seed int64) gokzg4844.Blob {
	var blob gokzg4844.Blob
	bytesPerBlob := gokzg4844.ScalarsPerBlob * gokzg4844.SerializedScalarSize
	for i := 0; i < bytesPerBlob; i += gokzg4844.SerializedScalarSize {
		fieldElementBytes := GetRandFieldElement(seed + int64(i))
		copy(blob[i:i+gokzg4844.SerializedScalarSize], fieldElementBytes[:])
	}
	return blob
}

func TestFetchL1Blob(t *testing.T) {
	blob := GetRandBlob(0xf00f00)
	commitment, err := kzgCtx.BlobToKZGCommitment(blob, 0)
	require.NoError(t, err)
	versionedHash := sha256.Sum256(commitment[:])
	versionedHash[0] = params.BlobTxHashVersion
	blobHash := eth.IndexedBlobHash{Hash: versionedHash, Index: 0xFACADE}
	l1Ref := eth.L1BlockRef{Time: 0}

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, blobFetcher, _, kv := createPrefetcher(t)
		storeBlob(t, kv, (eth.Bytes48)(commitment), (*eth.Blob)(&blob))

		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		blobFetcher.ExpectOnGetBlobSidecars(
			context.Background(),
			l1Ref,
			[]eth.IndexedBlobHash{blobHash},
			(eth.Bytes48)(commitment),
			[]*eth.Blob{(*eth.Blob)(&blob)},
			nil,
		)
		defer blobFetcher.AssertExpectations(t)

		blobs := oracle.GetBlob(l1Ref, blobHash)
		require.EqualValues(t, blobs[:], blob[:])
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, blobFetcher, _, _ := createPrefetcher(t)

		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		blobFetcher.ExpectOnGetBlobSidecars(
			context.Background(),
			l1Ref,
			[]eth.IndexedBlobHash{blobHash},
			(eth.Bytes48)(commitment),
			[]*eth.Blob{(*eth.Blob)(&blob)},
			nil,
		)
		defer blobFetcher.AssertExpectations(t)

		blobs := oracle.GetBlob(l1Ref, blobHash)
		require.EqualValues(t, blobs[:], blob[:])
	})
}

func TestFetchL2Block(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	block, rcpts := testutils.RandomBlock(rng, 10)
	hash := block.Hash()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, rcpts)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.BlockByHash(hash)
		require.EqualValues(t, block.Header(), result.Header())
		assertTransactionsEqual(t, block.Transactions(), result.Transactions())
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, _, l2Cl, _ := createPrefetcher(t)
		l2Cl.ExpectInfoAndTxsByHash(hash, eth.BlockToInfo(block), block.Transactions(), nil)
		defer l2Cl.MockL2Client.AssertExpectations(t)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.BlockByHash(hash)
		require.EqualValues(t, block.Header(), result.Header())
		assertTransactionsEqual(t, block.Transactions(), result.Transactions())
	})
}

func TestFetchL2Transactions(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	block, rcpts := testutils.RandomBlock(rng, 10)
	hash := block.Hash()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, rcpts)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.LoadTransactions(hash, block.TxHash())
		assertTransactionsEqual(t, block.Transactions(), result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, _, l2Cl, _ := createPrefetcher(t)
		l2Cl.ExpectInfoAndTxsByHash(hash, eth.BlockToInfo(block), block.Transactions(), nil)
		defer l2Cl.MockL2Client.AssertExpectations(t)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.LoadTransactions(hash, block.TxHash())
		assertTransactionsEqual(t, block.Transactions(), result)
	})
}

func TestFetchL2Node(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	node := testutils.RandomData(rng, 30)
	hash := crypto.Keccak256Hash(node)
	key := preimage.Keccak256Key(hash).PreimageKey()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		require.NoError(t, kv.Put(key, node))

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.NodeByHash(hash)
		require.EqualValues(t, node, result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, _, l2Cl, _ := createPrefetcher(t)
		l2Cl.ExpectNodeByHash(hash, node, nil)
		defer l2Cl.MockDebugClient.AssertExpectations(t)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.NodeByHash(hash)
		require.EqualValues(t, node, result)
	})
}

func TestFetchL2Code(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	code := testutils.RandomData(rng, 30)
	hash := crypto.Keccak256Hash(code)
	key := preimage.Keccak256Key(hash).PreimageKey()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		require.NoError(t, kv.Put(key, code))

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.CodeByHash(hash)
		require.EqualValues(t, code, result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, _, l2Cl, _ := createPrefetcher(t)
		l2Cl.ExpectCodeByHash(hash, code, nil)
		defer l2Cl.MockDebugClient.AssertExpectations(t)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.CodeByHash(hash)
		require.EqualValues(t, code, result)
	})
}

func TestBadHints(t *testing.T) {
	prefetcher, _, _, _, kv := createPrefetcher(t)
	hash := common.Hash{0xad}

	t.Run("NoSpace", func(t *testing.T) {
		// Accept the hint
		require.NoError(t, prefetcher.Hint(l1.HintL1BlockHeader))

		// But it will fail to prefetch when the pre-image isn't available
		pre, err := prefetcher.GetPreimage(context.Background(), hash)
		require.ErrorContains(t, err, "unsupported hint")
		require.Nil(t, pre)
	})

	t.Run("InvalidHash", func(t *testing.T) {
		// Accept the hint
		require.NoError(t, prefetcher.Hint(l1.HintL1BlockHeader+" asdfsadf"))

		// But it will fail to prefetch when the pre-image isn't available
		pre, err := prefetcher.GetPreimage(context.Background(), hash)
		require.ErrorContains(t, err, "invalid bytes")
		require.Nil(t, pre)
	})

	t.Run("UnknownType", func(t *testing.T) {
		// Accept the hint
		require.NoError(t, prefetcher.Hint("unknown "+hash.Hex()))

		// But it will fail to prefetch when the pre-image isn't available
		pre, err := prefetcher.GetPreimage(context.Background(), hash)
		require.ErrorContains(t, err, "unknown hint type")
		require.Nil(t, pre)
	})

	// Should not return hint errors if the preimage is already available
	t.Run("KeyExists", func(t *testing.T) {
		// Prepopulate the requested preimage
		value := []byte{1, 2, 3, 4}
		require.NoError(t, kv.Put(hash, value))

		// Hint is invalid
		require.NoError(t, prefetcher.Hint("asdfsadf"))
		// But fetching the key fails because prefetching isn't required
		pre, err := prefetcher.GetPreimage(context.Background(), hash)
		require.NoError(t, err)
		require.Equal(t, value, pre)
	})
}

func TestRetryWhenNotAvailableAfterPrefetching(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	node := testutils.RandomData(rng, 30)
	hash := crypto.Keccak256Hash(node)

	_, l1Source, l1BlobSource, l2Cl, kv := createPrefetcher(t)
	putsToIgnore := 2
	kv = &unreliableKvStore{KV: kv, putsToIgnore: putsToIgnore}
	prefetcher := NewPrefetcher(testlog.Logger(t, log.LvlInfo), l1Source, l1BlobSource, l2Cl, kv)

	// Expect one call for each ignored put, plus one more request for when the put succeeds
	for i := 0; i < putsToIgnore+1; i++ {
		l2Cl.ExpectNodeByHash(hash, node, nil)
	}
	defer l2Cl.MockDebugClient.AssertExpectations(t)

	oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
	result := oracle.NodeByHash(hash)
	require.EqualValues(t, node, result)
}

type unreliableKvStore struct {
	kvstore.KV
	putsToIgnore int
}

func (s *unreliableKvStore) Put(k common.Hash, v []byte) error {
	if s.putsToIgnore > 0 {
		s.putsToIgnore--
		return nil
	}
	println("storing")
	return s.KV.Put(k, v)
}

type l2Client struct {
	*testutils.MockL2Client
	*testutils.MockDebugClient
}

func (m *l2Client) OutputByRoot(ctx context.Context, root common.Hash) (eth.Output, error) {
	out := m.Mock.MethodCalled("OutputByRoot", root)
	return out[0].(eth.Output), *out[1].(*error)
}

func (m *l2Client) ExpectOutputByRoot(root common.Hash, output eth.Output, err error) {
	m.Mock.On("OutputByRoot", root).Once().Return(output, &err)
}

func createPrefetcher(t *testing.T) (*Prefetcher, *testutils.MockL1Source, *testutils.MockBlobsFetcher, *l2Client, kvstore.KV) {
	logger := testlog.Logger(t, log.LvlDebug)
	kv := kvstore.NewMemKV()

	l1Source := new(testutils.MockL1Source)
	l1BlobSource := new(testutils.MockBlobsFetcher)
	l2Source := &l2Client{
		MockL2Client:    new(testutils.MockL2Client),
		MockDebugClient: new(testutils.MockDebugClient),
	}

	prefetcher := NewPrefetcher(logger, l1Source, l1BlobSource, l2Source, kv)
	return prefetcher, l1Source, l1BlobSource, l2Source, kv
}

func storeBlock(t *testing.T, kv kvstore.KV, block *types.Block, receipts types.Receipts) {
	// Pre-store receipts
	opaqueRcpts, err := eth.EncodeReceipts(receipts)
	require.NoError(t, err)
	_, nodes := mpt.WriteTrie(opaqueRcpts)
	for _, p := range nodes {
		require.NoError(t, kv.Put(preimage.Keccak256Key(crypto.Keccak256Hash(p)).PreimageKey(), p))
	}

	// Pre-store transactions
	opaqueTxs, err := eth.EncodeTransactions(block.Transactions())
	require.NoError(t, err)
	_, txsNodes := mpt.WriteTrie(opaqueTxs)
	for _, p := range txsNodes {
		require.NoError(t, kv.Put(preimage.Keccak256Key(crypto.Keccak256Hash(p)).PreimageKey(), p))
	}

	// Pre-store block
	headerRlp, err := rlp.EncodeToBytes(block.Header())
	require.NoError(t, err)
	require.NoError(t, kv.Put(preimage.Keccak256Key(block.Hash()).PreimageKey(), headerRlp))
}

func storeBlob(t *testing.T, kv kvstore.KV, commitment eth.Bytes48, blob *eth.Blob) {
	// Pre-store versioned hash preimage (commitment)
	_ = kv.Put(preimage.Sha256Key(sha256.Sum256(commitment[:])).PreimageKey(), commitment[:])

	// Pre-store blob field elements
	blobKeyBuf := make([]byte, 80)
	copy(blobKeyBuf[:48], commitment[:])
	for i := 0; i < params.BlobTxFieldElementsPerBlob; i++ {
		binary.BigEndian.PutUint64(blobKeyBuf[:72], uint64(i))
		feKey := crypto.Keccak256Hash(blobKeyBuf)

		_ = kv.Put(preimage.BlobKey(feKey).PreimageKey(), blob[i<<5:(i+1)<<5])
	}
}

func asOracleFn(t *testing.T, prefetcher *Prefetcher) preimage.OracleFn {
	return func(key preimage.Key) []byte {
		pre, err := prefetcher.GetPreimage(context.Background(), key.PreimageKey())
		require.NoError(t, err)
		return pre
	}
}

func asHinter(t *testing.T, prefetcher *Prefetcher) preimage.HinterFn {
	return func(v preimage.Hint) {
		err := prefetcher.Hint(v.Hint())
		require.NoError(t, err)
	}
}

func assertTransactionsEqual(t *testing.T, blockTx types.Transactions, txs types.Transactions) {
	require.Equal(t, len(blockTx), len(txs))
	for i, tx := range txs {
		require.Equal(t, blockTx[i].Hash(), tx.Hash())
	}
}

func assertReceiptsEqual(t *testing.T, expectedRcpt types.Receipts, actualRcpt types.Receipts) {
	require.Equal(t, len(expectedRcpt), len(actualRcpt))
	for i, rcpt := range actualRcpt {
		// Make a copy of each to zero out fields we expect to be different
		expected := *expectedRcpt[i]
		actual := *rcpt
		expected.ContractAddress = common.Address{}
		actual.ContractAddress = common.Address{}
		require.Equal(t, expected, actual)
	}
}
