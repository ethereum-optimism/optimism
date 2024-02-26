package source

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
	SafeHeadAtL1Block(ctx context.Context, l1BlockNum uint64) (*eth.SafeHeadResponse, error)
}

type L1HeaderSource interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
}

type OutputSourceCreator struct {
	log          log.Logger
	rollupClient OutputRollupClient
	l1Client     L1HeaderSource
}

func NewOutputSourceCreator(logger log.Logger, rollupClient OutputRollupClient, l1Client L1HeaderSource) *OutputSourceCreator {
	return &OutputSourceCreator{
		log:          logger,
		rollupClient: rollupClient,
		l1Client:     l1Client,
	}
}

func (l *OutputSourceCreator) ForL1Head(ctx context.Context, l1Head common.Hash) (*RestrictedOutputSource, error) {
	head, err := l.l1Client.HeaderByHash(ctx, l1Head)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 head %v: %w", l1Head, err)
	}
	return NewRestrictedOutputSource(l.rollupClient, eth.HeaderBlockID(head)), nil
}
