package outputs

import (
	"context"

	"github.com/HashKeyChain/verse/op-challenger/game/fault/trace"
	"github.com/HashKeyChain/verse/op-challenger/game/fault/trace/alphabet"
	"github.com/HashKeyChain/verse/op-challenger/game/fault/trace/split"
	"github.com/HashKeyChain/verse/op-challenger/game/fault/trace/utils"
	"github.com/HashKeyChain/verse/op-challenger/game/fault/types"
	"github.com/HashKeyChain/verse/op-challenger/metrics"
	"github.com/HashKeyChain/verse/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func NewOutputAlphabetTraceAccessor(
	logger log.Logger,
	m metrics.Metricer,
	prestateProvider types.PrestateProvider,
	rollupClient OutputRollupClient,
	l2Client utils.L2HeaderSource,
	l1Head eth.BlockID,
	splitDepth types.Depth,
	prestateBlock uint64,
	poststateBlock uint64,
) (*trace.Accessor, error) {
	outputProvider := NewTraceProvider(logger, prestateProvider, rollupClient, l2Client, l1Head, splitDepth, prestateBlock, poststateBlock)
	alphabetCreator := func(ctx context.Context, localContext common.Hash, depth types.Depth, agreed utils.Proposal, claimed utils.Proposal) (types.TraceProvider, error) {
		provider := alphabet.NewTraceProvider(agreed.L2BlockNumber, depth)
		return provider, nil
	}
	cache := NewProviderCache(m, "output_alphabet_provider", alphabetCreator)
	selector := split.NewSplitProviderSelector(outputProvider, splitDepth, OutputRootSplitAdapter(outputProvider, cache.GetOrCreate))
	return trace.NewAccessor(selector), nil
}
