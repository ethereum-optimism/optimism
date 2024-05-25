package contracts

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodOracle = "oracle"
)

// VMContract is a binding that works with contracts implementing the IBigStepper interface
type VMContract struct {
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewVMContract(addr common.Address, caller *batching.MultiCaller) (*VMContract, error) {
	mipsAbi, err := bindings.MIPSMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load VM ABI: %w", err)
	}

	return &VMContract{
		multiCaller: caller,
		contract:    batching.NewBoundContract(mipsAbi, addr),
	}, nil
}

func (c *VMContract) Oracle(ctx context.Context) (*PreimageOracleContract, error) {
	results, err := c.multiCaller.SingleCall(ctx, batching.BlockLatest, c.contract.Call(methodOracle))
	if err != nil {
		return nil, fmt.Errorf("failed to load oracle address: %w", err)
	}
	return NewPreimageOracleContract(results.GetAddress(0), c.multiCaller)
}
