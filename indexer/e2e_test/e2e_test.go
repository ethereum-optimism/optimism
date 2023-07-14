package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	testSuite := createE2ETestSuite(t)
	t.Run("indexes block headers", func(t *testing.T) {
		// L1
		latestL1Header, err := testSuite.DB.Blocks.LatestL1BlockHeader()
		require.NoError(t, err)
		require.NotNil(t, latestL1Header)
		require.True(t, latestL1Header.Number.Int.Uint64() >= 9)

		// L1
		latestL2Header, err := testSuite.DB.Blocks.LatestL2BlockHeader()
		require.NoError(t, err)
		require.NotNil(t, latestL2Header)
		require.True(t, latestL2Header.Number.Int.Uint64() >= 9)
	})

	t.Run("indexes l2 checkpoints", func(t *testing.T) {
		latestOutput, err := testSuite.DB.Blocks.LatestCheckpointedOutput()
		require.NoError(t, err)
		require.NotNil(t, latestOutput)
		require.True(t, latestOutput.L2BlockNumber.Int.Uint64() >= 9)
	})
}
