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

type L2DataSource interface {
	ChainID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

type GameInputsSource interface {
	GetL1Head(ctx context.Context) (common.Hash, error)
	GetProposals(ctx context.Context) (agreed contracts.Proposal, disputed contracts.Proposal, err error)
}

func fetchLocalInputs(ctx context.Context, caller GameInputsSource, l2Client L2DataSource) (LocalGameInputs, error) {
	l1Head, err := caller.GetL1Head(ctx)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch L1 head: %w", err)
	}

	agreedOutput, claimedOutput, err := caller.GetProposals(ctx)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch proposals: %w", err)
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
