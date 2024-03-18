package resolved

import (
	"context"
	"errors"
	"math/big"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type GameContract interface {
	GetAllClaims(ctx context.Context, block rpcblock.Block) ([]faultTypes.Claim, error)
	GetClaimedBondFlag(ctx context.Context) (*big.Int, error)
}

type GameContractCreator func(game types.GameMetadata) (GameContract, error)

type ClaimValidatorMetrics interface {
	RecordUnexpectedClaimResolution()
}

type claimValidator struct {
	logger    log.Logger
	metrics   ClaimValidatorMetrics
	creator   GameContractCreator
	claimants []common.Address
}

func NewClaimValidator(l log.Logger, m ClaimValidatorMetrics, creator GameContractCreator, claimants ...common.Address) *claimValidator {
	return &claimValidator{
		logger:    l,
		metrics:   m,
		creator:   creator,
		claimants: claimants,
	}
}

func (v *claimValidator) Validate(ctx context.Context, block uint64, games []types.GameMetadata) (err error) {
	for _, game := range games {
		err = errors.Join(err, v.validateGame(ctx, rpcblock.ByNumber(block), game))
	}
	return err
}

func (v *claimValidator) validateGame(ctx context.Context, block rpcblock.Block, game types.GameMetadata) error {
	contract, err := v.creator(game)
	if err != nil {
		return err
	}

	claims, err := contract.GetAllClaims(ctx, block)
	if err != nil {
		return err
	}

	claimedBondFlag, err := contract.GetClaimedBondFlag(ctx)
	if err != nil {
		return err
	}

	for _, claim := range claims {
		countered := claim.CounteredBy != (common.Address{})
		maxBond := claim.Bond.Cmp(claimedBondFlag) == 0
		if v.isClaimant(claim.Claimant) && countered && maxBond {
			v.metrics.RecordUnexpectedClaimResolution()
			v.logger.Warn("Encountered unexpected claim resolution", "game", game.Proxy, "counter", claim.CounteredBy)
		}
	}
	return nil
}

func (v *claimValidator) isClaimant(claimant common.Address) bool {
	for _, c := range v.claimants {
		if c == claimant {
			return true
		}
	}
	return false
}
