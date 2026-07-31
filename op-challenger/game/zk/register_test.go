package zk

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/stretchr/testify/require"
)

type recordingRegistry struct {
	playerCreators map[gameTypes.GameType]scheduler.PlayerCreator
	bondCreators   map[gameTypes.GameType]claims.BondContractCreator
	bondCalls      []gameTypes.GameType
}

func (r *recordingRegistry) RegisterGameType(gameType gameTypes.GameType, creator scheduler.PlayerCreator) {
	r.playerCreators[gameType] = creator
}

func (r *recordingRegistry) RegisterBondContract(gameType gameTypes.GameType, creator claims.BondContractCreator) {
	r.bondCalls = append(r.bondCalls, gameType)
	r.bondCreators[gameType] = creator
}

func TestRegisterBondContracts(t *testing.T) {
	registry := &recordingRegistry{
		playerCreators: make(map[gameTypes.GameType]scheduler.PlayerCreator),
		bondCreators:   make(map[gameTypes.GameType]claims.BondContractCreator),
	}
	var caller *batching.MultiCaller

	RegisterBondContracts(metrics.NoopMetrics, registry, caller)

	require.Empty(t, registry.playerCreators)
	require.Len(t, registry.bondCreators, 1)
	require.Len(t, registry.bondCalls, 1)
	require.ElementsMatch(t, []gameTypes.GameType{gameTypes.ZKDisputeGameType}, registry.bondCalls)
	creator, ok := registry.bondCreators[gameTypes.ZKDisputeGameType]
	require.True(t, ok)
	require.NotNil(t, creator)
}
