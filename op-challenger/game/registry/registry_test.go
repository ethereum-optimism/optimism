package registry

import (
	"context"
	"math/big"
	"testing"

	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUnknownGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	player, err := registry.CreatePlayer(types.GameMetadata{GameType: 0}, "")
	require.ErrorIs(t, err, ErrUnsupportedGameType)
	require.Nil(t, player)
}

func TestKnownGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	expectedPlayer := &test.StubGamePlayer{}
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return expectedPlayer, nil
	}
	registry.RegisterGameType(0, creator, nil)
	player, err := registry.CreatePlayer(types.GameMetadata{GameType: 0}, "")
	require.NoError(t, err)
	require.Same(t, expectedPlayer, player)
}

func TestPanicsOnDuplicateGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return nil, nil
	}
	registry.RegisterGameType(0, creator, nil)
	require.Panics(t, func() {
		registry.RegisterGameType(0, creator, nil)
	})
}

func TestDeduplicateOracles(t *testing.T) {
	registry := NewGameTypeRegistry()
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return nil, nil
	}
	oracleA := stubPreimageOracle{0xaa}
	oracleB := stubPreimageOracle{0xbb}
	registry.RegisterGameType(0, creator, oracleA)
	registry.RegisterGameType(1, creator, oracleB)
	registry.RegisterGameType(2, creator, oracleB)
	oracles := registry.Oracles()
	require.Len(t, oracles, 2)
	require.Contains(t, oracles, oracleA)
	require.Contains(t, oracles, oracleB)
}

type stubPreimageOracle common.Address

func (s stubPreimageOracle) GetInputDataBlocks(_ context.Context, _ batching.Block, _ keccakTypes.LargePreimageIdent) ([]uint64, error) {
	panic("not supported")
}

func (s stubPreimageOracle) DecodeInputData(_ []byte) (*big.Int, keccakTypes.InputData, error) {
	panic("not supported")
}

func (s stubPreimageOracle) Addr() common.Address {
	return common.Address(s)
}

func (s stubPreimageOracle) GetActivePreimages(_ context.Context, _ common.Hash) ([]keccakTypes.LargePreimageMetaData, error) {
	return nil, nil
}
