package disputegame

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const defaultTimeout = 5 * time.Minute

type OutputGameHelper struct {
	t                     *testing.T
	require               *require.Assertions
	client                *ethclient.Client
	opts                  *bind.TransactOpts
	game                  *bindings.OutputBisectionGame
	factoryAddr           common.Address
	addr                  common.Address
	correctOutputProvider *outputs.OutputTraceProvider
	system                DisputeSystem
}

func (g *OutputGameHelper) Addr() common.Address {
	return g.addr
}

func (g *OutputGameHelper) SplitDepth(ctx context.Context) int64 {
	splitDepth, err := g.game.SplitDepth(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "failed to load split depth")
	return splitDepth.Int64()
}

func (g *OutputGameHelper) L2BlockNum(ctx context.Context) uint64 {
	blockNum, err := g.game.L2BlockNumber(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "failed to load l2 block number")
	return blockNum.Uint64()
}

func (g *OutputGameHelper) GenesisBlockNum(ctx context.Context) uint64 {
	blockNum, err := g.game.GenesisBlockNumber(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "failed to load genesis block number")
	return blockNum.Uint64()
}

// DisputeLastBlock posts claims from both the honest and dishonest actor to progress the output root part of the game
// through to the split depth and the claims are setup such that the last block in the game range is the block
// to execute cannon on. ie the first block the honest and dishonest actors disagree about is the l2 block of the game.
func (g *OutputGameHelper) DisputeLastBlock(ctx context.Context) *ClaimHelper {
	rootClaim := g.GetClaimValue(ctx, 0)
	disputeBlockNum := g.L2BlockNum(ctx)
	pos := types.NewPositionFromGIndex(big.NewInt(1))
	getClaimValue := func(parentClaim *ClaimHelper, claimPos types.Position) common.Hash {
		claimBlockNum, err := g.correctOutputProvider.BlockNumber(claimPos)
		g.require.NoError(err, "failed to calculate claim block number")
		// Use the correct output root for the challenger and incorrect for the defender
		if parentClaim.AgreesWithOutputRoot() || claimBlockNum < disputeBlockNum {
			return g.correctOutputRoot(ctx, claimPos)
		} else {
			return rootClaim
		}
	}

	claim := g.RootClaim(ctx)
	for !claim.IsOutputRootLeaf(ctx) {
		parentClaimBlockNum, err := g.correctOutputProvider.BlockNumber(pos)
		g.require.NoError(err, "failed to calculate parent claim block number")
		if parentClaimBlockNum >= disputeBlockNum {
			pos = pos.Attack()
			claim = claim.Attack(ctx, getClaimValue(claim, pos))
		} else {
			pos = pos.Defend()
			claim = claim.Defend(ctx, getClaimValue(claim, pos))
		}
	}
	return claim
}

func (g *OutputGameHelper) RootClaim(ctx context.Context) *ClaimHelper {
	claim := g.getClaim(ctx, 0)
	return newClaimHelper(g, 0, claim)
}

func (g *OutputGameHelper) WaitForCorrectOutputRoot(ctx context.Context, claimIdx int64) {
	g.WaitForClaimCount(ctx, claimIdx+1)
	claim := g.getClaim(ctx, claimIdx)
	output := g.correctOutputRoot(ctx, types.NewPositionFromGIndex(claim.Position))
	g.require.EqualValuesf(output, claim.Claim, "Incorrect output root at claim %v at position %v", claimIdx, claim.Position.Uint64())
}

func (g *OutputGameHelper) correctOutputRoot(ctx context.Context, pos types.Position) common.Hash {
	outputRoot, err := g.correctOutputProvider.Get(ctx, pos)
	g.require.NoErrorf(err, "Failed to get correct output for position %v", pos)
	return outputRoot
}

func (g *OutputGameHelper) GameDuration(ctx context.Context) time.Duration {
	duration, err := g.game.GameDuration(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "failed to get game duration")
	return time.Duration(duration) * time.Second
}

// WaitForClaimCount waits until there are at least count claims in the game.
// This does not check that the number of claims is exactly the specified count to avoid intermittent failures
// where a challenger posts an additional claim before this method sees the number of claims it was waiting for.
func (g *OutputGameHelper) WaitForClaimCount(ctx context.Context, count int64) {
	timedCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		actual, err := g.game.ClaimDataLen(&bind.CallOpts{Context: timedCtx})
		if err != nil {
			return false, err
		}
		g.t.Log("Waiting for claim count", "current", actual, "expected", count, "game", g.addr)
		return actual.Cmp(big.NewInt(count)) >= 0, nil
	})
	if err != nil {
		g.LogGameData(ctx)
		g.require.NoErrorf(err, "Did not find expected claim count %v", count)
	}
}

