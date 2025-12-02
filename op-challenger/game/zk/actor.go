package zk

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/generic"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type RootProvider interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type GameStatusProvider interface {
	GetGameStatus(ctx context.Context, idx uint64) (gameTypes.GameStatus, error)
}

type ChallengableContract interface {
	ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error)
	GetProposal(ctx context.Context) (common.Hash, uint64, error)
	GetChallengerMetadata(ctx context.Context, block rpcblock.Block) (contracts.ChallengerMetadata, error)
	ResolveTx() (txmgr.TxCandidate, error)
}

type Actor struct {
	logger             log.Logger
	l1Clock            ClockReader
	rootProvider       RootProvider
	gameStatusProvider GameStatusProvider
	contract           ChallengableContract
	txSender           TxSender
	l1Head             eth.BlockID
}

func ActorCreator(l1Clock ClockReader, rootProvider RootProvider, gameStatusProvider GameStatusProvider, contract ChallengableContract, txSender TxSender) generic.ActorCreator {
	return func(ctx context.Context, logger log.Logger, l1Head eth.BlockID) (generic.Actor, error) {
		return &Actor{
			logger:             logger,
			l1Clock:            l1Clock,
			rootProvider:       rootProvider,
			gameStatusProvider: gameStatusProvider,
			contract:           contract,
			txSender:           txSender,
			l1Head:             l1Head,
		}, nil
	}
}

func (a *Actor) Act(ctx context.Context) error {
	gameState, err := a.contract.GetChallengerMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to get zk game state: %w", err)
	}
	if resolved, err := a.tryResolve(ctx, gameState); err != nil {
		return err
	} else if resolved {
		return nil
	}
	if gameState.ProposalStatus != contracts.ProposalStatusUnchallenged {
		a.logger.Debug("Skipping unchallengeable zk game")
		return nil
	}

	// Check if we agree with the proposal
	proposalHash, proposalSeqNum, err := a.contract.GetProposal(ctx)
	if err != nil {
		return fmt.Errorf("failed to get zk game proposal: %w", err)
	}
	if valid, err := a.isValidProposal(ctx, proposalSeqNum, proposalHash); err != nil {
		return fmt.Errorf("failed to check if proposal is valid: %w", err)
	} else if valid {
		a.logger.Debug("Not challenging valid zk game")
		return nil
	}

	a.logger.Info("Challenging game")
	tx, err := a.contract.ChallengeTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to create challenge tx: %w", err)
	}
	if err := a.txSender.SendAndWaitSimple("challenge zk game", tx); err != nil {
		return fmt.Errorf("failed to challenge zk game: %w", err)
	}
	return nil
}

func (a *Actor) isValidProposal(ctx context.Context, proposalSeqNum uint64, proposalHash common.Hash) (bool, error) {
	canonicalOutput, err := a.rootProvider.OutputAtBlock(ctx, proposalSeqNum)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			if strings.Contains(strings.ToLower(rpcErr.Error()), "not found") {
				// There is no valid output at the proposal sequence number (it's in the future)
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to get canonical output at block %v: %w", proposalSeqNum, err)
	}
	if common.Hash(canonicalOutput.OutputRoot) != proposalHash {
		// Output root doesn't match so can't be valid
		return false, nil
	}
	return true, nil
}

func (a *Actor) tryResolve(ctx context.Context, gameState contracts.ChallengerMetadata) (bool, error) {
	if gameState.ProposalStatus == contracts.ProposalStatusResolved {
		a.logger.Trace("Skipping resolution of resolved zk game")
		return true, nil // Already resolved so skip challenging
	}
	deadlineExpired := gameState.Deadline.Before(a.l1Clock.Now())

	parentStatus, err := a.gameStatusProvider.GetGameStatus(ctx, uint64(gameState.ParentIndex))
	if err != nil {
		return false, fmt.Errorf("failed to get parent game status: %w", err)
	}
	if parentStatus == gameTypes.GameStatusInProgress {
		a.logger.Trace("Skipping resolution of zk game with parent in progress")
		return deadlineExpired, nil // skip challenging if deadline already expired
	}

	if gameState.ProposalStatus == contracts.ProposalStatusChallengedAndValidProofProvided ||
		gameState.ProposalStatus == contracts.ProposalStatusUnchallengedAndValidProofProvided {
		// Resolve if a valid proof is provided
		return a.resolve()
	}
	if deadlineExpired {
		// Resolve if the deadline has expired (either for challenging or proving)
		return a.resolve()
	}
	if parentStatus == gameTypes.GameStatusChallengerWon {
		// Resolve if the parent game is invalid
		return a.resolve()
	}
	return false, nil
}

func (a *Actor) resolve() (bool, error) {
	a.logger.Info("Resolving zk game")
	tx, err := a.contract.ResolveTx()
	if err != nil {
		return false, fmt.Errorf("failed to create resolve tx: %w", err)
	}
	if err := a.txSender.SendAndWaitSimple("resolve zk game", tx); err != nil {
		return false, fmt.Errorf("failed to resolve zk game: %w", err)
	}
	return true, nil
}

func (a *Actor) AdditionalStatus(_ context.Context) ([]any, error) {
	return nil, nil
}
