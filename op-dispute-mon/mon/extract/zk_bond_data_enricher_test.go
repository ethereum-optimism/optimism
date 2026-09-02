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

func TestNormalizeZKBonds(t *testing.T) {
	creator := common.Address{0x10}
	challenger := common.Address{0x20}
	prover := common.Address{0x30}
	challengerWon := gameTypes.GameStatusChallengerWon
	creatorLive := monTypes.BondRecord{Depositor: creator, Amount: big.NewInt(100)}
	creatorRemainder := monTypes.BondRecord{Depositor: creator, Amount: big.NewInt(70)}
	challengerLive := monTypes.BondRecord{Depositor: challenger, Amount: big.NewInt(30), ChallengerBond: true}

	tests := []struct {
		name       string
		mode       faultTypes.BondDistributionMode
		configure  func(*monTypes.ZKGameData)
		bonds      []monTypes.BondRecord
		credits    map[common.Address]*big.Int
		recipients map[common.Address]bool
	}{
		{
			name:       "live unchallenged",
			bonds:      []monTypes.BondRecord{creatorLive},
			credits:    map[common.Address]*big.Int{},
			recipients: map[common.Address]bool{creator: true},
		},
		{
			name: "live challenged",
			configure: func(game *monTypes.ZKGameData) {
				game.ProposalStatus = contracts.ProposalStatusChallenged
				game.Challenger = challenger
			},
			bonds:      []monTypes.BondRecord{creatorRemainder, challengerLive},
			credits:    map[common.Address]*big.Int{},
			recipients: map[common.Address]bool{creator: true, challenger: true},
		},
		{
			name: "challenger wins",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusChallengerWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Challenger = challenger
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: challenger, Amount: big.NewInt(70), Resolved: true, Forfeited: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{challenger: big.NewInt(100)},
			recipients: map[common.Address]bool{creator: true, challenger: true},
		},
		{
			name: "creator self challenges and wins",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusChallengerWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Challenger = creator
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(100)},
			recipients: map[common.Address]bool{creator: true},
		},
		{
			name: "invalid parent overrides valid proof",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusChallengerWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.ParentStatus = &challengerWon
				game.Challenger = challenger
				game.Prover = prover
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: challenger, Amount: big.NewInt(70), Resolved: true, Forfeited: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{challenger: big.NewInt(100)},
			recipients: map[common.Address]bool{creator: true, challenger: true, prover: true},
		},
		{
			name: "defender wins with distinct prover",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Challenger = challenger
				game.Prover = prover
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: prover, Amount: big.NewInt(30), Resolved: true, Forfeited: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(70), prover: big.NewInt(30)},
			recipients: map[common.Address]bool{creator: true, challenger: true, prover: true},
		},
		{
			name: "creator self challenges and third party proves",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Challenger = creator
				game.Prover = prover
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: creator, Recipient: prover, Amount: big.NewInt(30), Resolved: true, Forfeited: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(70), prover: big.NewInt(30)},
			recipients: map[common.Address]bool{creator: true, prover: true},
		},
		{
			name: "creator proves",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Challenger = challenger
				game.Prover = creator
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: creator, Amount: big.NewInt(30), Resolved: true, Forfeited: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(100)},
			recipients: map[common.Address]bool{creator: true, challenger: true},
		},
		{
			name: "challenger proves",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Challenger = challenger
				game.Prover = challenger
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(70), challenger: big.NewInt(30)},
			recipients: map[common.Address]bool{creator: true, challenger: true},
		},
		{
			name: "unchallenged valid proof rewards creator",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Prover = prover
			},
			bonds:      []monTypes.BondRecord{{Depositor: creator, Recipient: creator, Amount: big.NewInt(100), Resolved: true}},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(100)},
			recipients: map[common.Address]bool{creator: true, prover: true},
		},
		{
			name: "unchallenged deadline rewards creator",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
			},
			bonds:      []monTypes.BondRecord{{Depositor: creator, Recipient: creator, Amount: big.NewInt(100), Resolved: true}},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(100)},
			recipients: map[common.Address]bool{creator: true},
		},
		{
			name: "refund mode returns deposits",
			mode: faultTypes.RefundDistributionMode,
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusDefenderWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.Challenger = challenger
				game.Prover = prover
			},
			bonds: []monTypes.BondRecord{
				{Depositor: creator, Recipient: creator, Amount: big.NewInt(70), Resolved: true},
				{Depositor: challenger, Recipient: challenger, Amount: big.NewInt(30), Resolved: true, ChallengerBond: true},
			},
			credits:    map[common.Address]*big.Int{creator: big.NewInt(70), challenger: big.NewInt(30)},
			recipients: map[common.Address]bool{creator: true, challenger: true, prover: true},
		},
		{
			name: "invalid parent burns unchallenged creator bond",
			configure: func(game *monTypes.ZKGameData) {
				game.Status = gameTypes.GameStatusChallengerWon
				game.ProposalStatus = contracts.ProposalStatusResolved
				game.ParentStatus = &challengerWon
				game.Prover = prover
			},
			bonds: []monTypes.BondRecord{{
				Depositor: creator, Recipient: common.Address{}, Amount: big.NewInt(100), Resolved: true, Forfeited: true,
			}},
			credits:    map[common.Address]*big.Int{{}: big.NewInt(100)},
			recipients: map[common.Address]bool{creator: true, prover: true, {}: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			game := validZKBondGame(creator)
			if test.configure != nil {
				test.configure(game)
			}
			mode := test.mode
			if mode == 0 {
				mode = faultTypes.UndecidedDistributionMode
			}
			data, err := normalizeZKBondData(game, mode)
			require.NoError(t, err)
			require.Equal(t, test.bonds, data.Bonds)
			require.Equal(t, test.credits, data.ExpectedCredits)
			require.Equal(t, test.recipients, data.Recipients)
		})
	}

	t.Run("unchallenged total may be below configured challenger bond", func(t *testing.T) {
		game := validZKBondGame(creator)
		game.TotalBonds = big.NewInt(10)
		data, err := normalizeZKBondData(game, faultTypes.UndecidedDistributionMode)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(10), data.Bonds[0].Amount)
	})
}

