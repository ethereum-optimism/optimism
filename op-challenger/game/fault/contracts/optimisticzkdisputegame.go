package contracts

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

type OptimisticZKDisputeGameContract interface {
	DisputeGameContract
}

type OptimisticZKDisputeGameContractLatest struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewOptimisticZKDisputeGameContract(
	m metrics.ContractMetricer,
	addr common.Address,
	caller *batching.MultiCaller,
) (*OptimisticZKDisputeGameContractLatest, error) {
	abi := snapshots.LoadZKDisputeGameABI()
	return &OptimisticZKDisputeGameContractLatest{
		metrics:     m,
		multiCaller: caller,
		contract:    batching.NewBoundContract(abi, addr),
	}, nil
}

// GetMetadata returns the basic game metadata
func (g *OptimisticZKDisputeGameContractLatest) GetMetadata(ctx context.Context, block rpcblock.Block) (GenericGameMetadata, error) {
	defer g.metrics.StartContractRequest("GetMetadata")()
	results, err := g.multiCaller.Call(ctx, block,
		g.contract.Call(methodL1Head),
		g.contract.Call(methodL2SequenceNumber),
		g.contract.Call(methodRootClaim),
		g.contract.Call(methodStatus),
	)
	if err != nil {
		return GenericGameMetadata{}, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	if len(results) != 4 {
		return GenericGameMetadata{}, fmt.Errorf("expected 4 results but got %v", len(results))
	}
	l1Head := results[0].GetHash(0)
	l2SequenceNumber := results[1].GetBigInt(0).Uint64()
	rootClaim := results[2].GetHash(0)
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return GenericGameMetadata{}, fmt.Errorf("failed to convert game status: %w", err)
	}
	return GenericGameMetadata{
		L1Head:        l1Head,
		L2SequenceNum: l2SequenceNumber,
		ProposedRoot:  rootClaim,
		Status:        status,
	}, nil
}

func (g *OptimisticZKDisputeGameContractLatest) GetL1Head(ctx context.Context) (common.Hash, error) {
	defer g.metrics.StartContractRequest("GetL1Head")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodL1Head))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch L1 head: %w", err)
	}
	return result.GetHash(0), nil
}

func (g *OptimisticZKDisputeGameContractLatest) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	defer g.metrics.StartContractRequest("GetStatus")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodStatus))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (g *OptimisticZKDisputeGameContractLatest) GetGameRange(ctx context.Context) (prestateBlock uint64, poststateBlock uint64, retErr error) {
	defer g.metrics.StartContractRequest("GetGameRange")()
	results, err := g.multiCaller.Call(ctx, rpcblock.Latest,
		g.contract.Call(methodStartingBlockNumber),
		g.contract.Call(methodL2SequenceNumber))
	if err != nil {
		retErr = fmt.Errorf("failed to retrieve game block range: %w", err)
		return
	}
	if len(results) != 2 {
		retErr = fmt.Errorf("expected 2 results but got %v", len(results))
		return
	}
	prestateBlock = results[0].GetBigInt(0).Uint64()
	poststateBlock = results[1].GetBigInt(0).Uint64()
	return
}

func (g *OptimisticZKDisputeGameContractLatest) GetResolvedAt(ctx context.Context, block rpcblock.Block) (time.Time, error) {
	defer g.metrics.StartContractRequest("GetResolvedAt")()
	result, err := g.multiCaller.SingleCall(ctx, block, g.contract.Call(methodResolvedAt))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to retrieve resolution time: %w", err)
	}
	resolvedAt := time.Unix(int64(result.GetUint64(0)), 0)
	return resolvedAt, nil
}

func (g *OptimisticZKDisputeGameContractLatest) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	defer g.metrics.StartContractRequest("CallResolve")()
	call := g.resolveCall()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return gameTypes.GameStatusInProgress, fmt.Errorf("failed to call resolve: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (g *OptimisticZKDisputeGameContractLatest) ResolveTx() (txmgr.TxCandidate, error) {
	call := g.resolveCall()
	return call.ToTxCandidate()
}

func (g *OptimisticZKDisputeGameContractLatest) resolveCall() *batching.ContractCall {
	return g.contract.Call(methodResolve)
}

var _ DisputeGameContract = (*OptimisticZKDisputeGameContractLatest)(nil)
