package zkproposer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	devtestmetrics "github.com/ethereum-optimism/optimism/op-devstack/devtest/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/log"
)

const (
	metricPollInterval               = 100 * time.Millisecond
	stateWaitTimeout                 = 10 * time.Minute
	defenseTasksSpawnedMetric        = "kona_sp1_proposer_games_defense_spawned"
	peakConcurrentDefenseTasksMetric = "kona_sp1_proposer_peak_concurrent_defense_tasks"
	gameProvingFailuresMetric        = "kona_sp1_proposer_game_proving_error"
	metricsDisabledInstruction       = "ZK proposer metrics are disabled; pass presets.WithZKProposerOption(sysgo.WithZKMetrics()) when creating the preset"
)

// Runtime provides the process-owned metrics transport without exposing its endpoint.
type Runtime interface {
	MetricsClient() client.HTTP
}

// ZKProposer verifies the observable state of a running kona-sp1-proposer.
type ZKProposer struct {
	t       devtest.T
	log     log.Logger
	metrics *devtestmetrics.MetricsClient
}

// New creates a DSL facade for a running proposer runtime.
func New(t devtest.T, runtime Runtime) *ZKProposer {
	t.Require().NotNil(runtime, "ZK proposer runtime must not be nil")
	var metricsClient *devtestmetrics.MetricsClient
	if httpClient := runtime.MetricsClient(); httpClient != nil {
		metricsClient = devtestmetrics.NewMetricsClient(httpClient, devtestmetrics.WithWaitTimeout(stateWaitTimeout))
	}
	return &ZKProposer{t: t, log: t.Logger(), metrics: metricsClient}
}

// StateExpectation describes one exact proposer-state observation.
type StateExpectation struct {
	description string
	expected    int
	metric      string
}

// DefenseTasksSpawned expects the exact number of spawned defense tasks.
func DefenseTasksSpawned(expected int) *StateExpectation {
	return &StateExpectation{
		description: "defense tasks spawned",
		expected:    expected,
		metric:      defenseTasksSpawnedMetric,
	}
}

// PeakConcurrentDefenseTasks expects the peak number of concurrently scheduled defense tasks.
func PeakConcurrentDefenseTasks(expected int) *StateExpectation {
	return &StateExpectation{
		description: "peak concurrent defense tasks",
		expected:    expected,
		metric:      peakConcurrentDefenseTasksMetric,
	}
}

// ProvingFailures expects the exact number of proving failures.
func ProvingFailures(expected int) *StateExpectation {
	return &StateExpectation{
		description: "proving failures",
		expected:    expected,
		metric:      gameProvingFailuresMetric,
	}
}

// VerifyState waits for every expectation to hold in one metrics snapshot.
func (p *ZKProposer) VerifyState(expectations ...*StateExpectation) {
	p.t.Require().NoError(p.verifyState(p.t.Ctx(), expectations...), "verify ZK proposer state")
}

func (p *ZKProposer) verifyState(ctx context.Context, expectations ...*StateExpectation) error {
	if p.metrics == nil {
		return errors.New(metricsDisabledInstruction)
	}
	if len(expectations) == 0 {
		return errors.New("at least one ZK proposer state expectation is required")
	}
	for i, expectation := range expectations {
		if expectation == nil || expectation.metric == "" {
			return fmt.Errorf("ZK proposer state expectation %d is empty", i)
		}
	}

	p.log.Info("Waiting for ZK proposer state", "expectations", len(expectations))
	_, err := p.metrics.WaitForSnapshot(ctx, metricPollInterval, func(snapshot *devtestmetrics.Snapshot) error {
		for _, expectation := range expectations {
			observed, err := snapshot.Gauge(expectation.metric, nil)
			if err != nil {
				p.log.Info("ZK proposer state observation unavailable",
					"expectation", expectation.description,
					"expected", expectation.expected,
					"err", err)
				return fmt.Errorf("%s: %w", expectation.description, err)
			}
			p.log.Info("Observed ZK proposer state",
				"expectation", expectation.description,
				"expected", expectation.expected,
				"observed", observed)
			if observed != float64(expectation.expected) {
				return fmt.Errorf("%s expected %d but observed %v", expectation.description, expectation.expected, observed)
			}
		}
		return nil
	})
	return err
}
