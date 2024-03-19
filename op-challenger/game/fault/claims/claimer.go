package claims

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type BondClaimMetrics interface {
	RecordBondClaimed(amount uint64)
}

type BondContract interface {
	GetCredit(ctx context.Context, receipient common.Address) (*big.Int, error)
	ClaimCredit(receipient common.Address) (txmgr.TxCandidate, error)
}

type BondContractCreator func(game types.GameMetadata) (BondContract, error)

type Claimer struct {
	logger          log.Logger
	metrics         BondClaimMetrics
	contractCreator BondContractCreator
	txSender        types.TxSender
}

var _ BondClaimer = (*Claimer)(nil)

func NewBondClaimer(l log.Logger, m BondClaimMetrics, contractCreator BondContractCreator, txSender types.TxSender) *Claimer {
	return &Claimer{
		logger:          l,
		metrics:         m,
		contractCreator: contractCreator,
		txSender:        txSender,
	}
}

func (c *Claimer) ClaimBonds(ctx context.Context, games []types.GameMetadata) (err error) {
	for _, game := range games {
		err = errors.Join(err, c.claimBond(ctx, game))
	}
	return err
}

func (c *Claimer) claimBond(ctx context.Context, game types.GameMetadata) error {
	c.logger.Debug("Attempting to claim bonds for", "game", game.Proxy)

	contract, err := c.contractCreator(game)
	if err != nil {
		return fmt.Errorf("failed to create bond contract bindings: %w", err)
	}
	credit, err := contract.GetCredit(ctx, c.txSender.From())
	if err != nil {
		return fmt.Errorf("failed to get credit: %w", err)
	}

	if credit.Cmp(big.NewInt(0)) == 0 {
		c.logger.Debug("No credit to claim", "game", game.Proxy)
		return nil
	}

	candidate, err := contract.ClaimCredit(c.txSender.From())
	if err != nil {
		return fmt.Errorf("failed to create credit claim tx: %w", err)
	}

	if _, err = c.txSender.SendAndWait("claim credit", candidate); err != nil {
		return fmt.Errorf("failed to claim credit: %w", err)
	}

	c.metrics.RecordBondClaimed(credit.Uint64())
	return nil
}
