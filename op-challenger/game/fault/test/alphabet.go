package test

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

func NewAlphabetWithProofProvider(t *testing.T, maxDepth int, oracleError error) *alphabetWithProofProvider {
	return &alphabetWithProofProvider{
		alphabet.NewTraceProvider("abcdefghijklmnopqrstuvwxyz", uint64(maxDepth)),
		uint64(maxDepth),
		oracleError,
	}
}

func NewAlphabetClaimBuilder(t *testing.T, maxDepth int) *ClaimBuilder {
	alphabetProvider := NewAlphabetWithProofProvider(t, maxDepth, nil)
	return NewClaimBuilder(t, maxDepth, alphabetProvider)
}

type alphabetWithProofProvider struct {
	*alphabet.AlphabetTraceProvider
	depth       uint64
	OracleError error
}

func (a *alphabetWithProofProvider) GetStepData(ctx context.Context, i types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	preimage, _, _, err := a.AlphabetTraceProvider.GetStepData(ctx, i)
	if err != nil {
		return nil, nil, nil, err
	}
	traceIndex := i.TraceIndex(int(a.depth)).Uint64()
	data := types.NewPreimageOracleData([]byte{byte(traceIndex)}, []byte{byte(traceIndex - 1)}, uint32(traceIndex-1))
	return preimage, []byte{byte(traceIndex - 1)}, data, nil
}
