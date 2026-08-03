package extract

import (
	"context"
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

func TestNormalizeFaultBonds(t *testing.T) {
	creator := common.Address{0x11}
	counterer := common.Address{0x22}
	blockChallenger := common.Address{0x33}
	game := &monTypes.FaultGameData{
		BlockNumberChallenged: true,
		BlockNumberChallenger: blockChallenger,
		Claims: []monTypes.EnrichedClaim{
			{Claim: faultTypes.Claim{Claimant: creator, ClaimData: faultTypes.ClaimData{Bond: big.NewInt(10), Position: faultTypes.RootPosition}}, Resolved: true},
			{Claim: faultTypes.Claim{Claimant: creator, CounteredBy: counterer, ClaimData: faultTypes.ClaimData{Bond: big.NewInt(3), Position: faultTypes.NewPositionFromGIndex(big.NewInt(2))}}, Resolved: true},
			{Claim: faultTypes.Claim{Claimant: creator, ClaimData: faultTypes.ClaimData{Bond: big.NewInt(5), Position: faultTypes.NewPositionFromGIndex(big.NewInt(3))}}},
		},
	}

	require.NoError(t, normalizeFaultBonds(game, faultTypes.NormalDistributionMode))
	require.Equal(t, []monTypes.BondRecord{
		{Depositor: creator, Recipient: blockChallenger, Amount: big.NewInt(10), Resolved: true},
		{Depositor: creator, Recipient: counterer, Amount: big.NewInt(3), Resolved: true},
		{Depositor: creator, Amount: big.NewInt(5)},
	}, game.Bonds)
	require.Equal(t, big.NewInt(10), game.ExpectedCredits[blockChallenger])
	require.Equal(t, big.NewInt(3), game.ExpectedCredits[counterer])

	require.NoError(t, normalizeFaultBonds(game, faultTypes.RefundDistributionMode))
	require.Equal(t, big.NewInt(18), game.ExpectedCredits[creator])
	for _, bond := range game.Bonds {
		require.True(t, bond.Resolved)
		require.Equal(t, bond.Depositor, bond.Recipient)
	}
}

func TestNormalizeZKBonds(t *testing.T) {
	creator := common.Address{0x11}
	challenger := common.Address{0x22}
	prover := common.Address{0x33}
	tests := []struct {
		name       string
		status     gameTypes.GameStatus
		mode       faultTypes.BondDistributionMode
		challenger common.Address
		prover     common.Address
		want       []monTypes.BondRecord
	}{
		{
			name:   "live unchallenged",
			status: gameTypes.GameStatusInProgress,
			mode:   faultTypes.UndecidedDistributionMode,
			want:   []monTypes.BondRecord{{Depositor: creator, Amount: big.NewInt(100)}},
		},
		{
			name:       "live challenged",
			status:     gameTypes.GameStatusInProgress,
			mode:       faultTypes.UndecidedDistributionMode,
			challenger: challenger,
			want: []monTypes.BondRecord{
				{Depositor: creator, Amount: big.NewInt(70)},
				{Depositor: challenger, Amount: big.NewInt(30)},
			},
		},
		{
			name:       "challenger wins",
			status:     gameTypes.GameStatusChallengerWon,
			mode:       faultTypes.NormalDistributionMode,
			challenger: challenger,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: challenger, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true},
			},
		},
		{
			name:       "defender wins with distinct prover",
			status:     gameTypes.GameStatusDefenderWon,
			mode:       faultTypes.NormalDistributionMode,
			challenger: challenger,
			prover:     prover,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: prover, Amount: big.NewInt(30), Resolved: true},
			},
		},
		{
			name:       "defender wins with creator proving",
			status:     gameTypes.GameStatusDefenderWon,
			mode:       faultTypes.NormalDistributionMode,
			challenger: challenger,
			prover:     creator,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: creator, Amount: big.NewInt(30), Resolved: true},
			},
		},
		{
			name:       "refund",
			status:     gameTypes.GameStatusDefenderWon,
			mode:       faultTypes.RefundDistributionMode,
			challenger: challenger,
			prover:     prover,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true},
			},
		},
		{
			name:   "invalid parent burns unchallenged creator bond",
			status: gameTypes.GameStatusChallengerWon,
			mode:   faultTypes.NormalDistributionMode,
			want:   []monTypes.BondRecord{{Depositor: creator, Amount: big.NewInt(100), Resolved: true}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			game := &monTypes.ZKGameData{
				CommonGameData: monTypes.CommonGameData{Status: test.status},
				GameCreator:    creator,
				Challenger:     test.challenger,
				Prover:         test.prover,
				TotalBonds:     big.NewInt(100),
				ChallengerBond: big.NewInt(30),
			}
			require.NoError(t, normalizeZKBonds(game, test.mode))
			require.Equal(t, test.want, game.Bonds)
			for _, bond := range test.want {
				if bond.Resolved {
					require.Contains(t, game.ExpectedCredits, bond.Recipient)
				}
			}
		})
	}
}

