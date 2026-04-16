package faultproofs

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type gameArena interface {
	AdvanceTime(duration time.Duration)
	L1Client() *ethclient.Client
	GetProposalRoot(ctx context.Context, l2SequenceNumber uint64) common.Hash
	CreateChallenger(ctx context.Context)
	CreateHonestActor(ctx context.Context) *disputegame.OutputHonestHelper
}

type outputGameArena struct {
	t    *testing.T
	sys  *e2esys.System
	game *disputegame.OutputCannonGameHelper
}

func (o *outputGameArena) AdvanceTime(duration time.Duration) {
	o.sys.AdvanceTime(duration)
}

func (o *outputGameArena) L1Client() *ethclient.Client {
	return o.sys.NodeClient("l1")
}

func (o *outputGameArena) GetProposalRoot(ctx context.Context, l2SequenceNumber uint64) common.Hash {
	output, err := o.sys.RollupClient("sequencer").OutputAtBlock(ctx, l2SequenceNumber)
	require.NoError(o.t, err)
	return common.Hash(output.OutputRoot)
}

func (o *outputGameArena) CreateChallenger(ctx context.Context) {
	o.game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(o.sys.Cfg.Secrets.Alice))
}

func (o *outputGameArena) CreateHonestActor(ctx context.Context) *disputegame.OutputHonestHelper {
	return o.game.CreateHonestActor(ctx, "sequencer", disputegame.WithPrivKey(o.sys.Cfg.Secrets.Mallory))
}

func createOutputGameArena(t *testing.T, sys *e2esys.System, game *disputegame.OutputCannonGameHelper) gameArena {
	return &outputGameArena{
		t:    t,
		sys:  sys,
		game: game,
	}
}

