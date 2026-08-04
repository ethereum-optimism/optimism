package extract

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestFaultBondSnapshotCompatibility(t *testing.T) {
	claimant := common.Address{0x01}
	counterer := common.Address{0x02}
	blockChallenger := common.Address{0x03}
	pendingClaimant := common.Address{0x04}
	block := rpcblock.ByHash(common.Hash{0xaa})
	game := &monTypes.FaultGameData{
		CommonGameData:        monTypes.CommonGameData{GameMetadata: faultGameMetadata(10)},
		MaxClockDuration:      20,
		BlockNumberChallenged: true,
		BlockNumberChallenger: blockChallenger,
		Claims: []monTypes.EnrichedClaim{
			{
				Claim: faultTypes.Claim{
					ClaimData: faultTypes.ClaimData{
						Position: faultTypes.RootPosition,
						Bond:     big.NewInt(7),
					},
					Claimant:    claimant,
					CounteredBy: counterer,
				},
				Resolved: true,
			},
			{
				Claim: faultTypes.Claim{
					ClaimData: faultTypes.ClaimData{Bond: big.NewInt(11)},
					Claimant:  pendingClaimant,
				},
			},
		},
	}
	withdrawals := []*contracts.WithdrawalRequest{
		{Amount: big.NewInt(1), Timestamp: big.NewInt(2)},
		{Amount: big.NewInt(3), Timestamp: big.NewInt(4)},
		{Amount: big.NewInt(5), Timestamp: big.NewInt(6)},
	}
	credits := []*big.Int{big.NewInt(13), big.NewInt(17), big.NewInt(19)}
	caller := &bondSnapshotCaller{
		withdrawals: withdrawals,
		credits:     credits,
		mode:        faultTypes.RefundDistributionMode,
		balance:     big.NewInt(23),
		delay:       30 * time.Second,
		weth:        common.Address{0xee},
	}

	err := NewBondDataEnricher().Enrich(t.Context(), block, faultBondCaller(caller), game)
	require.NoError(t, err)
	data := game.BondGameData
	require.Equal(t, []string{"withdrawals", "credits", "mode", "balance"}, caller.trace)
	require.Equal(t, []rpcblock.Block{block, block, block, block}, caller.blocks)
	require.Equal(t, []common.Address{counterer, blockChallenger, pendingClaimant}, caller.recipients)
	require.Equal(t, map[common.Address]bool{
		counterer:       true,
		blockChallenger: true,
		pendingClaimant: true,
	}, data.Recipients)
	require.Equal(t, map[common.Address]*big.Int{blockChallenger: big.NewInt(7)}, data.ExpectedCredits)
	require.Equal(t, []monTypes.BondRecord{
		{Depositor: claimant, Recipient: counterer, Amount: big.NewInt(7), Resolved: true, Forfeited: true},
		{Depositor: pendingClaimant, Amount: big.NewInt(11)},
	}, data.Bonds)
	require.Equal(t, faultTypes.RefundDistributionMode, data.BondDistributionMode)
	require.Equal(t, common.Address{0xee}, data.WETHContract)
	require.Equal(t, 30*time.Second, data.WETHDelay)
	require.Equal(t, big.NewInt(23), data.ETHCollateral)
	require.Equal(t, time.Unix(60, 0), data.CreditWithdrawableAt)

	for i, recipient := range caller.recipients {
		require.Equal(t, credits[i], data.Credits[recipient])
		require.Equal(t, withdrawals[i], data.WithdrawalRequests[recipient])
	}
	credits[0].SetInt64(99)
	withdrawals[0].Amount.SetInt64(99)
	caller.balance.SetInt64(99)
	game.Claims[0].Bond.SetInt64(99)
	require.Equal(t, big.NewInt(13), data.Credits[counterer])
	require.Equal(t, big.NewInt(1), data.WithdrawalRequests[counterer].Amount)
	require.Equal(t, big.NewInt(23), data.ETHCollateral)
	require.Equal(t, big.NewInt(7), data.Bonds[0].Amount)
	require.Equal(t, big.NewInt(7), data.ExpectedCredits[blockChallenger])
}

