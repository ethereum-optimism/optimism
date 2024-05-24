package claims

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
}

type BondContractCreator func(game types.GameMetadata) (BondContract, error)

type Claimer struct {
	logger          log.Logger
	metrics         BondClaimMetrics
	contractCreator BondContractCreator
	txSender        TxSender
	claimants       []common.Address
}

var _ BondClaimer = (*Claimer)(nil)

func NewBondClaimer(l log.Logger, m BondClaimMetrics, contractCreator BondContractCreator, txSender TxSender, claimants ...common.Address) *Claimer {
	return &Claimer{
		logger:          l,
		metrics:         m,
		contractCreator: contractCreator,
		txSender:        txSender,
		claimants:       claimants,
	}
}

func (c *Claimer) ClaimBonds(ctx context.Context, games []types.GameMetadata) (err error) {
	for _, game := range games {
		for _, claimant := range c.claimants {
			err = errors.Join(err, c.claimBond(ctx, game, claimant))
		}
	}
	return err
}

func (c *Claimer) claimBond(ctx context.Context, game types.GameMetadata, addr common.Address) error {
	c.logger.Debug("Attempting to claim bonds for", "game", game.Proxy, "addr", addr)

	contract, err := c.contractCreator(game)
	if err != nil {
		return fmt.Errorf("failed to create bond contract: %w", err)
	}

	credit, status, err := contract.GetCredit(ctx, addr)
	if err != nil {
		return fmt.Errorf("failed to get credit: %w", err)
	}

	if status == types.GameStatusInProgress {
		c.logger.Debug("Not claiming credit from in progress game", "game", game.Proxy, "addr", addr, "status", status)
		return nil
	}
	if credit.Cmp(big.NewInt(0)) == 0 {
		c.logger.Debug("No credit to claim", "game", game.Proxy, "addr", addr)
		return nil
	}

	candidate, err := contract.ClaimCreditTx(ctx, addr)
	if errors.Is(err, contracts.ErrSimulationFailed) {
		c.logger.Debug("Credit still locked", "game", game.Proxy, "addr", addr)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to create credit claim tx: %w", err)
	}

	if err = c.txSender.SendAndWaitSimple("claim credit", candidate); err != nil {
		return fmt.Errorf("failed to claim credit: %w", err)
	}

	c.metrics.RecordBondClaimed(credit.Uint64())
	return nil
}
