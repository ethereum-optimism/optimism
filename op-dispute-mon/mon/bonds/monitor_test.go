package bonds

import (
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var frozen = time.Unix(int64(time.Hour.Seconds()), 0)

func TestCheckBonds(t *testing.T) {
	weth1 := common.Address{0x1a}
	weth1Balance := big.NewInt(4200)
	weth2 := common.Address{0x2b}
	weth2Balance := big.NewInt(10)
	game1 := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Credits:      map[common.Address]*big.Int{common.Address{0x01}: big.NewInt(2)},
		WETHContract: weth1, ETHCollateral: weth1Balance,
	}}
	game2 := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		Credits:      map[common.Address]*big.Int{common.Address{0x01}: big.NewInt(46)},
		WETHContract: weth2, ETHCollateral: weth2Balance,
	}}

	bonds, metricer, logs := setupBondMetricsTest(t)
	bonds.CheckBonds([]monTypes.BondedGame{game1, game2})

	require.Len(t, metricer.recorded, 2)
	require.Equal(t, uint64(2), bigs.Uint64Strict(metricer.recorded[weth1].Required))
	require.Equal(t, bigs.Uint64Strict(weth1Balance), bigs.Uint64Strict(metricer.recorded[weth1].Actual))
	require.Equal(t, uint64(46), bigs.Uint64Strict(metricer.recorded[weth2].Required))
	require.Equal(t, bigs.Uint64Strict(weth2Balance), bigs.Uint64Strict(metricer.recorded[weth2].Actual))
	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("Insufficient collateral"),
		testlog.NewAttributesFilter("delayedWETH", weth2.Hex()),
		testlog.NewAttributesFilter("required", "46"),
		testlog.NewAttributesFilter("actual", weth2Balance.String())))
	require.Nil(t, logs.FindLog(testlog.NewAttributesFilter("delayedWETH", weth1.Hex())))
}

func TestCheckRecipientCredit(t *testing.T) {
	addr1 := common.Address{0x11, 0xaa}
	addr2 := common.Address{0x22, 0xbb}
	addr3 := common.Address{0x3c}
	addr4 := common.Address{0x4d}
	newGame := func(proxy common.Address, withdrawable bool, expected, actual map[common.Address]*big.Int) *monTypes.FaultGameData {
		withdrawableAt := frozen.Add(time.Hour)
		if withdrawable {
			withdrawableAt = frozen
		}
		return &monTypes.FaultGameData{
			CommonGameData: monTypes.CommonGameData{GameMetadata: gameTypes.GameMetadata{Proxy: proxy}},
			BondGameData: monTypes.BondGameData{
				ExpectedCredits:      expected,
				Credits:              actual,
				CreditWithdrawableAt: withdrawableAt,
				WETHContract:         common.Address{0xff},
				ETHCollateral:        big.NewInt(6000),
			},
		}
	}
	game1 := newGame(common.Address{0x11}, false,
		map[common.Address]*big.Int{addr1: big.NewInt(15), addr2: big.NewInt(7), addr4: big.NewInt(3)},
		map[common.Address]*big.Int{addr1: big.NewInt(15), addr2: big.NewInt(2), addr3: big.NewInt(1)})
	game2 := newGame(common.Address{0x22}, true,
		map[common.Address]*big.Int{addr1: big.NewInt(17), addr2: big.NewInt(8), addr4: big.NewInt(4)},
		map[common.Address]*big.Int{addr1: big.NewInt(10), addr2: big.NewInt(8), addr3: big.NewInt(1), addr4: big.NewInt(4)})
	game3 := newGame(common.Address{0x33}, false,
		map[common.Address]*big.Int{addr1: big.NewInt(9), addr2: big.NewInt(6), addr4: big.NewInt(2)},
		map[common.Address]*big.Int{addr1: big.NewInt(9), addr2: big.NewInt(5), addr4: big.NewInt(3)})
	game4 := newGame(common.Address{0x44}, true,
		map[common.Address]*big.Int{addr1: big.NewInt(9), addr2: big.NewInt(6), addr4: big.NewInt(2)},
		map[common.Address]*big.Int{addr1: big.NewInt(9), addr2: big.NewInt(5), addr4: big.NewInt(3)})
	game4.CreditWithdrawableAt = frozen.Add(-time.Second)

	bonds, metricer, logs := setupBondMetricsTest(t)
	bonds.CheckBonds([]monTypes.BondedGame{game1, game2, game3, game4})

	require.Equal(t, map[metrics.CreditExpectation]int{
		metrics.CreditBelowWithdrawable:    2,
		metrics.CreditEqualWithdrawable:    3,
		metrics.CreditAboveWithdrawable:    2,
		metrics.CreditBelowNonWithdrawable: 3,
		metrics.CreditEqualNonWithdrawable: 2,
		metrics.CreditAboveNonWithdrawable: 2,
	}, metricer.credits)
	require.NotNil(t, findCreditLog(logs, log.LevelError, "Credit withdrawn early", game1.Proxy, addr2, "non_withdrawable"))
	require.NotNil(t, findCreditLog(logs, log.LevelWarn, "Credit above expected amount", game1.Proxy, addr3, "non_withdrawable"))
	require.NotNil(t, findCreditLog(logs, log.LevelError, "Credit withdrawn early", game1.Proxy, addr4, "non_withdrawable"))
	require.NotNil(t, findCreditLog(logs, log.LevelWarn, "Credit above expected amount", game2.Proxy, addr3, "withdrawable"))
	require.NotNil(t, findCreditLog(logs, log.LevelError, "Credit withdrawn early", game3.Proxy, addr2, "non_withdrawable"))
	require.NotNil(t, findCreditLog(logs, log.LevelWarn, "Credit above expected amount", game3.Proxy, addr4, "non_withdrawable"))
	require.NotNil(t, findCreditLog(logs, log.LevelWarn, "Credit above expected amount", game4.Proxy, addr4, "withdrawable"))
}

