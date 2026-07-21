package fault

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/stretchr/testify/require"
)

type recordingRegistry struct {
	players   map[gameTypes.GameType]scheduler.PlayerCreator
	bonds     map[gameTypes.GameType]claims.BondContractCreator
	bondCalls []gameTypes.GameType
}

func newRecordingRegistry() *recordingRegistry {
	return &recordingRegistry{
		players: make(map[gameTypes.GameType]scheduler.PlayerCreator),
		bonds:   make(map[gameTypes.GameType]claims.BondContractCreator),
	}
}

func (r *recordingRegistry) RegisterGameType(gameType gameTypes.GameType, creator scheduler.PlayerCreator) {
	r.players[gameType] = creator
}

func (r *recordingRegistry) RegisterBondContract(gameType gameTypes.GameType, creator claims.BondContractCreator) {
	r.bondCalls = append(r.bondCalls, gameType)
	r.bonds[gameType] = creator
}

func registeredGameTypes(creators map[gameTypes.GameType]claims.BondContractCreator) []gameTypes.GameType {
	gameTypes := make([]gameTypes.GameType, 0, len(creators))
	for gameType := range creators {
		gameTypes = append(gameTypes, gameType)
	}
	return gameTypes
}

func TestRegisterBondContracts_RegistersAllFaultLifecycleTypesWithoutPlayers(t *testing.T) {
	registry := newRecordingRegistry()

	RegisterBondContracts(context.Background(), metrics.NoopMetrics, registry, nil)

	expected := []gameTypes.GameType{
		gameTypes.AlphabetGameType,
		gameTypes.CannonGameType,
		gameTypes.CannonKonaGameType,
		gameTypes.PermissionedGameType,
		gameTypes.SuperPermissionedGameType,
		gameTypes.FastGameType,
		gameTypes.SuperCannonKonaGameType,
	}
	require.Len(t, registry.bonds, len(expected))
	require.ElementsMatch(t, expected, registeredGameTypes(registry.bonds))
	require.Len(t, registry.bondCalls, len(expected))
	require.ElementsMatch(t, expected, registry.bondCalls)
	require.Empty(t, registry.players)
	require.NotNil(t, registry.bonds[gameTypes.SuperPermissionedGameType])
}
