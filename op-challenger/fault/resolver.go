package fault

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ContractCaller is a minimal interface of [ethclient.Client] to allow for mocking.
type ContractCaller interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type Resolver struct {
	ContractCaller
	fdgAddr common.Address
	fdgAbi  *abi.ABI
}

// NewResolver creates a new [Resolver] instance.
func NewResolver(client ContractCaller, fdgAddr common.Address) (*Resolver, error) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Resolver{
		client,
		fdgAddr,
		fdgAbi,
	}, nil
}

// buildResolveData creates the transaction data for the Resolve function.
func (r *Resolver) buildResolveData() ([]byte, error) {
	return r.fdgAbi.Pack("resolve")
}

// CallResolve executes an eth_call to the Resolve function.
func (r *Resolver) CallResolve(ctx context.Context) (bool, error) {
	resolveData, err := r.buildResolveData()
	if err != nil {
		return false, err
	}
	_, err = r.CallContract(ctx, ethereum.CallMsg{
		To:   &r.fdgAddr,
		Data: resolveData,
	}, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}
