package outputs

import (
	"context"

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
	logger log.Logger,
	m metrics.Metricer,
	prestateProvider types.PrestateProvider,
	rollupClient OutputRollupClient,
	splitDepth uint64,
	prestateBlock uint64,
	poststateBlock uint64,
) (*trace.Accessor, error) {
	outputProvider := NewTraceProviderFromInputs(logger, prestateProvider, rollupClient, splitDepth, prestateBlock, poststateBlock)
	alphabetCreator := func(ctx context.Context, localContext common.Hash, depth uint64, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
		provider := alphabet.NewTraceProvider(localContext.Hex(), depth)
		return provider, nil
	}
	cache := NewProviderCache(m, "output_alphabet_provider", alphabetCreator)
	selector := split.NewSplitProviderSelector(outputProvider, int(splitDepth), OutputRootSplitAdapter(outputProvider, cache.GetOrCreate))
	return trace.NewAccessor(selector), nil
}
