package split

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var _ types.TraceProvider = (*SplitTraceProvider)(nil)

// CreateLowerProvider creates a new [types.TraceProvider] that is
// used for the bottom half of the [SplitTraceProvider].
type CreateLowerProvider func(ctx context.Context, preClaim common.Hash, postClaim common.Hash) (types.TraceProvider, error)

// AbsolutePrestateCommitmentFetcher is a function pointer to fetch the absolute prestate commitment
type AbsolutePrestateCommitmentFetcher func(ctx context.Context) (hash common.Hash, err error)

// AbsolutePrestateFetcher is a function pointer to fetch the absolute prestate
type AbsolutePrestateFetcher func(ctx context.Context) (preimage []byte, err error)

// SplitTraceProvider is a [types.TraceProvider] implementation that
// routes requests to the correct internal trace provider based on the
// depth of the requested trace.
type SplitTraceProvider struct {
	logger                            log.Logger
	upper                             types.TraceProvider
	lower                             map[*big.Int]types.TraceProvider
	lowerCreator                      CreateLowerProvider
	absolutePrestateCommitmentFetcher AbsolutePrestateCommitmentFetcher
	absolutePrestateFetcher           AbsolutePrestateFetcher
	topDepth                          uint64
}

// NewTraceProvider creates a new [SplitTraceProvider] instance.
// The [topDepth] parameter specifies the depth at which the internal
// [types.TraceProvider] should be switched.
func NewTraceProvider(logger log.Logger, upper types.TraceProvider, creator CreateLowerProvider, topDepth uint64) *SplitTraceProvider {
	return &SplitTraceProvider{
		logger:       logger,
		upper:        upper,
		lower:        make(map[*big.Int]types.TraceProvider),
		lowerCreator: creator,
		topDepth:     topDepth,
	}
}

func (s *SplitTraceProvider) buildLowerProvider(ctx context.Context, pos types.Position) (types.TraceProvider, error) {
	ancestorIndex := pos.IndexForParentDepth(int(s.topDepth))
	provider, ok := s.lower[ancestorIndex]
	if ok {
		return provider, nil
	}

	// Fetch the preclaim and postclaim for the ancestor.
	leftIndex := new(big.Int).Sub(ancestorIndex, big.NewInt(1))
	rightIndex := new(big.Int).Add(ancestorIndex, big.NewInt(1))
	prePos := types.NewPosition(int(s.topDepth), leftIndex)
	postPos := types.NewPosition(int(s.topDepth), rightIndex)
	preClaim, err := s.upper.Get(ctx, prePos)
	if err != nil {
		return nil, err
	}
	postClaim, err := s.upper.Get(ctx, postPos)
	if err != nil {
		return nil, err
	}
	provider, err = s.lowerCreator(ctx, preClaim, postClaim)
	if err != nil {
		return nil, err
	}
	s.absolutePrestateFetcher = provider.AbsolutePreState
	s.absolutePrestateCommitmentFetcher = provider.AbsolutePreStateCommitment
	return provider, nil
}

func (s *SplitTraceProvider) providerForPosition(ctx context.Context, pos types.Position) (uint64, types.TraceProvider, error) {
	if uint64(pos.Depth()) <= s.topDepth {
		return 0, s.upper, nil
	}
	lower, err := s.buildLowerProvider(ctx, pos)
	if err != nil {
		return 0, nil, err
	}
	return s.topDepth, lower, nil
}

// Get routes the Get request to the internal [types.TraceProvider] that
// that serves the trace index at the depth.
func (s *SplitTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	ancestorDepth, provider, err := s.providerForPosition(ctx, pos)
	if err != nil {
		return common.Hash{}, err
	}
	relativePosition, err := pos.RelativeToAncestorAtDepth(ancestorDepth)
	if err != nil {
		return common.Hash{}, err
	}
	return provider.Get(ctx, relativePosition)
}

// AbsolutePreStateCommitment returns the absolute prestate from the lowest internal [types.TraceProvider]
func (s *SplitTraceProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	if s.absolutePrestateCommitmentFetcher != nil {
		return s.absolutePrestateCommitmentFetcher(ctx)
	}
	return common.Hash{}, errors.New("missing lower trace provider to fetch absolute prestate commitment")
}

// AbsolutePreState routes the AbsolutePreState request to the lowest internal [types.TraceProvider].
func (s *SplitTraceProvider) AbsolutePreState(ctx context.Context) (preimage []byte, err error) {
	if s.absolutePrestateFetcher != nil {
		return s.absolutePrestateFetcher(ctx)
	}
	return nil, errors.New("missing lower trace provider to fetch absolute prestate")
}

// GetStepData routes the GetStepData request to the lowest internal [types.TraceProvider].
func (s *SplitTraceProvider) GetStepData(ctx context.Context, pos types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	ancestorDepth, provider, err := s.providerForPosition(ctx, pos)
	if err != nil {
		return nil, nil, nil, err
	}
	relativePosition, err := pos.RelativeToAncestorAtDepth(ancestorDepth)
	if err != nil {
		return nil, nil, nil, err
	}
	return provider.GetStepData(ctx, relativePosition)
}
