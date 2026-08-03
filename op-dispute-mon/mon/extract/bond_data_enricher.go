package extract

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrIncorrectCreditCount      = errors.New("incorrect credit count")
	ErrIncorrectWithdrawalsCount = errors.New("incorrect withdrawals count")
	ErrInvalidETHCollateral      = errors.New("invalid DelayedWETH collateral")
	ErrInvalidZKBondTotals       = errors.New("invalid ZK bond totals")
)

type BondCaller interface {
	GetCredits(context.Context, rpcblock.Block, ...common.Address) ([]*big.Int, error)
	GetBondDistributionMode(context.Context, rpcblock.Block) (faultTypes.BondDistributionMode, error)
}

type BalanceCaller interface {
	GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error)
}

// BondEnricher adds normalized bond, credit, withdrawal, and collateral data to a bond-bearing game.
type BondEnricher interface {
	Enrich(context.Context, rpcblock.Block, BondGameCaller, monTypes.BondedGame) error
}

// BondDataEnricher loads the shared bond state for fault and ZK games.
type BondDataEnricher struct{}

func NewBondDataEnricher() *BondDataEnricher {
	return &BondDataEnricher{}
}

var _ BondEnricher = (*BondDataEnricher)(nil)

func (b *BondDataEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller BondGameCaller, game monTypes.BondedGame) error {
	mode, err := caller.GetBondDistributionMode(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to fetch bond distribution mode: %w", err)
	}
	game.BondData().BondDistributionMode = mode
	if err := normalizeBonds(game, mode); err != nil {
		return err
	}

	// At this point the RPC-populated maps are empty, so the deterministic union is derived
	// from the normalized recipients, bonds, and expected credits used to pair batch results.
	recipients := game.BondData().RecipientAddresses()
	credits, err := caller.GetCredits(ctx, block, recipients...)
	if err != nil {
		return fmt.Errorf("failed to fetch credits: %w", err)
	}
	if len(credits) != len(recipients) {
		return fmt.Errorf("%w, requested %d values but got %d", ErrIncorrectCreditCount, len(recipients), len(credits))
	}
	withdrawals, err := caller.GetWithdrawals(ctx, block, recipients...)
	if err != nil {
		return fmt.Errorf("failed to fetch withdrawals: %w", err)
	}
	if len(withdrawals) != len(recipients) {
		return fmt.Errorf("%w, requested %d values but got %d", ErrIncorrectWithdrawalsCount, len(recipients), len(withdrawals))
	}

	data := game.BondData()
	data.Credits = make(map[common.Address]*big.Int, len(recipients))
	data.WithdrawalRequests = make(map[common.Address]*contracts.WithdrawalRequest, len(recipients))
	for i, recipient := range recipients {
		data.Credits[recipient] = normalizedBigInt(credits[i])
		data.WithdrawalRequests[recipient] = normalizedWithdrawal(withdrawals[i])
	}
	ethCollateral, wethDelay, wethContract, err := caller.GetBalanceAndDelay(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to fetch DelayedWETH balance and delay: %w", err)
	}
	if ethCollateral == nil {
		return ErrInvalidETHCollateral
	}
	data.ETHCollateral = new(big.Int).Set(ethCollateral)
	data.WETHDelay = wethDelay
	data.WETHContract = wethContract
	switch typed := game.(type) {
	case *monTypes.FaultGameData:
		data.CreditWithdrawableAt = time.Unix(int64(typed.Timestamp), 0).
			Add(time.Duration(typed.MaxClockDuration) * time.Second).
			Add(data.WETHDelay)
	}

	return nil
}

func normalizeBonds(game monTypes.BondedGame, mode faultTypes.BondDistributionMode) error {
	switch typed := game.(type) {
	case *monTypes.FaultGameData:
		return normalizeFaultBonds(typed, mode)
	case *monTypes.ZKGameData:
		return normalizeZKBonds(typed, mode)
	default:
		return fmt.Errorf("unsupported bonded game type %T", game)
	}
}

