package forge

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type nestedTupleProposal struct {
	Root             common.Hash
	L2SequenceNumber *big.Int
}

type nestedTupleInput struct {
	StartingAnchorRoot nestedTupleProposal
}

func TestGoStructToABITupleNestedTuple(t *testing.T) {
	tupleType, err := GoStructToABITuple(reflect.TypeFor[nestedTupleInput](), "nestedTupleInput")
	require.NoError(t, err)
	require.Len(t, tupleType.TupleElems, 1)
	require.Equal(t, []string{"StartingAnchorRoot"}, tupleType.TupleRawNames)
	require.Equal(t, "(bytes32,uint256)", tupleType.TupleElems[0].String())
	require.Equal(t, []string{"Root", "L2SequenceNumber"}, tupleType.TupleElems[0].TupleRawNames)

	args := abi.Arguments{{Type: tupleType}}
	tests := []struct {
		name     string
		proposal nestedTupleProposal
	}{
		{
			name: "zero sequence",
			proposal: nestedTupleProposal{
				Root:             common.Hash{0xde, 0xad},
				L2SequenceNumber: new(big.Int),
			},
		},
		{
			name: "non-zero sequence",
			proposal: nestedTupleProposal{
				Root:             common.HexToHash("0x02f4397b2de6fce03b3f9982378c2b4c4deff9c92c662dcc6f9643267aeb5e47"),
				L2SequenceNumber: big.NewInt(1234),
			},
		},
		{
			name: "maximum sequence",
			proposal: nestedTupleProposal{
				Root: common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
				L2SequenceNumber: new(big.Int).Sub(
					new(big.Int).Lsh(big.NewInt(1), 256),
					big.NewInt(1),
				),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := nestedTupleInput{StartingAnchorRoot: test.proposal}
			expected, err := args.Pack(input)
			require.NoError(t, err)
			encoded, err := (&BytesScriptEncoder[nestedTupleInput]{TypeName: "nestedTupleInput"}).Encode(input)
			require.NoError(t, err)
			require.Equal(t, expected, encoded)

			unpacked, err := args.Unpack(encoded)
			require.NoError(t, err)
			require.Len(t, unpacked, 1)

			actual := *abi.ConvertType(unpacked[0], new(nestedTupleInput)).(*nestedTupleInput)
			require.Equal(t, input.StartingAnchorRoot.Root, actual.StartingAnchorRoot.Root)
			require.Zero(t, input.StartingAnchorRoot.L2SequenceNumber.Cmp(actual.StartingAnchorRoot.L2SequenceNumber))
		})
	}
}
