package extract

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	gameContracts "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

var _ GameCaller = (*SuperPermissionedGameCaller)(nil)

type SuperPermissionedGameCaller struct {
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewSuperPermissionedGameCaller(_ GameCallerMetrics, addr common.Address, caller *batching.MultiCaller) *SuperPermissionedGameCaller {
	return &SuperPermissionedGameCaller{
		multiCaller: caller,
		contract:    batching.NewBoundContract(snapshots.LoadSuperPermissionedDisputeGameABI(), addr),
	}
}

func (s *SuperPermissionedGameCaller) GetExtendedMetadata(ctx context.Context, block rpcblock.Block) (gameContracts.GameMetadata, error) {
	results, err := s.multiCaller.Call(ctx, block,
		s.contract.Call("l1Head"),
		s.contract.Call("l2SequenceNumber"),
		s.contract.Call("rootClaim"),
		s.contract.Call("status"),
		s.contract.Call("maxClockDuration"),
	)
	if err != nil {
		return gameContracts.GameMetadata{}, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	if len(results) != 5 {
		return gameContracts.GameMetadata{}, fmt.Errorf("expected 5 results but got %v", len(results))
	}
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return gameContracts.GameMetadata{}, fmt.Errorf("failed to convert game status: %w", err)
	}
	l2SequenceNumber := uint64(math.MaxUint64)
	if result := results[1].GetBigInt(0); result.Sign() >= 0 && result.BitLen() <= 64 {
		l2SequenceNumber = bigs.Uint64Strict(result)
	}
	return gameContracts.GameMetadata{
		L1Head:           results[0].GetHash(0),
		L2SequenceNum:    l2SequenceNumber,
		RootClaim:        results[2].GetHash(0),
		Status:           status,
		MaxClockDuration: results[4].GetUint64(0),
	}, nil
}

func (s *SuperPermissionedGameCaller) GetAllClaims(context.Context, rpcblock.Block) ([]faultTypes.Claim, error) {
	return nil, nil
}

func (s *SuperPermissionedGameCaller) IsResolved(_ context.Context, _ rpcblock.Block, claims ...faultTypes.Claim) ([]bool, error) {
	return make([]bool, len(claims)), nil
}

func (s *SuperPermissionedGameCaller) GetWithdrawals(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*gameContracts.WithdrawalRequest, error) {
	withdrawals := make([]*gameContracts.WithdrawalRequest, len(recipients))
	for i := range withdrawals {
		withdrawals[i] = &gameContracts.WithdrawalRequest{
			Amount:    big.NewInt(0),
			Timestamp: big.NewInt(0),
		}
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
	return faultTypes.RefundDistributionMode, nil
}

func (s *SuperPermissionedGameCaller) GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	return big.NewInt(0), 0, common.Address{}, nil
}
