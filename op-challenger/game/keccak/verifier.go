package keccak

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/fetcher"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Fetcher interface {
	FetchInputs(ctx context.Context, blockHash common.Hash, oracle fetcher.Oracle, ident keccakTypes.LargePreimageIdent) ([]keccakTypes.InputData, error)
}

var ErrNotImplemented = errors.New("verify implementation not complete")

type PreimageVerifier struct {
	log     log.Logger
	fetcher Fetcher
}

func NewPreimageVerifier(logger log.Logger, fetcher Fetcher) *PreimageVerifier {
	return &PreimageVerifier{
		log:     logger,
		fetcher: fetcher,
	}
}

func (v *PreimageVerifier) CreateChallenge(ctx context.Context, blockHash common.Hash, oracle fetcher.Oracle, preimage keccakTypes.LargePreimageMetaData) (keccakTypes.Challenge, error) {
	inputs, err := v.fetcher.FetchInputs(ctx, blockHash, oracle, preimage.LargePreimageIdent)
	if err != nil {
		return keccakTypes.Challenge{}, fmt.Errorf("failed to fetch leaves: %w", err)
	}
	readers := make([]io.Reader, 0, len(inputs))
	var commitments []common.Hash
	for _, input := range inputs {
		readers = append(readers, bytes.NewReader(input.Input))
		commitments = append(commitments, input.Commitments...)
	}
	challenge, err := matrix.Challenge(io.MultiReader(readers...), commitments)
	if err != nil {
		return keccakTypes.Challenge{}, fmt.Errorf("failed to create challenge: %w", err)
	}
	return challenge, nil
}
