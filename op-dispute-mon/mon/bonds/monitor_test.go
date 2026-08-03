package bonds

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var frozen = time.Unix(int64(time.Hour.Seconds()), 0)

func TestCheckBondsAggregatesGameKindsAndZerosAgedOutWETH(t *testing.T) {
	weth := common.Address{0xaa}
	fault := bondedFault(weth, 100, []monTypes.BondRecord{{Amount: big.NewInt(2)}})
	zk := bondedZK(weth, 100, []monTypes.BondRecord{{Amount: big.NewInt(3)}})

	monitor, recorded, _ := setupBondMetricsTest(t, nil)
	monitor.CheckBonds([]monTypes.BondedGame{fault, zk})
	require.Equal(t, big.NewInt(5), recorded.collateral[weth].Required)
	require.Equal(t, big.NewInt(100), recorded.collateral[weth].Actual)

	monitor.CheckBonds(nil)
	require.Equal(t, big.NewInt(0), recorded.collateral[weth].Required)
	require.Equal(t, big.NewInt(0), recorded.collateral[weth].Actual)
}

func TestCheckBondsUsesWithdrawalAmountAfterCreditUnlock(t *testing.T) {
	recipient := common.Address{0x11}
	game := bondedZK(common.Address{0xaa}, 100, nil)
	game.ExpectedCredits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.Credits = map[common.Address]*big.Int{recipient: new(big.Int)}
	game.WithdrawalRequests = map[common.Address]*contracts.WithdrawalRequest{
		recipient: {Amount: big.NewInt(10), Timestamp: big.NewInt(frozen.Unix())},
	}
	game.WETHDelay = time.Second

	monitor, recorded, _ := setupBondMetricsTest(t, nil)
	monitor.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, recorded.credits[metrics.CreditEqualNonWithdrawable])
	require.Equal(t, 0, recorded.credits[metrics.CreditBelowNonWithdrawable])
}

func TestCheckBondsTreatsCompletedWithdrawalAsPaid(t *testing.T) {
	recipient := common.Address{0x11}
	game := bondedZK(common.Address{0xaa}, 100, nil)
	game.ExpectedCredits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.Credits = map[common.Address]*big.Int{recipient: new(big.Int)}
	game.WithdrawalRequests = map[common.Address]*contracts.WithdrawalRequest{
		recipient: {Amount: new(big.Int), Timestamp: big.NewInt(frozen.Add(-30 * time.Minute).Unix())},
	}
	game.WETHDelay = time.Second

	monitor, recorded, _ := setupBondMetricsTest(t, nil)
	monitor.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, recorded.credits[metrics.CreditEqualWithdrawable])
	require.Equal(t, 0, recorded.credits[metrics.CreditBelowWithdrawable])
	require.Equal(t, big.NewInt(0), recorded.collateral[game.WETHContract].Required)
}

func TestCheckBondsDoesNotTreatEarlyDisappearanceAsPaid(t *testing.T) {
	recipient := common.Address{0x11}
	game := bondedZK(common.Address{0xaa}, 100, nil)
	game.ExpectedCredits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.Credits = map[common.Address]*big.Int{recipient: new(big.Int)}
	game.WithdrawalRequests = map[common.Address]*contracts.WithdrawalRequest{
		recipient: {Amount: new(big.Int), Timestamp: big.NewInt(frozen.Unix())},
	}
	game.WETHDelay = time.Hour

	monitor, recorded, _ := setupBondMetricsTest(t, nil)
	monitor.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, recorded.credits[metrics.CreditBelowNonWithdrawable])
	require.Equal(t, 0, recorded.credits[metrics.CreditEqualNonWithdrawable])
}

func TestCheckBondsMissingZKObligationIsBelowNonWithdrawable(t *testing.T) {
	recipient := common.Address{0x11}
	game := bondedZK(common.Address{0xaa}, 100, nil)
	game.ExpectedCredits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.Credits = map[common.Address]*big.Int{recipient: new(big.Int)}
	game.WithdrawalRequests = map[common.Address]*contracts.WithdrawalRequest{
		recipient: {Amount: new(big.Int), Timestamp: new(big.Int)},
	}
	// A ZK credit cannot legitimately disappear until phase one atomically creates a
	// DelayedWETH request, even if a theoretical game-level deadline has passed.
	game.CreditWithdrawableAt = frozen.Add(-time.Hour)
	monitor, recorded, logs := setupBondMetricsTest(t, nil)
	monitor.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, recorded.credits[metrics.CreditBelowNonWithdrawable])
	require.Equal(t, 0, recorded.credits[metrics.CreditBelowWithdrawable])
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelError),
		testlog.NewMessageFilter("Credit withdrawn early"),
		testlog.NewAttributesFilter("recipient", recipient.Hex()),
	))
}

