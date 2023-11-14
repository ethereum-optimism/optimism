package disputegame

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
)

type OutputCannonGameHelper struct {
	FaultGameHelper
	rollupClient *sources.RollupClient
}

func (g *OutputCannonGameHelper) StartChallenger(
	ctx context.Context,
	rollupCfg *rollup.Config,
	l2Genesis *core.Genesis,
	rollupEndpoint string,
	l1Endpoint string,
	l2Endpoint string,
	name string,
	options ...challenger.Option,
) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithOutputCannon(g.t, rollupCfg, l2Genesis, rollupEndpoint, l2Endpoint),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, l1Endpoint, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

func (g *OutputCannonGameHelper) WaitForCorrectOutputRoot(ctx context.Context, claimIdx int64) {
	g.WaitForClaimCount(ctx, claimIdx+1)
	claim := g.getClaim(ctx, claimIdx)
	err, blockNum := g.blockNumForClaim(ctx, claim)
	g.require.NoError(err)
	output, err := g.rollupClient.OutputAtBlock(ctx, blockNum)
	g.require.NoErrorf(err, "Failed to get output at block %v", blockNum)
	g.require.EqualValuesf(output.OutputRoot, claim.Claim, "Incorrect output root at claim %v. Expected to be from block %v", claimIdx, blockNum)
}

func (g *OutputCannonGameHelper) blockNumForClaim(ctx context.Context, claim ContractClaim) (error, uint64) {
	proposals, err := g.game.Proposals(&bind.CallOpts{Context: ctx})
	g.require.NoError(err, "failed to retrieve proposals")
	prestateBlockNum := proposals.Starting.L2BlockNumber
	disputedBlockNum := proposals.Disputed.L2BlockNumber
	gameDepth := g.MaxDepth(ctx)

	// TODO(client-pod#43): Load this from the contract
	topDepth := gameDepth / 2
	traceIdx := types.NewPositionFromGIndex(claim.Position).TraceIndex(int(topDepth))
	blockNum := new(big.Int).Add(prestateBlockNum, traceIdx).Uint64() + 1
	if blockNum > disputedBlockNum.Uint64() {
		blockNum = disputedBlockNum.Uint64()
	}
	return err, blockNum
}
