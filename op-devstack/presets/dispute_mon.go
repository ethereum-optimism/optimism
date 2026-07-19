package presets

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	devtestmetrics "github.com/ethereum-optimism/optimism/op-devstack/devtest/metrics"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

const disputeMonMetricPollInterval = 100 * time.Millisecond

type DisputeMon struct {
	t       devtest.T
	metrics *devtestmetrics.MetricsClient
}

func newDisputeMon(t devtest.T, metricsURL string) *DisputeMon {
	httpClient := client.NewBasicHTTPClient(strings.TrimRight(metricsURL, "/"), t.Logger())
	return &DisputeMon{
		t:       t,
		metrics: devtestmetrics.NewMetricsClient(httpClient),
	}
}

func (d *DisputeMon) VerifyGameCount(gameType gameTypes.GameType, expected int) {
	err := d.metrics.WaitForGauge(d.t.Ctx(), devtestmetrics.GaugeDefinition{
		Name:     "op_dispute_mon_games",
		Labels:   map[string]string{"game_type": gameType.String()},
		Expected: float64(expected),
	}, disputeMonMetricPollInterval)
	d.t.Require().NoError(err, "expected dispute monitor to export %d games of type %s", expected, gameType)
}

type MonitoringBaseline struct {
	lastOutputFetch float64
	completedCycles uint64
}

type metricCheck func(*devtestmetrics.Snapshot) error

func (d *DisputeMon) VerifyHealthySuperPermissionedGame() {
	d.waitForMetrics(
		gaugeEquals("op_dispute_mon_up", nil, 1),
		gaugeEquals("op_dispute_mon_games", map[string]string{"game_type": gameTypes.SuperPermissionedGameType.String()}, 1),
		gaugeEquals("op_dispute_mon_failed_games", nil, 0),
		gaugeAtLeast("op_dispute_mon_last_output_fetch", nil, 1),
		gaugeEquals("op_dispute_mon_games_agreement", map[string]string{
			"status":             "agree_defender_wins",
			"completion":         "complete",
			"result_correctness": "correct",
			"root_agreement":     "agree",
		}, 1),
		gaugeEquals("op_dispute_mon_resolution_status", map[string]string{
			"completion":   "complete",
			"max_duration": "max_duration",
		}, 1),
	)
}

func (d *DisputeMon) VerifyInvalidProposalForecast(gameType gameTypes.GameType) {
	d.waitForMetrics(
		gaugeEquals("op_dispute_mon_games", map[string]string{"game_type": gameType.String()}, 1),
		gaugeEquals("op_dispute_mon_games_agreement", map[string]string{
			"status":             "disagree_defender_ahead",
			"completion":         "in_progress",
			"result_correctness": "incorrect",
			"root_agreement":     "disagree",
		}, 1),
		gaugeAtLeast("op_dispute_mon_latest_proposal", map[string]string{"root_agreement": "disagree"}, 1),
		gaugeAtLeast("op_dispute_mon_last_output_fetch", nil, 1),
		gaugeEquals("op_dispute_mon_failed_games", nil, 0),
	)
}

func (d *DisputeMon) VerifyIncorrectResolvedGame() {
	d.waitForMetrics(
		gaugeEquals("op_dispute_mon_games_agreement", map[string]string{
			"status":             "disagree_defender_wins",
			"completion":         "complete",
			"result_correctness": "incorrect",
			"root_agreement":     "disagree",
		}, 1),
		gaugeAtLeast("op_dispute_mon_latest_proposal", map[string]string{"root_agreement": "disagree"}, 1),
		gaugeEquals("op_dispute_mon_failed_games", nil, 0),
	)
}

func (d *DisputeMon) VerifyHealthyOutputFetch(gameType gameTypes.GameType) MonitoringBaseline {
	snapshot := d.waitForMetrics(
		gaugeEquals("op_dispute_mon_up", nil, 1),
		gaugeEquals("op_dispute_mon_games", map[string]string{"game_type": gameType.String()}, 1),
		gaugeEquals("op_dispute_mon_failed_games", nil, 0),
		gaugeAtLeast("op_dispute_mon_last_output_fetch", nil, 1),
		histogramCountAtLeast("op_dispute_mon_monitor_duration_seconds", nil, 1),
	)
	lastOutputFetch, err := snapshot.Gauge("op_dispute_mon_last_output_fetch", nil)
	d.t.Require().NoError(err, "read last successful output fetch")
	completedCycles, err := snapshot.HistogramCount("op_dispute_mon_monitor_duration_seconds", nil)
	d.t.Require().NoError(err, "read completed dispute monitor cycles")
	return MonitoringBaseline{
		lastOutputFetch: lastOutputFetch,
		completedCycles: completedCycles,
	}
}

