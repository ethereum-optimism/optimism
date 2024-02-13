package loader

import (
	"context"
	"math"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type OutputSourceCreator struct {
	log          log.Logger
	rollupClient OutputRollupClient
}

func NewOutputSourceCreator(logger log.Logger, rollupClient OutputRollupClient) *OutputSourceCreator {
	return &OutputSourceCreator{
		log:          logger,
		rollupClient: rollupClient,
	}
}

func (l *OutputSourceCreator) ForL1Head(ctx context.Context, l1Head common.Hash) (*RestrictedOutputLoader, error) {
	// TODO: Actually restrict the safe head
	return NewRestrictedOutputLoader(l.rollupClient, math.MaxUint64), nil
}
