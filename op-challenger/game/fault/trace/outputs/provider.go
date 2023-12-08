package outputs

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrGetStepData = errors.New("GetStepData not supported")
	ErrIndexTooBig = errors.New("trace index is greater than max uint64")
)

var _ types.TraceProvider = (*OutputTraceProvider)(nil)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

// OutputTraceProvider is a [types.TraceProvider] implementation that uses
// output roots for given L2 Blocks as a trace.
type OutputTraceProvider struct {
	types.PrestateProvider
	logger         log.Logger
	rollupClient   OutputRollupClient
	prestateBlock  uint64
	poststateBlock uint64
	gameDepth      uint64
}

func NewTraceProvider(ctx context.Context, logger log.Logger, rollupRpc string, gameDepth, prestateBlock, poststateBlock uint64) (*OutputTraceProvider, error) {
	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, dial.DefaultDialTimeout, logger, rollupRpc)
	if err != nil {
		return nil, err
	}
	prestateProvider := NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
	return NewTraceProviderFromInputs(logger, prestateProvider, rollupClient, gameDepth, prestateBlock, poststateBlock), nil
}

func NewTraceProviderFromInputs(logger log.Logger, prestateProvider types.PrestateProvider, rollupClient OutputRollupClient, gameDepth, prestateBlock, poststateBlock uint64) *OutputTraceProvider {
	return &OutputTraceProvider{
		PrestateProvider: prestateProvider,
		logger:           logger,
		rollupClient:     rollupClient,
		prestateBlock:    prestateBlock,
		poststateBlock:   poststateBlock,
		gameDepth:        gameDepth,
	}
}

func (o *OutputTraceProvider) BlockNumber(pos types.Position) (uint64, error) {
	traceIndex := pos.TraceIndex(int(o.gameDepth))
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
	output, err := o.rollupClient.OutputAtBlock(ctx, block)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", o.prestateBlock, err)
	}
	return common.Hash(output.OutputRoot), nil
}
