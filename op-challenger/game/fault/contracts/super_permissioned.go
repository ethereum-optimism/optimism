package contracts

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

type SuperPermissionedDisputeGameContract struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

var _ DisputeGameContract = (*SuperPermissionedDisputeGameContract)(nil)

func NewSuperPermissionedDisputeGameContract(metrics metrics.ContractMetricer, addr common.Address, caller *batching.MultiCaller) *SuperPermissionedDisputeGameContract {
	return &SuperPermissionedDisputeGameContract{
		metrics:     metrics,
		multiCaller: caller,
		contract:    batching.NewBoundContract(snapshots.LoadSuperPermissionedDisputeGameABI(), addr),
	}
}

func (s *SuperPermissionedDisputeGameContract) Addr() common.Address {
	return s.contract.Addr()
}

func (s *SuperPermissionedDisputeGameContract) GetL1Head(ctx context.Context) (common.Hash, error) {
	defer s.metrics.StartContractRequest("GetL1Head")()
	result, err := s.multiCaller.SingleCall(ctx, rpcblock.Latest, s.contract.Call(methodL1Head))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to retrieve l1 head: %w", err)
	}
	return result.GetHash(0), nil
}

func (s *SuperPermissionedDisputeGameContract) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	defer s.metrics.StartContractRequest("GetStatus")()
	result, err := s.multiCaller.SingleCall(ctx, rpcblock.Latest, s.contract.Call(methodStatus))
	if err != nil {
		return gameTypes.GameStatusInProgress, fmt.Errorf("failed to retrieve game status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (s *SuperPermissionedDisputeGameContract) GetRootClaim(ctx context.Context) (common.Hash, error) {
	defer s.metrics.StartContractRequest("GetRootClaim")()
	result, err := s.multiCaller.SingleCall(ctx, rpcblock.Latest, s.contract.Call(methodRootClaim))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to retrieve root claim: %w", err)
	}
	return result.GetHash(0), nil
}

func (s *SuperPermissionedDisputeGameContract) GetL2SequenceNumber(ctx context.Context) (uint64, error) {
	defer s.metrics.StartContractRequest("GetL2SequenceNumber")()
	result, err := s.multiCaller.SingleCall(ctx, rpcblock.Latest, s.contract.Call(methodL2SequenceNumber))
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve l2 sequence number: %w", err)
	}
	return bigs.Uint64Strict(result.GetBigInt(0)), nil
}

func (s *SuperPermissionedDisputeGameContract) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	defer s.metrics.StartContractRequest("CallResolve")()
	call := s.resolveCall()
	result, err := s.multiCaller.SingleCall(ctx, rpcblock.Latest, call)
	if err != nil {
		return gameTypes.GameStatusInProgress, fmt.Errorf("failed to call resolve: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (s *SuperPermissionedDisputeGameContract) ResolveTx() (txmgr.TxCandidate, error) {
	return s.resolveCall().ToTxCandidate()
}

func (s *SuperPermissionedDisputeGameContract) resolveCall() *batching.ContractCall {
	return s.contract.Call(methodResolve)
}

func (s *SuperPermissionedDisputeGameContract) GetResolvedAt(ctx context.Context, block rpcblock.Block) (time.Time, error) {
	defer s.metrics.StartContractRequest("GetResolvedAt")()
	result, err := s.multiCaller.SingleCall(ctx, block, s.contract.Call(methodResolvedAt))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to retrieve resolution time: %w", err)
	}
	return time.Unix(int64(result.GetUint64(0)), 0), nil
}

func (s *SuperPermissionedDisputeGameContract) GetGameRange(context.Context) (uint64, uint64, error) {
	return 0, 0, ErrUnsupportedGameType
}

func (s *SuperPermissionedDisputeGameContract) GetMetadata(ctx context.Context, block rpcblock.Block) (GenericGameMetadata, error) {
	defer s.metrics.StartContractRequest("GetMetadata")()
	results, err := s.multiCaller.Call(ctx, block,
		s.contract.Call(methodL1Head),
		s.contract.Call(methodL2SequenceNumber),
		s.contract.Call(methodRootClaim),
		s.contract.Call(methodStatus),
	)
	if err != nil {
		return GenericGameMetadata{}, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return GenericGameMetadata{}, fmt.Errorf("failed to decode game status: %w", err)
	}
	return GenericGameMetadata{
		L1Head:        results[0].GetHash(0),
		L2SequenceNum: bigs.Uint64Strict(results[1].GetBigInt(0)),
		ProposedRoot:  results[2].GetHash(0),
		Status:        status,
	}, nil
}

func (s *SuperPermissionedDisputeGameContract) GetCredit(ctx context.Context, _ common.Address) (*big.Int, gameTypes.GameStatus, error) {
	status, err := s.GetStatus(ctx)
	if err != nil {
		return nil, gameTypes.GameStatusInProgress, err
	}
	return big.NewInt(0), status, nil
}

func (s *SuperPermissionedDisputeGameContract) ClaimCreditTx(context.Context, common.Address) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, ErrSimulationFailed
}

func (s *SuperPermissionedDisputeGameContract) GetBondDistributionMode(context.Context, rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return faultTypes.RefundDistributionMode, nil
}

func (s *SuperPermissionedDisputeGameContract) CloseGameTx(context.Context) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, ErrCloseGameNotSupported
}
