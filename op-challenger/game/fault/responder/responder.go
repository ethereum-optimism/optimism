package responder

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// FaultResponder implements the [Responder] interface to send onchain transactions.
type FaultResponder struct {
	log log.Logger

	txMgr txmgr.TxManager

	fdgAddr common.Address
	fdgAbi  *abi.ABI
}

// NewFaultResponder returns a new [FaultResponder].
func NewFaultResponder(logger log.Logger, txManagr txmgr.TxManager, fdgAddr common.Address) (*FaultResponder, error) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &FaultResponder{
		log:     logger,
		txMgr:   txManagr,
		fdgAddr: fdgAddr,
		fdgAbi:  fdgAbi,
	}, nil
}

// buildFaultDefendData creates the transaction data for the Defend function.
func (r *FaultResponder) buildFaultDefendData(parentContractIndex int, pivot [32]byte) ([]byte, error) {
	return r.fdgAbi.Pack(
		"defend",
		big.NewInt(int64(parentContractIndex)),
		pivot,
	)
}

// buildFaultAttackData creates the transaction data for the Attack function.
func (r *FaultResponder) buildFaultAttackData(parentContractIndex int, pivot [32]byte) ([]byte, error) {
	return r.fdgAbi.Pack(
		"attack",
		big.NewInt(int64(parentContractIndex)),
		pivot,
	)
}

// buildResolveData creates the transaction data for the Resolve function.
func (r *FaultResponder) buildResolveData() ([]byte, error) {
	return r.fdgAbi.Pack("resolve")
}

// CallResolve determines if the resolve function on the fault dispute game contract
// would succeed. Returns the game status if the call would succeed, errors otherwise.
func (r *FaultResponder) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	txData, err := r.buildResolveData()
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
	var status uint8
	if err = r.fdgAbi.UnpackIntoInterface(&status, "resolve", res); err != nil {
		return gameTypes.GameStatusInProgress, err
	}
	return gameTypes.GameStatusFromUint8(status)
}

// Resolve executes a resolve transaction to resolve a fault dispute game.
func (r *FaultResponder) Resolve(ctx context.Context) error {
	txData, err := r.buildResolveData()
	if err != nil {
		return err
	}

	return r.sendTxAndWait(ctx, txData)
}

// buildResolveClaimData creates the transaction data for the ResolveClaim function.
func (r *FaultResponder) buildResolveClaimData(ctx context.Context, claimIdx uint64) ([]byte, error) {
	return r.fdgAbi.Pack("resolveClaim", big.NewInt(int64(claimIdx)))
}

// CallResolveClaim determines if the resolveClaim function on the fault dispute game contract
// would succeed.
func (r *FaultResponder) CallResolveClaim(ctx context.Context, claimIdx uint64) error {
	txData, err := r.buildResolveClaimData(ctx, claimIdx)
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
	txData, err := r.buildResolveClaimData(ctx, claimIdx)
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
			txData, err = r.buildFaultAttackData(action.ParentIdx, action.Value)
		} else {
			txData, err = r.buildFaultDefendData(action.ParentIdx, action.Value)
		}
	case types.ActionTypeStep:
		txData, err = r.buildStepTxData(uint64(action.ParentIdx), action.IsAttack, action.PreState, action.ProofData)
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

// buildStepTxData creates the transaction data for the step function.
func (r *FaultResponder) buildStepTxData(claimIdx uint64, isAttack bool, stateData []byte, proof []byte) ([]byte, error) {
	return r.fdgAbi.Pack(
		"step",
		big.NewInt(int64(claimIdx)),
		isAttack,
		stateData,
		proof,
	)
}
