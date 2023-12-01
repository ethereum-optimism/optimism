package outputs

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func NewOutputAlphabetTraceAccessor(
	ctx context.Context,
	logger log.Logger,
	v AbsolutePrestateValidator,
	m metrics.Metricer,
	cfg *config.Config,
	gameDepth uint64,
	splitDepth uint64,
	prestateBlock uint64,
	poststateBlock uint64,
) (*trace.Accessor, error) {
	bottomDepth := gameDepth - splitDepth
	outputProvider, err := NewTraceProvider(ctx, logger, cfg.RollupRpc, splitDepth, prestateBlock, poststateBlock)
	if err != nil {
		return nil, err
	}

	if err := v(ctx, outputProvider); err != nil {
		return nil, err
	}

	alphabetCreator := func(ctx context.Context, localContext common.Hash, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
		provider := alphabet.NewTraceProvider(localContext.Hex(), bottomDepth)
		return provider, nil
	}

	cache := NewProviderCache(m, "output_alphabet_provider", alphabetCreator)
	selector := split.NewSplitProviderSelector(outputProvider, int(splitDepth), OutputRootSplitAdapter(outputProvider, cache.GetOrCreate))
	return trace.NewAccessor(selector), nil
}
