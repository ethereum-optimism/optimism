package loader

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

var ErrExceedsL1Head = errors.New("output root beyond safe head for L1 head")

type RestrictedOutputLoader struct {
	rollupClient OutputRollupClient
	maxSafeHead  uint64
}

func NewRestrictedOutputLoader(rollupClient OutputRollupClient, maxSafeHead uint64) *RestrictedOutputLoader {
	return &RestrictedOutputLoader{
		rollupClient: rollupClient,
		maxSafeHead:  maxSafeHead,
	}
}

func (l *RestrictedOutputLoader) OutputAtBlock(ctx context.Context, blockNum uint64) (common.Hash, error) {
	if blockNum > l.maxSafeHead {
		// TODO: Should this just return the maxSafeHead hash instead or do we need special handling?
		return common.Hash{}, fmt.Errorf("%w, requested: %v max: %v", ErrExceedsL1Head, blockNum, l.maxSafeHead)
	}
	output, err := l.rollupClient.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", blockNum, err)
	}
	return common.Hash(output.OutputRoot), nil
}