func normalizeFaultBonds(game *monTypes.FaultGameData, mode faultTypes.BondDistributionMode) error {
	if mode != faultTypes.LegacyDistributionMode && mode != faultTypes.UndecidedDistributionMode &&
		mode != faultTypes.NormalDistributionMode && mode != faultTypes.RefundDistributionMode {
		return fmt.Errorf("unsupported fault bond distribution mode %d", mode)
	}
	data := game.BondData()
	data.Bonds = make([]monTypes.BondRecord, 0, len(game.Claims))
	data.Recipients = make(map[common.Address]bool)
	data.ExpectedCredits = make(map[common.Address]*big.Int)
	for _, claim := range game.Claims {
		data.Recipients[claim.Claimant] = true
		if claim.CounteredBy != (common.Address{}) {
			data.Recipients[claim.CounteredBy] = true
		}
		if claim.IsRoot() && game.BlockNumberChallenger != (common.Address{}) {
			data.Recipients[game.BlockNumberChallenger] = true
		}

		record := monTypes.BondRecord{Depositor: claim.Claimant, Amount: claim.Bond}
		switch {
		case mode == faultTypes.RefundDistributionMode:
			record.Resolved = true
			record.Recipient = claim.Claimant
		case !claim.Resolved:
		case claim.IsRoot() && game.BlockNumberChallenged:
			record.Resolved = true
			record.Recipient = game.BlockNumberChallenger
		case claim.CounteredBy != (common.Address{}):
			record.Resolved = true
			record.Recipient = claim.CounteredBy
		default:
			record.Resolved = true
			record.Recipient = claim.Claimant
		}
		data.Bonds = append(data.Bonds, record)
		addExpectedCredit(data.ExpectedCredits, record)
	}
	return nil
}

func normalizeZKBonds(game *monTypes.ZKGameData, mode faultTypes.BondDistributionMode) error {
	if mode != faultTypes.UndecidedDistributionMode && mode != faultTypes.NormalDistributionMode && mode != faultTypes.RefundDistributionMode {
		return fmt.Errorf("unsupported ZK bond distribution mode %d", mode)
	}
	if game.TotalBonds == nil || game.ChallengerBond == nil || game.TotalBonds.Sign() < 0 || game.ChallengerBond.Sign() < 0 {
		return ErrInvalidZKBondTotals
	}

	data := game.BondData()
	data.Recipients = map[common.Address]bool{
		game.GameCreator: true,
	}
	if game.Challenger != (common.Address{}) {
		data.Recipients[game.Challenger] = true
	}
	if game.Prover != (common.Address{}) {
		data.Recipients[game.Prover] = true
	}
	data.ExpectedCredits = make(map[common.Address]*big.Int)

	creatorBond := new(big.Int).Set(game.TotalBonds)
	if game.Challenger != (common.Address{}) {
		if game.TotalBonds.Cmp(game.ChallengerBond) < 0 {
			return fmt.Errorf("%w: total bonds %s below challenger bond %s", ErrInvalidZKBondTotals, game.TotalBonds, game.ChallengerBond)
		}
		creatorBond.Sub(creatorBond, game.ChallengerBond)
	}
	data.Bonds = []monTypes.BondRecord{{Depositor: game.GameCreator, Amount: creatorBond}}
	if game.Challenger != (common.Address{}) {
		data.Bonds = append(data.Bonds, monTypes.BondRecord{
			Depositor:      game.Challenger,
			Amount:         new(big.Int).Set(game.ChallengerBond),
			ChallengerBond: true,
		})
	}

	if game.Status == gameTypes.GameStatusInProgress {
		return nil
	}
	for i := range data.Bonds {
		record := &data.Bonds[i]
		record.Resolved = true
		switch {
		case mode == faultTypes.RefundDistributionMode:
			record.Recipient = record.Depositor
		case game.Status == gameTypes.GameStatusChallengerWon && game.Challenger == (common.Address{}):
			record.Burned = true
		case game.Status == gameTypes.GameStatusChallengerWon:
			record.Recipient = game.Challenger
		case game.Status == gameTypes.GameStatusDefenderWon && record.ChallengerBond &&
			game.Prover != (common.Address{}) && game.Prover != game.GameCreator:
			record.Recipient = game.Prover
		case game.Status == gameTypes.GameStatusDefenderWon:
			record.Recipient = game.GameCreator
		default:
			return fmt.Errorf("unsupported terminal ZK game status %s", game.Status)
		}
		data.Recipients[record.Recipient] = true
		addExpectedCredit(data.ExpectedCredits, *record)
	}
	return nil
}

func normalizedBigInt(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}

func normalizedWithdrawal(request *contracts.WithdrawalRequest) *contracts.WithdrawalRequest {
	if request == nil {
		return &contracts.WithdrawalRequest{Amount: new(big.Int), Timestamp: new(big.Int)}
	}
	return &contracts.WithdrawalRequest{
		Amount:    normalizedBigInt(request.Amount),
		Timestamp: normalizedBigInt(request.Timestamp),
	}
}

func addExpectedCredit(expected map[common.Address]*big.Int, record monTypes.BondRecord) {
	if !record.Resolved {
		return
	}
	amount := expected[record.Recipient]
	if amount == nil {
		amount = new(big.Int)
	}
	expected[record.Recipient] = new(big.Int).Add(amount, record.Amount)
}
