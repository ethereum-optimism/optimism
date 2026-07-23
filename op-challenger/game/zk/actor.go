package zk

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/generic"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	errNoChallengeRequired  = errors.New("no challenge required")
	errNoResolutionRequired = errors.New("no resolution required")
)

type SuperRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
}

type GameStatusProvider interface {
	GetGameStatus(ctx context.Context, idx uint64) (gameTypes.GameStatus, error)
}

type ChallengableContract interface {
	Addr() common.Address
	ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error)
	GetProposal(ctx context.Context) (common.Hash, uint64, error)
	GetChallengerMetadata(ctx context.Context, block rpcblock.Block) (contracts.ChallengerMetadata, error)
	ResolveTx() (txmgr.TxCandidate, error)
}

type Actor struct {
	logger             log.Logger
	l1Clock            ClockReader
	l1Head             eth.BlockID
	superRootProvider  SuperRootProvider
	gameStatusProvider GameStatusProvider
	contract           ChallengableContract
	txSender           TxSender
}

func ActorCreator(l1Clock ClockReader, superRootProvider SuperRootProvider, gameStatusProvider GameStatusProvider, contract ChallengableContract, txSender TxSender) generic.ActorCreator {
	return func(_ context.Context, logger log.Logger, l1Head eth.BlockID) (generic.Actor, error) {
		return &Actor{
			logger:             logger,
			l1Clock:            l1Clock,
			l1Head:             l1Head,
			superRootProvider:  superRootProvider,
			gameStatusProvider: gameStatusProvider,
			contract:           contract,
			txSender:           txSender,
		}, nil
	}
}

func (a *Actor) Act(ctx context.Context) error {
	gameState, err := a.contract.GetChallengerMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to get zk game state: %w", err)
	}

	var txs []txmgr.TxCandidate
	if tx, err := a.createChallengeTx(ctx, gameState); errors.Is(err, errNoChallengeRequired) {
		a.logger.Debug("No challenge required")
	} else if err != nil {
		return err
	} else {
		txs = append(txs, tx)
	}
	if tx, err := a.createResolveTx(ctx, gameState); errors.Is(err, errNoResolutionRequired) {
		a.logger.Debug("No resolution required")
	} else if err != nil {
		return err
	} else {
		txs = append(txs, tx)
	}

	if len(txs) == 0 {
		return nil
	}
	if err := a.txSender.SendAndWaitSimple(fmt.Sprintf("respond to game %v", a.contract.Addr()), txs...); err != nil {
		return fmt.Errorf("failed to send transactions for game %v: %w", a.contract.Addr(), err)
	}
	return nil
}

func (a *Actor) createChallengeTx(ctx context.Context, gameState contracts.ChallengerMetadata) (txmgr.TxCandidate, error) {
	if gameState.ProposalStatus != contracts.ProposalStatusUnchallenged || gameState.Deadline.Before(a.l1Clock.Now()) {
		a.logger.Trace("Skipping unchallengeable zk game")
		return txmgr.TxCandidate{}, errNoChallengeRequired
	}
	valid, err := a.isValidProposal(ctx)
	if errors.Is(err, gameTypes.ErrNotInSync) {
		a.logger.Debug("Waiting for source node to process past the game L1 head")
		return txmgr.TxCandidate{}, errNoChallengeRequired
	}
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to check if proposal is valid: %w", err)
	}
	if valid {
		a.logger.Trace("Not challenging valid zk game")
		return txmgr.TxCandidate{}, errNoChallengeRequired
	}

	a.logger.Info("Challenging game")
	return a.contract.ChallengeTx(ctx)
}

func (a *Actor) isValidProposal(ctx context.Context) (bool, error) {
	proposalHash, proposalTimestamp, err := a.contract.GetProposal(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get zk game proposal: %w", err)
	}
	resp, err := a.superRootProvider.SuperRootAtTimestamp(ctx, proposalTimestamp)
	if err != nil {
		return false, fmt.Errorf("failed to get canonical super root at timestamp %v: %w", proposalTimestamp, err)
	}
	if resp.CurrentL1.Number <= a.l1Head.Number {
		// Source node hasn't fully processed the game's L1 head yet — can't decide on a stale view.
		return false, gameTypes.ErrNotInSync
	}
	if resp.Data == nil {
		// No super root at this timestamp (future / not yet available) — cannot be valid.
		return false, nil
	}
	if common.Hash(resp.Data.SuperRoot) != proposalHash {
		// Super root doesn't match the proposal claim — cannot be valid.
		return false, nil
	}
	if resp.CurrentSafeTimestamp < proposalTimestamp {
		// Proposal timestamp is beyond the cross-safe tip, so it cannot be validated yet.
		a.logger.Debug("Proposed super root is not yet safe, treating as invalid",
			"safeTimestamp", resp.CurrentSafeTimestamp, "proposedTimestamp", proposalTimestamp)
		return false, nil
	}
	if resp.Data.VerifiedRequiredL1.Number > a.l1Head.Number {
		// Canonical and safe now but unprovable within this game's l1Head; accept, don't challenge.
		a.logger.Warn("ZK proposal canonical but not provable within game l1Head; not challenging",
			"proposalTimestamp", proposalTimestamp, "verifiedRequiredL1", resp.Data.VerifiedRequiredL1.Number, "gameL1Head", a.l1Head.Number)
	}
	return true, nil
}

func (a *Actor) createResolveTx(ctx context.Context, gameState contracts.ChallengerMetadata) (txmgr.TxCandidate, error) {
	if gameState.ProposalStatus == contracts.ProposalStatusResolved {
		a.logger.Trace("Skipping resolution of resolved zk game")
		return txmgr.TxCandidate{}, errNoResolutionRequired
	}
	deadlineExpired := gameState.Deadline.Before(a.l1Clock.Now())

	if gameState.ParentIndex != math.MaxUint32 {
		parentStatus, err := a.gameStatusProvider.GetGameStatus(ctx, uint64(gameState.ParentIndex))
		if err != nil {
			return txmgr.TxCandidate{}, fmt.Errorf("failed to get parent game status: %w", err)
		}
		if parentStatus == gameTypes.GameStatusInProgress {
			a.logger.Trace("Skipping resolution of zk game with parent in progress")
			return txmgr.TxCandidate{}, errNoResolutionRequired
		}
		if parentStatus == gameTypes.GameStatusChallengerWon {
			// Resolve if the parent game is invalid
			return a.contract.ResolveTx()
		}
	}

	if gameState.ProposalStatus == contracts.ProposalStatusChallengedAndValidProofProvided ||
		gameState.ProposalStatus == contracts.ProposalStatusUnchallengedAndValidProofProvided {
		// Resolve if a valid proof is provided
		return a.contract.ResolveTx()
	}
	if deadlineExpired {
		// Resolve if the deadline has expired (either for challenging or proving)
		return a.contract.ResolveTx()
	}
	return txmgr.TxCandidate{}, errNoResolutionRequired
}

func (a *Actor) AdditionalStatus(ctx context.Context) ([]any, error) {
	metadata, err := a.contract.GetChallengerMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenger metadata: %w", err)
	}
	return []any{"proposalStatus", metadata.ProposalStatus}, nil
}