type ContractClaim struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}

func (g *OutputGameHelper) MaxDepth(ctx context.Context) int64 {
	depth, err := g.game.MaxGameDepth(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "Failed to load game depth")
	return depth.Int64()
}

func (g *OutputGameHelper) waitForClaim(ctx context.Context, errorMsg string, predicate func(claim ContractClaim) bool) (int64, ContractClaim) {
	timedCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	var matchedClaim ContractClaim
	var matchClaimIdx int64
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		count, err := g.game.ClaimDataLen(&bind.CallOpts{Context: timedCtx})
		if err != nil {
			return false, fmt.Errorf("retrieve number of claims: %w", err)
		}
		// Search backwards because the new claims are at the end and more likely the ones we want.
		for i := count.Int64() - 1; i >= 0; i-- {
			claimData, err := g.game.ClaimData(&bind.CallOpts{Context: timedCtx}, big.NewInt(i))
			if err != nil {
				return false, fmt.Errorf("retrieve claim %v: %w", i, err)
			}
			if predicate(claimData) {
				matchClaimIdx = i
				matchedClaim = claimData
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil { // Avoid waiting time capturing game data when there's no error
		g.require.NoErrorf(err, "%v\n%v", errorMsg, g.gameData(ctx))
	}
	return matchClaimIdx, matchedClaim
}

func (g *OutputGameHelper) waitForNoClaim(ctx context.Context, errorMsg string, predicate func(claim ContractClaim) bool) {
	timedCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		count, err := g.game.ClaimDataLen(&bind.CallOpts{Context: timedCtx})
		if err != nil {
			return false, fmt.Errorf("retrieve number of claims: %w", err)
		}
		// Search backwards because the new claims are at the end and more likely the ones we will fail on.
		for i := count.Int64() - 1; i >= 0; i-- {
			claimData, err := g.game.ClaimData(&bind.CallOpts{Context: timedCtx}, big.NewInt(i))
			if err != nil {
				return false, fmt.Errorf("retrieve claim %v: %w", i, err)
			}
			if predicate(claimData) {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil { // Avoid waiting time capturing game data when there's no error
		g.require.NoErrorf(err, "%v\n%v", errorMsg, g.gameData(ctx))
	}
}

func (g *OutputGameHelper) GetClaimValue(ctx context.Context, claimIdx int64) common.Hash {
	g.WaitForClaimCount(ctx, claimIdx+1)
	claim := g.getClaim(ctx, claimIdx)
	return claim.Claim
}

func (g *OutputGameHelper) GetClaimPosition(ctx context.Context, claimIdx int64) types.Position {
	g.WaitForClaimCount(ctx, claimIdx+1)
	claim := g.getClaim(ctx, claimIdx)
	return types.NewPositionFromGIndex(claim.Position)
}

// getClaim retrieves the claim data for a specific index.
// Note that it is deliberately not exported as tests should use WaitForClaim to avoid race conditions.
func (g *OutputGameHelper) getClaim(ctx context.Context, claimIdx int64) ContractClaim {
	claimData, err := g.game.ClaimData(&bind.CallOpts{Context: ctx}, big.NewInt(claimIdx))
	if err != nil {
		g.require.NoErrorf(err, "retrieve claim %v", claimIdx)
	}
	return claimData
}

func (g *OutputGameHelper) WaitForClaimAtDepth(ctx context.Context, depth int) {
	g.waitForClaim(
		ctx,
		fmt.Sprintf("Could not find claim depth %v", depth),
		func(claim ContractClaim) bool {
			pos := types.NewPositionFromGIndex(claim.Position)
			return pos.Depth() == depth
		})
}

func (g *OutputGameHelper) WaitForClaimAtMaxDepth(ctx context.Context, countered bool) {
	maxDepth := g.MaxDepth(ctx)
	g.waitForClaim(
		ctx,
		fmt.Sprintf("Could not find claim depth %v with countered=%v", maxDepth, countered),
		func(claim ContractClaim) bool {
			pos := types.NewPositionFromGIndex(claim.Position)
			return int64(pos.Depth()) == maxDepth && claim.Countered == countered
		})
}

func (g *OutputGameHelper) WaitForAllClaimsCountered(ctx context.Context) {
	g.waitForNoClaim(
		ctx,
		"Did not find all claims countered",
		func(claim ContractClaim) bool {
			return !claim.Countered
		})
}

func (g *OutputGameHelper) Resolve(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	tx, err := g.game.Resolve(g.opts)
	g.require.NoError(err)
	_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err)
}

func (g *OutputGameHelper) Status(ctx context.Context) Status {
	status, err := g.game.Status(&bind.CallOpts{Context: ctx})
	g.require.NoError(err)
	return Status(status)
}

func (g *OutputGameHelper) WaitForGameStatus(ctx context.Context, expected Status) {
	g.t.Logf("Waiting for game %v to have status %v", g.addr, expected)
	timedCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		ctx, cancel := context.WithTimeout(timedCtx, 30*time.Second)
		defer cancel()
		status, err := g.game.Status(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, fmt.Errorf("game status unavailable: %w", err)
		}
		g.t.Logf("Game %v has state %v, waiting for state %v", g.addr, Status(status), expected)
		return expected == Status(status), nil
	})
	g.require.NoErrorf(err, "wait for game status. Game state: \n%v", g.gameData(ctx))
}

func (g *OutputGameHelper) WaitForInactivity(ctx context.Context, numInactiveBlocks int, untilGameEnds bool) {
	g.t.Logf("Waiting for game %v to have no activity for %v blocks", g.addr, numInactiveBlocks)
	headCh := make(chan *gethtypes.Header, 100)
	headSub, err := g.client.SubscribeNewHead(ctx, headCh)
	g.require.NoError(err)
	defer headSub.Unsubscribe()

	var lastActiveBlock uint64
	for {
		if untilGameEnds && g.Status(ctx) != StatusInProgress {
			break
		}
		select {
		case head := <-headCh:
			if lastActiveBlock == 0 {
				lastActiveBlock = head.Number.Uint64()
				continue
			} else if lastActiveBlock+uint64(numInactiveBlocks) < head.Number.Uint64() {
				return
			}
			block, err := g.client.BlockByNumber(ctx, head.Number)
			g.require.NoError(err)
			numActions := 0
			for _, tx := range block.Transactions() {
				if tx.To().Hex() == g.addr.Hex() {
					numActions++
				}
			}
			if numActions != 0 {
				g.t.Logf("Game %v has %v actions in block %d. Resetting inactivity timeout", g.addr, numActions, block.NumberU64())
				lastActiveBlock = head.Number.Uint64()
			}
		case err := <-headSub.Err():
			g.require.NoError(err)
		case <-ctx.Done():
			g.require.Fail("Context canceled", ctx.Err())
		}
	}
}

// Mover is a function that either attacks or defends the claim at parentClaimIdx
type Mover func(parentClaimIdx int64)

// Stepper is a function that attempts to perform a step against the claim at parentClaimIdx
type Stepper func(parentClaimIdx int64)

// DefendRootClaim uses the supplied Mover to perform moves in an attempt to defend the root claim.
// It is assumed that the output root being disputed is valid and that an honest op-challenger is already running.
// When the game has reached the maximum depth it waits for the honest challenger to counter the leaf claim with step.
func (g *OutputGameHelper) DefendRootClaim(ctx context.Context, performMove Mover) {
	maxDepth := g.MaxDepth(ctx)
	for claimCount := g.getClaimCount(ctx); claimCount < maxDepth; {
		g.LogGameData(ctx)
		claimCount++
		// Wait for the challenger to counter
		g.WaitForClaimCount(ctx, claimCount)
		g.LogGameData(ctx)

		// Respond with our own move
		performMove(claimCount - 1)
		claimCount++
		g.WaitForClaimCount(ctx, claimCount)
	}

	// Wait for the challenger to call step and counter our invalid claim
	g.WaitForClaimAtMaxDepth(ctx, true)
}

// ChallengeRootClaim uses the supplied Mover and Stepper to perform moves and steps in an attempt to challenge the root claim.
// It is assumed that the output root being disputed is invalid and that an honest op-challenger is already running.
// When the game has reached the maximum depth it calls the Stepper to attempt to counter the leaf claim.
// Since the output root is invalid, it should not be possible for the Stepper to call step successfully.
func (g *OutputGameHelper) ChallengeRootClaim(ctx context.Context, performMove Mover, attemptStep Stepper) {
	maxDepth := g.MaxDepth(ctx)

	for claimCount := g.getClaimCount(ctx); claimCount < maxDepth; {
		g.LogGameData(ctx)
		// Perform our move
		performMove(claimCount - 1)
		claimCount++
		g.WaitForClaimCount(ctx, claimCount)

		// Wait for the challenger to counter
		claimCount++
		g.WaitForClaimCount(ctx, claimCount)
	}

	// Confirm the game has reached max depth and the last claim hasn't been countered
	g.WaitForClaimAtMaxDepth(ctx, false)
	g.LogGameData(ctx)

	// It's on us to call step if we want to win but shouldn't be possible
	attemptStep(maxDepth)
}

func (g *OutputGameHelper) getClaimCount(ctx context.Context) int64 {
	claimCount, err := g.game.ClaimDataLen(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "Failed to get current claim count")
	return claimCount.Int64()
}

func (g *OutputGameHelper) WaitForNewClaim(ctx context.Context, checkPoint int64) (int64, error) {
	return g.waitForNewClaim(ctx, checkPoint, defaultTimeout)
}

func (g *OutputGameHelper) waitForNewClaim(ctx context.Context, checkPoint int64, timeout time.Duration) (int64, error) {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var newClaimLen int64
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		actual, err := g.game.ClaimDataLen(&bind.CallOpts{Context: ctx})
		if err != nil {
			return false, err
		}
		newClaimLen = actual.Int64()
		return actual.Cmp(big.NewInt(checkPoint)) > 0, nil
	})
	return newClaimLen, err
}

