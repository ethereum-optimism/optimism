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

func TestNormalizeFaultBondsReturnsResolvedUncounteredBondToClaimant(t *testing.T) {
	claimant := common.Address{0x11}
	game := &monTypes.FaultGameData{
		Claims: []monTypes.EnrichedClaim{{
			Claim: faultTypes.Claim{
				Claimant:  claimant,
				ClaimData: faultTypes.ClaimData{Bond: big.NewInt(10), Position: faultTypes.RootPosition},
			},
			Resolved: true,
		}},
	}

	require.NoError(t, normalizeFaultBonds(game, faultTypes.NormalDistributionMode))
	require.Equal(t, []monTypes.BondRecord{{
		Depositor: claimant,
		Recipient: claimant,
		Amount:    big.NewInt(10),
		Resolved:  true,
	}}, game.Bonds)
	require.Equal(t, map[common.Address]*big.Int{claimant: big.NewInt(10)}, game.ExpectedCredits)
}

func TestNormalizeFaultBondsRefundsEachDepositor(t *testing.T) {
	firstClaimant := common.Address{0x11}
	secondClaimant := common.Address{0x22}
	game := &monTypes.FaultGameData{Claims: []monTypes.EnrichedClaim{
		{Claim: faultTypes.Claim{
			Claimant:  firstClaimant,
			ClaimData: faultTypes.ClaimData{Bond: big.NewInt(10), Position: faultTypes.RootPosition},
		}},
		{Claim: faultTypes.Claim{
			Claimant: secondClaimant,
			ClaimData: faultTypes.ClaimData{
				Bond:     big.NewInt(3),
				Position: faultTypes.NewPositionFromGIndex(big.NewInt(2)),
			},
		}},
	}}

	require.NoError(t, normalizeFaultBonds(game, faultTypes.RefundDistributionMode))
	require.Equal(t, []monTypes.BondRecord{
		{Depositor: firstClaimant, Recipient: firstClaimant, Amount: big.NewInt(10), Resolved: true},
		{Depositor: secondClaimant, Recipient: secondClaimant, Amount: big.NewInt(3), Resolved: true},
	}, game.Bonds)
	require.Equal(t, map[common.Address]*big.Int{
		firstClaimant:  big.NewInt(10),
		secondClaimant: big.NewInt(3),
	}, game.ExpectedCredits)
}

