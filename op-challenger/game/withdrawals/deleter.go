package withdrawals

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// maxDeletesPerGame bounds the deletes sent for a single game in one run so a game with a large
// number of proofs against it can't starve the others. The remainder is picked up on later runs.
const maxDeletesPerGame = 100

type Portal interface {
	Addr() common.Address
	WithdrawalProvenExtension1Topic() common.Hash
	DecodeWithdrawalProvenExtension1(log *ethTypes.Log) (contracts.WithdrawalProof, error)
	GetProvenWithdrawals(ctx context.Context, block rpcblock.Block, proofs []contracts.WithdrawalProof) ([]contracts.ProvenWithdrawal, error)
	DeleteProvenWithdrawalTx(proof contracts.WithdrawalProof) (txmgr.TxCandidate, error)
}

type TxSender interface {
	SendAndWaitDetailed(txPurpose string, txs ...txmgr.TxCandidate) []error
}

type DeleterMetrics interface {
	RecordWithdrawalDeleted()
	RecordWithdrawalDeletionFailed()
}

type Deleter struct {
	log      log.Logger
	metrics  DeleterMetrics
	portal   Portal
	l1       L1Client
	txSender TxSender
}

func NewDeleter(logger log.Logger, m DeleterMetrics, portal Portal, l1 L1Client, txSender TxSender) *Deleter {
	return &Deleter{
		log:      logger,
		metrics:  m,
		portal:   portal,
		l1:       l1,
		txSender: txSender,
	}
}

// DeleteInvalidatedWithdrawals deletes the withdrawal proofs the portal still records against game,
// which the challenger winning game has invalidated. Proofs proven from scanFrom up to and including
// l1Head are considered. It reports whether every invalidated proof has now been deleted.
func (d *Deleter) DeleteInvalidatedWithdrawals(ctx context.Context, game common.Address, scanFrom uint64, l1Head eth.BlockID) (bool, error) {
	done, err := d.deleteForGame(ctx, game, scanFrom, l1Head)
	if err != nil {
		d.metrics.RecordWithdrawalDeletionFailed()
		return false, err
	}
	return done, nil
}

func (d *Deleter) deleteForGame(ctx context.Context, game common.Address, scanFrom uint64, l1Head eth.BlockID) (bool, error) {
	proofs, err := d.proofs(ctx, scanFrom, l1Head.Number)
	if err != nil {
		return false, err
	}
	invalidated, err := d.invalidated(ctx, game, rpcblock.ByHash(l1Head.Hash), proofs)
	if err != nil {
		return false, err
	}
	truncated := len(invalidated) > maxDeletesPerGame
	if truncated {
		d.log.Warn("Deferring some withdrawal proof deletions to a later run",
			"game", game, "invalidated", len(invalidated), "limit", maxDeletesPerGame)
		invalidated = invalidated[:maxDeletesPerGame]
	}
	if err := d.send(game, invalidated); err != nil {
		return false, err
	}
	return !truncated, nil
}

// invalidated selects the proofs the portal still records against game.
// Deleted proofs read back with a zero timestamp, which makes deletion idempotent.
func (d *Deleter) invalidated(ctx context.Context, game common.Address, block rpcblock.Block, proofs []contracts.WithdrawalProof) ([]contracts.WithdrawalProof, error) {
	if len(proofs) == 0 {
		return nil, nil
	}
	records, err := d.portal.GetProvenWithdrawals(ctx, block, proofs)
	if err != nil {
		return nil, err
	}
	var invalidated []contracts.WithdrawalProof
	for i, record := range records {
		if record.Timestamp != 0 && record.DisputeGameProxy == game {
			invalidated = append(invalidated, proofs[i])
		}
	}
	return invalidated, nil
}

func (d *Deleter) send(game common.Address, proofs []contracts.WithdrawalProof) error {
	if len(proofs) == 0 {
		return nil
	}
	candidates := make([]txmgr.TxCandidate, len(proofs))
	for i, proof := range proofs {
		candidate, err := d.portal.DeleteProvenWithdrawalTx(proof)
		if err != nil {
			return fmt.Errorf("failed to create delete tx for withdrawal %v: %w", proof.WithdrawalHash, err)
		}
		candidates[i] = candidate
	}
	d.log.Info("Deleting withdrawal proofs invalidated by challenger win", "game", game, "count", len(candidates))
	var errs error
	for _, err := range d.txSender.SendAndWaitDetailed("delete proven withdrawal", candidates...) {
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			d.metrics.RecordWithdrawalDeleted()
		}
	}
	return errs
}
