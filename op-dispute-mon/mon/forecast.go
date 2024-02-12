package mon

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"

	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrContractCreation = errors.New("failed to create contract")
	ErrMetadataFetch    = errors.New("failed to fetch game metadata")
	ErrClaimFetch       = errors.New("failed to fetch game claims")
	ErrResolver         = errors.New("failed to resolve game")
	ErrRootAgreement    = errors.New("failed to check root agreement")
)

type forecast struct {
	logger log.Logger
	// TODO(client-pod#536): Add forecast metrics.
	//                       These should only fire if a game is in progress.
	//                       otherwise, the detector should record the game status.
	creator   GameCallerCreator
	validator OutputValidator
}

func newForecast(logger log.Logger, creator GameCallerCreator, validator OutputValidator) *forecast {
	return &forecast{
		logger:    logger,
		creator:   creator,
		validator: validator,
	}
}

func (f *forecast) Forecast(ctx context.Context, games []types.GameMetadata) {
	for _, game := range games {
		if err := f.forecastGame(ctx, game); err != nil {
			f.logger.Error("Failed to forecast game", "err", err)
		}
	}
}

func (f *forecast) forecastGame(ctx context.Context, game types.GameMetadata) error {
	loader, err := f.creator.CreateContract(game)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrContractCreation, err)
	}

	// Get the game status, it must be in progress to forecast.
	l2BlockNum, rootClaim, status, err := loader.GetGameMetadata(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMetadataFetch, err)
	}
	if status != types.GameStatusInProgress {
		f.logger.Debug("Game is not in progress, skipping forecast", "game", game, "status", status)
		return nil
	}

	// Load all claims for the game.
	claims, err := loader.GetAllClaims(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrClaimFetch, err)
	}

	// Compute the resolution status of the game.
	status, err = Resolve(claims)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrResolver, err)
	}

	// Check the root agreement.
	agreement, expected, err := f.validator.CheckRootAgreement(ctx, l2BlockNum, rootClaim)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRootAgreement, err)
	}

	if agreement {
		// If we agree with the output root proposal, the Defender should win, defending that claim.
		if status == types.GameStatusChallengerWon {
			f.logger.Warn("Forecasting unexpected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		} else {
			f.logger.Debug("Forecasting expected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		}
	} else {
		// If we disagree with the output root proposal, the Challenger should win, challenging that claim.
		if status == types.GameStatusDefenderWon {
			f.logger.Warn("Forecasting unexpected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		} else {
			f.logger.Debug("Forecasting expected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		}
	}

	return nil
}
