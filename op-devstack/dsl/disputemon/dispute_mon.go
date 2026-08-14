package disputemon

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	devtestmetrics "github.com/ethereum-optimism/optimism/op-devstack/devtest/metrics"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

const (
	disputeMonMetricPollInterval = 100 * time.Millisecond
	disputeMonMaxOutputFetchAge  = 30 * time.Second
)

type DisputeMon struct {
	t       devtest.T
	metrics *devtestmetrics.MetricsClient
}

func New(t devtest.T, metricsURL string) *DisputeMon {
	httpClient := client.NewBasicHTTPClient(strings.TrimRight(metricsURL, "/"), t.Logger())
	return &DisputeMon{
		t:       t,
		metrics: devtestmetrics.NewMetricsClient(httpClient),
	}
}

type StateExpectation struct {
	check devtestmetrics.SnapshotCheck
}

type BondedGame interface {
	WETHAddress() common.Address
}

func allOf(expectations ...StateExpectation) StateExpectation {
	return StateExpectation{
		check: func(snapshot *devtestmetrics.Snapshot) error {
			for _, expectation := range expectations {
				if err := expectation.check(snapshot); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func GameCount(gameType gameTypes.GameType, expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_games",
			map[string]string{"game_type": gameType.String()},
			float64(expected),
		),
	}
}

func FailedGames(expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals("op_dispute_mon_failed_games", nil, float64(expected)),
	}
}

func GamesWaitingForRootSource(gameType gameTypes.GameType, expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_games_waiting_for_root_source",
			map[string]string{"game_type": gameType.String()},
			float64(expected),
		),
	}
}

func AgreedRoots(expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeSumEquals(
			"op_dispute_mon_games_agreement",
			map[string]string{"root_agreement": "agree"},
			float64(expected),
		),
	}
}

func DisagreedRoots(expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeSumEquals(
			"op_dispute_mon_games_agreement",
			map[string]string{"root_agreement": "disagree"},
			float64(expected),
		),
	}
}

func InvalidProposalObserved(game interface{ CreatedAt() uint64 }) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_latest_proposal",
			map[string]string{"root_agreement": "disagree"},
			float64(game.CreatedAt()),
		),
	}
}

func IncorrectDefenderAhead(expected int) StateExpectation {
	return gameAgreement(
		"disagree_defender_ahead",
		"in_progress",
		"incorrect",
		"disagree",
		expected,
	)
}

func IncorrectDefenderWins(expected int) StateExpectation {
	return gameAgreement(
		"disagree_defender_wins",
		"complete",
		"incorrect",
		"disagree",
		expected,
	)
}

func CorrectChallengerWins(expected int) StateExpectation {
	return gameAgreement(
		"disagree_challenger_wins",
		"complete",
		"correct",
		"disagree",
		expected,
	)
}

func CorrectDefenderAhead(expected int) StateExpectation {
	return gameAgreement("agree_defender_ahead", "in_progress", "correct", "agree", expected)
}

func CorrectDefenderWins(expected int) StateExpectation {
	return gameAgreement("agree_defender_wins", "complete", "correct", "agree", expected)
}

func CorrectChallengerAhead(expected int) StateExpectation {
	return gameAgreement("disagree_challenger_ahead", "in_progress", "correct", "disagree", expected)
}

func IncorrectChallengerAhead(expected int) StateExpectation {
	return gameAgreement("agree_challenger_ahead", "in_progress", "incorrect", "agree", expected)
}

func IncorrectChallengerWins(expected int) StateExpectation {
	return gameAgreement("agree_challenger_wins", "complete", "incorrect", "agree", expected)
}

func CompletedBeforeMaxDuration(expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_resolution_status",
			map[string]string{
				"completion":   "complete",
				"max_duration": "before_max_duration",
			},
			float64(expected),
		),
	}
}

func ResolvedClaims(expected int) StateExpectation {
	return claimCount(map[string]string{"resolved": "resolved"}, expected)
}