func TestFaultBondSnapshotStopsAfterErrors(t *testing.T) {
	testErr := errors.New("boom")
	tests := []struct {
		name          string
		configure     func(*bondSnapshotCaller)
		expectedTrace []string
		expectedError error
	}{
		{
			name:          "withdrawals",
			configure:     func(c *bondSnapshotCaller) { c.withdrawalsErr = testErr },
			expectedTrace: []string{"withdrawals"},
			expectedError: testErr,
		},
		{
			name:          "withdrawal count",
			configure:     func(c *bondSnapshotCaller) { c.withdrawals = append(c.withdrawals, &contracts.WithdrawalRequest{}) },
			expectedTrace: []string{"withdrawals"},
			expectedError: ErrIncorrectWithdrawalsCount,
		},
		{
			name:          "credits",
			configure:     func(c *bondSnapshotCaller) { c.creditsErr = testErr },
			expectedTrace: []string{"withdrawals", "credits"},
			expectedError: testErr,
		},
		{
			name:          "credit count",
			configure:     func(c *bondSnapshotCaller) { c.credits = append(c.credits, big.NewInt(1)) },
			expectedTrace: []string{"withdrawals", "credits"},
			expectedError: ErrIncorrectCreditCount,
		},
		{
			name:          "mode",
			configure:     func(c *bondSnapshotCaller) { c.modeErr = testErr },
			expectedTrace: []string{"withdrawals", "credits", "mode"},
			expectedError: testErr,
		},
		{
			name:          "balance",
			configure:     func(c *bondSnapshotCaller) { c.balanceErr = testErr },
			expectedTrace: []string{"withdrawals", "credits", "mode", "balance"},
			expectedError: testErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := &bondSnapshotCaller{
				withdrawals: []*contracts.WithdrawalRequest{{}},
				credits:     []*big.Int{big.NewInt(1)},
				balance:     big.NewInt(1),
			}
			test.configure(caller)
			game := &monTypes.FaultGameData{Claims: []monTypes.EnrichedClaim{{Claim: faultTypes.Claim{Claimant: common.Address{0x01}}}}}

			err := NewBondDataEnricher().Enrich(t.Context(), rpcblock.Latest, faultBondCaller(caller), game)
			require.ErrorIs(t, err, test.expectedError)
			require.Equal(t, monTypes.BondGameData{}, game.BondGameData)
			require.Equal(t, test.expectedTrace, caller.trace)
		})
	}
}

func TestFaultBondSnapshotPreservesNilAndZeroValues(t *testing.T) {
	recipient := common.Address{0x01}
	game := &monTypes.FaultGameData{Claims: []monTypes.EnrichedClaim{{Claim: faultTypes.Claim{Claimant: recipient}}}}
	tests := []struct {
		name       string
		withdrawal *contracts.WithdrawalRequest
	}{
		{name: "nil withdrawal"},
		{name: "nil withdrawal fields", withdrawal: &contracts.WithdrawalRequest{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := &bondSnapshotCaller{
				withdrawals: []*contracts.WithdrawalRequest{test.withdrawal},
				credits:     []*big.Int{nil},
			}

			err := NewBondDataEnricher().Enrich(t.Context(), rpcblock.Latest, faultBondCaller(caller), game)
			require.NoError(t, err)
			data := game.BondGameData
			require.Contains(t, data.Credits, recipient)
			require.Nil(t, data.Credits[recipient])
			require.Contains(t, data.WithdrawalRequests, recipient)
			if test.withdrawal == nil {
				require.Nil(t, data.WithdrawalRequests[recipient])
			} else {
				require.NotNil(t, data.WithdrawalRequests[recipient])
				require.Nil(t, data.WithdrawalRequests[recipient].Amount)
				require.Nil(t, data.WithdrawalRequests[recipient].Timestamp)
			}
			require.Nil(t, data.ETHCollateral)
			require.Zero(t, data.BondDistributionMode)
			require.Zero(t, data.WETHDelay)
			require.Zero(t, data.WETHContract)
		})
	}
}

func TestFaultBondSnapshotSumsExpectedCredits(t *testing.T) {
	recipient := common.Address{0x02}
	game := &monTypes.FaultGameData{Claims: []monTypes.EnrichedClaim{
		{
			Claim:    faultTypes.Claim{ClaimData: faultTypes.ClaimData{Bond: big.NewInt(2)}, Claimant: common.Address{0x01}, CounteredBy: recipient},
			Resolved: true,
		},
		{
			Claim:    faultTypes.Claim{ClaimData: faultTypes.ClaimData{Bond: big.NewInt(3)}, Claimant: common.Address{0x03}, CounteredBy: recipient},
			Resolved: true,
		},
	}}

	data := normalizeFaultBondData(game)
	require.Equal(t, map[common.Address]*big.Int{recipient: big.NewInt(5)}, data.ExpectedCredits)
}

