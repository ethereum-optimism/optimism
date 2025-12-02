package batching

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestDecodeCall(t *testing.T) {
	method := "approve"
	spender := common.Address{0xbb, 0xee}
	amount := big.NewInt(4242)
	testAbi, err := test.ERC20MetaData.GetAbi()
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

func TestDecodeEvent(t *testing.T) {
	testAbi, err := test.ERC20MetaData.GetAbi()
	require.NoError(t, err)

	// event Transfer(address indexed from, address indexed to, uint256 amount);
	event := testAbi.Events["Transfer"]

	contract := NewBoundContract(testAbi, common.Address{0xaa})
	t.Run("NoTopics", func(t *testing.T) {
		log := &types.Log{}
		_, _, err := contract.DecodeEvent(log)
		require.ErrorIs(t, err, ErrUnknownEvent)
	})

	t.Run("UnknownEvent", func(t *testing.T) {
		log := &types.Log{
			Topics: []common.Hash{{0xaa}},
		}
		_, _, err := contract.DecodeEvent(log)
		require.ErrorIs(t, err, ErrUnknownEvent)
	})

	t.Run("InvalidTopics", func(t *testing.T) {
		amount := big.NewInt(828274)
		data, err := event.Inputs.NonIndexed().Pack(amount)
		require.NoError(t, err)
		log := &types.Log{
			Topics: []common.Hash{
				event.ID,
				common.BytesToHash(common.Address{0xaa}.Bytes()),
				// Missing topic for to indexed value
			},
			Data: data,
		}
		_, _, err = contract.DecodeEvent(log)
		require.ErrorIs(t, err, ErrInvalidEvent)
	})

	t.Run("MissingData", func(t *testing.T) {
		log := &types.Log{
			Topics: []common.Hash{
				event.ID,
				common.BytesToHash(common.Address{0xaa}.Bytes()),
				common.BytesToHash(common.Address{0xbb}.Bytes()),
			},
		}
		_, _, err := contract.DecodeEvent(log)
		require.ErrorIs(t, err, ErrInvalidEvent)
	})

	t.Run("InvalidData", func(t *testing.T) {
		log := &types.Log{
			Topics: []common.Hash{
				event.ID,
				common.BytesToHash(common.Address{0xaa}.Bytes()),
				common.BytesToHash(common.Address{0xbb}.Bytes()),
			},
			Data: []byte{0xbb, 0xcc},
		}
		_, _, err := contract.DecodeEvent(log)
		require.ErrorIs(t, err, ErrInvalidEvent)
	})

	t.Run("ValidEvent", func(t *testing.T) {
		amount := big.NewInt(828274)
		data, err := event.Inputs.NonIndexed().Pack(amount)
		require.NoError(t, err)
		log := &types.Log{
			Topics: []common.Hash{
				event.ID,
				common.BytesToHash(common.Address{0xaa}.Bytes()),
				common.BytesToHash(common.Address{0xbb}.Bytes()),
			},
			Data: data,
		}
		name, result, err := contract.DecodeEvent(log)
		require.NoError(t, err)
		require.Equal(t, name, event.Name)
		require.Equal(t, common.Address{0xaa}, result.GetAddress(0))
		require.Equal(t, common.Address{0xbb}, result.GetAddress(1))
		require.Zerof(t, amount.Cmp(result.GetBigInt(2)), "expected %v but got %v", amount, result.GetBigInt(2))
	})
}
