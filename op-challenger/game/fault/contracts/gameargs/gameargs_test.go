package gameargs

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("Invalid-ZeroLength", func(t *testing.T) {
		_, err := Parse([]byte{})
		require.ErrorIs(t, err, ErrInvalidGameArgs)
	})

	t.Run("Invalid-TooLong", func(t *testing.T) {
		input := make([]byte, PermissionedArgsLength+1)
		_, err := Parse(input)
		require.ErrorIs(t, err, ErrInvalidGameArgs)
	})

	t.Run("Invalid-TooShort", func(t *testing.T) {
		input := make([]byte, PermissionlessArgsLength-1)
		_, err := Parse(input)
		require.ErrorIs(t, err, ErrInvalidGameArgs)
	})

	t.Run("Invalid-BetweenValidLengths", func(t *testing.T) {
		input := make([]byte, PermissionlessArgsLength+1)
		_, err := Parse(input)
		require.ErrorIs(t, err, ErrInvalidGameArgs)
	})

	t.Run("Valid-Permissionless", func(t *testing.T) {
		expected := fullGameArgs()
		expected.Proposer = common.Address{}
		expected.Challenger = common.Address{}
		input := expected.PackPermissionless()
		actual, err := Parse(input)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("Valid-Permissioned", func(t *testing.T) {
		expected := fullGameArgs()
		input := expected.PackPermissioned()
		actual, err := Parse(input)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func fullGameArgs() GameArgs {
	rng := rand.New(rand.NewSource(0))
	return GameArgs{
		AbsolutePrestate:    testutils.RandomHash(rng),
		Vm:                  testutils.RandomAddress(rng),
		AnchorStateRegistry: testutils.RandomAddress(rng),
		Weth:                testutils.RandomAddress(rng),
		L2ChainID:           eth.ChainIDFromBytes32(testutils.RandomHash(rng)),
		Proposer:            testutils.RandomAddress(rng),
		Challenger:          testutils.RandomAddress(rng),
	}
}
