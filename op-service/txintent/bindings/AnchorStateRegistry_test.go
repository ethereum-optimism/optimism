package bindings

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAnchorStateRegistryGetAnchorRootDecodesFullWidthSequence(t *testing.T) {
	expectedRoot := common.HexToHash("0x1234")
	expectedSequence := new(big.Int).Lsh(big.NewInt(1), 200)
	encodedResult := append(expectedRoot.Bytes(), common.LeftPadBytes(expectedSequence.Bytes(), 32)...)

	asr := NewBindings[AnchorStateRegistry]()
	call := asr.GetAnchorRoot()
	result, err := call.DecodeOutput(encodedResult)
	require.NoError(t, err)
	require.Equal(t, expectedRoot, result.Root)
	require.Zero(t, expectedSequence.Cmp(result.L2SequenceNumber))
}
