package contracts

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodGetAnchorRoot  = "getAnchorRoot"
	methodSetAnchorState = "setAnchorState"
)

type AnchorStateRegistryContract struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewAnchorStateRegistryContract(metrics metrics.ContractMetricer, addr common.Address, caller *batching.MultiCaller) *AnchorStateRegistryContract {
	return &AnchorStateRegistryContract{
		metrics:     metrics,
		multiCaller: caller,
		contract:    batching.NewBoundContract(snapshots.LoadAnchorStateRegistryABI(), addr),
	}
}

func (a *AnchorStateRegistryContract) Addr() common.Address {
	return a.contract.Addr()
}

// GetAnchorRoot returns the current anchor state root and its L2 sequence number.
func (a *AnchorStateRegistryContract) GetAnchorRoot(ctx context.Context, block rpcblock.Block) (common.Hash, *big.Int, error) {
	defer a.metrics.StartContractRequest("GetAnchorRoot")()
	result, err := a.multiCaller.SingleCall(ctx, block, a.contract.Call(methodGetAnchorRoot))
	if err != nil {
		return common.Hash{}, nil, fmt.Errorf("failed to retrieve anchor root: %w", err)
	}
	return result.GetHash(0), result.GetBigInt(1), nil
}

func (a *AnchorStateRegistryContract) SetAnchorStateTx(ctx context.Context, game common.Address) (txmgr.TxCandidate, error) {
	defer a.metrics.StartContractRequest("SetAnchorState")()
	call := a.contract.Call(methodSetAnchorState, game)
	if _, err := a.multiCaller.SingleCall(ctx, rpcblock.Latest, call); err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("%w: %w", ErrSimulationFailed, err)
	}
	return call.ToTxCandidate()
}
