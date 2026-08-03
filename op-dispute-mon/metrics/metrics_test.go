package metrics

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestGameAgreementStatusSentinelIsLast(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "metrics.go", nil, 0)
	require.NoError(t, err)
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		declaration, ok := node.(*ast.GenDecl)
		if !ok || declaration.Tok != token.CONST {
			return true
		}
		for specIndex, spec := range declaration.Specs {
			values := spec.(*ast.ValueSpec)
			for nameIndex, name := range values.Names {
				if name.Name != "gameAgreementStatusCount" {
					continue
				}
				found = true
				require.Equal(t, len(declaration.Specs)-1, specIndex, "gameAgreementStatusCount must remain the last agreement status")
				require.Equal(t, len(values.Names)-1, nameIndex, "gameAgreementStatusCount must remain the last agreement status")
			}
		}
		return true
	})
	require.True(t, found, "gameAgreementStatusCount sentinel not found")
}

func TestRecordGameAgreementsPreservesCanonicalSeries(t *testing.T) {
	metricer := NewMetrics()
	metricer.RecordGameAgreements(nil)
	require.Equal(t, GameAgreementStatus(8), gameAgreementStatusCount)
	require.Len(t, canonicalGameAgreementSeries, int(gameAgreementStatusCount))

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

	counts := map[GameAgreementStatus]int{
		AgreeChallengerAhead:    1,
		DisagreeChallengerAhead: 2,
		AgreeDefenderAhead:      3,
		DisagreeDefenderAhead:   4,
		AgreeDefenderWins:       5,
		DisagreeDefenderWins:    6,
		AgreeChallengerWins:     7,
		DisagreeChallengerWins:  8,
	}
	metricer.RecordGameAgreements(counts)
	firstCycle := gatherGameAgreements(t, metricer)
	require.Len(t, firstCycle.Metric, len(initial.Metric))
	require.Equal(t, float64(1), agreementValue(t, firstCycle, agreementLabels("agree_challenger_ahead", "in_progress", "incorrect", "agree")))
	require.Equal(t, float64(2), agreementValue(t, firstCycle, agreementLabels("disagree_challenger_ahead", "in_progress", "correct", "disagree")))
	require.Equal(t, float64(3), agreementValue(t, firstCycle, agreementLabels("agree_defender_ahead", "in_progress", "correct", "agree")))
	require.Equal(t, float64(4), agreementValue(t, firstCycle, agreementLabels("disagree_defender_ahead", "in_progress", "incorrect", "disagree")))
	require.Equal(t, float64(5), agreementValue(t, firstCycle, agreementLabels("agree_defender_wins", "complete", "correct", "agree")))
	require.Equal(t, float64(6), agreementValue(t, firstCycle, agreementLabels("disagree_defender_wins", "complete", "incorrect", "disagree")))
	require.Equal(t, float64(7), agreementValue(t, firstCycle, agreementLabels("agree_challenger_wins", "complete", "incorrect", "agree")))
	require.Equal(t, float64(8), agreementValue(t, firstCycle, agreementLabels("disagree_challenger_wins", "complete", "correct", "disagree")))

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
