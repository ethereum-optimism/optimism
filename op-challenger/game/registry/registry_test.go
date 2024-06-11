package registry

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
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
	registry.RegisterGameType(0, creator)
	player, err := registry.CreatePlayer(types.GameMetadata{GameType: 0}, "")
	require.NoError(t, err)
	require.Same(t, expectedPlayer, player)
}

func TestPanicsOnDuplicateGameType(t *testing.T) {
	registry := NewGameTypeRegistry()
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return nil, nil
	}
	registry.RegisterGameType(0, creator)
	require.Panics(t, func() {
		registry.RegisterGameType(0, creator)
	})
}

func TestBondContracts(t *testing.T) {
	t.Run("UnknownGameType", func(t *testing.T) {
		registry := NewGameTypeRegistry()
		contract, err := registry.CreateBondContract(types.GameMetadata{GameType: 0})
		require.ErrorIs(t, err, ErrUnsupportedGameType)
		require.Nil(t, contract)
	})
	t.Run("KnownGameType", func(t *testing.T) {
		registry := NewGameTypeRegistry()
		expected := &stubBondContract{}
		registry.RegisterBondContract(0, func(game types.GameMetadata) (claims.BondContract, error) {
			return expected, nil
		})
		creator, err := registry.CreateBondContract(types.GameMetadata{GameType: 0})
		require.NoError(t, err)
		require.Same(t, expected, creator)
	})
	t.Run("PanicsOnDuplicate", func(t *testing.T) {
		registry := NewGameTypeRegistry()
		creator := func(game types.GameMetadata) (claims.BondContract, error) {
			return nil, nil
		}
		registry.RegisterBondContract(0, creator)
		require.Panics(t, func() {
			registry.RegisterBondContract(0, creator)
		})
	})
}

type stubBondContract struct{}

func (s *stubBondContract) GetCredit(_ context.Context, _ common.Address) (*big.Int, types.GameStatus, error) {
	panic("not supported")
}

func (s *stubBondContract) ClaimCreditTx(_ context.Context, _ common.Address) (txmgr.TxCandidate, error) {
	panic("not supported")
}
