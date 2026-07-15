package bindings

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

func TestAnchorStateRegistryAnchorGameBinding(t *testing.T) {
	expectedAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")

	asr := NewBindings[AnchorStateRegistry]()
	call := asr.AnchorGame()
	calldata, err := call.EncodeInput()
	require.NoError(t, err)
	require.Equal(t, crypto.Keccak256([]byte("anchorGame()"))[:4], calldata)

	result, err := call.DecodeOutput(common.LeftPadBytes(expectedAddress.Bytes(), 32))
	require.NoError(t, err)
	require.Equal(t, expectedAddress, result)
}