func TestNormalizeZKBonds(t *testing.T) {
	creator := common.Address{0x11}
	challenger := common.Address{0x22}
	prover := common.Address{0x33}
	tests := []struct {
		name        string
		status      gameTypes.GameStatus
		mode        faultTypes.BondDistributionMode
		challenger  common.Address
		prover      common.Address
		want        []monTypes.BondRecord
		wantCredits map[common.Address]*big.Int
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
				{Depositor: challenger, Amount: big.NewInt(30), ChallengerBond: true},
			},
		},
		{
			name:       "challenger wins",
			status:     gameTypes.GameStatusChallengerWon,
			mode:       faultTypes.NormalDistributionMode,
			challenger: challenger,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: challenger, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			wantCredits: map[common.Address]*big.Int{challenger: big.NewInt(100)},
		},
		{
			name:       "defender wins with distinct prover",
			status:     gameTypes.GameStatusDefenderWon,
			mode:       faultTypes.NormalDistributionMode,
			challenger: challenger,
			prover:     prover,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: prover, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			wantCredits: map[common.Address]*big.Int{creator: big.NewInt(70), prover: big.NewInt(30)},
		},
		{
			name:       "defender wins after creator self-challenges",
			status:     gameTypes.GameStatusDefenderWon,
			mode:       faultTypes.NormalDistributionMode,
			challenger: creator,
			prover:     prover,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: creator, Recipient: prover, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			wantCredits: map[common.Address]*big.Int{creator: big.NewInt(70), prover: big.NewInt(30)},
		},
		{
			name:       "defender wins with creator proving",
			status:     gameTypes.GameStatusDefenderWon,
			mode:       faultTypes.NormalDistributionMode,
			challenger: challenger,
			prover:     creator,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: creator, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			wantCredits: map[common.Address]*big.Int{creator: big.NewInt(100)},
		},
		{
			name:   "defender wins after unchallenged proof",
			status: gameTypes.GameStatusDefenderWon,
			mode:   faultTypes.NormalDistributionMode,
			prover: prover,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(100), Resolved: true},
			},
			wantCredits: map[common.Address]*big.Int{creator: big.NewInt(100)},
		},
		{
			name:   "defender wins without challenger",
			status: gameTypes.GameStatusDefenderWon,
			mode:   faultTypes.NormalDistributionMode,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(100), Resolved: true},
			},
			wantCredits: map[common.Address]*big.Int{creator: big.NewInt(100)},
		},
		{
			name:       "refund",
			status:     gameTypes.GameStatusDefenderWon,
			mode:       faultTypes.RefundDistributionMode,
			challenger: challenger,
			prover:     prover,
			want: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			wantCredits: map[common.Address]*big.Int{creator: big.NewInt(70), challenger: big.NewInt(30)},
		},
		{
			name:        "invalid parent burns unchallenged creator bond",
			status:      gameTypes.GameStatusChallengerWon,
			mode:        faultTypes.NormalDistributionMode,
			want:        []monTypes.BondRecord{{Depositor: creator, Amount: big.NewInt(100), Resolved: true, Burned: true}},
			wantCredits: map[common.Address]*big.Int{common.Address{}: big.NewInt(100)},
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
			if test.wantCredits == nil {
				test.wantCredits = map[common.Address]*big.Int{}
			}
			require.Equal(t, test.wantCredits, game.ExpectedCredits)
		})
	}
}

func TestNormalizeZKBondsAllowsUnchallengedTotalBelowChallengerBond(t *testing.T) {
	creator := common.Address{0x11}
	game := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
		GameCreator:    creator,
		TotalBonds:     big.NewInt(29),
		ChallengerBond: big.NewInt(30),
	}

	require.NoError(t, normalizeZKBonds(game, faultTypes.UndecidedDistributionMode))
	require.Equal(t, []monTypes.BondRecord{{Depositor: creator, Amount: big.NewInt(29)}}, game.Bonds)
}

func TestNormalizeZKBondsRejectsInvalidTotals(t *testing.T) {
	tests := []struct {
		name           string
		totalBonds     *big.Int
		challengerBond *big.Int
	}{
		{name: "nil total", challengerBond: big.NewInt(30)},
		{name: "nil challenger bond", totalBonds: big.NewInt(100)},
		{name: "negative total", totalBonds: big.NewInt(-1), challengerBond: big.NewInt(30)},
		{name: "negative challenger bond", totalBonds: big.NewInt(100), challengerBond: big.NewInt(-1)},
		{name: "total below challenger bond", totalBonds: big.NewInt(29), challengerBond: big.NewInt(30)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			game := &monTypes.ZKGameData{
				CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
				GameCreator:    common.Address{0x11},
				Challenger:     common.Address{0x22},
				TotalBonds:     test.totalBonds,
				ChallengerBond: test.challengerBond,
			}
			require.ErrorIs(t, normalizeZKBonds(game, faultTypes.UndecidedDistributionMode), ErrInvalidZKBondTotals)
		})
	}
}

