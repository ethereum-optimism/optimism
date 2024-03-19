package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

var ErrExceedsL1Head = errors.New("output root beyond safe head for L1 head")

type RestrictedOutputSource struct {
	unrestricted *UnrestrictedOutputSource
	maxSafeHead  uint64
}

func NewRestrictedOutputSource(rollupClient OutputRollupClient, maxSafeHead uint64) *RestrictedOutputSource {
	return &RestrictedOutputSource{
		unrestricted: NewUnrestrictedOutputSource(rollupClient),
		maxSafeHead:  maxSafeHead,
	}
}

func (l *RestrictedOutputSource) OutputAtBlock(ctx context.Context, blockNum uint64) (common.Hash, error) {
	if blockNum > l.maxSafeHead {
		return common.Hash{}, fmt.Errorf("%w, requested: %v max: %v", ErrExceedsL1Head, blockNum, l.maxSafeHead)
	}
	return l.unrestricted.OutputAtBlock(ctx, blockNum)
}