func TestNormalizeZKBondsRejectsInvalidTotals(t *testing.T) {
	game := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
		GameCreator:    common.Address{0x11},
		Challenger:     common.Address{0x22},
		TotalBonds:     big.NewInt(29),
		ChallengerBond: big.NewInt(30),
	}
	require.ErrorIs(t, normalizeZKBonds(game, faultTypes.UndecidedDistributionMode), ErrInvalidZKBondTotals)
}

func TestBondDataEnricherLoadsPinnedStateForAllRecipients(t *testing.T) {
	creator := common.Address{0x11}
	challenger := common.Address{0x22}
	caller := &stubBondGameCaller{
		mode:        faultTypes.NormalDistributionMode,
		credits:     map[common.Address]*big.Int{creator: big.NewInt(70), challenger: big.NewInt(30)},
		withdrawals: map[common.Address]*contracts.WithdrawalRequest{},
		balance:     big.NewInt(100),
		delay:       time.Hour,
		weth:        common.Address{0xaa},
	}
	game := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusChallengerWon},
		GameCreator:    creator,
		Challenger:     challenger,
		TotalBonds:     big.NewInt(100),
		ChallengerBond: big.NewInt(30),
	}

	require.NoError(t, NewBondDataEnricher().Enrich(context.Background(), rpcblock.ByNumber(10), caller, game))
	require.Equal(t, []common.Address{creator, challenger}, caller.requestedCredits)
	require.Equal(t, caller.requestedCredits, caller.requestedWithdrawals)
	require.Equal(t, big.NewInt(100), game.ETHCollateral)
	require.Equal(t, big.NewInt(100), game.ExpectedCredits[challenger])
}

func TestBondDataEnricherLoadsZeroAddressBurnCredit(t *testing.T) {
	creator := common.Address{0x11}
	zero := common.Address{}
	caller := &stubBondGameCaller{
		mode:        faultTypes.NormalDistributionMode,
		credits:     map[common.Address]*big.Int{creator: new(big.Int), zero: big.NewInt(100)},
		withdrawals: map[common.Address]*contracts.WithdrawalRequest{},
		balance:     big.NewInt(100),
	}
	game := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusChallengerWon},
		GameCreator:    creator,
		TotalBonds:     big.NewInt(100),
		ChallengerBond: big.NewInt(30),
	}

	require.NoError(t, NewBondDataEnricher().Enrich(context.Background(), rpcblock.ByNumber(10), caller, game))
	require.Equal(t, []common.Address{zero, creator}, caller.requestedCredits)
	require.Equal(t, big.NewInt(100), game.ExpectedCredits[zero])
}

type stubBondGameCaller struct {
	mode                 faultTypes.BondDistributionMode
	credits              map[common.Address]*big.Int
	withdrawals          map[common.Address]*contracts.WithdrawalRequest
	balance              *big.Int
	delay                time.Duration
	weth                 common.Address
	requestedCredits     []common.Address
	requestedWithdrawals []common.Address
}

func (s *stubBondGameCaller) GetCredits(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	s.requestedCredits = append([]common.Address(nil), recipients...)
	result := make([]*big.Int, len(recipients))
	for i, recipient := range recipients {
		result[i] = s.credits[recipient]
	}
	return result, nil
}

func (s *stubBondGameCaller) GetBondDistributionMode(context.Context, rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return s.mode, nil
}

func (s *stubBondGameCaller) GetWithdrawals(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	s.requestedWithdrawals = append([]common.Address(nil), recipients...)
	result := make([]*contracts.WithdrawalRequest, len(recipients))
	for i, recipient := range recipients {
		request := s.withdrawals[recipient]
		if request == nil {
			request = &contracts.WithdrawalRequest{Amount: new(big.Int), Timestamp: new(big.Int)}
		}
		result[i] = request
	}
	return result, nil
}

func (s *stubBondGameCaller) GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	return s.balance, s.delay, s.weth, nil
}
