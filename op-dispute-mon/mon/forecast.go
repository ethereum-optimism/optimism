package mon

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/resolution"
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
	RecordClaimResolutionDelayMax(delay float64)
	RecordGameAgreement(status metrics.GameAgreementStatus, count int)
}

type forecast struct {
	logger    log.Logger
	metrics   ForecastMetrics
	validator OutputValidator
}

func newForecast(logger log.Logger, metrics ForecastMetrics, validator OutputValidator) *forecast {
	return &forecast{
		logger:    logger,
		metrics:   metrics,
		validator: validator,
	}
}

func (f *forecast) Forecast(ctx context.Context, games []*monTypes.EnrichedGameData) {
	batch := monTypes.ForecastBatch{}
	for _, game := range games {
		if err := f.forecastGame(ctx, game, &batch); err != nil {
			f.logger.Error("Failed to forecast game", "err", err)
		}
	}
	f.recordBatch(batch)
}

func (f *forecast) recordBatch(batch monTypes.ForecastBatch) {
	f.metrics.RecordGameAgreement(metrics.AgreeChallengerAhead, batch.AgreeChallengerAhead)
	f.metrics.RecordGameAgreement(metrics.DisagreeChallengerAhead, batch.DisagreeChallengerAhead)
	f.metrics.RecordGameAgreement(metrics.AgreeDefenderAhead, batch.AgreeDefenderAhead)
	f.metrics.RecordGameAgreement(metrics.DisagreeDefenderAhead, batch.DisagreeDefenderAhead)
}

func (f *forecast) forecastGame(ctx context.Context, game *monTypes.EnrichedGameData, metrics *monTypes.ForecastBatch) error {
	if game.Status != types.GameStatusInProgress {
		f.logger.Debug("Game is not in progress, skipping forecast", "game", game.Proxy, "status", game.Status)
		return nil
	}

	// Create the bidirectional tree of claims.
	tree := transform.CreateBidirectionalTree(game.Claims)

	// Compute the resolution status of the game.
	status := resolution.Resolve(tree)

	// Check the root agreement.
	agreement, expected, err := f.validator.CheckRootAgreement(ctx, game.L2BlockNumber, game.RootClaim)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRootAgreement, err)
	}

	if agreement {
		// If we agree with the output root proposal, the Defender should win, defending that claim.
		if status == types.GameStatusChallengerWon {
			metrics.AgreeChallengerAhead++
			f.logger.Warn("Forecasting unexpected game result", "status", status,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		} else {
			metrics.AgreeDefenderAhead++
			f.logger.Debug("Forecasting expected game result", "status", status,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		}
	} else {
		// If we disagree with the output root proposal, the Challenger should win, challenging that claim.
		if status == types.GameStatusDefenderWon {
			metrics.DisagreeDefenderAhead++
			f.logger.Warn("Forecasting unexpected game result", "status", status,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		} else {
			metrics.DisagreeChallengerAhead++
			f.logger.Debug("Forecasting expected game result", "status", status,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		}
	}

	return nil
}
