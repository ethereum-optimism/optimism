package responder

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type GameContract interface {
	CallResolve(ctx context.Context) (gameTypes.GameStatus, error)
	ResolveTx() (txmgr.TxCandidate, error)
	CallResolveClaim(ctx context.Context, claimIdx uint64) error
	ResolveClaimTx(claimIdx uint64) (txmgr.TxCandidate, error)
	AttackTx(parentContractIndex uint64, pivot common.Hash) (txmgr.TxCandidate, error)
	DefendTx(parentContractIndex uint64, pivot common.Hash) (txmgr.TxCandidate, error)
	StepTx(claimIdx uint64, isAttack bool, stateData []byte, proof []byte) (txmgr.TxCandidate, error)
	UpdateOracleTx(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error)
}

// FaultResponder implements the [Responder] interface to send onchain transactions.
type FaultResponder struct {
	log log.Logger

	txMgr    txmgr.TxManager
	contract GameContract
}

// NewFaultResponder returns a new [FaultResponder].
func NewFaultResponder(logger log.Logger, txMgr txmgr.TxManager, contract GameContract) (*FaultResponder, error) {
	return &FaultResponder{
		log:      logger,
		txMgr:    txMgr,
		contract: contract,
	}, nil
}

// CallResolve determines if the resolve function on the fault dispute game contract
// would succeed. Returns the game status if the call would succeed, errors otherwise.
func (r *FaultResponder) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	return r.contract.CallResolve(ctx)
}

// Resolve executes a resolve transaction to resolve a fault dispute game.
func (r *FaultResponder) Resolve(ctx context.Context) error {
	candidate, err := r.contract.ResolveTx()
	if err != nil {
		return err
	}

	return r.sendTxAndWait(ctx, candidate)
}

// CallResolveClaim determines if the resolveClaim function on the fault dispute game contract
// would succeed.
func (r *FaultResponder) CallResolveClaim(ctx context.Context, claimIdx uint64) error {
	return r.contract.CallResolveClaim(ctx, claimIdx)
}

// ResolveClaim executes a resolveClaim transaction to resolve a fault dispute game.
func (r *FaultResponder) ResolveClaim(ctx context.Context, claimIdx uint64) error {
	candidate, err := r.contract.ResolveClaimTx(claimIdx)
	if err != nil {
		return err
	}
	return r.sendTxAndWait(ctx, candidate)
}

func (r *FaultResponder) PerformAction(ctx context.Context, action types.Action) error {
	if action.OracleData != nil {
		r.log.Info("Updating oracle data", "key", action.OracleData.OracleKey)
		candidate, err := r.contract.UpdateOracleTx(ctx, uint64(action.ParentIdx), action.OracleData)
		if err != nil {
			return fmt.Errorf("failed to create pre-image oracle tx: %w", err)
		}
		if err := r.sendTxAndWait(ctx, candidate); err != nil {
			return fmt.Errorf("failed to populate pre-image oracle: %w", err)
		}
	}
	var candidate txmgr.TxCandidate
	var err error
	switch action.Type {
	case types.ActionTypeMove:
		if action.IsAttack {
			candidate, err = r.contract.AttackTx(uint64(action.ParentIdx), action.Value)
		} else {
			candidate, err = r.contract.DefendTx(uint64(action.ParentIdx), action.Value)
		}
	case types.ActionTypeStep:
		candidate, err = r.contract.StepTx(uint64(action.ParentIdx), action.IsAttack, action.PreState, action.ProofData)
	}
	if err != nil {
		return err
	}
	return r.sendTxAndWait(ctx, candidate)
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (r *FaultResponder) sendTxAndWait(ctx context.Context, candidate txmgr.TxCandidate) error {
	receipt, err := r.txMgr.Send(ctx, candidate)
	if err != nil {
		return err
	}
	if receipt.Status == ethtypes.ReceiptStatusFailed {
		r.log.Error("Responder tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		r.log.Debug("Responder tx successfully published", "tx_hash", receipt.TxHash)
	}
	return nil
}