func TestZKBondConsumersRecordAllCreditBuckets(t *testing.T) {
	recipient := common.Address{0x01}
	newGame := func(proxy byte, credit, requestAmount *big.Int, timestamp int64) *monTypes.ZKGameData {
		return &monTypes.ZKGameData{
			CommonGameData: monTypes.CommonGameData{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{proxy}}},
			BondGameData: monTypes.BondGameData{
				ExpectedCredits: map[common.Address]*big.Int{recipient: big.NewInt(10)},
				Credits:         map[common.Address]*big.Int{recipient: credit},
				WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
					recipient: {Amount: requestAmount, Timestamp: big.NewInt(timestamp)},
				},
				BondDistributionMode: faultTypes.NormalDistributionMode,
				WETHContract:         common.Address{0xee},
				WETHDelay:            10 * time.Second,
				ETHCollateral:        big.NewInt(1000),
			},
		}
	}
	matureTimestamp := frozen.Unix() - 10
	games := []monTypes.BondedGame{
		newGame(1, big.NewInt(9), new(big.Int), 0),
		newGame(2, big.NewInt(10), new(big.Int), 0),
		newGame(3, big.NewInt(11), new(big.Int), 0),
		newGame(4, new(big.Int), big.NewInt(9), matureTimestamp),
		newGame(5, new(big.Int), big.NewInt(10), matureTimestamp),
		newGame(6, new(big.Int), big.NewInt(11), matureTimestamp-1),
	}

	bonds, metricer, _ := setupBondMetricsTest(t)
	bonds.CheckBonds(games)
	require.Equal(t, map[metrics.CreditExpectation]int{
		metrics.CreditBelowWithdrawable:    1,
		metrics.CreditEqualWithdrawable:    1,
		metrics.CreditAboveWithdrawable:    1,
		metrics.CreditBelowNonWithdrawable: 1,
		metrics.CreditEqualNonWithdrawable: 1,
		metrics.CreditAboveNonWithdrawable: 1,
	}, metricer.credits)
}

func TestZKBondConsumersIgnoreEmptyCreditSlots(t *testing.T) {
	recipient := common.Address{0x01}
	game := &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
		ExpectedCredits: map[common.Address]*big.Int{},
		Credits:         map[common.Address]*big.Int{recipient: new(big.Int)},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			recipient: {Amount: new(big.Int), Timestamp: new(big.Int)},
		},
		WETHContract:  common.Address{0xee},
		ETHCollateral: big.NewInt(1000),
	}}

	bonds, metricer, _ := setupBondMetricsTest(t)
	bonds.CheckBonds([]monTypes.BondedGame{game})
	for _, count := range metricer.credits {
		require.Zero(t, count)
	}
}