func ResolvedClaimsInFirstHalf(expected int) StateExpectation {
	return claimCount(map[string]string{
		"resolved":         "resolved",
		"game_time_period": "first_half",
	}, expected)
}

func ResolvedClaimsInSecondHalf(expected int) StateExpectation {
	return claimCount(map[string]string{
		"resolved":         "resolved",
		"game_time_period": "second_half",
	}, expected)
}

func UnresolvedClaimsInFirstHalf(expected int) StateExpectation {
	return claimCount(map[string]string{
		"resolved":         "unresolved",
		"game_time_period": "first_half",
	}, expected)
}

func UnresolvedClaimsInSecondHalf(expected int) StateExpectation {
	return claimCount(map[string]string{
		"resolved":         "unresolved",
		"game_time_period": "second_half",
	}, expected)
}

func ExpiredClaims(expected int) StateExpectation {
	return claimCount(map[string]string{"clock": "expired"}, expected)
}

func UnexpiredClaims(expected int) StateExpectation {
	return claimCount(map[string]string{"clock": "not_expired"}, expected)
}

func ResolvableClaims(expected int) StateExpectation {
	return claimCount(map[string]string{
		"resolved":   "unresolved",
		"resolvable": "resolvable",
	}, expected)
}

func ExpectedNonWithdrawableCredits(expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_credits",
			map[string]string{
				"credit":       "expected",
				"withdrawable": "non_withdrawable",
			},
			float64(expected),
		),
	}
}

func ExcessCredits(expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeSumEquals(
			"op_dispute_mon_credits",
			map[string]string{"credit": "above"},
			float64(expected),
		),
	}
}

func DeficientNonWithdrawableCredits(expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_credits",
			map[string]string{
				"credit":       "below",
				"withdrawable": "non_withdrawable",
			},
			float64(expected),
		),
	}
}

func ExactNonWithdrawableCredits(expected int) StateExpectation {
	return allOf(
		ExpectedNonWithdrawableCredits(expected),
		ExcessCredits(0),
		DeficientNonWithdrawableCredits(0),
	)
}

func ExactWithdrawableCredits(expected int) StateExpectation {
	return allOf(
		StateExpectation{check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_credits",
			map[string]string{"credit": "expected", "withdrawable": "withdrawable"},
			float64(expected),
		)},
		StateExpectation{check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_credits",
			map[string]string{"credit": "below", "withdrawable": "withdrawable"},
			0,
		)},
		StateExpectation{check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_credits",
			map[string]string{"credit": "above", "withdrawable": "withdrawable"},
			0,
		)},
	)
}

func MatchingWithdrawalRequests(game BondedGame, expected int) StateExpectation {
	return withdrawalRequests(game, "matching", expected)
}

func DivergentWithdrawalRequests(game BondedGame, expected int) StateExpectation {
	return withdrawalRequests(game, "divergent", expected)
}

func NoWithdrawalRequests(game BondedGame) StateExpectation {
	return allOf(
		MatchingWithdrawalRequests(game, 0),
		DivergentWithdrawalRequests(game, 0),
	)
}

func SufficientCollateral(game BondedGame, expectedWei *big.Int) StateExpectation {
	return collateral(game, "sufficient", expectedWei)
}

func NoInsufficientCollateral(game BondedGame) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_bond_collateral_required",
			map[string]string{
				"delayedWETH": game.WETHAddress().Hex(),
				"balance":     "insufficient",
			},
			0,
		),
	}
}

func FullyCollateralized(game BondedGame, expectedWei *big.Int) StateExpectation {
	return allOf(
		SufficientCollateral(game, expectedWei),
		NoInsufficientCollateral(game),
	)
}

func AvailableCollateral(game BondedGame, expectedWei *big.Int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_bond_collateral_available",
			map[string]string{
				"delayedWETH": game.WETHAddress().Hex(),
				"balance":     "sufficient",
			},
			weiToEther(expectedWei),
		),
	}
}

