package mon

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
)

type statusLoader struct {
	caller *batching.MultiCaller
}

func NewStatusLoader(caller *batching.MultiCaller) *statusLoader {
	return &statusLoader{caller: caller}
}

func (s *statusLoader) GetStatus(ctx context.Context, proxy common.Address) (types.GameStatus, error) {
	fdg, err := contracts.NewFaultDisputeGameContract(proxy, s.caller)
	if err != nil {
		return 0, fmt.Errorf("failed to create FaultDisputeGameContract: %w", err)
	}
	return fdg.GetStatus(ctx)
}
