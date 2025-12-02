package test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

func NewAlphabetWithProofProvider(t *testing.T, startingL2BlockNumber *big.Int, maxDepth types.Depth, oracleError error) *AlphabetWithProofProvider {
	return &AlphabetWithProofProvider{
		alphabet.NewTraceProvider(startingL2BlockNumber, maxDepth),
		maxDepth,
		oracleError,
		nil,
	}
}

func NewAlphabetClaimBuilder(t *testing.T, startingL2BlockNumber *big.Int, maxDepth types.Depth) *ClaimBuilder {
	alphabetProvider := NewAlphabetWithProofProvider(t, startingL2BlockNumber, maxDepth, nil)
	return NewClaimBuilder(t, maxDepth, alphabetProvider)
}

type AlphabetWithProofProvider struct {
	*alphabet.AlphabetTraceProvider
	depth            types.Depth
	OracleError      error
	L2BlockChallenge *types.InvalidL2BlockNumberChallenge
}

func (a *AlphabetWithProofProvider) GetStepData(ctx context.Context, i types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	preimage, _, _, err := a.AlphabetTraceProvider.GetStepData(ctx, i)
	if err != nil {
		return nil, nil, nil, err
	}
	traceIndex := i.TraceIndex(a.depth).Uint64()
	data := types.NewPreimageOracleData([]byte{byte(traceIndex)}, []byte{byte(traceIndex - 1)}, uint32(traceIndex-1))
	return preimage, []byte{byte(traceIndex - 1)}, data, nil
}

func (c *AlphabetWithProofProvider) GetL2BlockNumberChallenge(_ context.Context) (*types.InvalidL2BlockNumberChallenge, error) {
	if c.L2BlockChallenge != nil {
		return c.L2BlockChallenge, nil
	} else {
		return nil, types.ErrL2BlockNumberValid
	}
}
