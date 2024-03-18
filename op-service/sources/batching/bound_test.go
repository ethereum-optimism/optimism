package batching

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDecodeCall(t *testing.T) {
	method := "approve"
	spender := common.Address{0xbb, 0xee}
	amount := big.NewInt(4242)
	testAbi, err := bindings.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	validData, err := testAbi.Pack(method, spender, amount)
	require.NoError(t, err)

	contract := NewBoundContract(testAbi, common.Address{0xaa})
	t.Run("TooShort", func(t *testing.T) {
		_, _, err := contract.DecodeCall([]byte{1, 2, 3})
		require.ErrorIs(t, err, ErrUnknownMethod)
	})

	t.Run("UnknownMethodId", func(t *testing.T) {
		_, _, err := contract.DecodeCall([]byte{1, 2, 3, 4})
		require.ErrorIs(t, err, ErrUnknownMethod)
	})

	t.Run("MissingArgs", func(t *testing.T) {
		// Truncate to just the 4 byte method selector
		_, _, err = contract.DecodeCall(validData[:4])
		require.ErrorIs(t, err, ErrInvalidCall)

		// Truncate to partial args
		_, _, err = contract.DecodeCall(validData[:6])
		require.ErrorIs(t, err, ErrInvalidCall)

		// Truncate to first arg but missing second
		_, _, err = contract.DecodeCall(validData[:24])
		require.ErrorIs(t, err, ErrInvalidCall)
	})

	t.Run("ValidCall", func(t *testing.T) {
		name, args, err := contract.DecodeCall(validData)
		require.NoError(t, err)
		require.Equal(t, name, method)
		require.Equal(t, spender, args.GetAddress(0))
		require.Zero(t, amount.Cmp(args.GetBigInt(1)))
	})
}