func TestNormalizeZKBondsCopiesAmounts(t *testing.T) {
	totalBonds := big.NewInt(100)
	challengerBond := big.NewInt(30)
	game := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
		GameCreator:    common.Address{0x11},
		Challenger:     common.Address{0x22},
		TotalBonds:     totalBonds,
		ChallengerBond: challengerBond,
	}
	require.NoError(t, normalizeZKBonds(game, faultTypes.UndecidedDistributionMode))

	totalBonds.SetInt64(200)
	challengerBond.SetInt64(40)
	require.Equal(t, big.NewInt(70), game.Bonds[0].Amount)
	require.Equal(t, big.NewInt(30), game.Bonds[1].Amount)

	game.Bonds[0].Amount.SetInt64(1)
	game.Bonds[1].Amount.SetInt64(2)
	require.Equal(t, big.NewInt(200), totalBonds)
	require.Equal(t, big.NewInt(40), challengerBond)
}

func TestBondDataEnricherLoadsPinnedStateForAllRecipients(t *testing.T) {
	creator := common.Address{0x11}
	challenger := common.Address{0x22}
	creatorWithdrawal := &contracts.WithdrawalRequest{Amount: big.NewInt(5), Timestamp: big.NewInt(6)}
	caller := &stubBondGameCaller{
		mode:        faultTypes.NormalDistributionMode,
		credits:     map[common.Address]*big.Int{creator: big.NewInt(70), challenger: big.NewInt(30)},
		withdrawals: map[common.Address]*contracts.WithdrawalRequest{creator: creatorWithdrawal},
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
	require.True(t, game.CreditWithdrawableAt.IsZero())

	caller.balance.SetInt64(200)
	caller.credits[creator].SetInt64(200)
	creatorWithdrawal.Amount.SetInt64(200)
	creatorWithdrawal.Timestamp.SetInt64(200)
	require.Equal(t, big.NewInt(100), game.ETHCollateral)
	require.Equal(t, big.NewInt(70), game.Credits[creator])
	require.Equal(t, big.NewInt(5), game.WithdrawalRequests[creator].Amount)
	require.Equal(t, big.NewInt(6), game.WithdrawalRequests[creator].Timestamp)
}

func TestBondDataEnricherNormalizesNilCredit(t *testing.T) {
	creator := common.Address{0x11}
	caller := &stubBondGameCaller{
		mode:        faultTypes.UndecidedDistributionMode,
		credits:     map[common.Address]*big.Int{creator: nil},
		withdrawals: map[common.Address]*contracts.WithdrawalRequest{},
		balance:     big.NewInt(100),
	}
	game := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
		GameCreator:    creator,
		TotalBonds:     big.NewInt(100),
		ChallengerBond: big.NewInt(30),
	}

	require.NoError(t, NewBondDataEnricher().Enrich(context.Background(), rpcblock.ByNumber(10), caller, game))
	require.NotNil(t, game.Credits[creator])
	require.Zero(t, game.Credits[creator].Sign())
}

func TestBondDataEnricherNormalizesNilWithdrawal(t *testing.T) {
	creator := common.Address{0x11}
	tests := []struct {
		name       string
		withdrawal *contracts.WithdrawalRequest
	}{
		{name: "nil request"},
		{name: "nil request fields", withdrawal: &contracts.WithdrawalRequest{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := &stubBondGameCaller{
				mode:        faultTypes.UndecidedDistributionMode,
				credits:     map[common.Address]*big.Int{creator: new(big.Int)},
				withdrawals: map[common.Address]*contracts.WithdrawalRequest{creator: test.withdrawal},
				balance:     big.NewInt(100),
				rawRequests: true,
			}
			game := &monTypes.ZKGameData{
				CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
				GameCreator:    creator,
				TotalBonds:     big.NewInt(100),
				ChallengerBond: big.NewInt(30),
			}

			require.NoError(t, NewBondDataEnricher().Enrich(context.Background(), rpcblock.ByNumber(10), caller, game))
			request := game.WithdrawalRequests[creator]
			require.NotNil(t, request)
			require.NotNil(t, request.Amount)
			require.NotNil(t, request.Timestamp)
			require.Zero(t, request.Amount.Sign())
			require.Zero(t, request.Timestamp.Sign())
		})
	}
}

