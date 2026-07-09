package fees

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// TestFlzCompressLen uses op-geth's full set of FlzCompressLen reference vectors.
func TestFlzCompressLen(t *testing.T) {
	emptyTx := types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       ptr.New(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")),
		Value:    big.NewInt(0),
		Gas:      0,
		GasPrice: big.NewInt(0),
	})
	emptyTxBytes, err := emptyTx.MarshalBinary()
	require.NoError(t, err)

	contractCallTx, err := hex.DecodeString("02f901550a758302df1483be21b88304743f94f8" +
		"0e51afb613d764fa61751affd3313c190a86bb870151bd62fd12adb8" +
		"e41ef24f3f0000000000000000000000000000000000000000000000" +
		"00000000000000006e000000000000000000000000af88d065e77c8c" +
		"c2239327c5edb3a432268e5831000000000000000000000000000000" +
		"000000000000000000000000000003c1e50000000000000000000000" +
		"00000000000000000000000000000000000000000000000000000000" +
		"000000000000000000000000000000000000000000000000a0000000" +
		"00000000000000000000000000000000000000000000000000000000" +
		"148c89ed219d02f1a5be012c689b4f5b731827bebe00000000000000" +
		"0000000000c001a033fd89cb37c31b2cba46b6466e040c61fc9b2a36" +
		"75a7f5f493ebd5ad77c497f8a07cdf65680e238392693019b4092f61" +
		"0222e71b7cec06449cb922b93b6a12744e")
	require.NoError(t, err)

	testCases := []struct {
		name        string
		input       []byte
		expectedLen uint32
	}{
		{"empty", []byte{}, 0},
		{"all ones", bytes.Repeat([]byte{1}, 1000), 21},
		{"all zeros", make([]byte, 1000), 21},
		{"empty tx", emptyTxBytes, 31},
		{"contract call tx", contractCallTx, 202},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedLen, FlzCompressLen(tc.input))
		})
	}
}
