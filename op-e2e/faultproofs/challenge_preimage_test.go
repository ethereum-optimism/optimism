package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestChallengeLargePreimages_ChallengeFirst(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	disputeGameFactory.StartChallenger(ctx, "Challenger",
		challenger.WithAlphabet(sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice))
	preimageHelper := disputeGameFactory.PreimageHelper(ctx)
	ident := preimageHelper.UploadLargePreimage(ctx, preimage.MinPreimageSize,
		preimage.WithReplacedCommitment(0, common.Hash{0xaa}))

	require.NotEqual(t, ident.Claimant, common.Address{})

	preimageHelper.WaitForChallenged(ctx, ident)
}

func TestChallengeLargePreimages_ChallengeMiddle(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	disputeGameFactory.StartChallenger(ctx, "Challenger",
		challenger.WithAlphabet(sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))
	preimageHelper := disputeGameFactory.PreimageHelper(ctx)
	ident := preimageHelper.UploadLargePreimage(ctx, preimage.MinPreimageSize,
		preimage.WithReplacedCommitment(10, common.Hash{0xaa}))

	require.NotEqual(t, ident.Claimant, common.Address{})

	preimageHelper.WaitForChallenged(ctx, ident)
}

func TestChallengeLargePreimages_ChallengeLast(t *testing.T) {
	op_e2e.InitParallel(t)
	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	disputeGameFactory.StartChallenger(ctx, "Challenger",
		challenger.WithAlphabet(sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))
	preimageHelper := disputeGameFactory.PreimageHelper(ctx)
	ident := preimageHelper.UploadLargePreimage(ctx, preimage.MinPreimageSize,
		preimage.WithLastCommitment(common.Hash{0xaa}))

	require.NotEqual(t, ident.Claimant, common.Address{})

	preimageHelper.WaitForChallenged(ctx, ident)
}
