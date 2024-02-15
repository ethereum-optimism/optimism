package mon

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/transform"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"

	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrContractCreation = errors.New("failed to create contract")
	ErrMetadataFetch    = errors.New("failed to fetch game metadata")
	ErrClaimFetch       = errors.New("failed to fetch game claims")
	ErrRootAgreement    = errors.New("failed to check root agreement")
)

type ForecastMetrics interface {
	RecordGameAgreement(status string, count int)
}

type forecast struct {
	logger    log.Logger
	metrics   ForecastMetrics
	creator   GameCallerCreator
	validator OutputValidator
}

func newForecast(logger log.Logger, metrics ForecastMetrics, creator GameCallerCreator, validator OutputValidator) *forecast {
	return &forecast{
		logger:    logger,
		metrics:   metrics,
		creator:   creator,
		validator: validator,
	}
}

func (f *forecast) Forecast(ctx context.Context, games []types.GameMetadata) {
	batch := monTypes.ForecastBatch{}
	for _, game := range games {
		if err := f.forecastGame(ctx, game, &batch); err != nil {
			f.logger.Error("Failed to forecast game", "err", err)
		}
	}
	f.recordBatch(batch)
}

func (f *forecast) recordBatch(batch monTypes.ForecastBatch) {
	f.metrics.RecordGameAgreement("agree_challenger_ahead", batch.AgreeChallengerAhead)
	f.metrics.RecordGameAgreement("disagree_challenger_ahead", batch.DisagreeChallengerAhead)
	f.metrics.RecordGameAgreement("agree_defender_ahead", batch.AgreeDefenderAhead)
	f.metrics.RecordGameAgreement("disagree_defender_ahead", batch.DisagreeDefenderAhead)
}

func (f *forecast) forecastGame(ctx context.Context, game types.GameMetadata, metrics *monTypes.ForecastBatch) error {
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

	// Create the bidirectional tree of claims.
	tree := transform.CreateBidirectionalTree(claims)

	// Compute the resolution status of the game.
	status = Resolve(tree)

	// Check the root agreement.
	agreement, expected, err := f.validator.CheckRootAgreement(ctx, l2BlockNum, rootClaim)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRootAgreement, err)
	}

	if agreement {
		// If we agree with the output root proposal, the Defender should win, defending that claim.
		if status == types.GameStatusChallengerWon {
			metrics.AgreeChallengerAhead++
			f.logger.Warn("Forecasting unexpected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		} else {
			metrics.AgreeDefenderAhead++
			f.logger.Debug("Forecasting expected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		}
	} else {
		// If we disagree with the output root proposal, the Challenger should win, challenging that claim.
		if status == types.GameStatusDefenderWon {
			metrics.DisagreeDefenderAhead++
			f.logger.Warn("Forecasting unexpected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		} else {
			metrics.DisagreeChallengerAhead++
			f.logger.Debug("Forecasting expected game result", "status", status, "game", game, "rootClaim", rootClaim, "expected", expected)
		}
	}

	return nil
}
