package disputegame

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type FaultGameHelper struct {
	t        *testing.T
	require  *require.Assertions
	client   *ethclient.Client
	opts     *bind.TransactOpts
	game     *bindings.FaultDisputeGame
	maxDepth int
	addr     common.Address
}

func (g *FaultGameHelper) GameDuration(ctx context.Context) time.Duration {
	duration, err := g.game.GAMEDURATION(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "failed to get game duration")
	return time.Duration(duration) * time.Second
}

func (g *FaultGameHelper) WaitForClaimCount(ctx context.Context, count int64) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	err := utils.WaitFor(ctx, time.Second, func() (bool, error) {
		actual, err := g.game.ClaimDataLen(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, err
		}
		g.t.Log("Waiting for claim count", "current", actual, "expected", count, "game", g.addr)
		return actual.Cmp(big.NewInt(count)) == 0, nil
	})
	g.require.NoError(err)
}

type ContractClaim struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}

func (g *FaultGameHelper) WaitForClaim(ctx context.Context, predicate func(claim ContractClaim) bool) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	err := utils.WaitFor(ctx, time.Second, func() (bool, error) {
		count, err := g.game.ClaimDataLen(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, fmt.Errorf("retrieve number of claims: %w", err)
		}
		// Search backwards because the new claims are at the end and more likely the ones we want.
		for i := count.Int64() - 1; i >= 0; i-- {
			claimData, err := g.game.ClaimData(&bind.CallOpts{Context: ctx}, big.NewInt(i))
			if err != nil {
				return false, fmt.Errorf("retrieve claim %v: %w", i, err)
			}
			if predicate(claimData) {
				return true, nil
			}
		}
		return false, nil
	})
	g.require.NoError(err)
}

func (g *FaultGameHelper) WaitForClaimAtMaxDepth(ctx context.Context, countered bool) {
	g.WaitForClaim(ctx, func(claim ContractClaim) bool {
		pos := types.NewPositionFromGIndex(claim.Position.Uint64())
		return pos.Depth() == g.maxDepth && claim.Countered == countered
	})
}

func (g *FaultGameHelper) Resolve(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	tx, err := g.game.Resolve(g.opts)
	g.require.NoError(err)
	_, err = utils.WaitReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err)
}

func (g *FaultGameHelper) WaitForGameStatus(ctx context.Context, expected Status) {
	g.t.Logf("Waiting for game %v to have status %v", g.addr, expected)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	err := utils.WaitFor(ctx, time.Second, func() (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		status, err := g.game.Status(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, fmt.Errorf("game status unavailable: %w", err)
		}
		g.t.Logf("Game %v has state %v, waiting for state %v", g.addr, Status(status), expected)
		return expected == Status(status), nil
	})
	g.require.NoError(err, "wait for game status")
}
