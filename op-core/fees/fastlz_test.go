package fees

import (
	"bytes"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/fees/internal/cfastlz"
)

var (
	// emptyTx is the binary encoding of an unsigned legacy transaction to
	// 0x095e7baea6a6c7c4c2dfeb977efac326af552d87 with every other field zero.
	emptyTx = common.FromHex("dd80808094095e7baea6a6c7c4c2dfeb977efac326af552d878080808080")

	contractCallTx = common.FromHex("02f901550a758302df1483be21b88304743f94f8" +
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
)

// ascending returns the bytes 1, 2, ..., n, among which no 3-byte sequence repeats.
func ascending(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i + 1)
	}
	return b
}

// TestFlzCompressLen pins FlzCompressLen on inputs that reach each branch of the compressor. The
// expected lengths are those of the C FastLZ library and of Solady's LibZip. The test checks each
// length against the C library, the original implementation that the Go, Solidity and Rust code
// port. op-revm's tests pin the same branch vectors and check them against LibZip.
func TestFlzCompressLen(t *testing.T) {
	testCases := []struct {
		name        string
		input       []byte
		expectedLen uint32
	}{
		{"empty", []byte{}, 0},
		// The match search starts at byte 2 and stops 13 bytes before the end, so inputs shorter
		// than 13 bytes skip it, and inputs up to 16 bytes are all literals even when they repeat.
		// With 16 zeros, the search finds a match on its last probe and discards it; 17 zeros is
		// the shortest input that keeps a match.
		{"one byte", []byte{0x01}, 2},
		{"three bytes", []byte{0xfa, 0xca, 0xde}, 4},
		{"twelve zeros", make([]byte, 12), 13},
		{"16 zeros", make([]byte, 16), 17},
		{"17 zeros", make([]byte, 17), 12},
		// Literal runs are split into runs of 32 bytes, here without and with a remainder.
		{"64 incompressible bytes", ascending(64), 66},
		{"100 incompressible bytes", ascending(100), 104},
		// A run of zeros is one long match, which is split every 262 bytes. These two lengths put
		// the remainder on each side of the boundary between its 2-byte and 3-byte encodings.
		{"277 zeros", make([]byte, 277), 14},
		{"278 zeros", make([]byte, 278), 15},
		// A repeat counts as a match at a distance of up to 8191 bytes.
		{"repeat 8191 bytes apart", slices.Concat(ascending(32), make([]byte, 8191-32), ascending(32)), 140},
		{"repeat 8192 bytes apart", slices.Concat(ascending(32), make([]byte, 8192-32), ascending(32)), 164},
		// 01 02 03 and bf 02 00 share a hash table slot, but their bytes differ, so they don't match.
		{"hash collision", slices.Concat([]byte{0xaa, 0xbb, 0x01, 0x02, 0x03, 0xbf, 0x02, 0x00}, ascending(13)), 22},
		{"all ones", bytes.Repeat([]byte{1}, 1000), 21},
		{"all zeros", make([]byte, 1000), 21},
		{"empty tx", emptyTx, 31},
		{"contract call tx", contractCallTx, 202},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedLen, FlzCompressLen(tc.input))
			require.Equal(t, int(tc.expectedLen), len(cfastlz.Compress(tc.input)), "C FastLZ")
		})
	}
}
