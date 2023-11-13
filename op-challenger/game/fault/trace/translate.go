package trace

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type translatingProvider struct {
	parentDepth uint64
	provider    types.TraceProvider
}

func Translate(provider types.TraceProvider, parentDepth uint64) types.TraceProvider {
	return &translatingProvider{
		parentDepth: parentDepth,
		provider:    provider,
	}
}

func (p translatingProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	relativePos, err := pos.RelativeToAncestorAtDepth(p.parentDepth)
	if err != nil {
		return common.Hash{}, err
	}
	return p.provider.Get(ctx, relativePos)
}

func (p translatingProvider) GetStepData(ctx context.Context, pos types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	relativePos, err := pos.RelativeToAncestorAtDepth(p.parentDepth)
	if err != nil {
		return nil, nil, nil, err
	}
	return p.provider.GetStepData(ctx, relativePos)
}

func (p translatingProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	return p.provider.AbsolutePreStateCommitment(ctx)
}

var _ types.TraceProvider = (*translatingProvider)(nil)
