package metrics

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestRecordGameAgreementsKeepsFixedTypedSeries(t *testing.T) {
	metricer := NewMetrics()
	metricer.RecordGameAgreements(nil)

	initial := gatherGameAgreements(t, metricer)
	require.Len(t, initial.Metric, len(gameTypes.SupportedLifecycleGameTypes)*8*2)
	require.Equal(t, float64(0), agreementValue(t, initial, map[string]string{
		"game_type":          gameTypes.ZKDisputeGameType.String(),
		"status":             "agree_challenger_ahead",
		"completion":         "in_progress",
		"result_correctness": "correct",
		"root_agreement":     "agree",
	}))

	metricer.RecordGameAgreements(map[GameAgreementKey]int{
		{GameType: gameTypes.CannonGameType, Status: AgreeDefenderAhead, Correct: true}:      2,
		{GameType: gameTypes.ZKDisputeGameType, Status: AgreeChallengerAhead, Correct: true}: 1,
	})
	firstCycle := gatherGameAgreements(t, metricer)
	require.Len(t, firstCycle.Metric, len(initial.Metric))
	require.Equal(t, float64(2), agreementValue(t, firstCycle, map[string]string{
		"game_type":          gameTypes.CannonGameType.String(),
		"status":             "agree_defender_ahead",
		"completion":         "in_progress",
		"result_correctness": "correct",
		"root_agreement":     "agree",
	}))
	require.Equal(t, float64(1), agreementValue(t, firstCycle, map[string]string{
		"game_type":          gameTypes.ZKDisputeGameType.String(),
		"status":             "agree_challenger_ahead",
		"completion":         "in_progress",
		"result_correctness": "correct",
		"root_agreement":     "agree",
	}))

	metricer.RecordGameAgreements(map[GameAgreementKey]int{
		{GameType: gameTypes.ZKDisputeGameType, Status: AgreeChallengerAhead, Correct: false}: 3,
	})
	secondCycle := gatherGameAgreements(t, metricer)
	require.Len(t, secondCycle.Metric, len(initial.Metric))
	require.Equal(t, float64(0), agreementValue(t, secondCycle, map[string]string{
		"game_type":          gameTypes.CannonGameType.String(),
		"status":             "agree_defender_ahead",
		"completion":         "in_progress",
		"result_correctness": "correct",
		"root_agreement":     "agree",
	}))
	require.Equal(t, float64(0), agreementValue(t, secondCycle, map[string]string{
		"game_type":          gameTypes.ZKDisputeGameType.String(),
		"status":             "agree_challenger_ahead",
		"completion":         "in_progress",
		"result_correctness": "correct",
		"root_agreement":     "agree",
	}))
	require.Equal(t, float64(3), agreementValue(t, secondCycle, map[string]string{
		"game_type":          gameTypes.ZKDisputeGameType.String(),
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

func agreementValue(t *testing.T, family *dto.MetricFamily, labels map[string]string) float64 {
	t.Helper()
	for _, metric := range family.Metric {
		actual := make(map[string]string, len(metric.Label))
		for _, label := range metric.Label {
			actual[label.GetName()] = label.GetValue()
		}
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
