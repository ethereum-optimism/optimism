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

const metricsLabel = "status_loader"

type statusLoader struct {
	cache  *caching.LRUCache[common.Address, *contracts.FaultDisputeGameContract]
	caller *batching.MultiCaller
}

func NewStatusLoader(m caching.Metrics, caller *batching.MultiCaller) *statusLoader {
	return &statusLoader{
		caller: caller,
		cache:  caching.NewLRUCache[common.Address, *contracts.FaultDisputeGameContract](m, metricsLabel, 100),
	}
}

func (s *statusLoader) GetStatus(ctx context.Context, proxy common.Address) (types.GameStatus, error) {
	if fdg, ok := s.cache.Get(proxy); ok {
		return fdg.GetStatus(ctx)
	}
	fdg, err := contracts.NewFaultDisputeGameContract(proxy, s.caller)
	if err != nil {
		return 0, fmt.Errorf("failed to create FaultDisputeGameContract: %w", err)
	}
	s.cache.Add(proxy, fdg)
	return fdg.GetStatus(ctx)
}

func (s *statusLoader) GetRootClaim(ctx context.Context, proxy common.Address) (common.Hash, error) {
	if fdg, ok := s.cache.Get(proxy); ok {
		return fdg.GetRootClaim(ctx)
	}
	fdg, err := contracts.NewFaultDisputeGameContract(proxy, s.caller)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create FaultDisputeGameContract: %w", err)
	}
	s.cache.Add(proxy, fdg)
	return fdg.GetRootClaim(ctx)
}

func (s *statusLoader) GetL2BlockNumber(ctx context.Context, proxy common.Address) (uint64, error) {
	if fdg, ok := s.cache.Get(proxy); ok {
		return fdg.GetL2BlockNumber(ctx)
	}
	fdg, err := contracts.NewFaultDisputeGameContract(proxy, s.caller)
	if err != nil {
		return 0, fmt.Errorf("failed to create FaultDisputeGameContract: %w", err)
	}
	s.cache.Add(proxy, fdg)
	return fdg.GetL2BlockNumber(ctx)
}
