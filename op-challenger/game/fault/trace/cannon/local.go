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

type LocalGameInputs struct {
	L1Head        common.Hash
	L2Head        common.Hash
	L2OutputRoot  common.Hash
	L2Claim       common.Hash
	L2BlockNumber *big.Int
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

func fetchLocalInputs(ctx context.Context, gameAddr common.Address, caller GameInputsSource, l2Client L2DataSource) (LocalGameInputs, error) {
	opts := &bind.CallOpts{Context: ctx}
	l1Head, err := caller.L1Head(opts)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch L1 head for game %v: %w", gameAddr, err)
	}

	proposals, err := caller.Proposals(opts)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch proposals: %w", err)
	}
	claimedOutput := proposals.Disputed
	agreedOutput := proposals.Starting
	agreedHeader, err := l2Client.HeaderByNumber(ctx, agreedOutput.L2BlockNumber)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch L2 block header %v: %w", agreedOutput.L2BlockNumber, err)
	}
	l2Head := agreedHeader.Hash()

	return LocalGameInputs{
		L1Head:        l1Head,
		L2Head:        l2Head,
		L2OutputRoot:  agreedOutput.OutputRoot,
		L2Claim:       claimedOutput.OutputRoot,
		L2BlockNumber: claimedOutput.L2BlockNumber,
	}, nil
}
