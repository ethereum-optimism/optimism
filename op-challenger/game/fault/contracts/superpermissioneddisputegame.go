package contracts

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

var ErrClaimCreditNotSupported = errors.New("contract does not support claiming credit")

const (
	methodSuperPermissionedAnchorStateRegistry = "anchorStateRegistry"
	methodSuperPermissionedL2SequenceNumber    = "l2SequenceNumber"
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

func (g *SuperPermissionedDisputeGameContract) HasBondsToClaim() bool {
	return false
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
	asrAddr, err := g.anchorStateRegistry(ctx)
	if err != nil {
		return false, err
	}
	gameSequence, err := g.l2SequenceNumber(ctx)
	if err != nil {
		return false, err
	}
	asr := NewAnchorStateRegistryContract(g.metrics, asrAddr, g.multiCaller)
	_, anchorSequence, err := asr.GetAnchorRoot(ctx, rpcblock.Latest)
	if err != nil {
		return false, err
	}
	return anchorSequence.Cmp(gameSequence) >= 0, nil
}

func (g *SuperPermissionedDisputeGameContract) CloseGameTx(ctx context.Context) (txmgr.TxCandidate, error) {
	asrAddr, err := g.anchorStateRegistry(ctx)
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	asr := NewAnchorStateRegistryContract(g.metrics, asrAddr, g.multiCaller)
	return asr.SetAnchorStateTx(ctx, g.contract.Addr())
}

func (g *SuperPermissionedDisputeGameContract) anchorStateRegistry(ctx context.Context) (common.Address, error) {
	defer g.metrics.StartContractRequest("GetAnchorStateRegistry")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodSuperPermissionedAnchorStateRegistry))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to retrieve anchor state registry: %w", err)
	}
	return result.GetAddress(0), nil
}

func (g *SuperPermissionedDisputeGameContract) l2SequenceNumber(ctx context.Context) (*big.Int, error) {
	defer g.metrics.StartContractRequest("GetL2SequenceNumber")()
	result, err := g.multiCaller.SingleCall(ctx, rpcblock.Latest, g.contract.Call(methodSuperPermissionedL2SequenceNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve L2 sequence number: %w", err)
	}
	return result.GetBigInt(0), nil
}
