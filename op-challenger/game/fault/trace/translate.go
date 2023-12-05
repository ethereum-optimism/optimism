package trace

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type TranslatingProvider struct {
	rootDepth uint64
	provider  types.TraceProvider
}

// Translate returns a new TraceProvider that translates any requested positions before passing them on to the
// specified provider.
// The translation is done such that the root node for provider is at rootDepth.
func Translate(provider types.TraceProvider, rootDepth uint64) types.TraceProvider {
	return &TranslatingProvider{
		rootDepth: rootDepth,
		provider:  provider,
	}
}

func (p *TranslatingProvider) Original() types.TraceProvider {
	return p.provider
}

func (p *TranslatingProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	relativePos, err := pos.RelativeToAncestorAtDepth(p.rootDepth)
	if err != nil {
		return common.Hash{}, err
	}
	return p.provider.Get(ctx, relativePos)
}

func (p *TranslatingProvider) GetStepData(ctx context.Context, pos types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	relativePos, err := pos.RelativeToAncestorAtDepth(p.rootDepth)
	if err != nil {
		return nil, nil, nil, err
	}
	return p.provider.GetStepData(ctx, relativePos)
}

func (p *TranslatingProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	return p.provider.AbsolutePreStateCommitment(ctx)
}

var _ types.TraceProvider = (*TranslatingProvider)(nil)
