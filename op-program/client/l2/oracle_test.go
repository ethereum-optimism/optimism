package l2

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

func mockPreimageOracle(t *testing.T) (po *PreimageOracle, hintsMock *mock.Mock, preimages map[common.Hash][]byte) {
	// Prepare the pre-images
	preimages = make(map[common.Hash][]byte)

	hintsMock = new(mock.Mock)

	po = &PreimageOracle{
		oracle: preimage.OracleFn(func(key preimage.Key) []byte {
			v, ok := preimages[key.PreimageKey()]
			require.True(t, ok, "preimage must exist")
			return v
		}),
		hint: preimage.HinterFn(func(v preimage.Hint) {
			hintsMock.MethodCalled("hint", v.Hint())
		}),
	}

	return po, hintsMock, preimages
}

// testBlock tests that the given block can be passed through the preimage oracle.
func testBlock(t *testing.T, block *types.Block) {
	po, hints, preimages := mockPreimageOracle(t)

	hdrBytes, err := rlp.EncodeToBytes(block.Header())
	require.NoError(t, err)
	preimages[preimage.Keccak256Key(block.Hash()).PreimageKey()] = hdrBytes

	opaqueTxs, err := eth.EncodeTransactions(block.Transactions())
	require.NoError(t, err)
	_, txsNodes := mpt.WriteTrie(opaqueTxs)
	for _, p := range txsNodes {
		preimages[preimage.Keccak256Key(crypto.Keccak256Hash(p)).PreimageKey()] = p
	}

	// Prepare a raw mock pre-image oracle that will serve the pre-image data and handle hints

	// Check if blocks with txs work
	hints.On("hint", BlockHeaderHint(block.Hash()).Hint()).Once().Return()
	hints.On("hint", TransactionsHint(block.Hash()).Hint()).Once().Return()
	gotBlock := po.BlockByHash(block.Hash())
	hints.AssertExpectations(t)

	require.Equal(t, gotBlock.Hash(), block.Hash())
	expectedTxs := block.Transactions()
	require.Equal(t, len(expectedTxs), len(gotBlock.Transactions()), "expecting equal tx list length")
	for i, tx := range gotBlock.Transactions() {
		require.Equalf(t, tx.Hash(), expectedTxs[i].Hash(), "expecting tx %d to match", i)
	}
}

func TestPreimageOracleBlockByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		block, _ := testutils.RandomBlock(rng, 10)
		t.Run(fmt.Sprintf("block_%d", i), func(t *testing.T) {
			testBlock(t, block)
		})
	}
}

func TestPreimageOracleNodeByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("node_%d", i), func(t *testing.T) {
			po, hints, preimages := mockPreimageOracle(t)

			node := make([]byte, 123)
			rng.Read(node)

			h := crypto.Keccak256Hash(node)
			preimages[preimage.Keccak256Key(h).PreimageKey()] = node

			hints.On("hint", StateNodeHint(h).Hint()).Once().Return()
			gotNode := po.NodeByHash(h)
			hints.AssertExpectations(t)
			require.Equal(t, hexutil.Bytes(node), hexutil.Bytes(gotNode), "node matches")
		})
	}
}

func TestPreimageOracleCodeByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("code_%d", i), func(t *testing.T) {
			po, hints, preimages := mockPreimageOracle(t)

			node := make([]byte, 123)
			rng.Read(node)

			h := crypto.Keccak256Hash(node)
			preimages[preimage.Keccak256Key(h).PreimageKey()] = node

			hints.On("hint", CodeHint(h).Hint()).Once().Return()
			gotNode := po.CodeByHash(h)
			hints.AssertExpectations(t)
			require.Equal(t, hexutil.Bytes(node), hexutil.Bytes(gotNode), "code matches")
		})
	}
}
