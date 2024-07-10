package claim

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var ErrClaimNotValid = errors.New("invalid claim")

type L2Source interface {
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2OutputRoot(uint64) (eth.Bytes32, error)
}

func ValidateClaim(log log.Logger, l2ClaimBlockNum uint64, claimedOutputRoot eth.Bytes32, src L2Source) error {
	l2Head, err := src.L2BlockRefByLabel(context.Background(), eth.Safe)
	if err != nil {
		return fmt.Errorf("cannot retrieve safe head: %w", err)
	}
	outputRoot, err := src.L2OutputRoot(min(l2ClaimBlockNum, l2Head.Number))
	if err != nil {
		return fmt.Errorf("calculate L2 output root: %w", err)
	}
	log.Info("Validating claim", "head", l2Head, "output", outputRoot, "claim", claimedOutputRoot)
	if claimedOutputRoot != outputRoot {
		return fmt.Errorf("%w: claim: %v actual: %v", ErrClaimNotValid, claimedOutputRoot, outputRoot)
	}
	return nil
}
