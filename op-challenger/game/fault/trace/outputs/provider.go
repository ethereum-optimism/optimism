package outputs

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrGetStepData = errors.New("GetStepData not supported")
	ErrIndexTooBig = errors.New("trace index is greater than max uint64")
)

var _ types.TraceProvider = (*OutputTraceProvider)(nil)

type OutputRootProvider interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (common.Hash, error)
}

// OutputTraceProvider is a [types.TraceProvider] implementation that uses
// output roots for given L2 Blocks as a trace.
type OutputTraceProvider struct {
	types.PrestateProvider
	logger         log.Logger
	rollupProvider OutputRootProvider
	prestateBlock  uint64
	poststateBlock uint64
	gameDepth      types.Depth
}

func NewTraceProviderFromInputs(logger log.Logger, prestateProvider types.PrestateProvider, rollupProvider OutputRootProvider, gameDepth types.Depth, prestateBlock, poststateBlock uint64) *OutputTraceProvider {
	return &OutputTraceProvider{
		PrestateProvider: prestateProvider,
		logger:           logger,
		rollupProvider:   rollupProvider,
		prestateBlock:    prestateBlock,
		poststateBlock:   poststateBlock,
		gameDepth:        gameDepth,
	}
}

func (o *OutputTraceProvider) BlockNumber(pos types.Position) (uint64, error) {
	traceIndex := pos.TraceIndex(o.gameDepth)
	if !traceIndex.IsUint64() {
		return 0, fmt.Errorf("%w: %v", ErrIndexTooBig, traceIndex)
	}
	outputBlock := traceIndex.Uint64() + o.prestateBlock + 1
	if outputBlock > o.poststateBlock {
		outputBlock = o.poststateBlock
	}
	return outputBlock, nil
}

func (o *OutputTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	outputBlock, err := o.BlockNumber(pos)
	if err != nil {
		return common.Hash{}, err
	}
	return o.outputAtBlock(ctx, outputBlock)
}

// GetStepData is not supported in the [OutputTraceProvider].
func (o *OutputTraceProvider) GetStepData(_ context.Context, _ types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	return nil, nil, nil, ErrGetStepData
}

func (o *OutputTraceProvider) outputAtBlock(ctx context.Context, block uint64) (common.Hash, error) {
	root, err := o.rollupProvider.OutputAtBlock(ctx, block)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", block, err)
	}
	return root, err
}
