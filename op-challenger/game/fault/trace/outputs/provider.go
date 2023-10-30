package outputs

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	GetStepDataErr      = fmt.Errorf("GetStepData not supported")
	AbsolutePreStateErr = fmt.Errorf("AbsolutePreState not supported")
)

var _ types.TraceProvider = (*OutputTraceProvider)(nil)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

// OutputTraceProvider is a [types.TraceProvider] implementation that uses
// output roots for given L2 Blocks as a trace.
type OutputTraceProvider struct {
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
	return NewTraceProviderFromInputs(logger, rollupClient, gameDepth, prestateBlock, poststateBlock), nil
}

func NewTraceProviderFromInputs(logger log.Logger, rollupClient OutputRollupClient, gameDepth, prestateBlock, poststateBlock uint64) *OutputTraceProvider {
	return &OutputTraceProvider{
		logger:         logger,
		rollupClient:   rollupClient,
		prestateBlock:  prestateBlock,
		poststateBlock: poststateBlock,
		gameDepth:      gameDepth,
	}
}

func (o *OutputTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	traceIndex := pos.TraceIndex(int(o.gameDepth))
	if !traceIndex.IsUint64() {
		return common.Hash{}, fmt.Errorf("trace index %v is greater than max uint64", traceIndex)
	}
	outputBlock := traceIndex.Uint64() + o.prestateBlock + 1
	if outputBlock > o.poststateBlock {
		outputBlock = o.poststateBlock
	}
	output, err := o.rollupClient.OutputAtBlock(ctx, outputBlock)
	if err != nil {
		o.logger.Error("Failed to fetch output", "blockNumber", outputBlock, "err", err)
		return common.Hash{}, err
	}
	return common.Hash(output.OutputRoot), nil
}

// AbsolutePreStateCommitment returns the absolute prestate at the configured prestateBlock.
func (o *OutputTraceProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	output, err := o.rollupClient.OutputAtBlock(ctx, o.prestateBlock)
	if err != nil {
		o.logger.Error("Failed to fetch output", "blockNumber", o.prestateBlock, "err", err)
		return common.Hash{}, err
	}
	return common.Hash(output.OutputRoot), nil
}

// AbsolutePreState is not supported in the [OutputTraceProvider].
func (o *OutputTraceProvider) AbsolutePreState(ctx context.Context) (preimage []byte, err error) {
	return nil, AbsolutePreStateErr
}

// GetStepData is not supported in the [OutputTraceProvider].
func (o *OutputTraceProvider) GetStepData(ctx context.Context, pos types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	return nil, nil, nil, GetStepDataErr
}
