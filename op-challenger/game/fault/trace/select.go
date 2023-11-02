package trace

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type simpleProviderSelector struct {
	provider types.TraceProvider
}

func NewSimpleProviderSelector(trace types.TraceProvider) types.ProviderSelector {
	return &simpleProviderSelector{trace}
}

func (s *simpleProviderSelector) SelectProvider(_ context.Context, _ types.Game, _ types.Claim, _ types.Position) (types.TraceProvider, error) {
	return s.provider, nil
}

type splitProviderSelector struct {
	top           types.TraceProvider
	topDepth      uint64
	bottomFactory func(ctx context.Context, pre common.Hash, post common.Hash) (types.TraceProvider, error)
}

func NewSplitProviderSelector(top types.TraceProvider, topDepth uint64, bottomFactory func(ctx context.Context, pre common.Hash, post common.Hash) (types.TraceProvider, error)) types.ProviderSelector {
	return &splitProviderSelector{
		top:           top,
		topDepth:      topDepth,
		bottomFactory: bottomFactory,
	}
}

func (s *splitProviderSelector) SelectProvider(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error) {
	if uint64(pos.Depth()) <= s.topDepth {
		return s.top, nil
	}
	// TODO: Walk back up from claim, pulling ancestors from game, until we find the pre and post claim for the top level
	var pre common.Hash
	var post common.Hash
	// TODO: Cache the bottom providers
	bottom, err := s.bottomFactory(ctx, pre, post)
	if err != nil {
		return nil, fmt.Errorf("create provider for pre %v and post %v: %w", pre, post, err)
	}
	return Translate(bottom, s.topDepth), nil
}
