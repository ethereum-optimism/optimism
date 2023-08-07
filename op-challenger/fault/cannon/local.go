package cannon

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type localGameInputs struct {
	l1Head        common.Hash
	l2ChainId     *big.Int
	l2Head        common.Hash
	l2OutputRoot  common.Hash
	l2Claim       common.Hash
	l2BlockNumber *big.Int
}

type L2DataSource interface {
	ChainID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

type GameInputsSource interface {
	L1Head(opts *bind.CallOpts) ([32]byte, error)
	Proposals(opts *bind.CallOpts) (struct {
		Starting bindings.IFaultDisputeGameOutputProposal
		Disputed bindings.IFaultDisputeGameOutputProposal
	}, error)
}

func fetchLocalInputs(ctx context.Context, gameAddr common.Address, caller GameInputsSource, l2Client L2DataSource) (localGameInputs, error) {
	opts := &bind.CallOpts{Context: ctx}
	l1Head, err := caller.L1Head(opts)
	if err != nil {
		return localGameInputs{}, fmt.Errorf("fetch L1 head for game %v: %w", gameAddr, err)
	}
	l2ChainId, err := l2Client.ChainID(ctx)
	if err != nil {
		return localGameInputs{}, fmt.Errorf("fetch L2 chain ID: %w", err)
	}

	proposals, err := caller.Proposals(opts)
	if err != nil {
		return localGameInputs{}, fmt.Errorf("fetch proposals: %w", err)
	}
	claimedOutput := proposals.Disputed
	agreedOutput := proposals.Starting
	agreedHeader, err := l2Client.HeaderByNumber(ctx, agreedOutput.L2BlockNumber)
	if err != nil {
		return localGameInputs{}, fmt.Errorf("fetch L2 block header %v: %w", agreedOutput.L2BlockNumber, err)
	}
	l2Head := agreedHeader.Hash()

	return localGameInputs{
		l1Head:        l1Head,
		l2ChainId:     l2ChainId,
		l2Head:        l2Head,
		l2OutputRoot:  agreedOutput.OutputRoot,
		l2Claim:       claimedOutput.OutputRoot,
		l2BlockNumber: claimedOutput.L2BlockNumber,
	}, nil
}
