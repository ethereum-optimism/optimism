package keccak

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/fetcher"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Fetcher interface {
	FetchLeaves(ctx context.Context, blockHash common.Hash, oracle fetcher.Oracle, ident types.LargePreimageIdent) ([]types.Leaf, error)
}

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

func (v *PreimageVerifier) Verify(ctx context.Context, blockHash common.Hash, oracle types.LargePreimageOracle, preimage types.LargePreimageMetaData) error {
	leaves, err := v.fetcher.FetchLeaves(ctx, blockHash, oracle, preimage.LargePreimageIdent)
	if err != nil {
		return fmt.Errorf("failed to fetch leaves: %w", err)
	}
	readers := make([]io.Reader, 0, len(leaves))
	for _, leaf := range leaves {
		readers = append(readers, bytes.NewReader(leaf.Input))
	}
	matrix := matrix.NewStateMatrix()
	validLeaves, err := matrix.AbsorbAll(io.MultiReader(readers...))
	if err != nil {
		return fmt.Errorf("failed to compute valid leaves: %w", err)
	}
	if len(validLeaves) != len(leaves) {
		// TODO: Challenge
		return errors.New("wrong number of leaves")
	}
	// TODO: Check leaf content
	return nil
}
