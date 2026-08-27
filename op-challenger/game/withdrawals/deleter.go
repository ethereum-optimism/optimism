package withdrawals

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
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
}

type Deleter struct {
	log        log.Logger
	metrics    DeleterMetrics
	portal     Portal
	gameStates GameStateReader
	txSender   TxSender
	scanner    *proofScanner
}

var _ InvalidatedWithdrawalDeleter = (*Deleter)(nil)

func NewDeleter(logger log.Logger, m DeleterMetrics, portal Portal, games GameStateReader, l1 L1Client, txSender TxSender) *Deleter {
	return &Deleter{
		log:        logger,
		metrics:    m,
		portal:     portal,
		gameStates: games,
		txSender:   txSender,
		scanner:    newProofScanner(l1, portal),
	}
}

func (d *Deleter) DeleteInvalidatedWithdrawals(ctx context.Context, blockNumber uint64, games []types.GameMetadata) error {
	addrs := make([]common.Address, len(games))
	for i, game := range games {
		addrs[i] = game.Proxy
	}
	d.scanner.Retain(addrs)
	states, err := d.gameStates.GetGameStates(ctx, rpcblock.ByNumber(blockNumber), addrs)
	if err != nil {
		return fmt.Errorf("failed to load game states: %w", err)
	}
	var errs error
	for i, state := range states {
		if state.Status != types.GameStatusChallengerWon {
			continue
		}
		game := resolvedGame{addr: addrs[i], l1Head: state.L1Head}
		if err := d.deleteForGame(ctx, game, blockNumber); err != nil {
			errs = errors.Join(errs, fmt.Errorf("game %v: %w", game.addr, err))
		}
	}
	return errs
}

func (d *Deleter) deleteForGame(ctx context.Context, game resolvedGame, blockNumber uint64) error {
	proofs, err := d.scanner.Proofs(ctx, game, blockNumber)
	if err != nil {
		return err
	}
	invalidated, err := d.invalidated(ctx, game.addr, rpcblock.ByNumber(blockNumber), proofs)
	if err != nil {
		return err
	}
	truncated := len(invalidated) > maxDeletesPerGame
	if truncated {
		d.log.Warn("Deferring some withdrawal proof deletions to a later run",
			"game", game.addr, "invalidated", len(invalidated), "limit", maxDeletesPerGame)
		invalidated = invalidated[:maxDeletesPerGame]
	}
	if err := d.send(game.addr, invalidated); err != nil {
		return err
	}
	if !truncated {
		d.scanner.Advance(game.addr, blockNumber)
	}
	return nil
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