func TestCheckBondsPreservesFaultWithdrawableDeadline(t *testing.T) {
	recipient := common.Address{0x11}
	game := bondedFault(common.Address{0xaa}, 100, nil)
	game.ExpectedCredits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.Credits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.WithdrawalRequests = map[common.Address]*contracts.WithdrawalRequest{
		recipient: {Amount: new(big.Int), Timestamp: new(big.Int)},
	}
	game.CreditWithdrawableAt = frozen.Add(-time.Second)

	monitor, recorded, _ := setupBondMetricsTest(t, nil)
	monitor.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, recorded.credits[metrics.CreditEqualWithdrawable])
	require.Equal(t, 0, recorded.credits[metrics.CreditEqualNonWithdrawable])
}

func TestCheckBondsUsesFaultUnlockTimestampAfterWithdrawalStarts(t *testing.T) {
	recipient := common.Address{0x11}
	game := bondedFault(common.Address{0xaa}, 100, nil)
	game.ExpectedCredits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.Credits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
	game.WithdrawalRequests = map[common.Address]*contracts.WithdrawalRequest{
		recipient: {Amount: big.NewInt(10), Timestamp: big.NewInt(frozen.Unix())},
	}
	game.WETHDelay = time.Hour
	game.CreditWithdrawableAt = frozen.Add(-time.Second)

	monitor, recorded, _ := setupBondMetricsTest(t, nil)
	monitor.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, recorded.credits[metrics.CreditEqualNonWithdrawable])
	require.Equal(t, 0, recorded.credits[metrics.CreditEqualWithdrawable])
}

func TestCheckBondsRecordsHonestActorsAcrossFaultAndZK(t *testing.T) {
	actor1 := common.Address{0x11}
	actor2 := common.Address{0x22}
	fault := bondedFault(common.Address{0xaa}, 100, []monTypes.BondRecord{
		{Depositor: actor1, Amount: big.NewInt(2)},
		{Depositor: actor1, Recipient: actor2, Amount: big.NewInt(3), Resolved: true},
	})
	zk := bondedZK(common.Address{0xaa}, 100, []monTypes.BondRecord{
		{Depositor: actor2, Recipient: actor2, Amount: big.NewInt(5), Resolved: true},
		{Depositor: actor1, Recipient: common.Address{}, Amount: big.NewInt(7), Resolved: true, Burned: true},
	})

	monitor, recorded, _ := setupBondMetricsTest(t, monTypes.NewHonestActors([]common.Address{actor1, actor2}))
	monitor.CheckBonds([]monTypes.BondedGame{fault, zk})
	require.Equal(t, big.NewInt(2), recorded.honest[actor1].Pending)
	require.Equal(t, big.NewInt(10), recorded.honest[actor1].Lost)
	require.Equal(t, big.NewInt(0), recorded.honest[actor1].Won)
	require.Equal(t, big.NewInt(0), recorded.honest[actor2].Pending)
	require.Equal(t, big.NewInt(0), recorded.honest[actor2].Lost)
	require.Equal(t, big.NewInt(3), recorded.honest[actor2].Won)
}

func TestCheckBondsUsesConservativeSharedBalance(t *testing.T) {
	weth := common.Address{0xaa}
	fault := bondedFault(weth, 100, nil)
	zk := bondedZK(weth, 90, nil)
	monitor, recorded, logs := setupBondMetricsTest(t, nil)

	monitor.CheckBonds([]monTypes.BondedGame{fault, zk})
	require.Equal(t, big.NewInt(90), recorded.collateral[weth].Actual)
	require.NotNil(t, logs.FindLog(testlog.NewMessageFilter("Games returned different balances for shared DelayedWETH")))
}

