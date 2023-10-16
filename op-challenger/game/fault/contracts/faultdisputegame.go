package contracts

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type FaultDisputeGameContract struct {
	multiCaller *MultiCaller
	addr        common.Address
	abi         *abi.ABI
}

func NewFaultDisputeGameContract(addr common.Address, caller *MultiCaller) (*FaultDisputeGameContract, error) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load fault dispute game ABI: %w", err)
	}

	return &FaultDisputeGameContract{
		multiCaller: caller,
		abi:         fdgAbi,
		addr:        addr,
	}, nil
}

func (f *FaultDisputeGameContract) GetGameDuration(ctx context.Context) (uint64, error) {
	result, err := f.multiCaller.SingleCallLatest(ctx, NewContractCall(f.abi, f.addr, "GAME_DURATION"))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch game duration: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *FaultDisputeGameContract) GetMaxGameDepth(ctx context.Context) (uint64, error) {
	result, err := f.multiCaller.SingleCallLatest(ctx, NewContractCall(f.abi, f.addr, "MAX_GAME_DEPTH"))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch max game depth: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *FaultDisputeGameContract) GetAbsolutePrestateHash(ctx context.Context) (common.Hash, error) {
	result, err := f.multiCaller.SingleCallLatest(ctx, NewContractCall(f.abi, f.addr, "ABSOLUTE_PRESTATE"))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch absolute prestate hash: %w", err)
	}
	return result.GetHash(0), nil
}

func (f *FaultDisputeGameContract) GetStatus(ctx context.Context) (gameTypes.GameStatus, error) {
	result, err := f.multiCaller.SingleCallLatest(ctx, NewContractCall(f.abi, f.addr, "status"))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch status: %w", err)
	}
	return gameTypes.GameStatusFromUint8(result.GetUint8(0))
}

func (f *FaultDisputeGameContract) GetClaimCount(ctx context.Context) (uint64, error) {
	result, err := f.multiCaller.SingleCallLatest(ctx, NewContractCall(f.abi, f.addr, "claimDataLen"))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch claim count: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *FaultDisputeGameContract) GetClaim(ctx context.Context, idx uint64) (types.Claim, error) {
	result, err := f.multiCaller.SingleCallLatest(ctx, NewContractCall(f.abi, f.addr, "claimData", new(big.Int).SetUint64(idx)))
	if err != nil {
		return types.Claim{}, fmt.Errorf("failed to fetch claim %v: %w", idx, err)
	}
	return f.decodeClaim(result, int(idx)), nil
}

func (f *FaultDisputeGameContract) GetAllClaims(ctx context.Context) ([]types.Claim, error) {
	count, err := f.GetClaimCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load claim count: %w", err)
	}

	calls := make([]*ContractCall, count)
	for i := uint64(0); i < count; i++ {
		calls[i] = NewContractCall(f.abi, f.addr, "claimData", new(big.Int).SetUint64(i))
	}

	results, err := f.multiCaller.CallLatest(ctx, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch claim data: %w", err)
	}

	var claims []types.Claim
	for idx, result := range results {
		claims = append(claims, f.decodeClaim(result, idx))
	}
	return claims, nil
}

func (f *FaultDisputeGameContract) decodeClaim(result *CallResult, contractIndex int) types.Claim {
	parentIndex := result.GetUint32(0)
	countered := result.GetBool(1)
	claim := result.GetHash(2)
	position := result.GetBigInt(3)
	clock := result.GetBigInt(4)
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    claim,
			Position: types.NewPositionFromGIndex(position),
		},
		Countered:           countered,
		Clock:               clock.Uint64(),
		ContractIndex:       contractIndex,
		ParentContractIndex: int(parentIndex),
	}
}
