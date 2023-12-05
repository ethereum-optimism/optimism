package outputs

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/dial"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type OutputPrestateProvider struct {
	prestateBlock uint64
	rollupClient  OutputRollupClient
}

func NewPrestateProvider(ctx context.Context, logger log.Logger, rollupRpc string, prestateBlock uint64) (*OutputPrestateProvider, error) {
	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, dial.DefaultDialTimeout, logger, rollupRpc)
	if err != nil {
		return nil, err
	}
	return &OutputPrestateProvider{
		prestateBlock: prestateBlock,
		rollupClient:  rollupClient,
	}, nil
}

func NewPrestateProviderFromInputs(logger log.Logger, rollupClient OutputRollupClient, prestateBlock uint64) *OutputPrestateProvider {
	return &OutputPrestateProvider{
		prestateBlock: prestateBlock,
		rollupClient:  rollupClient,
	}
}

func (o *OutputPrestateProvider) GenesisOutputRoot(ctx context.Context) (hash common.Hash, err error) {
	return o.outputAtBlock(ctx, 0)
}

func (o *OutputPrestateProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	return o.outputAtBlock(ctx, o.prestateBlock)
}

func (o *OutputPrestateProvider) outputAtBlock(ctx context.Context, block uint64) (common.Hash, error) {
	output, err := o.rollupClient.OutputAtBlock(ctx, block)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", o.prestateBlock, err)
	}
	return common.Hash(output.OutputRoot), nil
}