func HonestActorInvalidClaims(actor common.Address, expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_honest_actor_claims",
			map[string]string{
				"honest_actor_address": actor.Hex(),
				"state":                "invalid",
			},
			float64(expected),
		),
	}
}

func HonestActorLostBonds(actor common.Address, expectedWei *big.Int) StateExpectation {
	return honestActorBonds(actor, "lost", expectedWei)
}

func HonestActorPendingBonds(actor common.Address, expectedWei *big.Int) StateExpectation {
	return honestActorBonds(actor, "pending", expectedWei)
}

func HonestActorWonBonds(actor common.Address, expectedWei *big.Int) StateExpectation {
	return honestActorBonds(actor, "won", expectedWei)
}

func HonestActorPendingWithdrawals(actor common.Address, expectedWei *big.Int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_honest_actor_pending_withdrawals",
			map[string]string{"actor": actor.Hex()},
			weiToEther(expectedWei),
		),
	}
}

func PendingZKResolutions(expected int) StateExpectation {
	return pendingZKLifecycleAction("resolution", expected)
}

func PendingZKBondDistributions(expected int) StateExpectation {
	return pendingZKLifecycleAction("bond_distribution", expected)
}

func AnchorStateL2SequenceNumber(registry common.Address, expected uint64) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_anchor_state_l2_sequence_number",
			map[string]string{"anchor_state_registry": registry.Hex()},
			float64(expected),
		),
	}
}

func honestActorBonds(actor common.Address, state string, expectedWei *big.Int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_honest_actor_bonds",
			map[string]string{
				"honest_actor_address": actor.Hex(),
				"state":                state,
			},
			weiToEther(expectedWei),
		),
	}
}

func claimCount(labels map[string]string, expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeSumEquals(
			"op_dispute_mon_claims",
			labels,
			float64(expected),
		),
	}
}

func gameAgreement(status string, completion string, resultCorrectness string, rootAgreement string, expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_games_agreement",
			map[string]string{
				"status":             status,
				"completion":         completion,
				"result_correctness": resultCorrectness,
				"root_agreement":     rootAgreement,
			},
			float64(expected),
		),
	}
}

func withdrawalRequests(game BondedGame, credits string, expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_withdrawal_requests",
			map[string]string{
				"delayedWETH": game.WETHAddress().Hex(),
				"credits":     credits,
			},
			float64(expected),
		),
	}
}

func collateral(game BondedGame, balance string, expectedWei *big.Int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_bond_collateral_required",
			map[string]string{
				"delayedWETH": game.WETHAddress().Hex(),
				"balance":     balance,
			},
			weiToEther(expectedWei),
		),
	}
}

func pendingZKLifecycleAction(action string, expected int) StateExpectation {
	return StateExpectation{
		check: devtestmetrics.GaugeEquals(
			"op_dispute_mon_zk_games_pending_lifecycle_action",
			map[string]string{"action": action},
			float64(expected),
		),
	}
}

func (d *DisputeMon) VerifyState(expectations ...StateExpectation) {
	d.t.Require().NotEmpty(expectations, "at least one dispute monitor state expectation is required")
	for i, expectation := range expectations {
		d.t.Require().NotNil(expectation.check, "dispute monitor state expectation %d is empty", i)
	}
	d.waitForMetrics(func(snapshot *devtestmetrics.Snapshot) error {
		for _, expectation := range expectations {
			if err := expectation.check(snapshot); err != nil {
				return err
			}
		}
		return nil
	})
}

// VerifyCompletedCycleWithoutFailures waits for one completed monitor cycle
// that did not classify any game as failed.
func (d *DisputeMon) VerifyCompletedCycleWithoutFailures() {
	d.waitForMetrics(
		devtestmetrics.GaugeEquals("op_dispute_mon_failed_games", nil, 0),
		devtestmetrics.HistogramCountAtLeast("op_dispute_mon_monitor_duration_seconds", nil, 1),
	)
}

