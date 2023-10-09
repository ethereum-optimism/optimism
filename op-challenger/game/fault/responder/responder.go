package responder

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Abi interface {
	ResolveCallData() ([]byte, error)
	ParseResolveResult(res []byte) (gameTypes.GameStatus, error)
	ResolveClaimData(idx uint64) ([]byte, error)
	FaultAttackData(idx int, value common.Hash) ([]byte, error)
	FaultDefendData(idx int, value common.Hash) ([]byte, error)
	StepTxData(idx uint64, attack bool, state []byte, proof []byte) ([]byte, error)
}

// FaultResponder implements the [Responder] interface to send onchain transactions.
type FaultResponder struct {
	log log.Logger

	txMgr txmgr.TxManager

	fdgAddr  common.Address
	contract Abi // TODO: Should use interface for testability/substitutability
}

// NewFaultResponder returns a new [FaultResponder].
func NewFaultResponder(logger log.Logger, txManagr txmgr.TxManager, fdgAddr common.Address, contract Abi) (*FaultResponder, error) {
	return &FaultResponder{
		log:      logger,
		txMgr:    txManagr,
		fdgAddr:  fdgAddr,
		contract: contract,
	}, nil
}

// CallResolve determines if the resolve function on the fault dispute game contract
// would succeed. Returns the game status if the call would succeed, errors otherwise.
func (r *FaultResponder) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	txData, err := r.contract.ResolveCallData()
	if err != nil {
		return gameTypes.GameStatusInProgress, err
	}
	res, err := r.txMgr.Call(ctx, ethereum.CallMsg{
		To:   &r.fdgAddr,
		Data: txData,
	}, nil)
	if err != nil {
		return gameTypes.GameStatusInProgress, err
	}
	return r.contract.ParseResolveResult(res)
}

// Resolve executes a resolve transaction to resolve a fault dispute game.
func (r *FaultResponder) Resolve(ctx context.Context) error {
	txData, err := r.contract.ResolveCallData()
	if err != nil {
		return err
	}

	return r.sendTxAndWait(ctx, txData)
}

// CallResolveClaim determines if the resolveClaim function on the fault dispute game contract
// would succeed.
func (r *FaultResponder) CallResolveClaim(ctx context.Context, claimIdx uint64) error {
	txData, err := r.contract.ResolveClaimData(claimIdx)
	if err != nil {
		return err
	}
	_, err = r.txMgr.Call(ctx, ethereum.CallMsg{
		To:   &r.fdgAddr,
		Data: txData,
	}, nil)
	return err
}

// ResolveClaim executes a resolveClaim transaction to resolve a fault dispute game.
func (r *FaultResponder) ResolveClaim(ctx context.Context, claimIdx uint64) error {
	txData, err := r.contract.ResolveClaimData(claimIdx)
	if err != nil {
		return err
	}
	return r.sendTxAndWait(ctx, txData)
}

func (r *FaultResponder) PerformAction(ctx context.Context, action types.Action) error {
	var txData []byte
	var err error
	switch action.Type {
	case types.ActionTypeMove:
		if action.IsAttack {
			txData, err = r.contract.FaultAttackData(action.ParentIdx, action.Value)
		} else {
			txData, err = r.contract.FaultDefendData(action.ParentIdx, action.Value)
		}
	case types.ActionTypeStep:
		txData, err = r.contract.StepTxData(uint64(action.ParentIdx), action.IsAttack, action.PreState, action.ProofData)
	}
	if err != nil {
		return err
	}
	return r.sendTxAndWait(ctx, txData)
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (r *FaultResponder) sendTxAndWait(ctx context.Context, txData []byte) error {
	receipt, err := r.txMgr.Send(ctx, txmgr.TxCandidate{
		To:       &r.fdgAddr,
		TxData:   txData,
		GasLimit: 0,
	})
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
