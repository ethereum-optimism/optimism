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

	// ETH Mainnet block #18,663,292
	//
	// https://etherscan.io/tx/0x88b2d153a4e893ba91ac235325c44b1aa0c802fcb42657701e1a73e1c675f7ca
	//
	// NOTE: The block hash differs from the live block due to a state root mismatch. In order to generate
	//       a testdata database with only this block in it, the state root of the block was modified.
	//       Old State Root: 0xaf81a692d228d56d35c80d65aeba59636b4671403054f6c57446c0e3e4d951c8
	//       New State Root (Empty MPT): 0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421
	blockHash := common.HexToHash("0x6a229123d607c2232a8b0bdd36f90745945d05181018e64e60ff2b93ab6b52e5")
	res, err := FetchRethReceipts("../rethdb-reader/testdata/db", &blockHash)
	require.NoError(t, err)

	receipt := (*types.Receipt)(res[0])
	require.Equal(t, receipt.Type, uint8(2))
	require.Equal(t, receipt.Status, uint64(1))
	require.Equal(t, receipt.CumulativeGasUsed, uint64(115_316))
	require.Equal(t, receipt.Bloom, types.BytesToBloom(common.Hex2Bytes("00200000000000000000000080001000000000000000000000000000000000000000000000000000000000000000100002000100080000000000000000000000000000000000000000000008000000200000000400000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000400000000000001000000000000000100000000080000004000000000000000000000000000000000000002000000000000000000000000000000000000000006000000000000000000000000000000000000001000000000000000000000200000000000000100000000020000000000000000000000000000000010")))
	require.Equal(t, receipt.Logs[0].Address, common.HexToAddress("c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"))
	require.Equal(t, receipt.Logs[0].Topics[0], common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"))
	require.Equal(t, receipt.Logs[0].Topics[1], common.HexToHash("00000000000000000000000000000000003b3cc22af3ae1eac0440bcee416b40"))
	require.Equal(t, receipt.Logs[0].Data, common.Hex2Bytes("00000000000000000000000000000000000000000000000008a30cd230000000"))
	require.Equal(t, receipt.TxHash, common.HexToHash("0x88b2d153a4e893ba91ac235325c44b1aa0c802fcb42657701e1a73e1c675f7ca"))

	require.Equal(t, receipt.BlockHash, blockHash)
	require.Equal(t, receipt.BlockNumber, big.NewInt(18_663_292))
	require.Equal(t, receipt.TransactionIndex, uint(0))
}
