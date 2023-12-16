package outputs

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func NewOutputCannonTraceAccessor(
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	l2Client cannon.L2HeaderSource,
	contract cannon.L1HeadSource,
	prestateProvider types.PrestateProvider,
	rollupClient OutputRollupClient,
	dir string,
	splitDepth uint64,
	prestateBlock uint64,
	poststateBlock uint64,
) (*trace.Accessor, error) {
	outputProvider := NewTraceProviderFromInputs(logger, prestateProvider, rollupClient, splitDepth, prestateBlock, poststateBlock)
	cannonCreator := func(ctx context.Context, localContext common.Hash, depth uint64, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
		logger := logger.New("pre", agreed.OutputRoot, "post", claimed.OutputRoot, "localContext", localContext)
		subdir := filepath.Join(dir, localContext.Hex())
		localInputs, err := cannon.FetchLocalInputsFromProposals(ctx, contract, l2Client, agreed, claimed)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch cannon local inputs: %w", err)
		}
		provider := cannon.NewTraceProvider(logger, m, cfg, localContext, localInputs, subdir, depth)
		return provider, nil
	}

	cache := NewProviderCache(m, "output_cannon_provider", cannonCreator)
	selector := split.NewSplitProviderSelector(outputProvider, int(splitDepth), OutputRootSplitAdapter(outputProvider, cache.GetOrCreate))
	return trace.NewAccessor(selector), nil
}