func TestBondDataEnricherLoadsFaultState(t *testing.T) {
	creator := common.Address{0x11}
	counterer := common.Address{0x22}
	blockChallenger := common.Address{0x33}
	caller := &stubBondGameCaller{
		mode: faultTypes.NormalDistributionMode,
		credits: map[common.Address]*big.Int{
			creator: big.NewInt(10), counterer: big.NewInt(3), blockChallenger: big.NewInt(7),
		},
		withdrawals: map[common.Address]*contracts.WithdrawalRequest{},
		balance:     big.NewInt(20),
		delay:       time.Hour,
		weth:        common.Address{0xaa},
	}
	game := &monTypes.FaultGameData{
		CommonGameData:        monTypes.CommonGameData{GameMetadata: gameTypes.GameMetadata{Timestamp: 1_000}},
		MaxClockDuration:      50,
		BlockNumberChallenged: true,
		BlockNumberChallenger: blockChallenger,
		Claims: []monTypes.EnrichedClaim{
			{Claim: faultTypes.Claim{Claimant: creator, ClaimData: faultTypes.ClaimData{Bond: big.NewInt(7), Position: faultTypes.RootPosition}}, Resolved: true},
			{Claim: faultTypes.Claim{Claimant: creator, CounteredBy: counterer, ClaimData: faultTypes.ClaimData{Bond: big.NewInt(3), Position: faultTypes.NewPositionFromGIndex(big.NewInt(2))}}, Resolved: true},
		},
	}

	require.NoError(t, NewBondDataEnricher().Enrich(context.Background(), rpcblock.ByNumber(10), caller, game))
	require.Equal(t, []common.Address{creator, counterer, blockChallenger}, caller.requestedCredits)
	require.Equal(t, caller.requestedCredits, caller.requestedWithdrawals)
	require.Equal(t, map[common.Address]bool{creator: true, counterer: true, blockChallenger: true}, game.Recipients)
	require.Equal(t, faultTypes.NormalDistributionMode, game.BondDistributionMode)
	require.Equal(t, []monTypes.BondRecord{
		{Depositor: creator, Recipient: blockChallenger, Amount: big.NewInt(7), Resolved: true},
		{Depositor: creator, Recipient: counterer, Amount: big.NewInt(3), Resolved: true},
	}, game.Bonds)
	require.Equal(t, map[common.Address]*big.Int{
		blockChallenger: big.NewInt(7), counterer: big.NewInt(3),
	}, game.ExpectedCredits)
	require.Equal(t, caller.credits, game.Credits)
	require.Len(t, game.WithdrawalRequests, 3)
	for _, request := range game.WithdrawalRequests {
		require.Zero(t, request.Amount.Sign())
		require.Zero(t, request.Timestamp.Sign())
	}
	require.Equal(t, big.NewInt(20), game.ETHCollateral)
	require.Equal(t, time.Hour, game.WETHDelay)
	require.Equal(t, common.Address{0xaa}, game.WETHContract)
	require.Equal(t, time.Unix(1_000, 0).Add(50*time.Second).Add(time.Hour), game.CreditWithdrawableAt)
}

