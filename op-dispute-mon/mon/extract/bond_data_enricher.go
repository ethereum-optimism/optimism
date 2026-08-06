package extract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
)

// BondDataEnricher loads the normalized bond state for a fault game.
type BondDataEnricher struct{}

var (
	ErrIncorrectCreditCount      = errors.New("incorrect credit count")
	ErrIncorrectWithdrawalsCount = errors.New("incorrect withdrawals count")
)

type BondCaller interface {
	GetCredits(context.Context, rpcblock.Block, ...common.Address) ([]*big.Int, error)
	GetBondDistributionMode(context.Context, rpcblock.Block) (faultTypes.BondDistributionMode, error)
}

type BalanceCaller interface {
	GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error)
}

func NewBondDataEnricher() *BondDataEnricher {
	return &BondDataEnricher{}
}

var _ FaultEnricher = (*BondDataEnricher)(nil)

func (b *BondDataEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller FaultGameCaller, game *monTypes.FaultGameData) error {
	data, err := b.load(ctx, block, caller, game)
	if err != nil {
		return err
	}
	game.BondGameData = data
	return nil
}

func (*BondDataEnricher) load(ctx context.Context, block rpcblock.Block, caller BondGameCaller, game *monTypes.FaultGameData) (monTypes.BondGameData, error) {
	data := normalizeFaultBondData(game)
	recipients := slices.Collect(maps.Keys(data.Recipients))
	slices.SortFunc(recipients, func(a, b common.Address) int {
		return bytes.Compare(a[:], b[:])
	})

	withdrawals, err := caller.GetWithdrawals(ctx, block, recipients...)
	if err != nil {
		return monTypes.BondGameData{}, fmt.Errorf("failed to fetch withdrawals: %w", err)
	}
	if len(withdrawals) != len(recipients) {
		return monTypes.BondGameData{}, fmt.Errorf("%w, requested %v values but got %v", ErrIncorrectWithdrawalsCount, len(recipients), len(withdrawals))
	}

	credits, err := caller.GetCredits(ctx, block, recipients...)
	if err != nil {
		return monTypes.BondGameData{}, err
	}
	if len(credits) != len(recipients) {
		return monTypes.BondGameData{}, fmt.Errorf("%w, requested %v values but got %v", ErrIncorrectCreditCount, len(recipients), len(credits))
	}

	mode, err := caller.GetBondDistributionMode(ctx, block)
	if err != nil {
		return monTypes.BondGameData{}, err
	}
	balance, delay, wethContract, err := caller.GetBalanceAndDelay(ctx, block)
	if err != nil {
		return monTypes.BondGameData{}, fmt.Errorf("failed to fetch balance: %w", err)
	}

	data.Credits = make(map[common.Address]*big.Int, len(recipients))
	data.WithdrawalRequests = make(map[common.Address]*contracts.WithdrawalRequest, len(recipients))
	for i, recipient := range recipients {
		data.Credits[recipient] = cloneBigInt(credits[i])
		data.WithdrawalRequests[recipient] = cloneWithdrawal(withdrawals[i])
	}
	data.BondDistributionMode = mode
	data.ETHCollateral = cloneBigInt(balance)
	data.WETHDelay = delay
	data.WETHContract = wethContract
	data.CreditWithdrawableAt = time.Unix(int64(game.Timestamp), 0).
		Add(time.Duration(game.MaxClockDuration) * time.Second).
		Add(delay)
	return data, nil
}

func normalizeFaultBondData(game *monTypes.FaultGameData) monTypes.BondGameData {
	data := monTypes.BondGameData{
		Bonds:           make([]monTypes.BondRecord, 0, len(game.Claims)),
		Recipients:      make(map[common.Address]bool),
		ExpectedCredits: make(map[common.Address]*big.Int),
	}
	for _, claim := range game.Claims {
		rpcRecipient := claim.Claimant
		if claim.CounteredBy != (common.Address{}) {
			rpcRecipient = claim.CounteredBy
		}
		data.Recipients[rpcRecipient] = true

		record := monTypes.BondRecord{
			Depositor: claim.Claimant,
			Amount:    cloneBigInt(claim.Bond),
			Resolved:  claim.Resolved,
			Forfeited: claim.Resolved && claim.CounteredBy != (common.Address{}),
		}
		if claim.Resolved {
			record.Recipient = rpcRecipient
			expectedRecipient := rpcRecipient
			if claim.IsRoot() && game.BlockNumberChallenged {
				expectedRecipient = game.BlockNumberChallenger
			}
			addExpectedCredit(data.ExpectedCredits, expectedRecipient, claim.Bond)
		}
		data.Bonds = append(data.Bonds, record)
	}
	if game.BlockNumberChallenger != (common.Address{}) {
		data.Recipients[game.BlockNumberChallenger] = true
	}
	return data
}

func addExpectedCredit(credits map[common.Address]*big.Int, recipient common.Address, amount *big.Int) {
	current := credits[recipient]
	if current == nil {
		current = new(big.Int)
	}
	if amount == nil {
		credits[recipient] = new(big.Int).Set(current)
		return
	}
	credits[recipient] = new(big.Int).Add(current, amount)
}

func cloneBigInt(value *big.Int) *big.Int {
	if value == nil {
		return nil
	}
	return new(big.Int).Set(value)
}

func cloneWithdrawal(request *contracts.WithdrawalRequest) *contracts.WithdrawalRequest {
	if request == nil {
		return nil
	}
	return &contracts.WithdrawalRequest{
		Amount:    cloneBigInt(request.Amount),
		Timestamp: cloneBigInt(request.Timestamp),
	}
}
