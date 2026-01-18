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

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func runLegacyAndNew(t *testing.T, baseName string, fn func(t *testing.T, hintL2ChainIDs bool)) {
	t.Helper()
	t.Run("legacy_"+baseName, func(t *testing.T) {
		fn(t, false)
	})
	t.Run(baseName, func(t *testing.T) {
		fn(t, true)
	})
}

func mockPreimageOracle(t *testing.T, hintL2ChainIDs bool) (po *PreimageOracle, hintsMock *mock.Mock, preimages map[common.Hash][]byte) {
	// Prepare the pre-images
	preimages = make(map[common.Hash][]byte)

	hintsMock = new(mock.Mock)

	rawOracle := preimage.OracleFn(func(key preimage.Key) []byte {
		v, ok := preimages[key.PreimageKey()]
		require.True(t, ok, "preimage must exist")
		return v
	})
	hinter := preimage.HinterFn(func(v preimage.Hint) {
		hintsMock.MethodCalled("hint", v.Hint())
	})
	po = NewPreimageOracle(rawOracle, hinter, hintL2ChainIDs)
	return
}

// testBlock tests that the given block can be passed through the preimage oracle.
func testBlock(t *testing.T, block *types.Block, hintL2ChainIDs bool) {
	po, hints, preimages := mockPreimageOracle(t, hintL2ChainIDs)

	hdrBytes, err := rlp.EncodeToBytes(block.Header())
	require.NoError(t, err)
	preimages[preimage.Keccak256Key(block.Hash()).PreimageKey()] = hdrBytes

	opaqueTxs, err := eth.EncodeTransactions(block.Transactions())
	require.NoError(t, err)
	_, txsNodes := mpt.WriteTrie(opaqueTxs)
	for _, p := range txsNodes {
		preimages[preimage.Keccak256Key(crypto.Keccak256Hash(p)).PreimageKey()] = p
	}

	chainID := eth.ChainIDFromUInt64(4924)

	// Prepare a raw mock pre-image oracle that will serve the pre-image data and handle hints

	// Check if blocks with txs work
	if hintL2ChainIDs {
		hints.On("hint", BlockHeaderHint{Hash: block.Hash(), ChainID: chainID}.Hint()).Once().Return()
		hints.On("hint", TransactionsHint{Hash: block.Hash(), ChainID: chainID}.Hint()).Once().Return()
	} else {
		hints.On("hint", LegacyBlockHeaderHint(block.Hash()).Hint()).Once().Return()
		hints.On("hint", LegacyTransactionsHint(block.Hash()).Hint()).Once().Return()
	}
	gotBlock := po.BlockByHash(block.Hash(), chainID)
	hints.AssertExpectations(t)

	require.Equal(t, block.Hash(), gotBlock.Hash())
	expectedTxs := block.Transactions()
	require.Equal(t, len(expectedTxs), len(gotBlock.Transactions()), "expecting equal tx list length")
	for i, tx := range gotBlock.Transactions() {
		require.Equalf(t, expectedTxs[i].Hash(), tx.Hash(), "expecting tx %d to match", i)
	}
}

func TestPreimageOracleBlockByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		block, _ := testutils.RandomBlock(rng, 10)
		t.Run(fmt.Sprintf("legacy_block_%d", i), func(t *testing.T) {
			testBlock(t, block, false)
		})

		t.Run(fmt.Sprintf("block_%d", i), func(t *testing.T) {
			testBlock(t, block, true)
		})
	}
}

func TestPreimageOracleNodeByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		chainID := eth.ChainIDFromUInt64(rng.Uint64())
		runLegacyAndNew(t, fmt.Sprintf("node_%d", i), func(t *testing.T, hintL2ChainIDs bool) {
			po, hints, preimages := mockPreimageOracle(t, hintL2ChainIDs)

			node := make([]byte, 123)
			rng.Read(node)

			h := crypto.Keccak256Hash(node)
			preimages[preimage.Keccak256Key(h).PreimageKey()] = node

			if hintL2ChainIDs {
				hints.On("hint", StateNodeHint{Hash: h, ChainID: chainID}.Hint()).Once().Return()
			} else {
				hints.On("hint", LegacyStateNodeHint(h).Hint()).Once().Return()
			}

			gotNode := po.NodeByHash(h, chainID)
			hints.AssertExpectations(t)
			require.Equal(t, hexutil.Bytes(node), hexutil.Bytes(gotNode), "node matches")
		})
	}
}

func TestPreimageOracleCodeByHash(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		chainID := eth.ChainIDFromUInt64(rng.Uint64())
		runLegacyAndNew(t, fmt.Sprintf("code_%d", i), func(t *testing.T, hintL2ChainIDs bool) {
			po, hints, preimages := mockPreimageOracle(t, hintL2ChainIDs)

			code := make([]byte, 123)
			rng.Read(code)

			h := crypto.Keccak256Hash(code)
			preimages[preimage.Keccak256Key(h).PreimageKey()] = code

			if hintL2ChainIDs {
				hints.On("hint", CodeHint{Hash: h, ChainID: chainID}.Hint()).Once().Return()
			} else {
				hints.On("hint", LegacyCodeHint(h).Hint()).Once().Return()
			}

			gotCode := po.CodeByHash(h, chainID)
			hints.AssertExpectations(t)
			require.Equal(t, hexutil.Bytes(code), hexutil.Bytes(gotCode), "code matches")
		})
	}
}

func TestPreimageOracleOutputByRoot(t *testing.T) {
	rng := rand.New(rand.NewSource(123))

	for i := 0; i < 10; i++ {
		chainID := eth.ChainIDFromUInt64(rng.Uint64())
		runLegacyAndNew(t, fmt.Sprintf("output_%d", i), func(t *testing.T, hintL2ChainIDs bool) {
			po, hints, preimages := mockPreimageOracle(t, hintL2ChainIDs)
			output := testutils.RandomOutputV0(rng)

			h := common.Hash(eth.OutputRoot(output))
			preimages[preimage.Keccak256Key(h).PreimageKey()] = output.Marshal()

			if hintL2ChainIDs {
				hints.On("hint", L2OutputHint{Hash: h, ChainID: chainID}.Hint()).Once().Return()
			} else {
				hints.On("hint", LegacyL2OutputHint(h).Hint()).Once().Return()
			}

			gotOutput := po.OutputByRoot(h, chainID)
			hints.AssertExpectations(t)
			require.Equal(t, hexutil.Bytes(output.Marshal()), hexutil.Bytes(gotOutput.Marshal()), "output matches")
		})
	}
}
