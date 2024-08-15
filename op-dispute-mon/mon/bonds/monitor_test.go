package bonds

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	frozen = time.Unix(int64(time.Hour.Seconds()), 0)
)

func TestCheckBonds(t *testing.T) {
	weth1 := common.Address{0x1a}
	weth1Balance := big.NewInt(4200)
	weth2 := common.Address{0x2b}
	weth2Balance := big.NewInt(10) // Insufficient
	game1 := &monTypes.EnrichedGameData{
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(2),
		},
		WETHContract:  weth1,
		ETHCollateral: weth1Balance,
	}
	game2 := &monTypes.EnrichedGameData{
		Credits: map[common.Address]*big.Int{
			common.Address{0x01}: big.NewInt(46),
		},
		WETHContract:  weth2,
		ETHCollateral: weth2Balance,
	}

	bonds, metrics, logs := setupBondMetricsTest(t)
	bonds.CheckBonds([]*monTypes.EnrichedGameData{game1, game2})

	require.Len(t, metrics.recorded, 2)
	require.Contains(t, metrics.recorded, weth1)
	require.Contains(t, metrics.recorded, weth2)
	require.Equal(t, metrics.recorded[weth1].Required.Uint64(), uint64(2))
	require.Equal(t, metrics.recorded[weth1].Actual.Uint64(), weth1Balance.Uint64())
	require.Equal(t, metrics.recorded[weth2].Required.Uint64(), uint64(46))
	require.Equal(t, metrics.recorded[weth2].Actual.Uint64(), weth2Balance.Uint64())

	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("Insufficient collateral"),
		testlog.NewAttributesFilter("delayedWETH", weth2.Hex()),
		testlog.NewAttributesFilter("required", "46"),
		testlog.NewAttributesFilter("actual", weth2Balance.String())))
	// No messages about weth1 since it has sufficient collateral
	require.Nil(t, logs.FindLog(testlog.NewAttributesFilter("delayedWETH", weth1.Hex())))
}

