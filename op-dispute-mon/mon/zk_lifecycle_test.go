package mon

import (
	"math"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestZKLifecyclePendingActions(t *testing.T) {
	now := time.Unix(1_000, 0)
	inProgress := gameTypes.GameStatusInProgress
	defenderWon := gameTypes.GameStatusDefenderWon
	challengerWon := gameTypes.GameStatusChallengerWon

	tests := []struct {
		name             string
		game             *monTypes.ZKGameData
		wantResolution   int
		wantDistribution int
	}{
		{name: "live before deadline", game: zkLifecycleGame(now)},
		{name: "deadline is not expired at equality", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.Deadline = now
		})},
		{name: "expired deadline", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.Deadline = now.Add(-time.Second)
		}), wantResolution: 1},
		{name: "unchallenged valid proof", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.ProposalStatus = contracts.ProposalStatusUnchallengedAndValidProofProvided
		}), wantResolution: 1},
		{name: "challenged valid proof", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.ProposalStatus = contracts.ProposalStatusChallengedAndValidProofProvided
		}), wantResolution: 1},
		{name: "live parent blocks resolution", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.ParentIndex = 1
			game.ParentStatus = &inProgress
			game.ProposalStatus = contracts.ProposalStatusChallengedAndValidProofProvided
			game.Deadline = now.Add(-time.Second)
		})},
		{name: "challenger won parent", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.ParentIndex = 1
			game.ParentStatus = &challengerWon
		}), wantResolution: 1},
		{name: "defender won parent still needs another trigger", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.ParentIndex = 1
			game.ParentStatus = &defenderWon
		})},
		{name: "defender won parent falls through to deadline", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.ParentIndex = 1
			game.ParentStatus = &defenderWon
			game.Deadline = now.Add(-time.Second)
		}), wantResolution: 1},
		{name: "resolved proposal is not pending", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.ProposalStatus = contracts.ProposalStatusResolved
			game.Deadline = now.Add(-time.Second)
		})},
		{name: "terminal undecided distribution", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.Status = gameTypes.GameStatusDefenderWon
			game.ProposalStatus = contracts.ProposalStatusResolved
		}), wantDistribution: 1},
		{name: "terminal normal distribution", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.Status = gameTypes.GameStatusDefenderWon
			game.ProposalStatus = contracts.ProposalStatusResolved
			game.BondDistributionMode = faultTypes.NormalDistributionMode
		})},
		{name: "terminal refund distribution", game: zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
			game.Status = gameTypes.GameStatusChallengerWon
			game.ProposalStatus = contracts.ProposalStatusResolved
			game.BondDistributionMode = faultTypes.RefundDistributionMode
		})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metricer := metrics.NewMetrics()
			monitor := NewZKLifecycleMonitor(clock.NewDeterministicClock(now), metricer)
			monitor.CheckLifecycle([]*monTypes.ZKGameData{test.game})
			require.Equal(t, map[string]float64{
				"resolution":        float64(test.wantResolution),
				"bond_distribution": float64(test.wantDistribution),
			}, gatherZKLifecycleActions(t, metricer))
		})
	}

	t.Run("empty next poll zeroes both actions", func(t *testing.T) {
		metricer := metrics.NewMetrics()
		monitor := NewZKLifecycleMonitor(clock.NewDeterministicClock(now), metricer)
		monitor.CheckLifecycle([]*monTypes.ZKGameData{
			zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
				game.Deadline = now.Add(-time.Second)
			}),
			zkLifecycleGame(now, func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
			}),
		})
		monitor.CheckLifecycle(nil)
		require.Equal(t, map[string]float64{"resolution": 0, "bond_distribution": 0}, gatherZKLifecycleActions(t, metricer))
	})
}

func zkLifecycleGame(now time.Time, apply ...func(*monTypes.ZKGameData)) *monTypes.ZKGameData {
	game := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
		ParentIndex:    math.MaxUint32,
		ProposalStatus: contracts.ProposalStatusUnchallenged,
		Deadline:       now.Add(time.Hour),
	}
	for _, fn := range apply {
		fn(game)
	}
	return game
}

func gatherZKLifecycleActions(t *testing.T, metricer *metrics.Metrics) map[string]float64 {
	t.Helper()
	families, err := metricer.Registry().Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "op_dispute_mon_zk_games_pending_lifecycle_action" {
			continue
		}
		require.Len(t, family.Metric, 2)
		return zkLifecycleValues(family)
	}
	t.Fatal("ZK lifecycle metric family not found")
	return nil
}

func zkLifecycleValues(family *dto.MetricFamily) map[string]float64 {
	values := make(map[string]float64, len(family.Metric))
	for _, metric := range family.Metric {
		values[metric.Label[0].GetValue()] = metric.GetGauge().GetValue()
	}
	return values
}
