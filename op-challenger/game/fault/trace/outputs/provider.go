package outputs

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
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
	SafeHeadAtL1Block(ctx context.Context, l1BlockNum uint64) (*eth.SafeHeadResponse, error)
}

// OutputTraceProvider is a [types.TraceProvider] implementation that uses
// output roots for given L2 Blocks as a trace.
type OutputTraceProvider struct {
	types.PrestateProvider
	logger         log.Logger
	rollupProvider OutputRollupClient
	l2Client       utils.L2HeaderSource
	prestateBlock  uint64
	poststateBlock uint64
	l1Head         eth.BlockID
	gameDepth      types.Depth
}

func NewTraceProvider(logger log.Logger, prestateProvider types.PrestateProvider, rollupProvider OutputRollupClient, l2Client utils.L2HeaderSource, l1Head eth.BlockID, gameDepth types.Depth, prestateBlock, poststateBlock uint64) *OutputTraceProvider {
	return &OutputTraceProvider{
		PrestateProvider: prestateProvider,
		logger:           logger,
		rollupProvider:   rollupProvider,
		l2Client:         l2Client,
		prestateBlock:    prestateBlock,
		poststateBlock:   poststateBlock,
		l1Head:           l1Head,
		gameDepth:        gameDepth,
	}
}

// ClaimedBlockNumber returns the block number for a position restricted only by the claimed L2 block number.
// The returned block number may be after the safe head reached by processing batch data up to the game's L1 head
func (o *OutputTraceProvider) ClaimedBlockNumber(pos types.Position) (uint64, error) {
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

// HonestBlockNumber returns the block number for a position in the game restricted to the minimum of the claimed L2
// block number or the safe head reached by processing batch data up to the game's L1 head.
// This is used when posting honest output roots to ensure that only roots supported by L1 data are posted
func (o *OutputTraceProvider) HonestBlockNumber(ctx context.Context, pos types.Position) (uint64, error) {
	outputBlock, err := o.ClaimedBlockNumber(pos)
	if err != nil {
		return 0, err
	}
	resp, err := o.rollupProvider.SafeHeadAtL1Block(ctx, o.l1Head.Number)
	if err != nil {
		return 0, fmt.Errorf("failed to get safe head at L1 block %v: %w", o.l1Head, err)
	}
	maxSafeHead := resp.SafeHead.Number
	if outputBlock > maxSafeHead {
		outputBlock = maxSafeHead
	}
	return outputBlock, nil
}

func (o *OutputTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	outputBlock, err := o.HonestBlockNumber(ctx, pos)
	if err != nil {
		return common.Hash{}, err
	}
	return o.outputAtBlock(ctx, outputBlock)
}

// GetStepData is not supported in the [OutputTraceProvider].
func (o *OutputTraceProvider) GetStepData(_ context.Context, _ types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	return nil, nil, nil, ErrGetStepData
}

func (o *OutputTraceProvider) GetL2BlockNumberChallenge(ctx context.Context) (*types.InvalidL2BlockNumberChallenge, error) {
	outputBlock, err := o.HonestBlockNumber(ctx, types.RootPosition)
	if err != nil {
		return nil, err
	}
	claimedBlock, err := o.ClaimedBlockNumber(types.RootPosition)
	if err != nil {
		return nil, err
	}
	if claimedBlock == outputBlock {
		return nil, types.ErrL2BlockNumberValid
	}
	output, err := o.rollupProvider.OutputAtBlock(ctx, outputBlock)
	if err != nil {
		return nil, err
	}
	header, err := o.l2Client.HeaderByNumber(ctx, new(big.Int).SetUint64(outputBlock))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve L2 block header %v: %w", outputBlock, err)
	}
	return types.NewInvalidL2BlockNumberProof(output, header), nil
}

func (o *OutputTraceProvider) outputAtBlock(ctx context.Context, block uint64) (common.Hash, error) {
	output, err := o.rollupProvider.OutputAtBlock(ctx, block)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch output at block %v: %w", block, err)
	}
	return common.Hash(output.OutputRoot), nil
}
