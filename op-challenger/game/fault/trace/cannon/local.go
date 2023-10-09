package cannon

import (
	"context"
	"fmt"
	"math/big"

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
	L1Head(ctx context.Context) (common.Hash, error)
	Proposals(ctx context.Context) (agreedOutputRoot common.Hash, agreedBlockNumber *big.Int, disputedOutputRoot common.Hash, disputedBlockNumber *big.Int, err error)
}

func fetchLocalInputs(ctx context.Context, gameAddr common.Address, caller GameInputsSource, l2Client L2DataSource) (LocalGameInputs, error) {
	l1Head, err := caller.L1Head(ctx)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch L1 head for game %v: %w", gameAddr, err)
	}

	agreedOutputRoot, agreedBlockNumber, disputedOutputRoot, disputedBlockNumber, err := caller.Proposals(ctx)
	if err != nil {
		return LocalGameInputs{}, fmt.Errorf("fetch proposals: %w", err)
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
		L2Claim:       disputedOutputRoot,
		L2BlockNumber: disputedBlockNumber,
	}, nil
}
