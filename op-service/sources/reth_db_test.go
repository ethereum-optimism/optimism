package sources

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRethReceiptsLoad(t *testing.T) {
	t.Skip("Skipping test that requires a local L1 Goerli Reth DB")
	t.Parallel()

	// block = https://goerli.etherscan.io/block/994113
	blockHash := common.HexToHash("0x6f6f00553e4f74262a9812927afd11c341730c5c9210824fe172367457adb5f6")
	res, err := FetchRethReceipts("/path/to/goerli-db", &blockHash)
	require.NoError(t, err, "Failed to fetch receipts from Reth DB")
	require.Len(t, res, 2, "Expected 2 receipts to be returned")
	require.Equal(t, res[0].Type, 0)
	require.Equal(t, res[0].CumulativeGasUsed, uint64(93_787))
	require.Equal(t, res[0].Status, uint64(1))
}
