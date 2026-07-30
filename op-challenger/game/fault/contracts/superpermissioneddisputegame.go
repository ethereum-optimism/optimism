package contracts

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

var ErrClaimCreditNotSupported = errors.New("contract does not support claiming credit")

const (
	methodSuperPermissionedAnchorStateRegistry = "anchorStateRegistry"
	methodSuperPermissionedL1Head              = "l1Head"
	methodSuperPermissionedL2SequenceNumber    = "l2SequenceNumber"
	methodSuperPermissionedRootClaim           = "rootClaim"
	methodSuperPermissionedStatus              = "status"
)

type SuperPermissionedDisputeGameContract struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewSuperPermissionedDisputeGameContract(
	m metrics.ContractMetricer,
	addr common.Address,
	caller *batching.MultiCaller,
) *SuperPermissionedDisputeGameContract {
	return &SuperPermissionedDisputeGameContract{
		metrics:     m,
		multiCaller: caller,
		contract: batching.NewBoundContract(
			snapshots.LoadSuperPermissionedDisputeGameABI(), addr),
	}
}

func (g *SuperPermissionedDisputeGameContract) GetExtendedMetadata(
	ctx context.Context,
	block rpcblock.Block,
) (GameMetadata, error) {
	defer g.metrics.StartContractRequest("GetExtendedMetadata")()
	results, err := g.multiCaller.Call(
		ctx,
		block,
		g.contract.Call(methodSuperPermissionedL1Head),
		g.contract.Call(methodSuperPermissionedL2SequenceNumber),
		g.contract.Call(methodSuperPermissionedRootClaim),
		g.contract.Call(methodSuperPermissionedStatus),
	)
	if err != nil {
		return GameMetadata{}, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	if len(results) != 4 {
		return GameMetadata{}, fmt.Errorf("expected 4 results but got %v", len(results))
	}
	l2SequenceNumber := results[1].GetBigInt(0)
	l2SequenceNumberUint64 := uint64(math.MaxUint64)
	if l2SequenceNumber.IsUint64() {
		l2SequenceNumberUint64 = bigs.Uint64Strict(l2SequenceNumber)
	}
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return GameMetadata{}, fmt.Errorf("failed to convert game status: %w", err)
	}
	return GameMetadata{
		L1Head:        results[0].GetHash(0),
		L2SequenceNum: l2SequenceNumberUint64,
		RootClaim:     results[2].GetHash(0),
		Status:        status,
	}, nil
}

func (g *SuperPermissionedDisputeGameContract) GetAnchorStateRegistry(
	ctx context.Context,
	block rpcblock.Block,
) (common.Address, error) {
	defer g.metrics.StartContractRequest("GetAnchorStateRegistry")()
	result, err := g.multiCaller.SingleCall(ctx, block, g.contract.Call(methodSuperPermissionedAnchorStateRegistry))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to retrieve anchor state registry: %w", err)
	}
	return result.GetAddress(0), nil
}

func (g *SuperPermissionedDisputeGameContract) GetCredit(
	context.Context,
	common.Address,
) (*big.Int, gameTypes.GameStatus, error) {
	return big.NewInt(0), gameTypes.GameStatusDefenderWon, nil
}

func (g *SuperPermissionedDisputeGameContract) ClaimCreditTx(
	context.Context,
	common.Address,
) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, ErrClaimCreditNotSupported
}

func (g *SuperPermissionedDisputeGameContract) IsClosed(ctx context.Context) (bool, error) {
	defer g.metrics.StartContractRequest("IsClosed")()
	results, err := g.multiCaller.Call(
		ctx,
		rpcblock.Latest,
		g.contract.Call(methodSuperPermissionedAnchorStateRegistry),
		g.contract.Call(methodSuperPermissionedL2SequenceNumber),
	)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve game data: %w", err)
	}
	if len(results) != 2 {
		return false, fmt.Errorf("expected 2 results but got %v", len(results))
	}
	asr := NewAnchorStateRegistryContract(g.metrics, results[0].GetAddress(0), g.multiCaller)
	_, anchorSequence, err := asr.GetAnchorRoot(ctx, rpcblock.Latest)
	if err != nil {
		return false, err
	}
	return anchorSequence.Cmp(results[1].GetBigInt(0)) >= 0, nil
}

func (g *SuperPermissionedDisputeGameContract) CloseGameTx(ctx context.Context) (txmgr.TxCandidate, error) {
	asr, err := g.anchorStateRegistry(ctx)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	return asr.SetAnchorStateTx(ctx, g.contract.Addr())
}

func (g *SuperPermissionedDisputeGameContract) anchorStateRegistry(ctx context.Context) (*AnchorStateRegistryContract, error) {
	addr, err := g.GetAnchorStateRegistry(ctx, rpcblock.Latest)
	if err != nil {
		return nil, err
	}
	return NewAnchorStateRegistryContract(g.metrics, addr, g.multiCaller), nil
}
