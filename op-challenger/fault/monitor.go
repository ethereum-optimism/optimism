package fault

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/log"
)

type GameInfo interface {
	GetGameStatus(context.Context) (types.GameStatus, error)
	LogGameInfo(ctx context.Context)
}

type Actor interface {
	Act(ctx context.Context) error
}

func MonitorGame(ctx context.Context, logger log.Logger, agreeWithProposedOutput bool, actor Actor, caller GameInfo) error {
	logger.Info("Monitoring fault dispute game", "agreeWithOutput", agreeWithProposedOutput)

	for {
		done := progressGame(ctx, logger, agreeWithProposedOutput, actor, caller)
		if done {
			return nil
		}
		select {
		case <-time.After(300 * time.Millisecond):
		// Continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// progressGame checks the current state of the game, and attempts to progress it by performing moves, steps or resolving
// Returns true if the game is complete or false if it needs to be monitored further
func progressGame(ctx context.Context, logger log.Logger, agreeWithProposedOutput bool, actor Actor, caller GameInfo) bool {
	logger.Trace("Checking if actions are required")
	if err := actor.Act(ctx); err != nil {
		logger.Error("Error when acting on game", "err", err)
	}
	if status, err := caller.GetGameStatus(ctx); err != nil {
		logger.Warn("Unable to retrieve game status", "err", err)
	} else if status != 0 {
		var expectedStatus types.GameStatus
		if agreeWithProposedOutput {
			expectedStatus = types.GameStatusChallengerWon
		} else {
			expectedStatus = types.GameStatusDefenderWon
		}
		if expectedStatus == status {
			logger.Info("Game won", "status", GameStatusString(status))
		} else {
			logger.Error("Game lost", "status", GameStatusString(status))
		}
		return true
	} else {
		caller.LogGameInfo(ctx)
	}
	return false
}
