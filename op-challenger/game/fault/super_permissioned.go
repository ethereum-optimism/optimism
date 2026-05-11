package fault

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/generic"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type SuperPermissionedGameContract interface {
	generic.GenericGameLoader
	GetRootClaim(ctx context.Context) (common.Hash, error)
	GetL2SequenceNumber(ctx context.Context) (uint64, error)
	CallResolve(ctx context.Context) (gameTypes.GameStatus, error)
	ResolveTx() (txmgr.TxCandidate, error)
	ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error)
}

type SupervisorRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error)
}

type SuperNodeRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
}

type SuperPermissionedActor struct {
	log            log.Logger
	txSender       TxSender
	contract       SuperPermissionedGameContract
	rootProvider   SupervisorRootProvider
	superNode      SuperNodeRootProvider
	gameL1Head     eth.BlockID
	expectedRootFn func(context.Context, uint64) (common.Hash, bool, error)
}

var _ generic.Actor = (*SuperPermissionedActor)(nil)

func NewSuperPermissionedActor(
	logger log.Logger,
	txSender TxSender,
	contract SuperPermissionedGameContract,
	rootProvider SupervisorRootProvider,
	superNode SuperNodeRootProvider,
	gameL1Head eth.BlockID,
) *SuperPermissionedActor {
	return &SuperPermissionedActor{
		log:          logger,
		txSender:     txSender,
		contract:     contract,
		rootProvider: rootProvider,
		superNode:    superNode,
		gameL1Head:   gameL1Head,
	}
}

func (a *SuperPermissionedActor) Act(ctx context.Context) error {
	claim, err := a.contract.GetRootClaim(ctx)
	if err != nil {
		return fmt.Errorf("failed to load root claim: %w", err)
	}
	timestamp, err := a.contract.GetL2SequenceNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to load l2 sequence number: %w", err)
	}

	expected, valid, err := a.expectedRoot(ctx, timestamp)
	if err != nil {
		return err
	}
	if !valid || expected != claim {
		a.log.Info("Challenging invalid super permissioned game", "timestamp", timestamp, "claim", claim, "expected", expected, "valid", valid)
		tx, err := a.contract.ChallengeTx(ctx)
		if err != nil {
			return fmt.Errorf("failed to create challenge tx: %w", err)
		}
		return a.txSender.SendAndWaitSimple("challenge super permissioned game", tx)
	}

	status, err := a.contract.CallResolve(ctx)
	if err != nil {
		a.log.Debug("Super permissioned game not ready to resolve", "err", err)
		return nil
	}
	if status == gameTypes.GameStatusInProgress {
		return nil
	}
	tx, err := a.contract.ResolveTx()
	if err != nil {
		return fmt.Errorf("failed to create resolve tx: %w", err)
	}
	return a.txSender.SendAndWaitSimple("resolve super permissioned game", tx)
}

func (a *SuperPermissionedActor) AdditionalStatus(ctx context.Context) ([]any, error) {
	timestamp, err := a.contract.GetL2SequenceNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load l2 sequence number: %w", err)
	}
	return []any{"l2SequenceNumber", timestamp}, nil
}

func (a *SuperPermissionedActor) expectedRoot(ctx context.Context, timestamp uint64) (common.Hash, bool, error) {
	if a.expectedRootFn != nil {
		return a.expectedRootFn(ctx, timestamp)
	}
	if a.superNode != nil {
		return a.expectedRootFromSuperNode(ctx, timestamp)
	}
	return a.expectedRootFromSupervisor(ctx, timestamp)
}

func (a *SuperPermissionedActor) expectedRootFromSupervisor(ctx context.Context, timestamp uint64) (common.Hash, bool, error) {
	root, err := a.rootProvider.SuperRootAtTimestamp(ctx, hexutil.Uint64(timestamp))
	if errors.Is(err, ethereum.NotFound) {
		return common.Hash{}, false, nil
	}
	if err != nil {
		return common.Hash{}, false, fmt.Errorf("failed to retrieve super root at timestamp %v: %w", timestamp, err)
	}
	if root.CrossSafeDerivedFrom.Number > a.gameL1Head.Number {
		return common.Hash{}, false, nil
	}
	return common.Hash(root.SuperRoot), true, nil
}

func (a *SuperPermissionedActor) expectedRootFromSuperNode(ctx context.Context, timestamp uint64) (common.Hash, bool, error) {
	root, err := a.superNode.SuperRootAtTimestamp(ctx, timestamp)
	if err != nil {
		return common.Hash{}, false, fmt.Errorf("failed to retrieve super root at timestamp %v: %w", timestamp, err)
	}
	if root.CurrentL1.Number < a.gameL1Head.Number {
		return common.Hash{}, false, gameTypes.ErrNotInSync
	}
	if root.Data == nil {
		return common.Hash{}, false, nil
	}
	if root.Data.VerifiedRequiredL1.Number > a.gameL1Head.Number {
		return common.Hash{}, false, nil
	}
	return common.Hash(root.Data.SuperRoot), true, nil
}
