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

type L1HeadSource interface {
	L1Head(opts *bind.CallOpts) ([32]byte, error)
}

type GameInputsSource interface {
	L1HeadSource
	Proposals(opts *bind.CallOpts) (struct {
		Starting bindings.IFaultDisputeGameOutputProposal
		Disputed bindings.IFaultDisputeGameOutputProposal
	}, error)
}

func fetchLocalInputs(ctx context.Context, gameAddr common.Address, caller GameInputsSource, l2Client L2DataSource) (LocalGameInputs, error) {
	proposals, err := caller.Proposals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch proposals: %w", err)
	}
	claimedOutput := proposals.Disputed
	agreedOutput := proposals.Starting
	return fetchLocalInputsForOutputRoots(ctx, gameAddr, agreedOutput.OutputRoot, agreedOutput.L2BlockNumber, claimedOutput.OutputRoot, claimedOutput.L2BlockNumber, caller, l2Client)
}

func fetchLocalInputsForOutputRoots(
	ctx context.Context,
	gameAddr common.Address,
	agreedOutputRoot common.Hash,
	agreedBlockNumber *big.Int,
	claimedOutputRoot common.Hash,
	claimedOutputBlockNumber *big.Int,
	l1HeadSource L1HeadSource,
	l2Client L2DataSource,
) (LocalGameInputs, error) {
	opts := &bind.CallOpts{Context: ctx}
	l1Head, err := l1HeadSource.L1Head(opts)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch L1 head for game %v: %w", gameAddr, err)
	}

	agreedHeader, err := l2Client.HeaderByNumber(ctx, agreedBlockNumber)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch L2 block header %v: %w", agreedBlockNumber, err)
	}
	l2Head := agreedHeader.Hash()

	return LocalGameInputs{
		L1Head:        l1Head,
		L2Head:        l2Head,
		L2OutputRoot:  agreedOutputRoot,
		L2Claim:       claimedOutputRoot,
		L2BlockNumber: claimedOutputBlockNumber,
	}, nil
}
