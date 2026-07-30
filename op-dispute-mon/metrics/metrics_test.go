package metrics

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestRecordGameAgreementsPreservesCanonicalSeries(t *testing.T) {
	metricer := NewMetrics()
	metricer.RecordGameAgreements(nil)

	initial := gatherGameAgreements(t, metricer)
	require.Len(t, initial.Metric, 8)
	expectedLabels := map[string]map[string]string{
		"agree_challenger_ahead":    agreementLabels("agree_challenger_ahead", "in_progress", "incorrect", "agree"),
		"disagree_challenger_ahead": agreementLabels("disagree_challenger_ahead", "in_progress", "correct", "disagree"),
		"agree_defender_ahead":      agreementLabels("agree_defender_ahead", "in_progress", "correct", "agree"),
		"disagree_defender_ahead":   agreementLabels("disagree_defender_ahead", "in_progress", "incorrect", "disagree"),
		"agree_defender_wins":       agreementLabels("agree_defender_wins", "complete", "correct", "agree"),
		"disagree_defender_wins":    agreementLabels("disagree_defender_wins", "complete", "incorrect", "disagree"),
		"agree_challenger_wins":     agreementLabels("agree_challenger_wins", "complete", "incorrect", "agree"),
		"disagree_challenger_wins":  agreementLabels("disagree_challenger_wins", "complete", "correct", "disagree"),
	}
	for _, metric := range initial.Metric {
		labels := metricLabels(metric)
		require.Equal(t, expectedLabels[labels["status"]], labels)
		require.Zero(t, metric.GetGauge().GetValue())
	}

	metricer.RecordGameAgreements(map[GameAgreementStatus]int{
		AgreeDefenderAhead:     6,
		DisagreeChallengerWins: 2,
	})
	firstCycle := gatherGameAgreements(t, metricer)
	require.Len(t, firstCycle.Metric, len(initial.Metric))
	require.Equal(t, float64(6), agreementValue(t, firstCycle, map[string]string{
		"status":             "agree_defender_ahead",
		"completion":         "in_progress",
		"result_correctness": "correct",
		"root_agreement":     "agree",
	}))
	require.Equal(t, float64(2), agreementValue(t, firstCycle, map[string]string{
		"status":             "disagree_challenger_wins",
		"completion":         "complete",
		"result_correctness": "correct",
		"root_agreement":     "disagree",
	}))

	metricer.RecordGameAgreements(map[GameAgreementStatus]int{
		AgreeChallengerAhead: 3,
	})
	secondCycle := gatherGameAgreements(t, metricer)
	require.Len(t, secondCycle.Metric, len(initial.Metric))
	require.Equal(t, float64(0), agreementValue(t, secondCycle, map[string]string{
		"status":             "agree_defender_ahead",
		"completion":         "in_progress",
		"result_correctness": "correct",
		"root_agreement":     "agree",
	}))
	require.Equal(t, float64(3), agreementValue(t, secondCycle, map[string]string{
		"status":             "agree_challenger_ahead",
		"completion":         "in_progress",
		"result_correctness": "incorrect",
		"root_agreement":     "agree",
	}))
}

func gatherGameAgreements(t *testing.T, metricer *Metrics) *dto.MetricFamily {
	t.Helper()
	families, err := metricer.Registry().Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() == "op_dispute_mon_games_agreement" {
			return family
		}
	}
	t.Fatal("op_dispute_mon_games_agreement was not gathered")
	return nil
}

func agreementLabels(status, completion, correctness, rootAgreement string) map[string]string {
	return map[string]string{
		"status":             status,
		"completion":         completion,
		"result_correctness": correctness,
		"root_agreement":     rootAgreement,
	}
}

func metricLabels(metric *dto.Metric) map[string]string {
	labels := make(map[string]string, len(metric.Label))
	for _, label := range metric.Label {
		labels[label.GetName()] = label.GetValue()
	}
	return labels
}

func agreementValue(t *testing.T, family *dto.MetricFamily, labels map[string]string) float64 {
	t.Helper()
	for _, metric := range family.Metric {
		actual := metricLabels(metric)
		if len(actual) != len(labels) {
			continue
		}
		matches := true
		for name, value := range labels {
			if actual[name] != value {
				matches = false
				break
			}
		}
		if matches {
			return metric.GetGauge().GetValue()
		}
	}
	t.Fatalf("metric with labels %v not found", labels)
	return 0
}
