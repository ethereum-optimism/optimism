package cannon

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
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

type L2HeaderSource interface {
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

type L1HeadSource interface {
	GetL1Head(ctx context.Context) (common.Hash, error)
}

type GameInputsSource interface {
	L1HeadSource
	GetProposals(ctx context.Context) (agreed contracts.Proposal, disputed contracts.Proposal, err error)
}

func FetchLocalInputs(ctx context.Context, caller GameInputsSource, l2Client L2HeaderSource) (LocalGameInputs, error) {
	agreedOutput, claimedOutput, err := caller.GetProposals(ctx)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch proposals: %w", err)
	}
	return FetchLocalInputsFromProposals(ctx, caller, l2Client, agreedOutput, claimedOutput)
}

func FetchLocalInputsFromProposals(ctx context.Context, caller L1HeadSource, l2Client L2HeaderSource, agreedOutput contracts.Proposal, claimedOutput contracts.Proposal) (LocalGameInputs, error) {
	l1Head, err := caller.GetL1Head(ctx)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch L1 head: %w", err)
	}

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
