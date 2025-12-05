package zk

import (
	"context"
	"errors"
	"fmt"
	"math"
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

var (
	errNoChallengeRequired  = errors.New("no challenge required")
	errNoResolutionRequired = errors.New("no resolution required")
)

type RootProvider interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
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
	if valid, err := a.isValidProposal(ctx); err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to check if proposal is valid: %w", err)
	} else if valid {
		a.logger.Trace("Not challenging valid zk game")
		return txmgr.TxCandidate{}, errNoChallengeRequired
	}

	a.logger.Info("Challenging game")
	return a.contract.ChallengeTx(ctx)
}

func (a *Actor) isValidProposal(ctx context.Context) (bool, error) {
	proposalHash, proposalSeqNum, err := a.contract.GetProposal(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get zk game proposal: %w", err)
	}
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
	if canonicalOutput.Status.SafeL2.Number < proposalSeqNum {
		// Note this deliberately uses the simpler check of if the proposed block is currently unsafe
		// The proposal is not necessarily supported by data on L1 up to the game's L1 head
		// but we don't need to challenge it as long as supporting data has since become available
		// and the output matches the canonical chain.
		a.logger.Debug("Proposed block is not yet safe, treating as invalid", "safe", canonicalOutput.Status.SafeL2.Number, "proposed", proposalSeqNum)
		return false, nil
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