func TestCheckRecipientCredit(t *testing.T) {
	addr1 := common.Address{0x11, 0xaa}
	addr2 := common.Address{0x22, 0xbb}
	addr3 := common.Address{0x3c}
	addr4 := common.Address{0x4d}
	notRootPosition := types.NewPositionFromGIndex(big.NewInt(2))
	// Game has not reached max duration
	game1 := &monTypes.EnrichedGameData{
		MaxClockDuration: 50000,
		WETHDelay:        30 * time.Minute,
		GameMetadata: gameTypes.GameMetadata{
			Proxy:     common.Address{0x11},
			Timestamp: uint64(frozen.Unix()),
		},
		Claims: []monTypes.EnrichedClaim{
			{ // Expect 10 credits for addr1
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(10),
						Position: types.RootPosition,
					},
					Claimant: addr1,
				},
				Resolved: true,
			},
			{ // No expected credits as not resolved
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(15),
						Position: notRootPosition,
					},
					Claimant: addr1,
				},
				Resolved: false,
			},
			{ // Expect 5 credits for addr1
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(5),
						Position: notRootPosition,
					},
					Claimant: addr1,
				},
				Resolved: true,
			},
			{ // Expect 7 credits for addr2
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(7),
						Position: notRootPosition,
					},
					Claimant:    addr3,
					CounteredBy: addr2,
				},
				Resolved: true,
			},
			{ // Expect 3 credits for addr4
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(3),
						Position: notRootPosition,
					},
					Claimant: addr4,
				},
				Resolved: true,
			},
		},
		Credits: map[common.Address]*big.Int{
			// addr1 has correct credits
			addr1: big.NewInt(10 + 5),
			// addr2 has too few credits
			addr2: big.NewInt(2),
			// addr3 has too many credits
			addr3: big.NewInt(1),
			// addr4 has too few (no) credits
		},
		WETHContract:  common.Address{0xff},
		ETHCollateral: big.NewInt(6000),
	}
	// Max duration has been reached
	game2 := &monTypes.EnrichedGameData{
		MaxClockDuration: 5,
		WETHDelay:        5 * time.Second,
		GameMetadata: gameTypes.GameMetadata{
			Proxy:     common.Address{0x22},
			Timestamp: uint64(frozen.Unix()) - 11,
		},
		Claims: []monTypes.EnrichedClaim{
			{ // Expect 11 credits for addr1
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(11),
						Position: types.RootPosition,
					},
					Claimant: addr1,
				},
				Resolved: true,
			},
			{ // No expected credits as not resolved
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(15),
						Position: notRootPosition,
					},
					Claimant: addr1,
				},
				Resolved: false,
			},
			{ // Expect 6 credits for addr1
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(6),
						Position: notRootPosition,
					},
					Claimant: addr1,
				},
				Resolved: true,
			},
			{ // Expect 8 credits for addr2
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(8),
						Position: notRootPosition,
					},
					Claimant:    addr3,
					CounteredBy: addr2,
				},
				Resolved: true,
			},
			{ // Expect 4 credits for addr4
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(4),
						Position: notRootPosition,
					},
					Claimant: addr4,
				},
				Resolved: true,
			},
		},
		Credits: map[common.Address]*big.Int{
			// addr1 has too few credits
			addr1: big.NewInt(10),
			// addr2 has correct credits
			addr2: big.NewInt(8),
			// addr3 has too many credits
			addr3: big.NewInt(1),
			// addr4 has correct credits
			addr4: big.NewInt(4),
		},
		WETHContract:  common.Address{0xff},
		ETHCollateral: big.NewInt(6000),
	}

	// Game has not reached max duration
	game3 := &monTypes.EnrichedGameData{
		MaxClockDuration: 50000,
		WETHDelay:        10 * time.Hour,
		GameMetadata: gameTypes.GameMetadata{
			Proxy:     common.Address{0x33},
			Timestamp: uint64(frozen.Unix()) - 11,
		},
		Claims: []monTypes.EnrichedClaim{
			{ // Expect 9 credits for addr1
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(9),
						Position: types.RootPosition,
					},
					Claimant: addr1,
				},
				Resolved: true,
			},
			{ // Expect 6 credits for addr2
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(6),
						Position: notRootPosition,
					},
					Claimant:    addr4,
					CounteredBy: addr2,
				},
				Resolved: true,
			},
			{ // Expect 2 credits for addr4
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(2),
						Position: notRootPosition,
					},
					Claimant: addr4,
				},
				Resolved: true,
			},
		},
		Credits: map[common.Address]*big.Int{
			// addr1 has correct credits
			addr1: big.NewInt(9),
			// addr2 has too few credits
			addr2: big.NewInt(5),
			// addr3 is not involved in this game
			// addr4 has too many credits
			addr4: big.NewInt(3),
		},
		WETHContract:  common.Address{0xff},
		ETHCollateral: big.NewInt(6000),
	}

	// Game has not reached max duration
	game4 := &monTypes.EnrichedGameData{
		MaxClockDuration: 10,
		WETHDelay:        10 * time.Second,
		GameMetadata: gameTypes.GameMetadata{
			Proxy:     common.Address{0x44},
			Timestamp: uint64(frozen.Unix()) - 22,
		},
		BlockNumberChallenged: true,
		BlockNumberChallenger: addr1,
		Claims: []monTypes.EnrichedClaim{
			{ // Expect 9 credits for addr1 as the block number challenger
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(9),
						Position: types.RootPosition,
					},
					Claimant:    addr2,
					CounteredBy: addr3,
				},
				Resolved: true,
			},
			{ // Expect 6 credits for addr2
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(6),
						Position: notRootPosition,
					},
					Claimant:    addr4,
					CounteredBy: addr2,
				},
				Resolved: true,
			},
			{ // Expect 2 credits for addr4
				Claim: types.Claim{
					ClaimData: types.ClaimData{
						Bond:     big.NewInt(2),
						Position: notRootPosition,
					},
					Claimant: addr4,
				},
				Resolved: true,
			},
		},
		Credits: map[common.Address]*big.Int{
			// addr1 has correct credits
			addr1: big.NewInt(9),
			// addr2 has too few credits
			addr2: big.NewInt(5),
			// addr3 is not involved in this game
			// addr4 has too many credits
			addr4: big.NewInt(3),
		},
		WETHContract:  common.Address{0xff},
		ETHCollateral: big.NewInt(6000),
	}

	bonds, m, logs := setupBondMetricsTest(t)
	bonds.CheckBonds([]*monTypes.EnrichedGameData{game1, game2, game3, game4})

	require.Len(t, m.credits, 6)
	require.Contains(t, m.credits, metrics.CreditBelowWithdrawable)
	require.Contains(t, m.credits, metrics.CreditEqualWithdrawable)
	require.Contains(t, m.credits, metrics.CreditAboveWithdrawable)
	require.Contains(t, m.credits, metrics.CreditBelowNonWithdrawable)
	require.Contains(t, m.credits, metrics.CreditEqualNonWithdrawable)
	require.Contains(t, m.credits, metrics.CreditAboveNonWithdrawable)

	// Game 2 and 4 recipients added here as it has reached max duration
	require.Equal(t, 2, m.credits[metrics.CreditBelowWithdrawable], "CreditBelowWithdrawable")
	require.Equal(t, 3, m.credits[metrics.CreditEqualWithdrawable], "CreditEqualWithdrawable")
	require.Equal(t, 2, m.credits[metrics.CreditAboveWithdrawable], "CreditAboveWithdrawable")

	// Game 1 and 3 recipients added here as it hasn't reached max duration
	require.Equal(t, 3, m.credits[metrics.CreditBelowNonWithdrawable], "CreditBelowNonWithdrawable")
	require.Equal(t, 2, m.credits[metrics.CreditEqualNonWithdrawable], "CreditEqualNonWithdrawable")
	require.Equal(t, 2, m.credits[metrics.CreditAboveNonWithdrawable], "CreditAboveNonWithdrawable")

	// Logs from game1
	// addr1 is correct so has no logs
	// addr2 is below expected before max duration, so warn about early withdrawal
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelError),
		testlog.NewMessageFilter("Credit withdrawn early"),
		testlog.NewAttributesFilter("game", game1.Proxy.Hex()),
		testlog.NewAttributesFilter("recipient", addr2.Hex()),
		testlog.NewAttributesFilter("withdrawable", "non_withdrawable")))
	// addr3 is above expected
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewMessageFilter("Credit above expected amount"),
		testlog.NewAttributesFilter("game", game1.Proxy.Hex()),
		testlog.NewAttributesFilter("recipient", addr3.Hex()),
		testlog.NewAttributesFilter("withdrawable", "non_withdrawable")))
	// addr4 is below expected before max duration, so warn about early withdrawal
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelError),
		testlog.NewMessageFilter("Credit withdrawn early"),
		testlog.NewAttributesFilter("game", game1.Proxy.Hex()),
		testlog.NewAttributesFilter("recipient", addr4.Hex()),
		testlog.NewAttributesFilter("withdrawable", "non_withdrawable")))

	// Logs from game 2
	// addr1 is below expected - no warning as withdrawals may now be possible
	// addr2 is correct
	// addr3 is above expected - warn
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewMessageFilter("Credit above expected amount"),
		testlog.NewAttributesFilter("game", game2.Proxy.Hex()),
		testlog.NewAttributesFilter("recipient", addr3.Hex()),
		testlog.NewAttributesFilter("withdrawable", "withdrawable")))
	// addr4 is correct

	// Logs from game 3
	// addr1 is correct so has no logs
	// addr2 is below expected before max duration, so warn about early withdrawal
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelError),
		testlog.NewMessageFilter("Credit withdrawn early"),
		testlog.NewAttributesFilter("game", game3.Proxy.Hex()),
		testlog.NewAttributesFilter("recipient", addr2.Hex()),
		testlog.NewAttributesFilter("withdrawable", "non_withdrawable")))
	// addr3 is not involved so no logs
	// addr4 is above expected before max duration, so warn
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewMessageFilter("Credit above expected amount"),
		testlog.NewAttributesFilter("game", game3.Proxy.Hex()),
		testlog.NewAttributesFilter("recipient", addr4.Hex()),
		testlog.NewAttributesFilter("withdrawable", "non_withdrawable")))

	// Logs from game 4
	// addr1 is correct
	// addr2 is below expected before max duration, no log because withdrawals may be possible
	// addr3 is not involved so no logs
	// addr4 is above expected before max duration, so warn
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewMessageFilter("Credit above expected amount"),
		testlog.NewAttributesFilter("game", game4.Proxy.Hex()),
		testlog.NewAttributesFilter("recipient", addr4.Hex()),
		testlog.NewAttributesFilter("withdrawable", "withdrawable")))
}

func setupBondMetricsTest(t *testing.T) (*Bonds, *stubBondMetrics, *testlog.CapturingHandler) {
	logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
	metrics := &stubBondMetrics{
		credits:  make(map[metrics.CreditExpectation]int),
		recorded: make(map[common.Address]Collateral),
	}
	bonds := NewBonds(logger, metrics, clock.NewDeterministicClock(frozen))
	return bonds, metrics, logs
}

type stubBondMetrics struct {
	credits  map[metrics.CreditExpectation]int
	recorded map[common.Address]Collateral
}

func (s *stubBondMetrics) RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int) {
	s.recorded[addr] = Collateral{
		Required: required,
		Actual:   available,
	}
}

func (s *stubBondMetrics) RecordCredit(expectation metrics.CreditExpectation, count int) {
	s.credits[expectation] = count
}
