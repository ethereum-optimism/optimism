package outputs

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var _ types.PrestateProvider = (*OutputPrestateProvider)(nil)

type OutputPrestateProvider struct {
	prestateBlock uint64
	rollupClient  OutputRootProvider
	l1Head        common.Hash
}

func NewPrestateProvider(ctx context.Context, logger log.Logger, rollupClient OutputRootProvider, l1Head common.Hash, prestateBlock uint64) *OutputPrestateProvider {
	return &OutputPrestateProvider{
		prestateBlock: prestateBlock,
		rollupClient:  rollupClient,
		l1Head:        l1Head,
	}
}

func (o *OutputPrestateProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	return o.outputAtBlock(ctx, o.prestateBlock)
}

func (o *OutputPrestateProvider) outputAtBlock(ctx context.Context, block uint64) (common.Hash, error) {
	root, err := o.rollupClient.OutputAtBlock(ctx, o.l1Head, block)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", block, err)
	}
	return root, nil
}
