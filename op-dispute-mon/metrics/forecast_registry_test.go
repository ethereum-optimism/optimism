package metrics_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

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
