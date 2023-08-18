package disputegame

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type FaultGameHelper struct {
	t           *testing.T
	require     *require.Assertions
	client      *ethclient.Client
	opts        *bind.TransactOpts
	game        *bindings.FaultDisputeGame
	factoryAddr common.Address
	addr        common.Address
}

func (g *FaultGameHelper) GameDuration(ctx context.Context) time.Duration {
	duration, err := g.game.GAMEDURATION(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "failed to get game duration")
	return time.Duration(duration) * time.Second
}

func (g *FaultGameHelper) WaitForClaimCount(ctx context.Context, count int64) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
		actual, err := g.game.ClaimDataLen(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, err
		}
		g.t.Log("Waiting for claim count", "current", actual, "expected", count, "game", g.addr)
		return actual.Cmp(big.NewInt(count)) == 0, nil
	})
	g.require.NoErrorf(err, "Did not find expected claim count %v", count)
}

type ContractClaim struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}

func (g *FaultGameHelper) MaxDepth(ctx context.Context) int64 {
	depth, err := g.game.MAXGAMEDEPTH(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "Failed to load game depth")
	return depth.Int64()
}

func (g *FaultGameHelper) WaitForClaim(ctx context.Context, predicate func(claim ContractClaim) bool) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
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

// getClaim retrieves the claim data for a specific index.
// Note that it is deliberately not exported as tests should use WaitForClaim to avoid race conditions.
func (g *FaultGameHelper) getClaim(ctx context.Context, claimIdx int64) ContractClaim {
	claimData, err := g.game.ClaimData(&bind.CallOpts{Context: ctx}, big.NewInt(claimIdx))
	if err != nil {
		g.require.NoErrorf(err, "retrieve claim %v", claimIdx)
	}
	return claimData
}

func (g *FaultGameHelper) WaitForClaimAtMaxDepth(ctx context.Context, countered bool) {
	maxDepth := g.MaxDepth(ctx)
	g.WaitForClaim(ctx, func(claim ContractClaim) bool {
		pos := types.NewPositionFromGIndex(claim.Position.Uint64())
		return int64(pos.Depth()) == maxDepth && claim.Countered == countered
	})
}

func (g *FaultGameHelper) Resolve(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	tx, err := g.game.Resolve(g.opts)
	g.require.NoError(err)
	_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err)
}

func (g *FaultGameHelper) WaitForGameStatus(ctx context.Context, expected Status) {
	g.t.Logf("Waiting for game %v to have status %v", g.addr, expected)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
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

func (g *FaultGameHelper) Attack(ctx context.Context, claimIdx int64, claim common.Hash) {
	tx, err := g.game.Attack(g.opts, big.NewInt(claimIdx), claim)
	g.require.NoError(err, "Attack transaction did not send")
	_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err, "Attack transaction was not OK")
}

func (g *FaultGameHelper) Defend(ctx context.Context, claimIdx int64, claim common.Hash) {
	tx, err := g.game.Defend(g.opts, big.NewInt(claimIdx), claim)
	g.require.NoError(err, "Defend transaction did not send")
	_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err, "Defend transaction was not OK")
}

func (g *FaultGameHelper) LogGameData(ctx context.Context) {
	opts := &bind.CallOpts{Context: ctx}
	maxDepth := int(g.MaxDepth(ctx))
	claimCount, err := g.game.ClaimDataLen(opts)
	info := fmt.Sprintf("Claim count: %v\n", claimCount)
	g.require.NoError(err, "Fetching claim count")
	for i := int64(0); i < claimCount.Int64(); i++ {
		claim, err := g.game.ClaimData(opts, big.NewInt(i))
		g.require.NoErrorf(err, "Fetch claim %v", i)

		pos := types.NewPositionFromGIndex(claim.Position.Uint64())
		info = info + fmt.Sprintf("%v - Position: %v, Depth: %v, IndexAtDepth: %v Trace Index: %v, Value: %v, Countered: %v\n",
			i, claim.Position.Int64(), pos.Depth(), pos.IndexAtDepth(), pos.TraceIndex(maxDepth), common.Hash(claim.Claim).Hex(), claim.Countered)
	}
	status, err := g.game.Status(opts)
	g.require.NoError(err, "Load game status")
	g.t.Logf("Game %v (%v):\n%v\n", g.addr, Status(status), info)
}
