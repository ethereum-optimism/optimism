package contracts

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
)

var (
	methodWithdrawals = "withdrawals"
)

type DelayedWETHContract struct {
	metrics     metrics.ContractMetricer
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

type WithdrawalRequest struct {
	Amount    *big.Int
	Timestamp *big.Int
}

func NewDelayedWETHContract(metrics metrics.ContractMetricer, addr common.Address, caller *batching.MultiCaller) *DelayedWETHContract {
	contractAbi := snapshots.LoadDelayedWETHABI()
	return &DelayedWETHContract{
		metrics:     metrics,
		multiCaller: caller,
		contract:    batching.NewBoundContract(contractAbi, addr),
	}
}

// GetWithdrawals returns all withdrawals made from the contract since the given block.
func (d *DelayedWETHContract) GetWithdrawals(ctx context.Context, block rpcblock.Block, gameAddr common.Address, recipients ...common.Address) ([]*WithdrawalRequest, error) {
	defer d.metrics.StartContractRequest("GetWithdrawals")()
	calls := make([]batching.Call, 0, len(recipients))
	for _, recipient := range recipients {
		calls = append(calls, d.contract.Call(methodWithdrawals, gameAddr, recipient))
	}
	results, err := d.multiCaller.Call(ctx, block, calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch withdrawals: %w", err)
	}
	withdrawals := make([]*WithdrawalRequest, len(recipients))
	for i, result := range results {
		withdrawals[i] = &WithdrawalRequest{
			Amount:    result.GetBigInt(0),
			Timestamp: result.GetBigInt(1),
		}
	}
	return withdrawals, nil
}
