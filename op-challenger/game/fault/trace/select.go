package trace

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type providerSelector interface {
	// SelectProvider selects the appropriate TraceProvider to use at the specified position, when responding to
	// the specified reference claim within a game.
	// The returned TraceProvider
	selectProvider(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error)
}

type simpleProviderSelector struct {
	provider types.TraceProvider
}

func NewSimpleTraceAccessor(trace types.TraceProvider) *Accessor {
	return &Accessor{&simpleProviderSelector{trace}}
}

func (s *simpleProviderSelector) selectProvider(_ context.Context, _ types.Game, _ types.Claim, _ types.Position) (types.TraceProvider, error) {
	return s.provider, nil
}

type splitProviderSelector struct {
	top           types.TraceProvider
	topDepth      uint64
	bottomFactory func(ctx context.Context, pre common.Hash, post common.Hash) (types.TraceProvider, error)
}

func NewSplitTraceAccessor(top types.TraceProvider, topDepth uint64, bottomFactory func(ctx context.Context, pre common.Hash, post common.Hash) (types.TraceProvider, error)) *Accessor {
	selector := &splitProviderSelector{
		top:           top,
		topDepth:      topDepth,
		bottomFactory: bottomFactory,
	}
	return &Accessor{selector}
}

func (s *splitProviderSelector) selectProvider(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error) {
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

type Accessor struct {
	selector providerSelector
}

func (t *Accessor) Get(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (common.Hash, error) {
	provider, err := t.selector.selectProvider(ctx, game, ref, pos)
	if err != nil {
		return common.Hash{}, err
	}
	return provider.Get(ctx, pos)
}

func (t *Accessor) GetStepData(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	provider, err := t.selector.selectProvider(ctx, game, ref, pos)
	if err != nil {
		return nil, nil, nil, err
	}
	return provider.GetStepData(ctx, pos)
}

var _ types.TraceAccessor = (*Accessor)(nil)
