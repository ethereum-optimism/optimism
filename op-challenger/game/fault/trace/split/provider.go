package split

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	GetStepDataErr = fmt.Errorf("GetStepData not supported")
	NoProvidersErr = fmt.Errorf("no trace providers configured")
)

// todo(refcell): the Get method traceIndex is update to be a Position so we have the depth
// var _ types.TraceProvider = (*SplitTraceProvider)(nil)

// SplitTraceProvider is a [types.TraceProvider] implementation that
// routes requests to the correct internal trace provider based on the
// depth of the requested trace.
type SplitTraceProvider struct {
	logger     log.Logger
	providers  []types.TraceProvider
	depthTiers []uint64
}

func NewTraceProvider(logger log.Logger, providers []types.TraceProvider, depthTiers []uint64) *SplitTraceProvider {
	return &SplitTraceProvider{
		logger:     logger,
		providers:  providers,
		depthTiers: depthTiers,
	}
}

func (s *SplitTraceProvider) providerForDepth(depth uint64) (uint64, types.TraceProvider) {
	reduced := uint64(0)
	for i, tier := range s.depthTiers {
		if depth <= tier {
			return reduced, s.providers[i]
		}
		if i < len(s.providers)-1 {
			reduced += tier
		}
	}
	return reduced, s.providers[len(s.providers)-1]
}

// Get routes the Get request to the internal [types.TraceProvider] that
// that serves the trace index at the depth.
func (s *SplitTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	if len(s.providers) == 0 {
		return common.Hash{}, NoProvidersErr
	}
	reduced, provider := s.providerForDepth(uint64(pos.Depth()))
	localizedPosition := pos.Localize(reduced)
	// todo(refcell): we should just pass the localized position once `Get` is updated to accept a Position
	return provider.Get(ctx, localizedPosition.ToGIndex())
}

// AbsolutePreStateCommitment returns the absolute prestate from the lowest internal [types.TraceProvider]
func (s *SplitTraceProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	if len(s.providers) == 0 {
		return common.Hash{}, NoProvidersErr
	}
	return s.providers[len(s.providers)-1].AbsolutePreStateCommitment(ctx)
}

// AbsolutePreState routes the AbsolutePreState request to the lowest internal [types.TraceProvider].
func (s *SplitTraceProvider) AbsolutePreState(ctx context.Context) (preimage []byte, err error) {
	if len(s.providers) == 0 {
		return nil, NoProvidersErr
	}
	return s.providers[len(s.providers)-1].AbsolutePreState(ctx)
}

// GetStepData routes the GetStepData request to the lowest internal [types.TraceProvider].
func (s *SplitTraceProvider) GetStepData(ctx context.Context, i uint64) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	if len(s.providers) == 0 {
		return nil, nil, nil, NoProvidersErr
	}
	return s.providers[len(s.providers)-1].GetStepData(ctx, i)
}
