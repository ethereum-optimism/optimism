package contracts

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

const methodRespectedGameType = "respectedGameType"

type AnchorStateRegistry struct {
	caller         *batching.MultiCaller
	contract       *batching.BoundContract
	networkTimeout time.Duration
}

func NewAnchorStateRegistry(addr common.Address, caller *batching.MultiCaller, networkTimeout time.Duration) *AnchorStateRegistry {
	return &AnchorStateRegistry{
		caller:         caller,
		contract:       batching.NewBoundContract(snapshots.LoadAnchorStateRegistryABI(), addr),
		networkTimeout: networkTimeout,
	}
}

func (a *AnchorStateRegistry) RespectedGameType(ctx context.Context) (uint32, error) {
	cCtx, cancel := context.WithTimeout(ctx, a.networkTimeout)
	defer cancel()
	result, err := a.caller.SingleCall(cCtx, rpcblock.Latest, a.contract.Call(methodRespectedGameType))
	if err != nil {
		return 0, fmt.Errorf("failed to get respected game type: %w", err)
	}
	return result.GetUint32(0), nil
}
