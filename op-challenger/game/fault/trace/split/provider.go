package split

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var _ types.TraceProvider = (*SplitTraceProvider)(nil)

// SplitTraceProvider is a [types.TraceProvider] implementation that
// routes requests to the correct internal trace provider based on the
// depth of the requested trace.
type SplitTraceProvider struct {
	logger         log.Logger
	topProvider    types.TraceProvider
	bottomProvider types.TraceProvider
	topDepth       uint64
}

// NewTraceProvider creates a new [SplitTraceProvider] instance.
// The [topDepth] parameter specifies the depth at which the internal
// [types.TraceProvider] should be switched.
func NewTraceProvider(logger log.Logger, topProvider types.TraceProvider, bottomProvider types.TraceProvider, topDepth uint64) *SplitTraceProvider {
	return &SplitTraceProvider{
		logger:         logger,
		topProvider:    topProvider,
		bottomProvider: bottomProvider,
		topDepth:       topDepth,
	}
}

func (s *SplitTraceProvider) providerForDepth(depth uint64) (uint64, types.TraceProvider) {
	if depth <= s.topDepth {
		return 0, s.topProvider
	}
	return s.topDepth, s.bottomProvider
}

// Get routes the Get request to the internal [types.TraceProvider] that
// that serves the trace index at the depth.
func (s *SplitTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	ancestorDepth, provider := s.providerForDepth(uint64(pos.Depth()))
	relativePosition, err := pos.RelativeToAncestorAtDepth(ancestorDepth)
	if err != nil {
		return common.Hash{}, err
	}
	return provider.Get(ctx, relativePosition)
}

// AbsolutePreStateCommitment returns the absolute prestate from the lowest internal [types.TraceProvider]
func (s *SplitTraceProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	return s.bottomProvider.AbsolutePreStateCommitment(ctx)
}

// GetStepData routes the GetStepData request to the lowest internal [types.TraceProvider].
func (s *SplitTraceProvider) GetStepData(ctx context.Context, pos types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	ancestorDepth, provider := s.providerForDepth(uint64(pos.Depth()))
	relativePosition, err := pos.RelativeToAncestorAtDepth(ancestorDepth)
	if err != nil {
		return nil, nil, nil, err
	}
	return provider.GetStepData(ctx, relativePosition)
}
