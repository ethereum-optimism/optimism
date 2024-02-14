package mon

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
)

const metricsLabel = "binding_creator"

type MetadataLoader interface {
	GetGameMetadata(context.Context) (uint64, common.Hash, types.GameStatus, error)
}

type metadataCreator struct {
	cache  *caching.LRUCache[common.Address, *contracts.FaultDisputeGameContract]
	caller *batching.MultiCaller
}

func NewMetadataCreator(m caching.Metrics, caller *batching.MultiCaller) *metadataCreator {
	return &metadataCreator{
		caller: caller,
		cache:  caching.NewLRUCache[common.Address, *contracts.FaultDisputeGameContract](m, metricsLabel, 100),
	}
}

func (m *metadataCreator) CreateContract(game types.GameMetadata) (MetadataLoader, error) {
	if fdg, ok := m.cache.Get(game.Proxy); ok {
		return fdg, nil
	}
	switch game.GameType {
	case faultTypes.CannonGameType, faultTypes.AlphabetGameType:
		fdg, err := contracts.NewFaultDisputeGameContract(game.Proxy, m.caller)
		if err != nil {
			return nil, fmt.Errorf("failed to create FaultDisputeGameContract: %w", err)
		}
		m.cache.Add(game.Proxy, fdg)
		return fdg, nil
	default:
		return nil, fmt.Errorf("unsupported game type: %d", game.GameType)
	}
}
