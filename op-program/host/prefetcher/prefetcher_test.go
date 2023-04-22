package prefetcher

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

func TestNoHint(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		prefetcher, _, _, _ := createPrefetcher(t)
		res, err := prefetcher.GetPreimage(context.Background(), common.Hash{0xab})
		require.ErrorIs(t, err, kvstore.ErrNotFound)
		require.Nil(t, res)
	})

	t.Run("Exists", func(t *testing.T) {
		prefetcher, _, _, kv := createPrefetcher(t)
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
		prefetcher, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, rcpts)

		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.HeaderByBlockHash(hash)
		require.Equal(t, eth.HeaderBlockInfo(block.Header()), result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, l1Cl, _, _ := createPrefetcher(t)
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
		prefetcher, _, _, kv := createPrefetcher(t)

		storeBlock(t, kv, block, rcpts)

		// Check the data is available (note the oracle does not know about the block, only the kvstore does)
		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		header, txs := oracle.TransactionsByBlockHash(hash)
		require.EqualValues(t, hash, header.Hash())
		assertTransactionsEqual(t, block.Transactions(), txs)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, l1Cl, _, _ := createPrefetcher(t)
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
		prefetcher, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, receipts)

		// Check the data is available (note the oracle does not know about the block, only the kvstore does)
		oracle := l1.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		header, actualReceipts := oracle.ReceiptsByBlockHash(hash)
		require.EqualValues(t, hash, header.Hash())
		assertReceiptsEqual(t, receipts, actualReceipts)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, l1Cl, _, _ := createPrefetcher(t)
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
		prefetcher, l1Cl, _, kv := createPrefetcher(t)
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

func TestFetchL2Block(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	block, rcpts := testutils.RandomBlock(rng, 10)
	hash := block.Hash()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, rcpts)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.BlockByHash(hash)
		require.EqualValues(t, block.Header(), result.Header())
		assertTransactionsEqual(t, block.Transactions(), result.Transactions())
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, l2Cl, _ := createPrefetcher(t)
		l2Cl.ExpectInfoAndTxsByHash(hash, eth.BlockToInfo(block), block.Transactions(), nil)
		defer l2Cl.MockL2Client.AssertExpectations(t)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.BlockByHash(hash)
		require.EqualValues(t, block.Header(), result.Header())
		assertTransactionsEqual(t, block.Transactions(), result.Transactions())
	})
}

func TestFetchL2Node(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	node := testutils.RandomData(rng, 30)
	hash := crypto.Keccak256Hash(node)
	key := preimage.Keccak256Key(hash).PreimageKey()

	t.Run("AlreadyKnown", func(t *testing.T) {
		prefetcher, _, _, kv := createPrefetcher(t)
		require.NoError(t, kv.Put(key, node))

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.NodeByHash(hash)
		require.EqualValues(t, node, result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, l2Cl, _ := createPrefetcher(t)
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
		prefetcher, _, _, kv := createPrefetcher(t)
		require.NoError(t, kv.Put(key, code))

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.CodeByHash(hash)
		require.EqualValues(t, code, result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, l2Cl, _ := createPrefetcher(t)
		l2Cl.ExpectCodeByHash(hash, code, nil)
		defer l2Cl.MockDebugClient.AssertExpectations(t)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.CodeByHash(hash)
		require.EqualValues(t, code, result)
	})
}

func TestBadHints(t *testing.T) {
	prefetcher, _, _, kv := createPrefetcher(t)
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
		require.ErrorContains(t, err, "invalid hash")
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

type l2Client struct {
	*testutils.MockL2Client
	*testutils.MockDebugClient
}

func createPrefetcher(t *testing.T) (*Prefetcher, *testutils.MockL1Source, *l2Client, kvstore.KV) {
	logger := testlog.Logger(t, log.LvlDebug)
	kv := kvstore.NewMemKV()

	l1Source := new(testutils.MockL1Source)
	l2Source := &l2Client{
		MockL2Client:    new(testutils.MockL2Client),
		MockDebugClient: new(testutils.MockDebugClient),
	}

	prefetcher := NewPrefetcher(logger, l1Source, l2Source, kv)
	return prefetcher, l1Source, l2Source, kv
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
