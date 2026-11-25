// Package metrics_test stays external because mon imports metrics, while these tests exercise monitor components.
package metrics_test

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/bonds"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestMetricsExposeGamesWaitingForRootSource(t *testing.T) {
	metricer := metrics.NewMetrics()
	metricer.RecordGamesWaitingForRootSource(map[string]int{"zk": 2, "super-cannon-kona": 1})

	family := gatherMetricFamily(t, metricer, "op_dispute_mon_games_waiting_for_root_source")
	require.Len(t, family.Metric, 2)
	expected := map[string]float64{"zk": 2, "super-cannon-kona": 1}
	for _, sample := range family.Metric {
		labels := metricLabels(sample)
		require.Contains(t, expected, labels["game_type"])
		require.Equal(t, expected[labels["game_type"]], sample.GetGauge().GetValue())
	}

	metricer.RecordGamesWaitingForRootSource(map[string]int{"zk": 0})
	family = gatherMetricFamily(t, metricer, "op_dispute_mon_games_waiting_for_root_source")
	require.Len(t, family.Metric, 1)
	require.Equal(t, map[string]string{"game_type": "zk"}, metricLabels(family.Metric[0]))
	require.Zero(t, family.Metric[0].GetGauge().GetValue())
}

func TestForecastRecordsCanonicalAgreementSeries(t *testing.T) {
	// Mutation killed: omitting or relabeling one RecordGameAgreement call survives
	// mock-metric unit tests but changes the real registry's public series.
	metricer := metrics.NewMetrics()
	forecast := mon.NewForecast(testlog.Logger(t, 0), metricer)
	forecast.Forecast([]monTypes.EnrichedGame{
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusInProgress, AgreeWithClaim: true}, BlockNumberChallenged: true},
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusInProgress, AgreeWithClaim: false}, BlockNumberChallenged: true},
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusInProgress, AgreeWithClaim: true}},
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusInProgress, AgreeWithClaim: false}},
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusDefenderWon, AgreeWithClaim: true}},
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusDefenderWon, AgreeWithClaim: false}},
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusChallengerWon, AgreeWithClaim: true}},
		&monTypes.FaultGameData{CommonGameData: monTypes.CommonGameData{Status: types.GameStatusChallengerWon, AgreeWithClaim: false}},
	}, 0, 0)

	expected := map[string]map[string]string{
		"agree_challenger_ahead":    agreementLabels("agree_challenger_ahead", "in_progress", "incorrect", "agree"),
		"disagree_challenger_ahead": agreementLabels("disagree_challenger_ahead", "in_progress", "correct", "disagree"),
		"agree_defender_ahead":      agreementLabels("agree_defender_ahead", "in_progress", "correct", "agree"),
		"disagree_defender_ahead":   agreementLabels("disagree_defender_ahead", "in_progress", "incorrect", "disagree"),
		"agree_defender_wins":       agreementLabels("agree_defender_wins", "complete", "correct", "agree"),
		"disagree_defender_wins":    agreementLabels("disagree_defender_wins", "complete", "incorrect", "disagree"),
		"agree_challenger_wins":     agreementLabels("agree_challenger_wins", "complete", "incorrect", "agree"),
		"disagree_challenger_wins":  agreementLabels("disagree_challenger_wins", "complete", "correct", "disagree"),
	}
	family := gatherMetricFamily(t, metricer, "op_dispute_mon_games_agreement")
	require.Len(t, family.Metric, len(expected))
	for _, sample := range family.Metric {
		labels := metricLabels(sample)
		require.Equal(t, expected[labels["status"]], labels)
		require.Equal(t, float64(1), sample.GetGauge().GetValue())
	}

	forecast.Forecast(nil, 0, 0)
	family = gatherMetricFamily(t, metricer, "op_dispute_mon_games_agreement")
	require.Len(t, family.Metric, len(expected))
	for _, sample := range family.Metric {
		require.Zero(t, sample.GetGauge().GetValue())
	}
}

func TestBondsRecordsCanonicalHonestActorSeries(t *testing.T) {
	actor := common.Address{0xaa}
	other := common.Address{0xbb}
	metricer := metrics.NewMetrics()
	monitor := bonds.NewBonds(
		testlog.Logger(t, 0),
		metricer,
		clock.NewDeterministicClock(time.Unix(0, 0)),
		monTypes.NewHonestActors([]common.Address{actor}),
	)
	monitor.CheckBonds([]monTypes.BondedGame{&monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Bonds: []monTypes.BondRecord{
			{Depositor: actor, Amount: eth.Ether(1).ToBig()},
			{Depositor: actor, Recipient: other, Amount: eth.Ether(2).ToBig(), Resolved: true, Forfeited: true},
			{Depositor: other, Recipient: actor, Amount: eth.Ether(3).ToBig(), Resolved: true, Forfeited: true},
		},
		WETHContract:  common.Address{0xcc},
		ETHCollateral: eth.Ether(6).ToBig(),
	}}})

	expected := map[string]float64{"pending": 1, "lost": 2, "won": 3}
	family := gatherMetricFamily(t, metricer, "op_dispute_mon_honest_actor_bonds")
	require.Len(t, family.Metric, len(expected))
	for _, sample := range family.Metric {
		labels := metricLabels(sample)
		require.Equal(t, actor.Hex(), labels["honest_actor_address"])
		require.Equal(t, expected[labels["state"]], sample.GetGauge().GetValue())
	}

	monitor.CheckBonds(nil)
	family = gatherMetricFamily(t, metricer, "op_dispute_mon_honest_actor_bonds")
	require.Len(t, family.Metric, len(expected))
	for _, sample := range family.Metric {
		require.Zero(t, sample.GetGauge().GetValue())
	}
}

func gatherMetricFamily(t *testing.T, metricer *metrics.Metrics, name string) *dto.MetricFamily {
	t.Helper()
	families, err := metricer.Registry().Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	t.Fatalf("metric family %q not found", name)
	return nil
}

func agreementLabels(status, completion, correctness, agreement string) map[string]string {
	return map[string]string{
		"status":             status,
		"completion":         completion,
		"result_correctness": correctness,
		"root_agreement":     agreement,
	}
}

func metricLabels(metric *dto.Metric) map[string]string {
	labels := make(map[string]string, len(metric.Label))
	for _, label := range metric.Label {
		labels[label.GetName()] = label.GetValue()
	}
	return labels
}
