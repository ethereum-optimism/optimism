package extract

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodL1Head           = "l1Head"
	methodL2SequenceNumber = "l2SequenceNumber"
	methodRootClaim        = "rootClaim"
	methodStatus           = "status"
)

type SuperPermissionedGameCaller struct {
	metrics     contractMetrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewSuperPermissionedGameCaller(metrics contractMetrics.ContractMetricer, addr common.Address, caller *batching.MultiCaller) *SuperPermissionedGameCaller {
	return &SuperPermissionedGameCaller{
		metrics:     metrics,
		multiCaller: caller,
		contract:    batching.NewBoundContract(snapshots.LoadSuperPermissionedDisputeGameABI(), addr),
	}
}

func (s *SuperPermissionedGameCaller) GetExtendedMetadata(ctx context.Context, block rpcblock.Block) (contracts.GameMetadata, error) {
	defer s.metrics.StartContractRequest("GetExtendedMetadata")()
	results, err := s.multiCaller.Call(ctx, block,
		s.contract.Call(methodL1Head),
		s.contract.Call(methodL2SequenceNumber),
		s.contract.Call(methodRootClaim),
		s.contract.Call(methodStatus),
	)
	if err != nil {
		return contracts.GameMetadata{}, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	if len(results) != 4 {
		return contracts.GameMetadata{}, fmt.Errorf("expected 4 results but got %v", len(results))
	}
	l2SequenceNumber := results[1].GetBigInt(0)
	l2SequenceNumberUint64 := uint64(math.MaxUint64)
	if l2SequenceNumber.IsUint64() {
		l2SequenceNumberUint64 = bigs.Uint64Strict(l2SequenceNumber)
	}
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return contracts.GameMetadata{}, fmt.Errorf("failed to convert game status: %w", err)
	}
	return contracts.GameMetadata{
		L1Head:        results[0].GetHash(0),
		L2SequenceNum: l2SequenceNumberUint64,
		RootClaim:     results[2].GetHash(0),
		Status:        status,
	}, nil
}

func (s *SuperPermissionedGameCaller) GetAllClaims(context.Context, rpcblock.Block) ([]faultTypes.Claim, error) {
	return nil, nil
}

func (s *SuperPermissionedGameCaller) IsResolved(_ context.Context, _ rpcblock.Block, claims ...faultTypes.Claim) ([]bool, error) {
	return make([]bool, len(claims)), nil
}

func (s *SuperPermissionedGameCaller) GetWithdrawals(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	withdrawals := make([]*contracts.WithdrawalRequest, len(recipients))
	for i := range withdrawals {
		withdrawals[i] = &contracts.WithdrawalRequest{Timestamp: big.NewInt(0), Amount: big.NewInt(0)}
	}
	return withdrawals, nil
}

func (s *SuperPermissionedGameCaller) GetCredits(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	credits := make([]*big.Int, len(recipients))
	for i := range credits {
		credits[i] = big.NewInt(0)
	}
	return credits, nil
}

func (s *SuperPermissionedGameCaller) GetBondDistributionMode(context.Context, rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return faultTypes.UndecidedDistributionMode, nil
}

func (s *SuperPermissionedGameCaller) GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	return big.NewInt(0), 0, common.Address{}, nil
}
