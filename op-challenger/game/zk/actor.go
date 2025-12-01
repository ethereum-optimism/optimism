package zk

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-challenger/game/generic"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type RootProvider interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type ChallengableContract interface {
	CanChallenge(ctx context.Context) (bool, error)
	ChallengeTx(ctx context.Context) (txmgr.TxCandidate, error)
	GetProposal(ctx context.Context) (common.Hash, uint64, error)
}

type Actor struct {
	logger       log.Logger
	rootProvider RootProvider
	contract     ChallengableContract
	txSender     TxSender
	l1Head       eth.BlockID
}

func ActorCreator(rootProvider RootProvider, contract ChallengableContract, txSender TxSender) generic.ActorCreator {
	return func(ctx context.Context, logger log.Logger, l1Head eth.BlockID) (generic.Actor, error) {
		return &Actor{
			logger:       logger,
			rootProvider: rootProvider,
			contract:     contract,
			txSender:     txSender,
			l1Head:       l1Head,
		}, nil
	}
}

func (a *Actor) Act(ctx context.Context) error {
	canChallenge, err := a.contract.CanChallenge(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if game can be challenged: %w", err)
	}
	if !canChallenge {
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

func (a *Actor) AdditionalStatus(ctx context.Context) ([]any, error) {
	return nil, nil
}
