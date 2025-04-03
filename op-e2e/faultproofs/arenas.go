package faultproofs

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/interop"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

type superGameArena struct {
	t    *testing.T
	sys  interop.SuperSystem
	game *disputegame.SuperCannonGameHelper
}

func (s *superGameArena) AdvanceTime(duration time.Duration) {
	s.sys.AdvanceL1Time(duration)
}

func (s *superGameArena) L1Client() *ethclient.Client {
	return s.sys.L1GethClient()
}

func (s *superGameArena) GetProposalRoot(ctx context.Context, l2SequenceNumber uint64) common.Hash {
	output, err := s.sys.SupervisorClient().SuperRootAtTimestamp(ctx, hexutil.Uint64(l2SequenceNumber))
	require.NoError(s.t, err)
	return common.Hash(output.SuperRoot)
}

func (s *superGameArena) CreateChallenger(ctx context.Context) {
	s.game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(s.t)), challenger.WithDepset(s.t, s.sys.DependencySet()))
}

func (s *superGameArena) CreateHonestActor(ctx context.Context) *disputegame.OutputHonestHelper {
	return s.game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(s.t)), func(c *disputegame.HonestActorConfig) {
		c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(s.t, s.sys.DependencySet()))
	})
}

func createSuperGameArena(t *testing.T, sys interop.SuperSystem, game *disputegame.SuperCannonGameHelper) gameArena {
	return &superGameArena{
		t:    t,
		sys:  sys,
		game: game,
	}
}
