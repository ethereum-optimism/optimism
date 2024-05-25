package trace

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/stretchr/testify/require"
)

func TestTranslate(t *testing.T) {
	orig := alphabet.NewTraceProvider("abcdefghij", 4)
	translated := Translate(orig, 3)
	// All nodes on the first translated layer, map to GIndex 1
	for i := int64(8); i <= 15; i++ {
		requireSameValue(t, orig, 1, translated, i)
	}
	// Nodes on the second translated layer map to GIndex 2 and 3 alternately
	for i := int64(16); i <= 31; i += 2 {
		requireSameValue(t, orig, 2, translated, i)
		requireSameValue(t, orig, 3, translated, i+1)
	}
	// Nodes on the third translated layer map to GIndex 4, 5, 6 and 7
	for i := int64(32); i <= 61; i += 4 {
		requireSameValue(t, orig, 4, translated, i)
		requireSameValue(t, orig, 5, translated, i+1)
		requireSameValue(t, orig, 6, translated, i+2)
		requireSameValue(t, orig, 7, translated, i+3)
	}
}

func requireSameValue(t *testing.T, a types.TraceProvider, aGIdx int64, b types.TraceProvider, bGIdx int64) {
	// Check Get returns the same results
	aValue, err := a.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(aGIdx)))
	require.NoError(t, err)
	bValue, err := b.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(bGIdx)))
	require.NoError(t, err)
	require.Equal(t, aValue, bValue)

	// Check GetStepData returns the same results
	aPrestate, aProofData, aPreimageData, err := a.GetStepData(context.Background(), types.NewPositionFromGIndex(big.NewInt(aGIdx)))
	require.NoError(t, err)
	bPrestate, bProofData, bPreimageData, err := b.GetStepData(context.Background(), types.NewPositionFromGIndex(big.NewInt(bGIdx)))
	require.NoError(t, err)
	require.Equal(t, aPrestate, bPrestate)
	require.Equal(t, aProofData, bProofData)
	require.Equal(t, aPreimageData, bPreimageData)
}

func TestTranslate_AbsolutePreStateCommitment(t *testing.T) {
	orig := alphabet.NewTraceProvider("abcdefghij", 4)
	translated := Translate(orig, 3)
	origValue, err := orig.AbsolutePreStateCommitment(context.Background())
	require.NoError(t, err)
	translatedValue, err := translated.AbsolutePreStateCommitment(context.Background())
	require.NoError(t, err)
	require.Equal(t, origValue, translatedValue)
}