type MonitoringBaseline struct {
	lastOutputFetch float64
	completedCycles uint64
}

func (d *DisputeMon) VerifyDisputeMonHealthy() {
	snapshot, err := d.metrics.Fetch(d.t.Ctx())
	d.t.Require().NoError(err, "fetch initial dispute monitor metrics")
	previousOutputFetch, err := snapshot.Gauge("op_dispute_mon_last_output_fetch", nil)
	d.t.Require().NoError(err, "read initial output fetch timestamp")

	d.waitForMetrics(
		devtestmetrics.GaugeEquals("op_dispute_mon_up", nil, 1),
		func(snapshot *devtestmetrics.Snapshot) error {
			outputFetch, err := snapshot.Gauge("op_dispute_mon_last_output_fetch", nil)
			if err != nil {
				return err
			}
			if outputFetch <= previousOutputFetch {
				return fmt.Errorf("last output fetch timestamp did not update from %v", previousOutputFetch)
			}
			fetchedAt := time.Unix(int64(outputFetch), 0)
			age := time.Since(fetchedAt)
			if age < 0 {
				return fmt.Errorf("last output fetch timestamp %v is in the future", fetchedAt)
			}
			if age > disputeMonMaxOutputFetchAge {
				return fmt.Errorf("last output fetch timestamp %v is older than %v", fetchedAt, disputeMonMaxOutputFetchAge)
			}
			return nil
		},
	)
}

func (d *DisputeMon) VerifyHealthyOutputFetch(gameType gameTypes.GameType) MonitoringBaseline {
	snapshot := d.waitForMetrics(
		devtestmetrics.GaugeEquals("op_dispute_mon_up", nil, 1),
		devtestmetrics.GaugeEquals("op_dispute_mon_games", map[string]string{"game_type": gameType.String()}, 1),
		devtestmetrics.GaugeEquals("op_dispute_mon_failed_games", nil, 0),
		devtestmetrics.GaugeAtLeast("op_dispute_mon_last_output_fetch", nil, 1),
		devtestmetrics.HistogramCountAtLeast("op_dispute_mon_monitor_duration_seconds", nil, 1),
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
		devtestmetrics.GaugeEquals("op_dispute_mon_up", nil, 1),
		devtestmetrics.GaugeEquals("op_dispute_mon_failed_games", nil, 1),
		devtestmetrics.GaugeAtLeast("op_dispute_mon_last_output_fetch", nil, baseline.lastOutputFetch),
		devtestmetrics.HistogramCountAtLeast("op_dispute_mon_monitor_duration_seconds", nil, baseline.completedCycles+1),
	)
	lastOutputFetch, err := firstFailure.Gauge("op_dispute_mon_last_output_fetch", nil)
	d.t.Require().NoError(err, "read output fetch timestamp after reference node failure")
	completedCycles, err := firstFailure.HistogramCount("op_dispute_mon_monitor_duration_seconds", nil)
	d.t.Require().NoError(err, "read completed cycles after reference node failure")

	d.waitForMetrics(
		devtestmetrics.GaugeEquals("op_dispute_mon_up", nil, 1),
		devtestmetrics.GaugeEquals("op_dispute_mon_failed_games", nil, 1),
		devtestmetrics.GaugeEquals("op_dispute_mon_last_output_fetch", nil, lastOutputFetch),
		devtestmetrics.HistogramCountAtLeast("op_dispute_mon_monitor_duration_seconds", nil, completedCycles+1),
	)
}

func (d *DisputeMon) waitForMetrics(checks ...devtestmetrics.SnapshotCheck) *devtestmetrics.Snapshot {
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

func weiToEther(wei *big.Int) float64 {
	value := new(big.Rat).SetInt(wei)
	value.Quo(value, big.NewRat(params.Ether, 1))
	asFloat, _ := value.Float64()
	return asFloat
}