func TestFaultBondSnapshotDistinguishesRefundsFromSelfCounters(t *testing.T) {
	refunded := common.Address{0x01}
	selfCountered := common.Address{0x02}
	game := &monTypes.FaultGameData{Claims: []monTypes.EnrichedClaim{
		{
			Claim:    faultTypes.Claim{ClaimData: faultTypes.ClaimData{Bond: big.NewInt(2)}, Claimant: refunded},
			Resolved: true,
		},
		{
			Claim: faultTypes.Claim{
				ClaimData:   faultTypes.ClaimData{Bond: big.NewInt(3)},
				Claimant:    selfCountered,
				CounteredBy: selfCountered,
			},
			Resolved: true,
		},
	}}

	data := normalizeFaultBondData(game)
	require.Equal(t, []monTypes.BondRecord{
		{Depositor: refunded, Recipient: refunded, Amount: big.NewInt(2), Resolved: true},
		{Depositor: selfCountered, Recipient: selfCountered, Amount: big.NewInt(3), Resolved: true, Forfeited: true},
	}, data.Bonds)
}

func TestFaultBondSnapshotRetainsZeroAddressBlockChallengeCredit(t *testing.T) {
	claimant := common.Address{0x01}
	counterer := common.Address{0x02}
	game := &monTypes.FaultGameData{
		BlockNumberChallenged: true,
		Claims: []monTypes.EnrichedClaim{{
			Claim: faultTypes.Claim{
				ClaimData:   faultTypes.ClaimData{Position: faultTypes.RootPosition, Bond: big.NewInt(7)},
				Claimant:    claimant,
				CounteredBy: counterer,
			},
			Resolved: true,
		}},
	}

	data := normalizeFaultBondData(game)
	require.Equal(t, map[common.Address]*big.Int{common.Address{}: big.NewInt(7)}, data.ExpectedCredits)
	require.NotContains(t, data.Recipients, common.Address{})
	require.Equal(t, map[common.Address]bool{counterer: true}, data.Recipients)
	require.Equal(t, counterer, data.Bonds[0].Recipient)
}

func faultGameMetadata(timestamp uint64) gameTypes.GameMetadata {
	return gameTypes.GameMetadata{Timestamp: timestamp}
}

type bondSnapshotCaller struct {
	trace          []string
	blocks         []rpcblock.Block
	recipients     []common.Address
	withdrawals    []*contracts.WithdrawalRequest
	withdrawalsErr error
	credits        []*big.Int
	creditsErr     error
	mode           faultTypes.BondDistributionMode
	modeErr        error
	balance        *big.Int
	balanceErr     error
	delay          time.Duration
	weth           common.Address
}

type faultBondSnapshotCaller struct {
	*mockGameCaller
	bond *bondSnapshotCaller
}

func faultBondCaller(bond *bondSnapshotCaller) *faultBondSnapshotCaller {
	return &faultBondSnapshotCaller{mockGameCaller: &mockGameCaller{}, bond: bond}
}

func (c *faultBondSnapshotCaller) GetWithdrawals(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	return c.bond.GetWithdrawals(ctx, block, recipients...)
}

func (c *faultBondSnapshotCaller) GetCredits(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	return c.bond.GetCredits(ctx, block, recipients...)
}

func (c *faultBondSnapshotCaller) GetBondDistributionMode(ctx context.Context, block rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return c.bond.GetBondDistributionMode(ctx, block)
}

func (c *faultBondSnapshotCaller) GetBalanceAndDelay(ctx context.Context, block rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	return c.bond.GetBalanceAndDelay(ctx, block)
}

func (c *bondSnapshotCaller) record(name string, block rpcblock.Block) {
	c.trace = append(c.trace, name)
	c.blocks = append(c.blocks, block)
}

func (c *bondSnapshotCaller) GetWithdrawals(_ context.Context, block rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	c.record("withdrawals", block)
	c.recipients = append([]common.Address(nil), recipients...)
	return c.withdrawals, c.withdrawalsErr
}

func (c *bondSnapshotCaller) GetCredits(_ context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	c.record("credits", block)
	requireSameRecipients(c.recipients, recipients)
	return c.credits, c.creditsErr
}

func (c *bondSnapshotCaller) GetBondDistributionMode(_ context.Context, block rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	c.record("mode", block)
	return c.mode, c.modeErr
}

func (c *bondSnapshotCaller) GetBalanceAndDelay(_ context.Context, block rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	c.record("balance", block)
	return c.balance, c.delay, c.weth, c.balanceErr
}

func requireSameRecipients(expected, actual []common.Address) {
	if len(expected) != len(actual) {
		panic("recipient arguments changed between RPCs")
	}
	for i := range expected {
		if expected[i] != actual[i] {
			panic("recipient arguments changed between RPCs")
		}
	}
}