func TestNormalizeZKBondsRejectsInvalidInputs(t *testing.T) {
	creator := common.Address{0x10}
	challenger := common.Address{0x20}
	challengerWon := gameTypes.GameStatusChallengerWon
	tests := []struct {
		name      string
		mode      faultTypes.BondDistributionMode
		configure func(*monTypes.ZKGameData)
	}{
		{name: "zero creator", configure: func(game *monTypes.ZKGameData) { game.GameCreator = common.Address{} }},
		{name: "nil total", configure: func(game *monTypes.ZKGameData) { game.TotalBonds = nil }},
		{name: "nil challenger bond", configure: func(game *monTypes.ZKGameData) { game.ChallengerBond = nil }},
		{name: "negative total", configure: func(game *monTypes.ZKGameData) { game.TotalBonds = big.NewInt(-1) }},
		{name: "negative challenger bond", configure: func(game *monTypes.ZKGameData) { game.ChallengerBond = big.NewInt(-1) }},
		{name: "unsupported mode", mode: faultTypes.LegacyDistributionMode},
		{name: "unsupported status", configure: func(game *monTypes.ZKGameData) { game.Status = gameTypes.GameStatus(99) }},
		{name: "live decided mode", mode: faultTypes.NormalDistributionMode},
		{name: "challenged underflow", configure: func(game *monTypes.ZKGameData) {
			game.ProposalStatus = contracts.ProposalStatusChallenged
			game.Challenger = challenger
			game.TotalBonds = big.NewInt(20)
		}},
		{name: "invalid live participants", configure: func(game *monTypes.ZKGameData) { game.Challenger = challenger }},
		{name: "challenged defender win without prover", configure: func(game *monTypes.ZKGameData) {
			game.Status = gameTypes.GameStatusDefenderWon
			game.ProposalStatus = contracts.ProposalStatusResolved
			game.Challenger = challenger
		}},
		{name: "defender win with invalid parent", configure: func(game *monTypes.ZKGameData) {
			game.Status = gameTypes.GameStatusDefenderWon
			game.ProposalStatus = contracts.ProposalStatusResolved
			game.ParentStatus = &challengerWon
		}},
		{name: "challenger win after proof without invalid parent", configure: func(game *monTypes.ZKGameData) {
			game.Status = gameTypes.GameStatusChallengerWon
			game.ProposalStatus = contracts.ProposalStatusResolved
			game.Challenger = challenger
			game.Prover = common.Address{0x30}
		}},
		{name: "unchallenged loss without invalid parent", configure: func(game *monTypes.ZKGameData) {
			game.Status = gameTypes.GameStatusChallengerWon
			game.ProposalStatus = contracts.ProposalStatusResolved
		}},
		{name: "global proposal mismatch", configure: func(game *monTypes.ZKGameData) { game.ProposalStatus = contracts.ProposalStatusResolved }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			game := validZKBondGame(creator)
			if test.configure != nil {
				test.configure(game)
			}
			_, err := normalizeZKBondData(game, test.mode)
			require.ErrorIs(t, err, ErrInvalidZKBondData)
		})
	}
}