func TestZKBondConsumersRecognizeCompletedMaturePayout(t *testing.T) {
	recipient := common.Address{0x01}
	game := &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
		ExpectedCredits: map[common.Address]*big.Int{recipient: big.NewInt(10)},
		Credits:         map[common.Address]*big.Int{recipient: new(big.Int)},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{
			recipient: {Amount: new(big.Int), Timestamp: big.NewInt(frozen.Unix() - 10)},
		},
		BondDistributionMode: faultTypes.NormalDistributionMode,
		WETHContract:         common.Address{0xee},
		WETHDelay:            10 * time.Second,
		ETHCollateral:        big.NewInt(1000),
	}}

	bonds, metricer, _ := setupBondMetricsTest(t)
	bonds.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, metricer.credits[metrics.CreditEqualWithdrawable])
	require.Zero(t, metricer.credits[metrics.CreditBelowWithdrawable])
}

func TestFaultBondConsumersPreserveEmptyCreditSlots(t *testing.T) {
	recipient := common.Address{0x01}
	game := &monTypes.FaultGameData{BondGameData: monTypes.BondGameData{
		ExpectedCredits:      map[common.Address]*big.Int{},
		Credits:              map[common.Address]*big.Int{recipient: new(big.Int)},
		CreditWithdrawableAt: frozen.Add(time.Hour),
		WETHContract:         common.Address{0xee},
		ETHCollateral:        big.NewInt(1000),
	}}

	bonds, metricer, _ := setupBondMetricsTest(t)
	bonds.CheckBonds([]monTypes.BondedGame{game})
	require.Equal(t, 1, metricer.credits[metrics.CreditEqualNonWithdrawable])
}

func TestCheckBondsRecordsHonestActorBonds(t *testing.T) {
	actor1 := common.Address{0x01}
	actor2 := common.Address{0x02}
	game1 := &monTypes.FaultGameData{
		CommonGameData: monTypes.CommonGameData{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0xaa}}},
		BondGameData: monTypes.BondGameData{
			Bonds: []monTypes.BondRecord{
				{Depositor: actor1, Amount: big.NewInt(5)},
				{Depositor: actor1, Recipient: common.Address{0x03}, Amount: big.NewInt(2), Resolved: true, Forfeited: true},
				{Depositor: actor2, Recipient: actor2, Amount: big.NewInt(1), Resolved: true},
				{Depositor: common.Address{0x04}, Recipient: actor2, Amount: big.NewInt(3), Resolved: true, Forfeited: true},
				{Depositor: actor2, Recipient: actor2, Amount: big.NewInt(4), Resolved: true, Forfeited: true},
			},
			WETHContract:  common.Address{0xff},
			ETHCollateral: big.NewInt(100),
		},
	}
	game2 := &monTypes.FaultGameData{
		CommonGameData: monTypes.CommonGameData{GameMetadata: gameTypes.GameMetadata{Proxy: common.Address{0xbb}}},
		BondGameData:   game1.BondGameData,
	}

	bonds, metricer, _ := setupBondMetricsTestWithHonestActors(t, monTypes.NewHonestActors([]common.Address{{}, actor1, actor2}))
	bonds.CheckBonds([]monTypes.BondedGame{game1, game2})

	require.Equal(t, big.NewInt(10), metricer.honest[actor1].Pending)
	require.Equal(t, big.NewInt(4), metricer.honest[actor1].Lost)
	require.Equal(t, big.NewInt(0), metricer.honest[actor1].Won)
	require.Equal(t, big.NewInt(0), metricer.honest[actor2].Pending)
	require.Equal(t, big.NewInt(8), metricer.honest[actor2].Lost)
	require.Equal(t, big.NewInt(14), metricer.honest[actor2].Won)
	// Intentional existing-output change: address zero is a contract sentinel, not an honest actor.
	require.NotContains(t, metricer.honest, common.Address{})
}