func TestCheckBondsRecordsEveryCreditBucket(t *testing.T) {
	recipient := common.Address{0x11}
	tests := []struct {
		expectation  metrics.CreditExpectation
		actual       int64
		withdrawable bool
	}{
		{metrics.CreditBelowWithdrawable, 9, true},
		{metrics.CreditEqualWithdrawable, 10, true},
		{metrics.CreditAboveWithdrawable, 11, true},
		{metrics.CreditBelowNonWithdrawable, 9, false},
		{metrics.CreditEqualNonWithdrawable, 10, false},
		{metrics.CreditAboveNonWithdrawable, 11, false},
	}
	games := make([]monTypes.BondedGame, 0, len(tests))
	for i, test := range tests {
		game := bondedFault(common.Address{byte(i + 1)}, 100, nil)
		game.ExpectedCredits = map[common.Address]*big.Int{recipient: big.NewInt(10)}
		game.Credits = map[common.Address]*big.Int{recipient: big.NewInt(test.actual)}
		game.Recipients = map[common.Address]bool{recipient: true}
		game.CreditWithdrawableAt = frozen.Add(time.Second)
		if test.withdrawable {
			game.CreditWithdrawableAt = frozen.Add(-time.Second)
		}
		games = append(games, game)
	}

	monitor, recorded, _ := setupBondMetricsTest(t, nil)
	monitor.CheckBonds(games)
	for _, test := range tests {
		require.Equal(t, 1, recorded.credits[test.expectation], test.expectation)
	}
}

func TestCheckBondsLogsInsufficientCollateral(t *testing.T) {
	weth := common.Address{0xaa}
	game := bondedZK(weth, 9, []monTypes.BondRecord{{Amount: big.NewInt(10)}})
	monitor, recorded, logs := setupBondMetricsTest(t, nil)

	monitor.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, big.NewInt(10), recorded.collateral[weth].Required)
	require.Equal(t, big.NewInt(9), recorded.collateral[weth].Actual)
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelError),
		testlog.NewMessageFilter("Insufficient collateral"),
		testlog.NewAttributesFilter("delayedWETH", weth.Hex()),
	))
}

func TestCheckBondsTreatsMissingCollateralAsZero(t *testing.T) {
	weth := common.Address{0xaa}
	game := bondedZK(weth, 0, []monTypes.BondRecord{{Amount: big.NewInt(10)}})
	game.ETHCollateral = nil
	monitor, recorded, logs := setupBondMetricsTest(t, nil)

	require.NotPanics(t, func() { monitor.CheckBonds([]monTypes.BondedGame{game}) })
	require.Equal(t, big.NewInt(10), recorded.collateral[weth].Required)
	require.Equal(t, big.NewInt(0), recorded.collateral[weth].Actual)
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelError),
		testlog.NewMessageFilter("Insufficient collateral"),
		testlog.NewAttributesFilter("delayedWETH", weth.Hex()),
		testlog.NewAttributesFilter("required", "10"),
		testlog.NewAttributesFilter("actual", "0"),
	))
}

func bondedFault(weth common.Address, balance int64, records []monTypes.BondRecord) *monTypes.FaultGameData {
	return &monTypes.FaultGameData{BondGameData: bondData(weth, balance, records)}
}

func bondedZK(weth common.Address, balance int64, records []monTypes.BondRecord) *monTypes.ZKGameData {
	return &monTypes.ZKGameData{BondGameData: bondData(weth, balance, records)}
}

func bondData(weth common.Address, balance int64, records []monTypes.BondRecord) monTypes.BondGameData {
	return monTypes.BondGameData{
		Bonds:              records,
		Credits:            map[common.Address]*big.Int{},
		ExpectedCredits:    map[common.Address]*big.Int{},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{},
		WETHContract:       weth,
		ETHCollateral:      big.NewInt(balance),
	}
}

func setupBondMetricsTest(t *testing.T, honestActors monTypes.HonestActors) (*Bonds, *stubBondMetrics, *testlog.CapturingHandler) {
	logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
	recorded := &stubBondMetrics{
		credits:    make(map[metrics.CreditExpectation]int),
		collateral: make(map[common.Address]Collateral),
		honest:     make(map[common.Address]metrics.HonestActorBondData),
	}
	return NewBonds(logger, recorded, clock.NewDeterministicClock(frozen), honestActors), recorded, logs
}

type stubBondMetrics struct {
	credits    map[metrics.CreditExpectation]int
	collateral map[common.Address]Collateral
	honest     map[common.Address]metrics.HonestActorBondData
}

func (s *stubBondMetrics) RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int) {
	s.collateral[addr] = Collateral{Required: new(big.Int).Set(required), Actual: new(big.Int).Set(available)}
}

func (s *stubBondMetrics) RecordCredit(expectation metrics.CreditExpectation, count int) {
	s.credits[expectation] = count
}

func (s *stubBondMetrics) RecordHonestActorBonds(address common.Address, data *metrics.HonestActorBondData) {
	s.honest[address] = metrics.HonestActorBondData{
		Pending: new(big.Int).Set(data.Pending),
		Lost:    new(big.Int).Set(data.Lost),
		Won:     new(big.Int).Set(data.Won),
	}
}