func (d *DisputeMon) VerifyReferenceNodeFailure(baseline MonitoringBaseline) {
	firstFailure := d.waitForMetrics(
		gaugeEquals("op_dispute_mon_up", nil, 1),
		gaugeEquals("op_dispute_mon_failed_games", nil, 1),
		gaugeAtLeast("op_dispute_mon_last_output_fetch", nil, baseline.lastOutputFetch),
		histogramCountAtLeast("op_dispute_mon_monitor_duration_seconds", nil, baseline.completedCycles+1),
	)
	lastOutputFetch, err := firstFailure.Gauge("op_dispute_mon_last_output_fetch", nil)
	d.t.Require().NoError(err, "read output fetch timestamp after reference node failure")
	completedCycles, err := firstFailure.HistogramCount("op_dispute_mon_monitor_duration_seconds", nil)
	d.t.Require().NoError(err, "read completed cycles after reference node failure")

	d.waitForMetrics(
		gaugeEquals("op_dispute_mon_up", nil, 1),
		gaugeEquals("op_dispute_mon_failed_games", nil, 1),
		gaugeEquals("op_dispute_mon_last_output_fetch", nil, lastOutputFetch),
		histogramCountAtLeast("op_dispute_mon_monitor_duration_seconds", nil, completedCycles+1),
	)
}

func (d *DisputeMon) VerifyResolvedGameAccounting(game *proofs.FaultDisputeGame) {
	weth := game.WETHAddress().Hex()
	rootBond := weiToEther(game.RootClaim().Bond())
	d.waitForMetrics(
		gaugeEquals("op_dispute_mon_failed_games", nil, 0),
		gaugeEquals("op_dispute_mon_resolution_status", map[string]string{
			"completion":   "complete",
			"max_duration": "before_max_duration",
		}, 1),
		gaugeSumEquals("op_dispute_mon_claims", map[string]string{
			"resolved":         "resolved",
			"game_time_period": "first_half",
		}, 1),
		gaugeSumEquals("op_dispute_mon_claims", map[string]string{
			"resolved":   "unresolved",
			"resolvable": "resolvable",
		}, 0),
		gaugeEquals("op_dispute_mon_credits", map[string]string{
			"credit":       "expected",
			"withdrawable": "non_withdrawable",
		}, 1),
		gaugeSumEquals("op_dispute_mon_credits", map[string]string{"credit": "above"}, 0),
		gaugeEquals("op_dispute_mon_credits", map[string]string{
			"credit":       "below",
			"withdrawable": "non_withdrawable",
		}, 0),
		gaugeEquals("op_dispute_mon_withdrawal_requests", map[string]string{
			"delayedWETH": weth,
			"credits":     "matching",
		}, 0),
		gaugeEquals("op_dispute_mon_withdrawal_requests", map[string]string{
			"delayedWETH": weth,
			"credits":     "divergent",
		}, 0),
		gaugeEquals("op_dispute_mon_bond_collateral_required", map[string]string{
			"delayedWETH": weth,
			"balance":     "sufficient",
		}, rootBond),
		gaugeEquals("op_dispute_mon_bond_collateral_required", map[string]string{
			"delayedWETH": weth,
			"balance":     "insufficient",
		}, 0),
	)
}

func (d *DisputeMon) VerifyHonestActorLoss(actor common.Address, rootBond *big.Int) {
	d.waitForMetrics(
		gaugeEquals("op_dispute_mon_honest_actor_claims", map[string]string{
			"honest_actor_address": actor.Hex(),
			"state":                "invalid",
		}, 1),
		gaugeEquals("op_dispute_mon_honest_actor_bonds", map[string]string{
			"honest_actor_address": actor.Hex(),
			"state":                "lost",
		}, weiToEther(rootBond)),
		gaugeEquals("op_dispute_mon_games_agreement", map[string]string{
			"status":             "disagree_challenger_wins",
			"completion":         "complete",
			"result_correctness": "correct",
			"root_agreement":     "disagree",
		}, 1),
	)
}

func (d *DisputeMon) waitForMetrics(checks ...metricCheck) *devtestmetrics.Snapshot {
	snapshot, err := d.metrics.WaitForSnapshot(d.t.Ctx(), disputeMonMetricPollInterval, func(snapshot *devtestmetrics.Snapshot) error {
		for _, check := range checks {
			if err := check(snapshot); err != nil {
				return err
			}
		}
		return nil
	})
	d.t.Require().NoError(err, "wait for dispute monitor metrics")
	return snapshot
}

func gaugeEquals(name string, labels map[string]string, expected float64) metricCheck {
	return func(snapshot *devtestmetrics.Snapshot) error {
		observed, err := snapshot.Gauge(name, labels)
		if err != nil {
			return err
		}
		if observed != expected {
			return fmt.Errorf("metric %s with labels %v expected %v but observed %v", name, labels, expected, observed)
		}
		return nil
	}
}

