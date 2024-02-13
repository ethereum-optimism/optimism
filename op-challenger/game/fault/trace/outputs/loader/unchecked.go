package loader

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type UncheckedOutputLoader struct {
	rollupClient OutputRollupClient
}

func NewUncheckedOutputRootProvider(rollupClient OutputRollupClient) *UncheckedOutputLoader {
	return &UncheckedOutputLoader{rollupClient: rollupClient}
}

func (l *UncheckedOutputLoader) OutputAtBlock(ctx context.Context, blockNum uint64) (common.Hash, error) {
	output, err := l.rollupClient.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", blockNum, err)
	}
	return common.Hash(output.OutputRoot), nil
}
