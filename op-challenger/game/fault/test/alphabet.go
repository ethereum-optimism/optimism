package test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

func NewAlphabetWithProofProvider(t *testing.T, startingL2BlockNumber *big.Int, maxDepth types.Depth, oracleError error) *alphabetWithProofProvider {
	return &alphabetWithProofProvider{
		alphabet.NewTraceProvider(startingL2BlockNumber, maxDepth),
		maxDepth,
		oracleError,
	}
}

func NewAlphabetClaimBuilder(t *testing.T, startingL2BlockNumber *big.Int, maxDepth types.Depth) *ClaimBuilder {
	alphabetProvider := NewAlphabetWithProofProvider(t, startingL2BlockNumber, maxDepth, nil)
	return NewClaimBuilder(t, maxDepth, alphabetProvider)
}

type alphabetWithProofProvider struct {
	*alphabet.AlphabetTraceProvider
	depth       types.Depth
	OracleError error
}

func (a *alphabetWithProofProvider) GetStepData(ctx context.Context, i types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	preimage, _, _, err := a.AlphabetTraceProvider.GetStepData(ctx, i)
	if err != nil {
		return nil, nil, nil, err
	}
	traceIndex := i.TraceIndex(a.depth).Uint64()
	data := types.NewPreimageOracleData([]byte{byte(traceIndex)}, []byte{byte(traceIndex - 1)}, uint32(traceIndex-1))
	return preimage, []byte{byte(traceIndex - 1)}, data, nil
}
