package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

var ErrExceedsL1Head = errors.New("output root beyond safe head for L1 head")

type RestrictedOutputSource struct {
	rollupClient OutputRollupClient
	unrestricted *UnrestrictedOutputSource
	l1Head       eth.BlockID
}

func NewRestrictedOutputSource(rollupClient OutputRollupClient, l1Head eth.BlockID) *RestrictedOutputSource {
	return &RestrictedOutputSource{
		rollupClient: rollupClient,
		unrestricted: NewUnrestrictedOutputSource(rollupClient),
		l1Head:       l1Head,
	}
}

func (l *RestrictedOutputSource) OutputAtBlock(ctx context.Context, blockNum uint64) (common.Hash, error) {
	resp, err := l.rollupClient.SafeHeadAtL1Block(ctx, l.l1Head.Number)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get safe head at L1 block %v: %w", l.l1Head, err)
	}
	maxSafeHead := resp.SafeHead.Number
	if blockNum > maxSafeHead {
		return common.Hash{}, fmt.Errorf("%w, requested: %v max: %v", ErrExceedsL1Head, blockNum, maxSafeHead)
	}
	return l.unrestricted.OutputAtBlock(ctx, blockNum)
}
