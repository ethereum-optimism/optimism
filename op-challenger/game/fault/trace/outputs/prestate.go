package outputs

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

var _ types.PrestateProvider = (*OutputPrestateProvider)(nil)

type OutputPrestateProvider struct {
	prestateBlock uint64
	rollupClient  OutputRootProvider
}

func NewPrestateProvider(rollupClient OutputRootProvider, prestateBlock uint64) *OutputPrestateProvider {
	return &OutputPrestateProvider{
		prestateBlock: prestateBlock,
		rollupClient:  rollupClient,
	}
}

func (o *OutputPrestateProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	return o.outputAtBlock(ctx, o.prestateBlock)
}

func (o *OutputPrestateProvider) outputAtBlock(ctx context.Context, block uint64) (common.Hash, error) {
	root, err := o.rollupClient.OutputAtBlock(ctx, block)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", block, err)
	}
	return root, nil
}
