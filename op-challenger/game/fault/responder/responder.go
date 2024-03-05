package responder

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/preimages"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"

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
	GetRequiredBond(ctx context.Context, position types.Position) (*big.Int, error)
}

type Oracle interface {
	GlobalDataExists(ctx context.Context, data *types.PreimageOracleData) (bool, error)
}

// FaultResponder implements the [Responder] interface to send onchain transactions.
type FaultResponder struct {
	log      log.Logger
	sender   gameTypes.TxSender
	contract GameContract
	uploader preimages.PreimageUploader
	oracle   Oracle
}

// NewFaultResponder returns a new [FaultResponder].
func NewFaultResponder(logger log.Logger, sender gameTypes.TxSender, contract GameContract, uploader preimages.PreimageUploader, oracle Oracle) (*FaultResponder, error) {
	return &FaultResponder{
		log:      logger,
		sender:   sender,
		contract: contract,
		uploader: uploader,
		oracle:   oracle,
	}, nil
}

// CallResolve determines if the resolve function on the fault dispute game contract
// would succeed. Returns the game status if the call would succeed, errors otherwise.
func (r *FaultResponder) CallResolve(ctx context.Context) (gameTypes.GameStatus, error) {
	return r.contract.CallResolve(ctx)
}

// Resolve executes a resolve transaction to resolve a fault dispute game.
func (r *FaultResponder) Resolve() error {
	candidate, err := r.contract.ResolveTx()
	if err != nil {
		return err
	}

	return r.sendTxAndWait("resolve game", candidate)
}

// CallResolveClaim determines if the resolveClaim function on the fault dispute game contract
// would succeed.
func (r *FaultResponder) CallResolveClaim(ctx context.Context, claimIdx uint64) error {
	return r.contract.CallResolveClaim(ctx, claimIdx)
}

// ResolveClaim executes a resolveClaim transaction to resolve a fault dispute game.
func (r *FaultResponder) ResolveClaim(claimIdx uint64) error {
	candidate, err := r.contract.ResolveClaimTx(claimIdx)
	if err != nil {
		return err
	}
	return r.sendTxAndWait("resolve claim", candidate)
}

func (r *FaultResponder) PerformAction(ctx context.Context, action types.Action) error {
	if action.OracleData != nil {
		var preimageExists bool
		var err error
		if !action.OracleData.IsLocal {
			preimageExists, err = r.oracle.GlobalDataExists(ctx, action.OracleData)
			if err != nil {
				return fmt.Errorf("failed to check if preimage exists: %w", err)
			}
		}
		// Always upload local preimages
		if !preimageExists {
			err := r.uploader.UploadPreimage(ctx, uint64(action.ParentIdx), action.OracleData)
			if errors.Is(err, preimages.ErrChallengePeriodNotOver) {
				r.log.Debug("Large Preimage Squeeze failed, challenge period not over")
				return nil
			} else if err != nil {
				return fmt.Errorf("failed to upload preimage: %w", err)
			}
		}
	}
	var candidate txmgr.TxCandidate
	var err error
	switch action.Type {
	case types.ActionTypeMove:
		var movePos types.Position
		if action.IsAttack {
			movePos = action.ParentPosition.Attack()
			candidate, err = r.contract.AttackTx(uint64(action.ParentIdx), action.Value)
		} else {
			movePos = action.ParentPosition.Defend()
			candidate, err = r.contract.DefendTx(uint64(action.ParentIdx), action.Value)
		}

		bondValue, err := r.contract.GetRequiredBond(ctx, movePos)
		if err != nil {
			return err
		}
		candidate.Value = bondValue
	case types.ActionTypeStep:
		candidate, err = r.contract.StepTx(uint64(action.ParentIdx), action.IsAttack, action.PreState, action.ProofData)
	}
	if err != nil {
		return err
	}
	return r.sendTxAndWait("perform action", candidate)
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (r *FaultResponder) sendTxAndWait(purpose string, candidate txmgr.TxCandidate) error {
	_, err := r.sender.SendAndWait(purpose, candidate)
	return err
}
