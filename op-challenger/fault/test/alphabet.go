package test

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
)

func NewAlphabetWithProofProvider(t *testing.T, maxDepth int, oracleError error) *alphabetWithProofProvider {
	return &alphabetWithProofProvider{
		alphabet.NewTraceProvider("abcdefghijklmnopqrstuvwxyz", uint64(maxDepth)),
		oracleError,
	}
}

func NewAlphabetClaimBuilder(t *testing.T, maxDepth int) *ClaimBuilder {
	alphabetProvider := NewAlphabetWithProofProvider(t, maxDepth, nil)
	return NewClaimBuilder(t, maxDepth, alphabetProvider)
}

type alphabetWithProofProvider struct {
	*alphabet.AlphabetTraceProvider
	OracleError error
}

func (a *alphabetWithProofProvider) GetPreimage(ctx context.Context, i uint64) ([]byte, []byte, error) {
	preimage, _, err := a.AlphabetTraceProvider.GetPreimage(ctx, i)
	if err != nil {
		return nil, nil, err
	}
	return preimage, []byte{byte(i)}, nil
}

func (a *alphabetWithProofProvider) GetOracleData(ctx context.Context, i uint64) (*types.PreimageOracleData, error) {
	if a.OracleError != nil {
		return &types.PreimageOracleData{}, a.OracleError
	}
	data := types.NewPreimageOracleData([]byte{byte(i)}, []byte{byte(i)})
	return &data, nil
}