func TestNormalizeZKBondsCopiesAmounts(t *testing.T) {
	creator := common.Address{0x10}
	challenger := common.Address{0x20}
	game := validZKBondGame(creator)
	game.ProposalStatus = contracts.ProposalStatusChallenged
	game.Challenger = challenger
	data, err := normalizeZKBondData(game, faultTypes.UndecidedDistributionMode)
	require.NoError(t, err)

	game.TotalBonds.SetInt64(200)
	game.ChallengerBond.SetInt64(40)
	require.Equal(t, big.NewInt(70), data.Bonds[0].Amount)
	require.Equal(t, big.NewInt(30), data.Bonds[1].Amount)
	data.Bonds[0].Amount.SetInt64(1)
	require.Equal(t, big.NewInt(200), game.TotalBonds)
}

func TestZKBondSnapshot(t *testing.T) {
	creator := common.Address{0x10}
	challenger := common.Address{0x20}
	block := rpcblock.ByHash(common.Hash{0xaa})
	withdrawals := []*contracts.WithdrawalRequest{
		{Amount: new(big.Int), Timestamp: new(big.Int)},
		{Amount: big.NewInt(100), Timestamp: big.NewInt(200)},
	}
	credits := []*big.Int{new(big.Int), new(big.Int)}
	caller := &bondSnapshotCaller{
		mode:        faultTypes.NormalDistributionMode,
		withdrawals: withdrawals,
		credits:     credits,
		balance:     big.NewInt(150),
		delay:       time.Hour,
		weth:        common.Address{0xee},
	}
	game := validZKBondGame(creator)
	game.Status = gameTypes.GameStatusChallengerWon
	game.ProposalStatus = contracts.ProposalStatusResolved
	game.Challenger = challenger

	require.NoError(t, NewBondDataEnricher().EnrichZK(t.Context(), block, zkBondCaller(caller), game))
	require.Equal(t, []string{"mode", "withdrawals", "credits", "balance"}, caller.trace)
	require.Equal(t, []rpcblock.Block{block, block, block, block}, caller.blocks)
	require.Equal(t, []common.Address{creator, challenger}, caller.recipients)
	require.Equal(t, faultTypes.NormalDistributionMode, game.BondDistributionMode)
	require.Equal(t, common.Address{0xee}, game.WETHContract)
	require.Equal(t, time.Hour, game.WETHDelay)
	require.Equal(t, big.NewInt(150), game.ETHCollateral)
	require.Zero(t, game.CreditWithdrawableAt)
	require.Equal(t, big.NewInt(100), game.ExpectedCredits[challenger])
	require.Zero(t, game.Credits[challenger].Sign())
	require.Equal(t, big.NewInt(100), game.WithdrawalRequests[challenger].Amount)

	credits[1].SetInt64(1)
	withdrawals[1].Amount.SetInt64(2)
	caller.balance.SetInt64(3)
	require.Zero(t, game.Credits[challenger].Sign())
	require.Equal(t, big.NewInt(100), game.WithdrawalRequests[challenger].Amount)
	require.Equal(t, big.NewInt(150), game.ETHCollateral)
}

