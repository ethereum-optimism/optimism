package trace

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

func NewSimpleTraceAccessor(trace types.TraceProvider) *Accessor {
	selector := func(_ context.Context, _ types.Game, _ types.Claim, _ types.Position) (types.TraceProvider, error) {
		return trace, nil
	}
	return &Accessor{selector}
}

func NewSplitTraceAccessor(top types.TraceProvider, topDepth uint64, bottomFactory func(ctx context.Context, pre common.Hash, post common.Hash) (types.TraceProvider, error)) *Accessor {
	selector := func(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error) {
		if uint64(pos.Depth()) <= topDepth {
			return top, nil
		}
		// TODO: Walk back up from claim, pulling ancestors from game, until we find the pre and post claim for the top level
		var pre common.Hash
		var post common.Hash
		// TODO: Cache the bottom providers
		bottom, err := bottomFactory(ctx, pre, post)
		if err != nil {
			return nil, fmt.Errorf("create provider for pre %v and post %v: %w", pre, post, err)
		}
		return Translate(bottom, topDepth), nil
	}
	return &Accessor{selector}
}

type Accessor struct {
	selector func(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (types.TraceProvider, error)
}

func (t *Accessor) Get(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (common.Hash, error) {
	provider, err := t.selector(ctx, game, ref, pos)
	if err != nil {
		return common.Hash{}, err
	}
	return provider.Get(ctx, pos)
}

func (t *Accessor) GetStepData(ctx context.Context, game types.Game, ref types.Claim, pos types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	provider, err := t.selector(ctx, game, ref, pos)
	if err != nil {
		return nil, nil, nil, err
	}
	return provider.GetStepData(ctx, pos)
}

var _ types.TraceAccessor = (*Accessor)(nil)
