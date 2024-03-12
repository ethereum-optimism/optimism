package bonds

import (
	"context"
	"fmt"
	"math/big"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/maps"
)

type BondContract interface {
	GetCredits(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error)
}

// CalculateRequiredCollateral determines the minimum balance required for a fault dispute game contract in order
// to pay the outstanding bonds and credits.
// It returns the sum of unpaid bonds from claims, plus the sum of allocated but unclaimed credits.
func CalculateRequiredCollateral(ctx context.Context, contract BondContract, blockHash common.Hash, claims []faultTypes.Claim) (*big.Int, error) {
	unpaidBonds := big.NewInt(0)
	recipients := make(map[common.Address]bool)
	for _, claim := range claims {
		if monTypes.ResolvedBondAmount.Cmp(claim.Bond) != 0 {
			unpaidBonds = new(big.Int).Add(unpaidBonds, claim.Bond)
		}
		recipients[claim.Claimant] = true
		if claim.CounteredBy != (common.Address{}) {
			recipients[claim.CounteredBy] = true
		}
	}

	credits, err := contract.GetCredits(ctx, rpcblock.ByHash(blockHash), maps.Keys(recipients)...)
	if err != nil {
		return nil, fmt.Errorf("failed to load credits: %w", err)
	}
	for _, credit := range credits {
		unpaidBonds = new(big.Int).Add(unpaidBonds, credit)
	}
	return unpaidBonds, nil
}