func TestZKBondSnapshotRejectsInvalidRPCDataAtomically(t *testing.T) {
	testErr := errors.New("boom")
	tests := []struct {
		name      string
		configure func(*bondSnapshotCaller)
		wantErr   error
	}{
		{name: "mode error", configure: func(c *bondSnapshotCaller) { c.modeErr = testErr }, wantErr: testErr},
		{name: "withdrawal error", configure: func(c *bondSnapshotCaller) { c.withdrawalsErr = testErr }, wantErr: testErr},
		{name: "withdrawal count", configure: func(c *bondSnapshotCaller) { c.withdrawals = nil }, wantErr: ErrIncorrectWithdrawalsCount},
		{name: "nil withdrawal", configure: func(c *bondSnapshotCaller) { c.withdrawals[0] = nil }, wantErr: ErrInvalidZKBondData},
		{name: "nil withdrawal amount", configure: func(c *bondSnapshotCaller) { c.withdrawals[0].Amount = nil }, wantErr: ErrInvalidZKBondData},
		{name: "nil withdrawal timestamp", configure: func(c *bondSnapshotCaller) { c.withdrawals[0].Timestamp = nil }, wantErr: ErrInvalidZKBondData},
		{name: "negative withdrawal amount", configure: func(c *bondSnapshotCaller) { c.withdrawals[0].Amount = big.NewInt(-1) }, wantErr: ErrInvalidZKBondData},
		{name: "negative withdrawal timestamp", configure: func(c *bondSnapshotCaller) { c.withdrawals[0].Timestamp = big.NewInt(-1) }, wantErr: ErrInvalidZKBondData},
		{name: "overflowing withdrawal timestamp", configure: func(c *bondSnapshotCaller) {
			c.withdrawals[0].Timestamp = new(big.Int).Lsh(big.NewInt(1), 64)
		}, wantErr: ErrInvalidZKBondData},
		{name: "credit error", configure: func(c *bondSnapshotCaller) { c.creditsErr = testErr }, wantErr: testErr},
		{name: "credit count", configure: func(c *bondSnapshotCaller) { c.credits = nil }, wantErr: ErrIncorrectCreditCount},
		{name: "nil credit", configure: func(c *bondSnapshotCaller) { c.credits[0] = nil }, wantErr: ErrInvalidZKBondData},
		{name: "negative credit", configure: func(c *bondSnapshotCaller) { c.credits[0] = big.NewInt(-1) }, wantErr: ErrInvalidZKBondData},
		{name: "balance error", configure: func(c *bondSnapshotCaller) { c.balanceErr = testErr }, wantErr: testErr},
		{name: "nil balance", configure: func(c *bondSnapshotCaller) { c.balance = nil }, wantErr: ErrInvalidZKBondData},
		{name: "negative balance", configure: func(c *bondSnapshotCaller) { c.balance = big.NewInt(-1) }, wantErr: ErrInvalidZKBondData},
		{name: "negative delay", configure: func(c *bondSnapshotCaller) { c.delay = -time.Second }, wantErr: ErrInvalidZKBondData},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := &bondSnapshotCaller{
				withdrawals: []*contracts.WithdrawalRequest{{Amount: new(big.Int), Timestamp: new(big.Int)}},
				credits:     []*big.Int{new(big.Int)},
				balance:     big.NewInt(100),
			}
			test.configure(caller)
			game := validZKBondGame(common.Address{0x10})
			game.BondGameData = monTypes.BondGameData{ETHCollateral: big.NewInt(7)}

			err := NewBondDataEnricher().EnrichZK(t.Context(), rpcblock.Latest, zkBondCaller(caller), game)
			require.ErrorIs(t, err, test.wantErr)
			require.Equal(t, monTypes.BondGameData{ETHCollateral: big.NewInt(7)}, game.BondGameData)
		})
	}
}

type zkBondSnapshotCaller struct {
	*mockGameCaller
	bond *bondSnapshotCaller
}

func zkBondCaller(bond *bondSnapshotCaller) *zkBondSnapshotCaller {
	return &zkBondSnapshotCaller{mockGameCaller: &mockGameCaller{}, bond: bond}
}

func (c *zkBondSnapshotCaller) GetMetadata(context.Context, rpcblock.Block) (contracts.GenericGameMetadata, error) {
	return contracts.GenericGameMetadata{}, nil
}

func (c *zkBondSnapshotCaller) GetChallengerMetadata(context.Context, rpcblock.Block) (contracts.ChallengerMetadata, error) {
	return contracts.ChallengerMetadata{}, nil
}

func (c *zkBondSnapshotCaller) GetBondMetadata(context.Context, rpcblock.Block) (contracts.ZKBondMetadata, error) {
	return contracts.ZKBondMetadata{}, nil
}

func (c *zkBondSnapshotCaller) GetWithdrawals(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	return c.bond.GetWithdrawals(ctx, block, recipients...)
}

func (c *zkBondSnapshotCaller) GetCredits(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*big.Int, error) {
	return c.bond.GetCredits(ctx, block, recipients...)
}

func (c *zkBondSnapshotCaller) GetBondDistributionMode(ctx context.Context, block rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return c.bond.GetBondDistributionMode(ctx, block)
}

func (c *zkBondSnapshotCaller) GetBalanceAndDelay(ctx context.Context, block rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	return c.bond.GetBalanceAndDelay(ctx, block)
}

func validZKBondGame(creator common.Address) *monTypes.ZKGameData {
	return &monTypes.ZKGameData{
		CommonGameData: monTypes.CommonGameData{Status: gameTypes.GameStatusInProgress},
		ProposalStatus: contracts.ProposalStatusUnchallenged,
		GameCreator:    creator,
		TotalBonds:     big.NewInt(100),
		ChallengerBond: big.NewInt(30),
	}
}
