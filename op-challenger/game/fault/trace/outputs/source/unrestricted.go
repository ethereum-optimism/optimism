package source

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type UnrestrictedOutputSource struct {
	rollupClient OutputRollupClient
}

func NewUnrestrictedOutputSource(rollupClient OutputRollupClient) *UnrestrictedOutputSource {
	return &UnrestrictedOutputSource{rollupClient: rollupClient}
}

func (l *UnrestrictedOutputSource) OutputAtBlock(ctx context.Context, blockNum uint64) (common.Hash, error) {
	output, err := l.rollupClient.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", blockNum, err)
	}
	return common.Hash(output.OutputRoot), nil
}
