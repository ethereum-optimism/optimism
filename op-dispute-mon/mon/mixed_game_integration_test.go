package mon

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/bonds"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	metricstest "github.com/ethereum-optimism/optimism/op-service/metrics/test"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestMonitorMixedFaultAndZKGames(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	createdAt := now.Add(-time.Minute)
	cl := clock.NewDeterministicClock(now)
	logger := testlog.Logger(t, log.LvlDebug)
	metricer := metrics.NewMetrics()
	actor := common.Address{0xa1}
	weth := common.Address{0xee}
	faultBond := big.NewInt(params.Ether)
	zkBond := new(big.Int).Mul(big.NewInt(2), big.NewInt(params.Ether))
	totalBonds := new(big.Int).Add(new(big.Int).Set(faultBond), zkBond)
	rootClaim := common.Hash{0xca, 0xfe}
	maxClockDuration := uint64(time.Hour / time.Second)

	faultGame := &monTypes.FaultGameData{
		CommonGameData: monTypes.CommonGameData{
			GameMetadata: gameTypes.GameMetadata{
				Index:     0,
				GameType:  uint32(gameTypes.SuperCannonKonaGameType),
				Timestamp: uint64(createdAt.Unix()),
				Proxy:     common.Address{0xf1},
			},
			LastUpdateTime:    now,
			L2SequenceNumber:  100,
			RootClaim:         rootClaim,
			ExpectedRootClaim: rootClaim,
			Status:            gameTypes.GameStatusInProgress,
			AgreeWithClaim:    true,
		},
		BondGameData: monTypes.BondGameData{
			Bonds:           []monTypes.BondRecord{{Depositor: actor, Amount: faultBond}},
			Recipients:      map[common.Address]bool{actor: true},
			Credits:         map[common.Address]*big.Int{actor: new(big.Int)},
			ExpectedCredits: map[common.Address]*big.Int{},
			WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
				actor: {Amount: new(big.Int), Timestamp: new(big.Int)},
			},
			BondDistributionMode: faultTypes.UndecidedDistributionMode,
			WETHContract:         weth,
			WETHDelay:            time.Hour,
			ETHCollateral:        totalBonds,
			CreditWithdrawableAt: now.Add(24 * time.Hour),
		},
		MaxClockDuration: maxClockDuration,
		Claims: []monTypes.EnrichedClaim{{
			Claim: faultTypes.Claim{
				ClaimData: faultTypes.ClaimData{
					Value:    rootClaim,
					Bond:     faultBond,
					Position: faultTypes.RootPosition,
				},
				Claimant:            actor,
				Clock:               faultTypes.NewClock(0, createdAt),
				ContractIndex:       0,
				ParentContractIndex: -1,
			},
		}},
	}
	zkGame := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{
			GameMetadata: gameTypes.GameMetadata{
				Index:     1,
				GameType:  uint32(gameTypes.ZKDisputeGameType),
				Timestamp: uint64(createdAt.Unix()),
				Proxy:     common.Address{0xf2},
			},
			LastUpdateTime:    now,
			L2SequenceNumber:  101,
			RootClaim:         rootClaim,
			ExpectedRootClaim: rootClaim,
			Status:            gameTypes.GameStatusInProgress,
			AgreeWithClaim:    true,
		},
		BondGameData: monTypes.BondGameData{
			Bonds:           []monTypes.BondRecord{{Depositor: actor, Amount: zkBond}},
			Recipients:      map[common.Address]bool{actor: true},
			Credits:         map[common.Address]*big.Int{actor: new(big.Int)},
			ExpectedCredits: map[common.Address]*big.Int{},
			WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
				actor: {Amount: new(big.Int), Timestamp: new(big.Int)},
			},
			BondDistributionMode: faultTypes.UndecidedDistributionMode,
			WETHContract:         weth,
			WETHDelay:            time.Hour,
			ETHCollateral:        totalBonds,
		},
		ParentIndex:    math.MaxUint32,
		ProposalStatus: contracts.ProposalStatusUnchallenged,
		Deadline:       createdAt.Add(time.Hour),
		GameCreator:    actor,
		TotalBonds:     zkBond,
	}

	games := []monTypes.EnrichedGame{faultGame, zkGame}
	fetchHead := func(context.Context) (eth.L1BlockRef, error) {
		return eth.L1BlockRef{Number: 10, Hash: common.Hash{0x10}}, nil
	}
	extractGames := func(_ context.Context, blockHash common.Hash, minTimestamp uint64) ([]monTypes.EnrichedGame, int, int, error) {
		require.Equal(t, common.Hash{0x10}, blockHash)
		require.LessOrEqual(t, minTimestamp, faultGame.Timestamp)
		require.LessOrEqual(t, minTimestamp, zkGame.Timestamp)
		return games, 0, 0, nil
	}

	honestActors := monTypes.NewHonestActors([]common.Address{actor})
	forecast := NewForecast(logger, metricer)
	bondMonitor := bonds.NewBonds(logger, metricer, cl, honestActors)
	resolutionMonitor := NewResolutionMonitor(logger, metricer, cl)
	claimMonitor := NewClaimMonitor(logger, cl, honestActors, metricer)
	withdrawalMonitor := NewWithdrawalMonitor(logger, cl, metricer, honestActors)
	l2ChallengesMonitor := NewL2ChallengesMonitor(logger, metricer)
	updateTimeMonitor := NewUpdateTimeMonitor(cl, metricer)
	nodeEndpointErrorsMonitor := NewNodeEndpointErrorsMonitor(logger, metricer)
	nodeEndpointErrorCountMonitor := NewNodeEndpointErrorCountMonitor(logger, metricer)
	nodeEndpointOutOfSyncMonitor := NewNodeEndpointOutOfSyncMonitor(logger, metricer)
	mixedAvailabilityMonitor := NewMixedAvailability(logger, metricer)
	mixedSafetyMonitor := NewMixedSafetyMonitor(logger, metricer)
	differentRootMonitor := NewDifferentRootMonitor(logger, metricer)
	gameTypeMonitor := NewGameTypeMonitor(metricer)
	zkLifecycleMonitor := NewZKLifecycleMonitor(cl, metricer)
	anchorStateMonitor := NewAnchorStateMonitor(logger, metricer, func(common.Address) AnchorRootProvider {
		t.Fatal("test games unexpectedly referenced an anchor state registry")
		return nil
	})
	monitor := newGameMonitor(
		t.Context(), logger, cl, metricer, time.Second, 24*time.Hour,
		fetchHead, extractGames, forecast.Forecast, anchorStateMonitor.CheckAnchorState,
		[]CommonMonitor{
			updateTimeMonitor.CheckUpdateTimes,
			nodeEndpointErrorsMonitor.CheckNodeEndpointErrors,
			nodeEndpointErrorCountMonitor.CheckNodeEndpointErrorCount,
			nodeEndpointOutOfSyncMonitor.CheckNodeEndpointOutOfSync,
			mixedAvailabilityMonitor.CheckMixedAvailability,
			mixedSafetyMonitor.CheckMixedSafety,
			differentRootMonitor.CheckDifferentRoots,
			gameTypeMonitor.CheckGameTypes,
		},
		[]FaultMonitor{
			resolutionMonitor.CheckResolutions,
			claimMonitor.CheckClaims,
			l2ChallengesMonitor.CheckL2Challenges,
		},
		[]BondMonitor{
			bondMonitor.CheckBonds,
			withdrawalMonitor.CheckWithdrawals,
		},
		[]ZKMonitor{zkLifecycleMonitor.CheckLifecycle},
	)

	require.NoError(t, monitor.monitorGames())
	snapshot := metricstest.NewMetricChecker(t, metricer.Registry())
	requireGauge(t, snapshot, "op_dispute_mon_games", map[string]string{"game_type": gameTypes.SuperCannonKonaGameType.String()}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_games", map[string]string{"game_type": gameTypes.ZKDisputeGameType.String()}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_failed_games", nil, 0)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_games_agreement", map[string]string{"root_agreement": "agree"}, 2)
	requireGauge(t, snapshot, "op_dispute_mon_games_agreement", map[string]string{"status": "agree_defender_ahead", "completion": "in_progress", "result_correctness": "correct", "root_agreement": "agree"}, 2)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_claims", map[string]string{"resolved": "unresolved", "game_time_period": "first_half"}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_credits", map[string]string{"credit": "expected", "withdrawable": "non_withdrawable"}, 1)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_credits", map[string]string{"credit": "above"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_credits", map[string]string{"credit": "below", "withdrawable": "non_withdrawable"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_withdrawal_requests", map[string]string{"delayedWETH": weth.Hex(), "credits": "matching"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_withdrawal_requests", map[string]string{"delayedWETH": weth.Hex(), "credits": "divergent"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_required", map[string]string{"delayedWETH": weth.Hex(), "balance": "sufficient"}, 3)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_required", map[string]string{"delayedWETH": weth.Hex(), "balance": "insufficient"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_available", map[string]string{"delayedWETH": weth.Hex(), "balance": "sufficient"}, 3)
	requireGauge(t, snapshot, "op_dispute_mon_honest_actor_bonds", map[string]string{"honest_actor_address": actor.Hex(), "state": "pending"}, 3)
	requireGauge(t, snapshot, "op_dispute_mon_zk_games_pending_lifecycle_action", map[string]string{"action": "resolution"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_zk_games_pending_lifecycle_action", map[string]string{"action": "bond_distribution"}, 0)

	cl.AdvanceTime(time.Hour + time.Second)
	require.NoError(t, monitor.monitorGames())
	snapshot = metricstest.NewMetricChecker(t, metricer.Registry())
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_claims", map[string]string{"resolved": "unresolved", "resolvable": "resolvable"}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_zk_games_pending_lifecycle_action", map[string]string{"action": "resolution"}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_zk_games_pending_lifecycle_action", map[string]string{"action": "bond_distribution"}, 0)

	faultGame.Status = gameTypes.GameStatusDefenderWon
	faultGame.Claims[0].Resolved = true
	faultGame.Bonds[0].Resolved = true
	faultGame.Bonds[0].Recipient = actor
	faultGame.BondDistributionMode = faultTypes.NormalDistributionMode
	faultGame.ExpectedCredits[actor] = faultBond
	faultGame.Credits[actor] = faultBond
	zkGame.Status = gameTypes.GameStatusDefenderWon
	zkGame.ProposalStatus = contracts.ProposalStatusResolved
	zkGame.Bonds[0].Resolved = true
	zkGame.Bonds[0].Recipient = actor
	zkGame.ExpectedCredits[actor] = zkBond
	zkGame.Credits[actor] = zkBond
	require.NoError(t, monitor.monitorGames())
	snapshot = metricstest.NewMetricChecker(t, metricer.Registry())
	requireGauge(t, snapshot, "op_dispute_mon_games", map[string]string{"game_type": gameTypes.SuperCannonKonaGameType.String()}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_games", map[string]string{"game_type": gameTypes.ZKDisputeGameType.String()}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_failed_games", nil, 0)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_games_agreement", map[string]string{"root_agreement": "agree"}, 2)
	requireGauge(t, snapshot, "op_dispute_mon_games_agreement", map[string]string{"status": "agree_defender_wins", "completion": "complete", "result_correctness": "correct", "root_agreement": "agree"}, 2)
	requireGauge(t, snapshot, "op_dispute_mon_resolution_status", map[string]string{"completion": "complete", "max_duration": "before_max_duration"}, 1)
	requireGauge(t, snapshot, "op_dispute_mon_resolution_status", map[string]string{"completion": "resolvable", "max_duration": "before_max_duration"}, 0)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_claims", map[string]string{"resolved": "resolved"}, 1)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_claims", map[string]string{"resolved": "unresolved", "resolvable": "resolvable"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_credits", map[string]string{"credit": "expected", "withdrawable": "non_withdrawable"}, 2)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_credits", map[string]string{"credit": "above"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_credits", map[string]string{"credit": "below", "withdrawable": "non_withdrawable"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_required", map[string]string{"delayedWETH": weth.Hex(), "balance": "sufficient"}, 3)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_required", map[string]string{"delayedWETH": weth.Hex(), "balance": "insufficient"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_available", map[string]string{"delayedWETH": weth.Hex(), "balance": "sufficient"}, 3)
	requireGauge(t, snapshot, "op_dispute_mon_honest_actor_bonds", map[string]string{"honest_actor_address": actor.Hex(), "state": "pending"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_zk_games_pending_lifecycle_action", map[string]string{"action": "resolution"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_zk_games_pending_lifecycle_action", map[string]string{"action": "bond_distribution"}, 1)

	requestTime := big.NewInt(cl.Now().Unix())
	faultGame.WithdrawalRequests[actor] = &contracts.WithdrawalRequest{Amount: faultBond, Timestamp: requestTime}
	zkGame.BondDistributionMode = faultTypes.NormalDistributionMode
	zkGame.Credits[actor] = new(big.Int)
	zkGame.WithdrawalRequests[actor] = &contracts.WithdrawalRequest{Amount: zkBond, Timestamp: requestTime}
	require.NoError(t, monitor.monitorGames())
	snapshot = metricstest.NewMetricChecker(t, metricer.Registry())
	requireGauge(t, snapshot, "op_dispute_mon_withdrawal_requests", map[string]string{"delayedWETH": weth.Hex(), "credits": "matching"}, 2)
	requireGauge(t, snapshot, "op_dispute_mon_withdrawal_requests", map[string]string{"delayedWETH": weth.Hex(), "credits": "divergent"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_credits", map[string]string{"credit": "expected", "withdrawable": "non_withdrawable"}, 2)
	requireGaugeSum(t, metricer.Registry(), "op_dispute_mon_credits", map[string]string{"credit": "above"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_credits", map[string]string{"credit": "below", "withdrawable": "non_withdrawable"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_required", map[string]string{"delayedWETH": weth.Hex(), "balance": "sufficient"}, 3)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_required", map[string]string{"delayedWETH": weth.Hex(), "balance": "insufficient"}, 0)
	requireGauge(t, snapshot, "op_dispute_mon_bond_collateral_available", map[string]string{"delayedWETH": weth.Hex(), "balance": "sufficient"}, 3)
	requireGauge(t, snapshot, "op_dispute_mon_zk_games_pending_lifecycle_action", map[string]string{"action": "bond_distribution"}, 0)
}

func requireGauge(t *testing.T, checker *metricstest.MetricFamiliesChecker, name string, labels map[string]string, expected float64) {
	t.Helper()
	metric := checker.FindByName(name).FindByLabels(labels)
	require.NotNil(t, metric.GetGauge(), "metric %s with labels %v is not a gauge", name, labels)
	require.Equal(t, expected, metric.GetGauge().GetValue(), "metric %s with labels %v", name, labels)
}

func requireGaugeSum(t *testing.T, registry *prometheus.Registry, name string, labels map[string]string, expected float64) {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	var value float64
	foundFamily := false
	found := false
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		foundFamily = true
		for _, metric := range family.GetMetric() {
			matches := true
			for labelName, labelValue := range labels {
				matchedLabel := false
				for _, label := range metric.GetLabel() {
					if label.GetName() == labelName && label.GetValue() == labelValue {
						matchedLabel = true
						break
					}
				}
				if !matchedLabel {
					matches = false
					break
				}
			}
			if matches {
				found = true
				require.NotNil(t, metric.GetGauge(), "metric %s matching labels %v is not a gauge", name, labels)
				value += metric.GetGauge().GetValue()
			}
		}
		break
	}
	require.True(t, foundFamily, "metric family %s not found", name)
	require.True(t, found, "metric %s has no series matching labels %v", name, labels)
	require.Equal(t, expected, value, "metric %s matching labels %v", name, labels)
}
