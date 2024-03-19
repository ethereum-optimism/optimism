package source

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

func (l *OutputSourceCreator) ForL1Head(ctx context.Context, l1Head common.Hash) (*RestrictedOutputSource, error) {
	// TODO(client-pod#416): Run op-program to detect the latest safe head supported by l1Head
	return NewRestrictedOutputSource(l.rollupClient, math.MaxUint64), nil
}
