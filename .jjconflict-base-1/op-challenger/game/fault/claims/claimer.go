package claims

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type TxSender interface {
	SendAndWaitSimple(txPurpose string, txs ...txmgr.TxCandidate) error
}

type BondClaimMetrics interface {
	RecordBondClaimed(amount uint64)
}

type BondContract interface {
	GetCredit(ctx context.Context, recipient common.Address) (*big.Int, types.GameStatus, error)
	ClaimCreditTx(ctx context.Context, recipient common.Address) (txmgr.TxCandidate, error)
	GetBondDistributionMode(ctx context.Context, block rpcblock.Block) (faultTypes.BondDistributionMode, error)
	CloseGameTx(ctx context.Context) (txmgr.TxCandidate, error)
}

type BondContractCreator func(game types.GameMetadata) (BondContract, error)

type Claimer struct {
	logger          log.Logger
	metrics         BondClaimMetrics
	contractCreator BondContractCreator
	txSender        TxSender
	claimants       []common.Address
	selective       bool
}

var _ BondClaimer = (*Claimer)(nil)

func NewBondClaimer(l log.Logger, m BondClaimMetrics, contractCreator BondContractCreator, txSender TxSender, selective bool, claimants ...common.Address) *Claimer {
	return &Claimer{
		logger:          l,
		metrics:         m,
		contractCreator: contractCreator,
		txSender:        txSender,
		claimants:       claimants,
		selective:       selective,
	}
}

func (c *Claimer) ClaimBonds(ctx context.Context, games []types.GameMetadata) (err error) {
	for _, game := range games {
		contract, contractErr := c.contractCreator(game)
		if contractErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to create bond contract: %w", contractErr))
			continue
		}

		anyCreditFound, claimErr := c.claimBonds(ctx, contract, game)
		err = errors.Join(err, claimErr)

		if !anyCreditFound {
			err = errors.Join(err, c.closeGame(ctx, contract, game))
		}
	}
	return err
}

func (c *Claimer) claimBonds(ctx context.Context, contract BondContract, game types.GameMetadata) (bool, error) {
	anyCreditFound := false
	var claimErr error
	for _, claimant := range c.claimants {
		hasCredit, err := c.claimBond(ctx, contract, game, claimant)
		claimErr = errors.Join(claimErr, err)
		anyCreditFound = anyCreditFound || hasCredit
	}
	return anyCreditFound, claimErr
}

// claimBond attempts to claim credit for a single address.
// Returns true if the address had credit > 0 (regardless of whether the claim succeeded).
func (c *Claimer) claimBond(ctx context.Context, contract BondContract, game types.GameMetadata, addr common.Address) (bool, error) {
	c.logger.Debug("Attempting to claim bonds for", "game", game.Proxy, "addr", addr)

	credit, status, err := contract.GetCredit(ctx, addr)
	if err != nil {
		return false, fmt.Errorf("failed to get credit: %w", err)
	}

	if status == types.GameStatusInProgress {
		c.logger.Debug("Not claiming credit from in progress game", "game", game.Proxy, "addr", addr, "status", status)
		return true, nil // Game is in progress, don't try to close it
	}
	if credit.Cmp(big.NewInt(0)) == 0 {
		c.logger.Debug("No credit to claim", "game", game.Proxy, "addr", addr)
		return false, nil
	}

	candidate, err := contract.ClaimCreditTx(ctx, addr)
	if errors.Is(err, contracts.ErrSimulationFailed) {
		c.logger.Debug("Credit still locked", "game", game.Proxy, "addr", addr)
		return true, nil // Credit exists but is locked
	} else if err != nil {
		return true, fmt.Errorf("failed to create credit claim tx: %w", err)
	}

	if err = c.txSender.SendAndWaitSimple("claim credit", candidate); err != nil {
		return true, fmt.Errorf("failed to claim credit: %w", err)
	}

	c.metrics.RecordBondClaimed(credit.Uint64())
	return true, nil
}

func (c *Claimer) closeGame(ctx context.Context, contract BondContract, game types.GameMetadata) error {
	if c.selective {
		c.logger.Debug("Skipping game close in selective claim resolution mode", "game", game.Proxy)
		return nil
	}

	bondMode, err := contract.GetBondDistributionMode(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to get bond distribution mode: %w", err)
	}
	if bondMode != faultTypes.UndecidedDistributionMode {
		c.logger.Debug("Game already closed", "game", game.Proxy, "bondMode", bondMode)
		return nil
	}

	candidate, err := contract.CloseGameTx(ctx)
	if errors.Is(err, contracts.ErrSimulationFailed) {
		c.logger.Debug("Game not ready to close", "game", game.Proxy)
		return nil
	} else if errors.Is(err, contracts.ErrCloseGameNotSupported) {
		c.logger.Debug("Contract version does not support closeGame", "game", game.Proxy)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to create close game tx: %w", err)
	}

	c.logger.Info("Closing game to update anchor state", "game", game.Proxy)
	if err = c.txSender.SendAndWaitSimple("close game", candidate); err != nil {
		return fmt.Errorf("failed to close game: %w", err)
	}

	return nil
}
