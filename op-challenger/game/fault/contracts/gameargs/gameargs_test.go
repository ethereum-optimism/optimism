package gameargs

import (
	"encoding/binary"
	"math/big"
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

func TestZKGameArgsPack(t *testing.T) {
	prestate := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	verifier := common.HexToAddress("0x1111111111111111111111111111111111111111")
	asr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	weth := common.HexToAddress("0x3333333333333333333333333333333333333333")
	bond := big.NewInt(1e18)
	chainID := big.NewInt(42)

	got := ZKGameArgs{
		AbsolutePrestate:     prestate,
		Verifier:             verifier,
		MaxChallengeDuration: 3600,
		MaxProveDuration:     7200,
		ChallengerBond:       bond,
		AnchorStateRegistry:  asr,
		Weth:                 weth,
		L2ChainID:            chainID,
	}.Pack()

	require.Len(t, got, ZKArgsLength)
	require.Equal(t, prestate[:], got[0:32])
	require.Equal(t, verifier[:], got[32:52])
	require.Equal(t, uint64(3600), binary.BigEndian.Uint64(got[52:60]))
	require.Equal(t, uint64(7200), binary.BigEndian.Uint64(got[60:68]))
	require.Equal(t, bond, new(big.Int).SetBytes(got[68:100]))
	require.Equal(t, asr[:], got[100:120])
	require.Equal(t, weth[:], got[120:140])
	require.Equal(t, chainID, new(big.Int).SetBytes(got[140:172]))
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
