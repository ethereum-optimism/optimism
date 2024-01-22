package keccak

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/fetcher"
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
	_, err := v.fetcher.FetchLeaves(ctx, blockHash, oracle, preimage.LargePreimageIdent)
	if err != nil {
		return fmt.Errorf("failed to fetch leaves: %w", err)
	}
	return nil
}
