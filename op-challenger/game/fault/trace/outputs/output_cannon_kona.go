package outputs

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func NewOutputCannonKonaTraceAccessor(
	logger log.Logger,
	m metrics.Metricer,
	cfg vm.Config,
	serverExecutor vm.OracleServerExecutor,
	l2Client utils.L2HeaderSource,
	prestateProvider types.PrestateProvider,
	cannonPrestate string,
	rollupClient OutputRollupClient,
	dir string,
	l1Head eth.BlockID,
	splitDepth types.Depth,
	prestateBlock uint64,
	poststateBlock uint64,
) (*trace.Accessor, error) {
	outputProvider := NewTraceProvider(logger, prestateProvider, rollupClient, l2Client, l1Head, splitDepth, prestateBlock, poststateBlock)
	cannonKonaCreator := func(ctx context.Context, localContext common.Hash, depth types.Depth, agreed utils.Proposal, claimed utils.Proposal) (types.TraceProvider, error) {
		logger := logger.New("pre", agreed.OutputRoot, "post", claimed.OutputRoot, "localContext", localContext)
		subdir := filepath.Join(dir, localContext.Hex())
		localInputs, err := utils.FetchLocalInputsFromProposals(ctx, l1Head.Hash, l2Client, agreed, claimed)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch cannon kona local inputs: %w", err)
		}
		// Use the regular cannon trace provider but with the Kona server executor
		provider := cannon.NewTraceProvider(logger, m.ToTypedVmMetrics(cfg.VmType.String()), cfg, serverExecutor, prestateProvider, cannonPrestate, localInputs, subdir, depth)
		return provider, nil
	}

	cache := NewProviderCache(m, "output_cannon_kona_provider", cannonKonaCreator)
	selector := split.NewSplitProviderSelector(outputProvider, splitDepth, OutputRootSplitAdapter(outputProvider, cache.GetOrCreate))
	return trace.NewAccessor(selector), nil
}