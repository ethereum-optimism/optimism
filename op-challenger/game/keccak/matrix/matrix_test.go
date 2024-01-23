package matrix

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/commitments.json
var refTests []byte

func TestStateCommitment(t *testing.T) {
	tests := []struct {
		expectedPacked string
		matrix         []uint64 // Automatically padded with 0s to the required length
	}{
		{
			expectedPacked: "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			expectedPacked: "000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000009000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000b000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000d000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000f0000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001100000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000013000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000150000000000000000000000000000000000000000000000000000000000000016000000000000000000000000000000000000000000000000000000000000001700000000000000000000000000000000000000000000000000000000000000180000000000000000000000000000000000000000000000000000000000000019",
			matrix:         []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
		},
		{
			expectedPacked: "000000000000000000000000000000000000000000000000ffffffffffffffff000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			matrix:         []uint64{18446744073709551615},
		},
	}
	for _, test := range tests {
		test := test
		t.Run("", func(t *testing.T) {
			state := NewStateMatrix()
			copy(state.s.a[:], test.matrix)
			expected := crypto.Keccak256Hash(common.Hex2Bytes(test.expectedPacked))
			actual := state.StateCommitment()
			require.Equal(t, test.expectedPacked, common.Bytes2Hex(state.PackState()))
			require.Equal(t, expected, actual)
		})
	}
}

type testData struct {
	Input       []byte        `json:"input"`
	Commitments []common.Hash `json:"commitments"`
}

func TestReferenceCommitments(t *testing.T) {
	var tests []testData
	require.NoError(t, json.Unmarshal(refTests, &tests))

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Ref-%v", i), func(t *testing.T) {
			s := NewStateMatrix()
			commitments := []common.Hash{s.StateCommitment()}
			for i := 0; i < len(test.Input); i += types.BlockSize {
				end := min(i+types.BlockSize, len(test.Input))
				s.absorbLeafInput(test.Input[i:end], end == len(test.Input))
				commitments = append(commitments, s.StateCommitment())
			}
			if len(test.Input) == 0 {
				s.absorbLeafInput(nil, true)
				commitments = append(commitments, s.StateCommitment())
			}
			actual := s.Hash()
			expected := crypto.Keccak256Hash(test.Input)
			require.Equal(t, expected, actual)
			require.Equal(t, test.Commitments, commitments)
		})
	}
}

func TestReferenceCommitmentsFromReader(t *testing.T) {
	var tests []testData
	require.NoError(t, json.Unmarshal(refTests, &tests))

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Ref-%v", i), func(t *testing.T) {
			s := NewStateMatrix()
			commitments := []common.Hash{s.StateCommitment()}
			in := bytes.NewReader(test.Input)
			for {
				_, err := s.absorbNextLeafInput(in)
				if errors.Is(err, io.EOF) {
					commitments = append(commitments, s.StateCommitment())
					break
				}
				// Shouldn't get any error except EOF
				require.NoError(t, err)
				commitments = append(commitments, s.StateCommitment())
			}
			actual := s.Hash()
			expected := crypto.Keccak256Hash(test.Input)
			require.Equal(t, expected, actual)
			require.Equal(t, test.Commitments, commitments)
		})
	}
}

func TestAbsorbUpTo_ReferenceCommitments(t *testing.T) {
	var tests []testData
	require.NoError(t, json.Unmarshal(refTests, &tests))

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Ref-%v", i), func(t *testing.T) {
			s := NewStateMatrix()
			commitments := []common.Hash{s.StateCommitment()}
			in := bytes.NewReader(test.Input)
			for {
				input, err := s.AbsorbUpTo(in, types.BlockSize*3)
				if errors.Is(err, io.EOF) {
					commitments = append(commitments, input.Commitments...)
					break
				}
				// Shouldn't get any error except EOF
				require.NoError(t, err)
				commitments = append(commitments, input.Commitments...)
			}
			actual := s.Hash()
			expected := crypto.Keccak256Hash(test.Input)
			require.Equal(t, expected, actual)
			require.Equal(t, test.Commitments, commitments)
		})
	}
}

func TestAbsorbUpTo_LimitsDataRead(t *testing.T) {
	s := NewStateMatrix()
	data := testutils.RandomData(rand.New(rand.NewSource(2424)), types.BlockSize*6+20)
	in := bytes.NewReader(data)
	// Should fully read the first four leaves worth
	inputData, err := s.AbsorbUpTo(in, types.BlockSize*4)
	require.NoError(t, err)
	require.Equal(t, data[0:types.BlockSize*4], inputData.Input)
	require.Len(t, inputData.Commitments, 4)
	require.False(t, inputData.Finalize)

	// Should read the remaining data and return EOF
	inputData, err = s.AbsorbUpTo(in, types.BlockSize*10)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, data[types.BlockSize*4:], inputData.Input)
	require.Len(t, inputData.Commitments, 3, "2 full leaves plus the final partial leaf")
	require.True(t, inputData.Finalize)
}

func TestAbsorbUpTo_InvalidLengths(t *testing.T) {
	s := NewStateMatrix()
	lengths := []int{-types.BlockSize, -1, 0, 1, types.BlockSize - 1, types.BlockSize + 1, 2*types.BlockSize + 1}
	for _, length := range lengths {
		_, err := s.AbsorbUpTo(bytes.NewReader(nil), length)
		require.ErrorIsf(t, err, ErrInvalidMaxLen, "Should get invalid length for length %v", length)
	}
}

func TestMatrix_AbsorbNextLeaf(t *testing.T) {
	fullLeaf := make([]byte, types.BlockSize)
	for i := 0; i < types.BlockSize; i++ {
		fullLeaf[i] = byte(i)
	}
	tests := []struct {
		name       string
		input      []byte
		leafInputs [][]byte
		errs       []error
	}{
		{
			name:       "empty",
			input:      []byte{},
			leafInputs: [][]byte{{}},
			errs:       []error{io.EOF},
		},
		{
			name:       "single",
			input:      fullLeaf,
			leafInputs: [][]byte{fullLeaf},
			errs:       []error{io.EOF},
		},
		{
			name:       "single-overflow",
			input:      append(fullLeaf, byte(9)),
			leafInputs: [][]byte{fullLeaf, {byte(9)}},
			errs:       []error{nil, io.EOF},
		},
		{
			name:       "double",
			input:      append(fullLeaf, fullLeaf...),
			leafInputs: [][]byte{fullLeaf, fullLeaf},
			errs:       []error{nil, io.EOF},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			state := NewStateMatrix()
			in := bytes.NewReader(test.input)
			for i, leaf := range test.leafInputs {
				buf, err := state.absorbNextLeafInput(in)
				if errors.Is(err, io.EOF) {
					require.Equal(t, test.errs[i], err)
					break
				}
				require.NoError(t, err)
				require.Equal(t, leaf, buf)
			}
		})
	}
}

func FuzzKeccak(f *testing.F) {
	f.Fuzz(func(t *testing.T, number, time uint64, data []byte) {
		s := NewStateMatrix()
		for i := 0; i < len(data); i += types.BlockSize {
			end := min(i+types.BlockSize, len(data))
			s.absorbLeafInput(data[i:end], end == len(data))
		}
		actual := s.Hash()
		expected := crypto.Keccak256Hash(data)
		require.Equal(t, expected, actual)
	})
}
