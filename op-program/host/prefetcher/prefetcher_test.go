package prefetcher

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	l1test "github.com/ethereum-optimism/optimism/op-program/client/l1/test"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	l2test "github.com/ethereum-optimism/optimism/op-program/client/l2/test"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestNoHint(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		prefetcher, _, _, _, _ := createPrefetcher(t)
		res, err := prefetcher.GetPreimage(common.Hash{0xab})
		require.ErrorIs(t, err, kvstore.ErrNotFound)
		require.Nil(t, res)
	})

	t.Run("Exists", func(t *testing.T) {
		prefetcher, _, _, _, kv := createPrefetcher(t)
		data := []byte{1, 2, 3}
		hash := crypto.Keccak256Hash(data)
		require.NoError(t, kv.Put(hash, data))

		res, err := prefetcher.GetPreimage(hash)
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
		prefetcher, l1Oracle, _, _, _ := createPrefetcher(t)
		l1Oracle.Blocks[hash] = eth.HeaderBlockInfo(block.Header())
		require.NoError(t, prefetcher.Hint(l1.BlockHeaderHint(hash).Hint()))
		result, err := prefetcher.GetPreimage(key)
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
		prefetcher, l1Oracle, _, _, _ := createPrefetcher(t)
		l1Oracle.Blocks[hash] = eth.BlockToInfo(block)
		l1Oracle.Txs[hash] = block.Transactions()

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
		prefetcher, l1Oracle, _, _, _ := createPrefetcher(t)
		l1Oracle.Blocks[hash] = eth.BlockToInfo(block)
		l1Oracle.Txs[hash] = block.Transactions()
		l1Oracle.Rcpts[hash] = receipts

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
		prefetcher, _, _, _, kv := createPrefetcher(t)
		storeBlock(t, kv, block, rcpts)

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.BlockByHash(hash)
		require.EqualValues(t, block.Header(), result.Header())
		assertTransactionsEqual(t, block.Transactions(), result.Transactions())
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, l2BlockOracle, _, _ := createPrefetcher(t)
		l2BlockOracle.Blocks[hash] = block

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
		prefetcher, _, _, _, kv := createPrefetcher(t)
		require.NoError(t, kv.Put(key, node))

		oracle := l2.NewPreimageOracle(asOracleFn(t, prefetcher), asHinter(t, prefetcher))
		result := oracle.NodeByHash(hash)
		require.EqualValues(t, node, result)
	})

	t.Run("Unknown", func(t *testing.T) {
		prefetcher, _, _, l2StateOracle, _ := createPrefetcher(t)
		l2StateOracle.Data[hash] = node

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
		prefetcher, _, _, l2StateOracle, _ := createPrefetcher(t)
		l2StateOracle.Code[hash] = code

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
		pre, err := prefetcher.GetPreimage(hash)
		require.ErrorContains(t, err, "unsupported hint")
		require.Nil(t, pre)
	})

	t.Run("InvalidHash", func(t *testing.T) {
		// Accept the hint
		require.NoError(t, prefetcher.Hint(l1.HintL1BlockHeader+" asdfsadf"))

		// But it will fail to prefetch when the pre-image isn't available
		pre, err := prefetcher.GetPreimage(hash)
		require.ErrorContains(t, err, "invalid hash")
		require.Nil(t, pre)
	})

	t.Run("UnknownType", func(t *testing.T) {
		// Accept the hint
		require.NoError(t, prefetcher.Hint("unknown "+hash.Hex()))

		// But it will fail to prefetch when the pre-image isn't available
		pre, err := prefetcher.GetPreimage(hash)
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
		pre, err := prefetcher.GetPreimage(hash)
		require.NoError(t, err)
		require.Equal(t, value, pre)
	})
}

func createPrefetcher(t *testing.T) (*Prefetcher, *l1test.StubOracle, *l2test.StubBlockOracle, *l2test.StubStateOracle, kvstore.KV) {
	kv := kvstore.NewMemKV()
	l1Oracle := l1test.NewStubOracle(t)
	l2BlockOracle, l2StateOracle := l2test.NewStubOracle(t)
	prefetcher := NewPrefetcher(l1Oracle, l2BlockOracle, kv)
	return prefetcher, l1Oracle, l2BlockOracle, l2StateOracle, kv
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
		pre, err := prefetcher.GetPreimage(key.PreimageKey())
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
