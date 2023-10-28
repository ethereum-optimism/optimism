//go:build rethdb

package sources

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRethDBReceiptsLoad(t *testing.T) {
	t.Parallel()

	// Goerli block #9942861, with only the first transaction persisted to the DB
	//
	// https://goerli.etherscan.io/tx/0x12c0074a4a7916fe6f39de8417fe93f1fa77bcadfd5fc31a317fb6c344f66602
	blockHash := common.HexToHash("0xbcc3fb97b87bb4b14bacde74255cbfcf52675c0ad5e06fa264c0e5d6c0afd96e")
	res, err := FetchRethReceipts("../rethdb-reader/testdata/db", &blockHash)
	require.NoError(t, err)

	receipt := (*types.Receipt)(res[0])
	require.Equal(t, receipt.Type, uint8(2))
	require.Equal(t, receipt.Status, uint64(1))
	require.Equal(t, receipt.CumulativeGasUsed, uint64(241_404))
	require.Equal(t, receipt.Bloom, types.BytesToBloom(common.Hex2Bytes("00000000000000000000000000000000000000000100008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000004000000000000000010020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000800000000000000000000000000000000000000000000000000000000")))
	require.Equal(t, receipt.Logs[0].Address, common.HexToAddress("4ce63f351597214ef0b9a319124eea9e0f9668bb"))
	require.Equal(t, receipt.Logs[0].Topics[0], common.HexToHash("0cdbd8bd7813095001c5fe7917bd69d834dc01db7c1dfcf52ca135bd20384413"))
	require.Equal(t, receipt.Logs[0].Topics[1], common.HexToHash("00000000000000000000000000000000000000000000000000000000000000c2"))
	require.Equal(t, receipt.Logs[0].Data, []byte{})
	require.Equal(t, receipt.TxHash, common.HexToHash("0x12c0074a4a7916fe6f39de8417fe93f1fa77bcadfd5fc31a317fb6c344f66602"))

	require.Equal(t, receipt.BlockHash, common.HexToHash("0xbcc3fb97b87bb4b14bacde74255cbfcf52675c0ad5e06fa264c0e5d6c0afd96e"))
	require.Equal(t, receipt.BlockNumber, big.NewInt(9942861))
	require.Equal(t, receipt.TransactionIndex, uint(0))
}