func TestZKBondConsumersRecordHonestActors(t *testing.T) {
	creator := common.Address{0x01}
	challenger := common.Address{0x02}
	prover := common.Address{0x03}
	newGame := func(bonds []monTypes.BondRecord) *monTypes.ZKGameData {
		return &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
			Bonds:              bonds,
			Credits:            map[common.Address]*big.Int{},
			ExpectedCredits:    map[common.Address]*big.Int{},
			WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{},
			WETHContract:       common.Address{0xee},
			ETHCollateral:      big.NewInt(1000),
		}}
	}
	games := []monTypes.BondedGame{
		newGame([]monTypes.BondRecord{
			{Depositor: creator, Amount: big.NewInt(70)},
			{Depositor: challenger, Amount: big.NewInt(30), ChallengerBond: true},
		}),
		newGame([]monTypes.BondRecord{
			{Depositor: creator, Recipient: challenger, Amount: big.NewInt(70), Resolved: true, Forfeited: true},
			{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
		}),
		newGame([]monTypes.BondRecord{
			{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
			{Depositor: challenger, Recipient: prover, Amount: big.NewInt(30), Resolved: true, Forfeited: true, ChallengerBond: true},
		}),
		newGame([]monTypes.BondRecord{
			{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
			{Depositor: creator, Recipient: prover, Amount: big.NewInt(30), Resolved: true, Forfeited: true, ChallengerBond: true},
		}),
		newGame([]monTypes.BondRecord{{
			Depositor: creator, Recipient: common.Address{}, Amount: big.NewInt(100), Resolved: true, Forfeited: true,
		}}),
	}

	bonds, metricer, _ := setupBondMetricsTestWithHonestActors(t, monTypes.NewHonestActors([]common.Address{{}, creator, challenger, prover}))
	bonds.CheckBonds(games)
	require.Equal(t, metrics.HonestActorBondData{Pending: big.NewInt(70), Lost: big.NewInt(200), Won: new(big.Int)}, metricer.honest[creator])
	require.Equal(t, metrics.HonestActorBondData{Pending: big.NewInt(30), Lost: big.NewInt(30), Won: big.NewInt(70)}, metricer.honest[challenger])
	require.Equal(t, metrics.HonestActorBondData{Pending: new(big.Int), Lost: new(big.Int), Won: big.NewInt(60)}, metricer.honest[prover])
	require.NotContains(t, metricer.honest, common.Address{})
}

func TestZKBondConsumersDoNotForfeitReturnedOverlappingRoles(t *testing.T) {
	creator := common.Address{0x01}
	challenger := common.Address{0x02}
	game := &monTypes.ZKGameData{BondGameData: monTypes.BondGameData{
		Bonds: []monTypes.BondRecord{
			{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
			{Depositor: creator, Recipient: creator, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
		},
		Credits:            map[common.Address]*big.Int{},
		ExpectedCredits:    map[common.Address]*big.Int{},
		WithdrawalRequests: map[common.Address]*contracts.WithdrawalRequest{},
		WETHContract:       common.Address{0xee},
		ETHCollateral:      big.NewInt(1000),
	}}

	bonds, metricer, _ := setupBondMetricsTestWithHonestActors(t, monTypes.NewHonestActors([]common.Address{creator, challenger}))
	bonds.CheckBonds([]monTypes.BondedGame{game})
	zero := metrics.HonestActorBondData{Pending: new(big.Int), Lost: new(big.Int), Won: new(big.Int)}
	require.Equal(t, zero, metricer.honest[creator])
	require.Equal(t, zero, metricer.honest[challenger])
}

func findCreditLog(logs *testlog.CapturingHandler, level slog.Level, message string, game, recipient common.Address, withdrawable string) *testlog.CapturedRecord {
	return logs.FindLog(
		testlog.NewLevelFilter(level),
		testlog.NewMessageFilter(message),
		testlog.NewAttributesFilter("game", game.Hex()),
		testlog.NewAttributesFilter("recipient", recipient.Hex()),
		testlog.NewAttributesFilter("withdrawable", withdrawable))
}

func setupBondMetricsTest(t *testing.T) (*Bonds, *stubBondMetrics, *testlog.CapturingHandler) {
	return setupBondMetricsTestWithHonestActors(t, nil)
}

func setupBondMetricsTestWithHonestActors(t *testing.T, honestActors monTypes.HonestActors) (*Bonds, *stubBondMetrics, *testlog.CapturingHandler) {
	logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
	metricer := &stubBondMetrics{
		credits:  make(map[metrics.CreditExpectation]int),
		recorded: make(map[common.Address]Collateral),
		honest:   make(map[common.Address]metrics.HonestActorBondData),
	}
	return NewBonds(logger, metricer, clock.NewDeterministicClock(frozen), honestActors), metricer, logs
}

type stubBondMetrics struct {
	credits  map[metrics.CreditExpectation]int
	recorded map[common.Address]Collateral
	honest   map[common.Address]metrics.HonestActorBondData
}

func (s *stubBondMetrics) RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int) {
	s.recorded[addr] = Collateral{Required: required, Actual: available}
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