func TestBondDataEnricherErrors(t *testing.T) {
	errRPC := errors.New("boom")
	tests := []struct {
		name     string
		mutate   func(*stubBondGameCaller)
		wantErr  error
		contains string
	}{
		{name: "distribution mode", mutate: func(c *stubBondGameCaller) { c.modeErr = errRPC }, wantErr: errRPC},
		{name: "unsupported mode", mutate: func(c *stubBondGameCaller) { c.mode = faultTypes.BondDistributionMode(99) }, contains: "unsupported ZK bond distribution mode 99"},
		{name: "credits", mutate: func(c *stubBondGameCaller) { c.creditsErr = errRPC }, wantErr: errRPC},
		{name: "credit count", mutate: func(c *stubBondGameCaller) { c.wrongCreditCount = true }, wantErr: ErrIncorrectCreditCount},
		{name: "withdrawals", mutate: func(c *stubBondGameCaller) { c.withdrawalsErr = errRPC }, wantErr: errRPC},
		{name: "withdrawal count", mutate: func(c *stubBondGameCaller) { c.wrongWithdrawalCount = true }, wantErr: ErrIncorrectWithdrawalsCount},
		{name: "balance", mutate: func(c *stubBondGameCaller) { c.balanceErr = errRPC }, wantErr: errRPC},
		{name: "nil balance", mutate: func(c *stubBondGameCaller) { c.balance = nil }, wantErr: ErrInvalidETHCollateral},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			creator := common.Address{0x11}
			caller := &stubBondGameCaller{
				mode:        faultTypes.UndecidedDistributionMode,
				credits:     map[common.Address]*big.Int{creator: new(big.Int)},
				withdrawals: map[common.Address]*contracts.WithdrawalRequest{},
				balance:     big.NewInt(100),
			}
			test.mutate(caller)
			game := &monTypes.ZKGameData{
				CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
				BondGameData:   monTypes.BondGameData{ETHCollateral: big.NewInt(7)},
				GameCreator:    creator,
				TotalBonds:     big.NewInt(100),
				ChallengerBond: big.NewInt(30),
			}
			err := NewBondDataEnricher().Enrich(context.Background(), rpcblock.ByNumber(10), caller, game)
			require.Error(t, err)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
			}
			if test.contains != "" {
				require.ErrorContains(t, err, test.contains)
			}
			if test.name == "balance" {
				require.Equal(t, big.NewInt(7), game.ETHCollateral)
			}
		})
	}
}

func TestNormalizeBondsRejectsUnsupportedVariantStates(t *testing.T) {
	fault := &monTypes.FaultGameData{}
	require.ErrorContains(t, normalizeFaultBonds(fault, faultTypes.BondDistributionMode(99)), "unsupported fault bond distribution mode 99")

	zk := &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatus(99)},
		GameCreator:    common.Address{0x11},
		TotalBonds:     big.NewInt(100),
		ChallengerBond: big.NewInt(30),
	}
	require.ErrorContains(t, normalizeZKBonds(zk, faultTypes.NormalDistributionMode), "unsupported terminal ZK game status")
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
	modeErr              error
	creditsErr           error
	withdrawalsErr       error
	balanceErr           error
	wrongCreditCount     bool
	wrongWithdrawalCount bool
	rawRequests          bool
}

func (s *stubBondGameCaller) GetCredits(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	s.requestedCredits = append([]common.Address(nil), recipients...)
	if s.creditsErr != nil {
		return nil, s.creditsErr
	}
	result := make([]*big.Int, len(recipients))
	for i, recipient := range recipients {
		result[i] = s.credits[recipient]
	}
	if s.wrongCreditCount && len(result) > 0 {
		result = result[:len(result)-1]
	}
	return result, nil
}

func (s *stubBondGameCaller) GetBondDistributionMode(context.Context, rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return s.mode, s.modeErr
}

func (s *stubBondGameCaller) GetWithdrawals(_ context.Context, _ rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	s.requestedWithdrawals = append([]common.Address(nil), recipients...)
	if s.withdrawalsErr != nil {
		return nil, s.withdrawalsErr
	}
	result := make([]*contracts.WithdrawalRequest, len(recipients))
	for i, recipient := range recipients {
		request := s.withdrawals[recipient]
		if request == nil && !s.rawRequests {
			request = &contracts.WithdrawalRequest{Amount: new(big.Int), Timestamp: new(big.Int)}
		}
		result[i] = request
	}
	if s.wrongWithdrawalCount && len(result) > 0 {
		result = result[:len(result)-1]
	}
	return result, nil
}

func (s *stubBondGameCaller) GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	return s.balance, s.delay, s.weth, s.balanceErr
}
