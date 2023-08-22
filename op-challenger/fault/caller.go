package fault

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type FaultDisputeGameCaller interface {
	Status(opts *bind.CallOpts) (uint8, error)
	ClaimDataLen(opts *bind.CallOpts) (*big.Int, error)
}

type FaultCaller struct {
	contract FaultDisputeGameCaller
}

func NewFaultCaller(caller FaultDisputeGameCaller) *FaultCaller {
	return &FaultCaller{
		caller,
	}
}

func NewFaultCallerFromBindings(fdgAddr common.Address, client *ethclient.Client) (*FaultCaller, error) {
	caller, err := bindings.NewFaultDisputeGameCaller(fdgAddr, client)
	if err != nil {
		return nil, err
	}
	return &FaultCaller{
		caller,
	}, nil
}

// GetGameStatus returns the current game status.
// 0: In Progress
// 1: Challenger Won
// 2: Defender Won
func (fc *FaultCaller) GetGameStatus(ctx context.Context) (types.GameStatus, error) {
	status, err := fc.contract.Status(&bind.CallOpts{Context: ctx})
	return types.GameStatus(status), err
}

// GetClaimCount returns the number of claims in the game.
func (fc *FaultCaller) GetClaimCount(ctx context.Context) (uint64, error) {
	count, err := fc.contract.ClaimDataLen(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	return count.Uint64(), nil
}