func (g *OutputGameHelper) Attack(ctx context.Context, claimIdx int64, claim common.Hash) {
	tx, err := g.game.Attack(g.opts, big.NewInt(claimIdx), claim)
	if err != nil {
		g.require.NoErrorf(err, "Attack transaction did not send. Game state: \n%v", g.gameData(ctx))
	}
	_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
	if err != nil {
		g.require.NoErrorf(err, "Attack transaction was not OK. Game state: \n%v", g.gameData(ctx))
	}
}

func (g *OutputGameHelper) Defend(ctx context.Context, claimIdx int64, claim common.Hash) {
	tx, err := g.game.Defend(g.opts, big.NewInt(claimIdx), claim)
	if err != nil {
		g.require.NoErrorf(err, "Defend transaction did not send. Game state: \n%v", g.gameData(ctx))
	}
	_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
	if err != nil {
		g.require.NoErrorf(err, "Defend transaction was not OK. Game state: \n%v", g.gameData(ctx))
	}
}

type ErrWithData interface {
	ErrorData() interface{}
}

// StepFails attempts to call step and verifies that it fails with ValidStep()
func (g *OutputGameHelper) StepFails(claimIdx int64, isAttack bool, stateData []byte, proof []byte) {
	g.t.Logf("Attempting step against claim %v isAttack: %v", claimIdx, isAttack)
	_, err := g.game.Step(g.opts, big.NewInt(claimIdx), isAttack, stateData, proof)
	errData, ok := err.(ErrWithData)
	g.require.Truef(ok, "Error should provide ErrorData method: %v", err)
	g.require.Equal("0xfb4e40dd", errData.ErrorData(), "Revert reason should be abi encoded ValidStep()")
}

