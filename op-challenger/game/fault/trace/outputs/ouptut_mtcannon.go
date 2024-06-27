package outputs

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/mtcannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func NewOutputMTCannonTraceAccessor(
	logger log.Logger,
	m metrics.Metricer,
	cfg vm.Config,
	l2Client utils.L2HeaderSource,
	prestateProvider types.PrestateProvider,
	mtCannonPrestate string,
	rollupClient OutputRollupClient,
	dir string,
	l1Head eth.BlockID,
	splitDepth types.Depth,
	prestateBlock uint64,
	poststateBlock uint64,
) (*trace.Accessor, error) {
	outputProvider := NewTraceProvider(logger, prestateProvider, rollupClient, l2Client, l1Head, splitDepth, prestateBlock, poststateBlock)
	mtCannonCreator := func(ctx context.Context, localContext common.Hash, depth types.Depth, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
		logger := logger.New("pre", agreed.OutputRoot, "post", claimed.OutputRoot, "localContext", localContext)
		subdir := filepath.Join(dir, localContext.Hex())
		localInputs, err := utils.FetchLocalInputsFromProposals(ctx, l1Head.Hash, l2Client, agreed, claimed)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch mt-cannon local inputs: %w", err)
		}
		provider := mtcannon.NewTraceProvider(logger, m, cfg, prestateProvider, mtCannonPrestate, localInputs, subdir, depth)
		return provider, nil
	}

	cache := NewProviderCache(m, "output_mt_cannon_provider", mtCannonCreator)
	selector := split.NewSplitProviderSelector(outputProvider, splitDepth, OutputRootSplitAdapter(outputProvider, cache.GetOrCreate))
	return trace.NewAccessor(selector), nil
}