func gaugeAtLeast(name string, labels map[string]string, minimum float64) metricCheck {
	return func(snapshot *devtestmetrics.Snapshot) error {
		observed, err := snapshot.Gauge(name, labels)
		if err != nil {
			return err
		}
		if observed < minimum {
			return fmt.Errorf("metric %s with labels %v expected at least %v but observed %v", name, labels, minimum, observed)
		}
		return nil
	}
}

func gaugeSumEquals(name string, labels map[string]string, expected float64) metricCheck {
	return func(snapshot *devtestmetrics.Snapshot) error {
		observed, err := snapshot.GaugeSum(name, labels)
		if err != nil {
			return err
		}
		if observed != expected {
			return fmt.Errorf("metric %s sum with labels %v expected %v but observed %v", name, labels, expected, observed)
		}
		return nil
	}
}

func histogramCountAtLeast(name string, labels map[string]string, minimum uint64) metricCheck {
	return func(snapshot *devtestmetrics.Snapshot) error {
		observed, err := snapshot.HistogramCount(name, labels)
		if err != nil {
			return err
		}
		if observed < minimum {
			return fmt.Errorf("metric %s histogram count with labels %v expected at least %d but observed %d", name, labels, minimum, observed)
		}
		return nil
	}
}

func weiToEther(wei *big.Int) float64 {
	value := new(big.Rat).SetInt(wei)
	value.Quo(value, big.NewRat(params.Ether, 1))
	asFloat, _ := value.Float64()
	return asFloat
}

type disputeMonOptions struct {
	rollupRPCs    []string
	supernodeRPCs []string
	honestActors  []common.Address
}

type DisputeMonOption func(*disputeMonOptions)

func WithDisputeMonRollupNodes(nodes ...*dsl.L2CLNode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.rollupRPCs = append(opts.rollupRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func WithDisputeMonSupernodes(nodes ...*dsl.Supernode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.supernodeRPCs = append(opts.supernodeRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func WithDisputeMonHonestActors(actors ...common.Address) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		opts.honestActors = append(opts.honestActors, actors...)
	}
}

func (s *SingleChainInterop) StartDisputeMon() *DisputeMon {
	return StartDisputeMon(
		s.T,
		s.L1EL,
		s.L2ChainA.DisputeGameFactoryProxyAddr(),
		WithDisputeMonSupernodes(s.SuperRoots),
	)
}

func StartDisputeMon(
	t devtest.T,
	l1EL *dsl.L1ELNode,
	factory common.Address,
	options ...DisputeMonOption,
) *DisputeMon {
	t.Require().NotNil(l1EL, "L1 EL is required to start dispute monitor")
	opts := &disputeMonOptions{}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}
	t.Require().NotEmpty(
		append(append([]string{}, opts.rollupRPCs...), opts.supernodeRPCs...),
		"at least one rollup node or supernode is required to start dispute monitor",
	)
	runtime := sysgo.StartDisputeMon(t, sysgo.DisputeMonConfig{
		L1RPC:              l1EL.Escape().UserRPC(),
		GameFactoryAddress: factory,
		RollupRPCs:         opts.rollupRPCs,
		SupernodeRPCs:      opts.supernodeRPCs,
		HonestActors:       opts.honestActors,
	})
	return newDisputeMon(t, runtime.MetricsURL())
}

type disputeMonOptions struct {
	rollupRPCs    []string
	supernodeRPCs []string
}

type DisputeMonOption func(*disputeMonOptions)

func WithDisputeMonRollupNodes(nodes ...*dsl.L2CLNode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.rollupRPCs = append(opts.rollupRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func WithDisputeMonSupernodes(nodes ...*dsl.Supernode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.supernodeRPCs = append(opts.supernodeRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func StartDisputeMon(
	t devtest.T,
	l1EL *dsl.L1ELNode,
	factory common.Address,
	options ...DisputeMonOption,
) *DisputeMon {
	t.Helper()
	t.Require().NotNil(l1EL, "L1 EL is required to start dispute monitor")
	opts := &disputeMonOptions{}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}
	t.Require().NotEmpty(
		append(append([]string{}, opts.rollupRPCs...), opts.supernodeRPCs...),
		"at least one rollup node or supernode is required to start dispute monitor",
	)
	runtime := sysgo.StartDisputeMon(t, sysgo.DisputeMonConfig{
		L1RPC:              l1EL.Escape().UserRPC(),
		GameFactoryAddress: factory,
		RollupRPCs:         opts.rollupRPCs,
		SupernodeRPCs:      opts.supernodeRPCs,
	})
	return newDisputeMon(t, runtime.MetricsURL())
}
