package loader

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type SafeOutputLoader struct {
	log          log.Logger
	rollupClient OutputRollupClient
}

func NewSafeOutputLoader(logger log.Logger, rollupClient OutputRollupClient) *SafeOutputLoader {
	return &SafeOutputLoader{
		log:          logger,
		rollupClient: rollupClient,
	}
}

func (l *SafeOutputLoader) OutputAtBlock(ctx context.Context, l1Head common.Hash, blockNum uint64) (common.Hash, error) {
	output, err := l.rollupClient.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", blockNum, err)
	}
	return common.Hash(output.OutputRoot), nil
}