// ResolveClaim resolves a single subgame
func (g *OutputGameHelper) ResolveClaim(ctx context.Context, claimIdx int64) {
	tx, err := g.game.ResolveClaim(g.opts, big.NewInt(claimIdx))
	g.require.NoError(err, "ResolveClaim transaction did not send")
	_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
	g.require.NoError(err, "ResolveClaim transaction was not OK")
}

func (g *OutputGameHelper) gameData(ctx context.Context) string {
	opts := &bind.CallOpts{Context: ctx}
	maxDepth := int(g.MaxDepth(ctx))
	splitDepth := int(g.SplitDepth(ctx))
	claimCount, err := g.game.ClaimDataLen(opts)
	info := fmt.Sprintf("Claim count: %v\n", claimCount)
	g.require.NoError(err, "Fetching claim count")
	for i := int64(0); i < claimCount.Int64(); i++ {
		claim, err := g.game.ClaimData(opts, big.NewInt(i))
		g.require.NoErrorf(err, "Fetch claim %v", i)

		pos := types.NewPositionFromGIndex(claim.Position)
		extra := ""
		if pos.Depth() <= splitDepth {
			blockNum, err := g.correctOutputProvider.BlockNumber(pos)
			if err != nil {
			} else {
				extra = fmt.Sprintf("Block num: %v", blockNum)
			}
		}
		info = info + fmt.Sprintf("%v - Position: %v, Depth: %v, IndexAtDepth: %v Trace Index: %v, Value: %v, Countered: %v, ParentIndex: %v %v\n",
			i, claim.Position.Int64(), pos.Depth(), pos.IndexAtDepth(), pos.TraceIndex(maxDepth), common.Hash(claim.Claim).Hex(), claim.Countered, claim.ParentIndex, extra)
	}
	l2BlockNum := g.L2BlockNum(ctx)
	status, err := g.game.Status(opts)
	g.require.NoError(err, "Load game status")
	return fmt.Sprintf("Game %v - %v - L2 Block: %v - Split Depth: %v - Max Depth: %v:\n%v\n",
		g.addr, Status(status), l2BlockNum, splitDepth, maxDepth, info)
}

func (g *OutputGameHelper) LogGameData(ctx context.Context) {
	g.t.Log(g.gameData(ctx))
}
