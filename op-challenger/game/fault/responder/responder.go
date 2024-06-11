package responder

import (
	"context"
	"errors"
	"fmt"

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
	AttackTx(ctx context.Context, parent types.Claim, pivot common.Hash) (txmgr.TxCandidate, error)
	DefendTx(ctx context.Context, parent types.Claim, pivot common.Hash) (txmgr.TxCandidate, error)
	StepTx(claimIdx uint64, isAttack bool, stateData []byte, proof []byte) (txmgr.TxCandidate, error)
	ChallengeL2BlockNumberTx(challenge *types.InvalidL2BlockNumberChallenge) (txmgr.TxCandidate, error)
}

type Oracle interface {
	GlobalDataExists(ctx context.Context, data *types.PreimageOracleData) (bool, error)
}

type TxSender interface {
	SendAndWaitSimple(txPurpose string, txs ...txmgr.TxCandidate) error
}

// FaultResponder implements the [Responder] interface to send onchain transactions.
type FaultResponder struct {
	log      log.Logger
	sender   TxSender
	contract GameContract
	uploader preimages.PreimageUploader
	oracle   Oracle
}

// NewFaultResponder returns a new [FaultResponder].
func NewFaultResponder(logger log.Logger, sender TxSender, contract GameContract, uploader preimages.PreimageUploader, oracle Oracle) (*FaultResponder, error) {
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

	return r.sender.SendAndWaitSimple("resolve game", candidate)
}

// CallResolveClaim determines if the resolveClaim function on the fault dispute game contract
// would succeed.
func (r *FaultResponder) CallResolveClaim(ctx context.Context, claimIdx uint64) error {
	return r.contract.CallResolveClaim(ctx, claimIdx)
}

// ResolveClaims executes resolveClaim transactions to resolve claims in a dispute game.
func (r *FaultResponder) ResolveClaims(claimIdxs ...uint64) error {
	txs := make([]txmgr.TxCandidate, 0, len(claimIdxs))
	for _, claimIdx := range claimIdxs {
		candidate, err := r.contract.ResolveClaimTx(claimIdx)
		if err != nil {
			return err
		}
		txs = append(txs, candidate)
	}
	return r.sender.SendAndWaitSimple("resolve claim", txs...)
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
			err := r.uploader.UploadPreimage(ctx, uint64(action.ParentClaim.ContractIndex), action.OracleData)
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
		if action.IsAttack {
			candidate, err = r.contract.AttackTx(ctx, action.ParentClaim, action.Value)
		} else {
			candidate, err = r.contract.DefendTx(ctx, action.ParentClaim, action.Value)
		}
	case types.ActionTypeStep:
		candidate, err = r.contract.StepTx(uint64(action.ParentClaim.ContractIndex), action.IsAttack, action.PreState, action.ProofData)
	case types.ActionTypeChallengeL2BlockNumber:
		candidate, err = r.contract.ChallengeL2BlockNumberTx(action.InvalidL2BlockNumberChallenge)
	}
	if err != nil {
		return err
	}
	return r.sender.SendAndWaitSimple("perform action", candidate)
}
