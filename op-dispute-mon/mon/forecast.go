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
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrRootAgreement = errors.New("failed to check root agreement")
)

type OutputValidator interface {
	CheckRootAgreement(ctx context.Context, l1HeadNum uint64, l2BlockNum uint64, root common.Hash) (bool, common.Hash, error)
}

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
	f.metrics.RecordGameAgreement(metrics.AgreeDefenderWins, batch.AgreeDefenderWins)
	f.metrics.RecordGameAgreement(metrics.DisagreeDefenderWins, batch.DisagreeDefenderWins)
	f.metrics.RecordGameAgreement(metrics.AgreeChallengerWins, batch.AgreeChallengerWins)
	f.metrics.RecordGameAgreement(metrics.DisagreeChallengerWins, batch.DisagreeChallengerWins)

	f.metrics.RecordGameAgreement(metrics.AgreeChallengerAhead, batch.AgreeChallengerAhead)
	f.metrics.RecordGameAgreement(metrics.DisagreeChallengerAhead, batch.DisagreeChallengerAhead)
	f.metrics.RecordGameAgreement(metrics.AgreeDefenderAhead, batch.AgreeDefenderAhead)
	f.metrics.RecordGameAgreement(metrics.DisagreeDefenderAhead, batch.DisagreeDefenderAhead)
}

func (f *forecast) forecastGame(ctx context.Context, game *monTypes.EnrichedGameData, metrics *monTypes.ForecastBatch) error {
	// Check the root agreement.
	agreement, expected, err := f.validator.CheckRootAgreement(ctx, game.L1HeadNum, game.L2BlockNumber, game.RootClaim)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRootAgreement, err)
	}

	expectedResult := types.GameStatusDefenderWon
	if !agreement {
		expectedResult = types.GameStatusChallengerWon
	}

	if game.Status != types.GameStatusInProgress {
		if game.Status != expectedResult {
			f.logger.Error("Unexpected game result",
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"expectedResult", expectedResult, "actualResult", game.Status,
				"rootClaim", game.RootClaim, "correctClaim", expected)
		}
		switch game.Status {
		case types.GameStatusDefenderWon:
			if agreement {
				metrics.AgreeDefenderWins++
			} else {
				metrics.DisagreeDefenderWins++
			}
		case types.GameStatusChallengerWon:
			if agreement {
				metrics.AgreeChallengerWins++
			} else {
				metrics.DisagreeChallengerWins++
			}
		}
		return nil
	}

	// Create the bidirectional tree of claims.
	tree := transform.CreateBidirectionalTree(game.Claims)

	// Compute the resolution status of the game.
	forecastStatus := resolution.Resolve(tree)

	if agreement {
		// If we agree with the output root proposal, the Defender should win, defending that claim.
		if forecastStatus == types.GameStatusChallengerWon {
			metrics.AgreeChallengerAhead++
			f.logger.Warn("Forecasting unexpected game result", "status", forecastStatus,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		} else {
			metrics.AgreeDefenderAhead++
			f.logger.Debug("Forecasting expected game result", "status", forecastStatus,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		}
	} else {
		// If we disagree with the output root proposal, the Challenger should win, challenging that claim.
		if forecastStatus == types.GameStatusDefenderWon {
			metrics.DisagreeDefenderAhead++
			f.logger.Warn("Forecasting unexpected game result", "status", forecastStatus,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		} else {
			metrics.DisagreeChallengerAhead++
			f.logger.Debug("Forecasting expected game result", "status", forecastStatus,
				"game", game.Proxy, "blockNum", game.L2BlockNumber,
				"rootClaim", game.RootClaim, "expected", expected)
		}
	}

	return nil
}
