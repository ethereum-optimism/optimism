package mon

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
)

const metricsLabel = "metadata_loader"

type metadataLoader struct {
	cache  *caching.LRUCache[common.Address, *contracts.FaultDisputeGameContract]
	caller *batching.MultiCaller
}

func NewMetadataLoader(m caching.Metrics, caller *batching.MultiCaller) *metadataLoader {
	return &metadataLoader{
		caller: caller,
		cache:  caching.NewLRUCache[common.Address, *contracts.FaultDisputeGameContract](m, metricsLabel, 100),
	}
}

func (s *metadataLoader) GetGameMetadata(ctx context.Context, proxy common.Address) (uint64, common.Hash, types.GameStatus, error) {
	if fdg, ok := s.cache.Get(proxy); ok {
		return fdg.GetGameMetadata(ctx)
	}
	fdg, err := contracts.NewFaultDisputeGameContract(proxy, s.caller)
	if err != nil {
		return 0, common.Hash{}, 0, fmt.Errorf("failed to create FaultDisputeGameContract: %w", err)
	}
	s.cache.Add(proxy, fdg)
	return fdg.GetGameMetadata(ctx)
}
