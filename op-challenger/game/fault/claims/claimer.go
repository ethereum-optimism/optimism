package claims

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var _ BondClaimer = (*claimer)(nil)

type BondClaimer interface {
	ClaimBonds(ctx context.Context, games []types.GameMetadata) error
}

type BondClaimMetrics interface {
	RecordBondClaimed(amount uint64)
}

type BondContract interface {
	GetCredit(ctx context.Context, receipient common.Address) (*big.Int, error)
	ClaimCredit(receipient common.Address) (txmgr.TxCandidate, error)
}

type claimer struct {
	logger  log.Logger
	metrics BondClaimMetrics

	caller   *batching.MultiCaller
	txSender types.TxSender
}

func NewBondClaimer(l log.Logger, m BondClaimMetrics, c *batching.MultiCaller, txSender types.TxSender) *claimer {
	return &claimer{
		logger:   l,
		metrics:  m,
		caller:   c,
		txSender: txSender,
	}
}

func (c *claimer) ClaimBonds(ctx context.Context, games []types.GameMetadata) (err error) {
	for _, game := range games {
		err = errors.Join(err, c.claimBond(ctx, game.Proxy))
	}
	return err
}

func (c *claimer) claimBond(ctx context.Context, gameAddr common.Address) error {
	c.logger.Debug("attempting to claim bonds for", "game", gameAddr)

	contract, err := contracts.NewFaultDisputeGameContract(gameAddr, c.caller)
	if err != nil {
		return fmt.Errorf("failed to create contract: %w", err)
	}

	credit, err := contract.GetCredit(ctx, c.txSender.From())
	if err != nil {
		return fmt.Errorf("failed to get credit: %w", err)
	}

	if credit.Cmp(big.NewInt(0)) == 0 {
		c.logger.Debug("no credit to claim", "game", gameAddr)
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
