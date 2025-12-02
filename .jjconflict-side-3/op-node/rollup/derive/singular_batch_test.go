package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingularBatchForBatchInterface(t *testing.T) {
	rng := rand.New(rand.NewSource(0x543331))
	chainID := big.NewInt(rng.Int63n(1000))
	txCount := 1 + rng.Intn(8)

	singularBatch := RandomSingularBatch(rng, txCount, chainID)

	require.Equal(t, SingularBatchType, singularBatch.GetBatchType())
	require.Equal(t, singularBatch.Timestamp, singularBatch.GetTimestamp())
	require.Equal(t, singularBatch.EpochNum, singularBatch.GetEpochNum())
}
