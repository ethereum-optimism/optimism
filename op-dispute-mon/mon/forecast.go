package mon

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/transform"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrRootAgreement = errors.New("failed to check root agreement")
)

type ForecastMetrics interface {
	RecordGameAgreement(status metrics.GameAgreementStatus, count int)
	RecordLatestValidProposalL2Block(validL2Block uint64)
	RecordLatestProposals(validTimestamp, invalidTimestamp uint64)
	RecordIgnoredGames(count int)
	RecordFailedGames(count int)
}

type forecastBatch struct {
	AgreeDefenderAhead      int
	DisagreeDefenderAhead   int
	AgreeChallengerAhead    int
	DisagreeChallengerAhead int

	AgreeDefenderWins      int
	DisagreeDefenderWins   int
	AgreeChallengerWins    int
	DisagreeChallengerWins int

	LatestValidProposalL2Block uint64
	LatestInvalidProposal      uint64
	LatestValidProposal        uint64
}

type Forecast struct {
	logger  log.Logger
	metrics ForecastMetrics
}

func NewForecast(logger log.Logger, metrics ForecastMetrics) *Forecast {
	return &Forecast{
		logger:  logger,
		metrics: metrics,
	}
}

func (f *Forecast) Forecast(games []*monTypes.EnrichedGameData, ignoredCount, failedCount int) {
	batch := forecastBatch{}
	for _, game := range games {
		if err := f.forecastGame(game, &batch); err != nil {
			f.logger.Error("Failed to forecast game", "err", err)
		}
	}
	f.recordBatch(batch, ignoredCount, failedCount)
}

func (f *Forecast) recordBatch(batch forecastBatch, ignoredCount, failedCount int) {
	f.metrics.RecordGameAgreement(metrics.AgreeDefenderWins, batch.AgreeDefenderWins)
	f.metrics.RecordGameAgreement(metrics.DisagreeDefenderWins, batch.DisagreeDefenderWins)
	f.metrics.RecordGameAgreement(metrics.AgreeChallengerWins, batch.AgreeChallengerWins)
	f.metrics.RecordGameAgreement(metrics.DisagreeChallengerWins, batch.DisagreeChallengerWins)

	f.metrics.RecordGameAgreement(metrics.AgreeChallengerAhead, batch.AgreeChallengerAhead)
	f.metrics.RecordGameAgreement(metrics.DisagreeChallengerAhead, batch.DisagreeChallengerAhead)
	f.metrics.RecordGameAgreement(metrics.AgreeDefenderAhead, batch.AgreeDefenderAhead)
	f.metrics.RecordGameAgreement(metrics.DisagreeDefenderAhead, batch.DisagreeDefenderAhead)

	f.metrics.RecordLatestValidProposalL2Block(batch.LatestValidProposalL2Block)
	f.metrics.RecordLatestProposals(batch.LatestValidProposal, batch.LatestInvalidProposal)

	f.metrics.RecordIgnoredGames(ignoredCount)
	f.metrics.RecordFailedGames(failedCount)
}

func (f *Forecast) forecastGame(game *monTypes.EnrichedGameData, metrics *forecastBatch) error {
	// Check the root agreement.
	agreement := game.AgreeWithClaim
	expected := game.ExpectedRootClaim

	expectedResult := types.GameStatusDefenderWon
	if !agreement {
		expectedResult = types.GameStatusChallengerWon
		if metrics.LatestInvalidProposal < game.Timestamp {
			metrics.LatestInvalidProposal = game.Timestamp
		}
	} else {
		if metrics.LatestValidProposal < game.Timestamp {
			metrics.LatestValidProposal = game.Timestamp
		}
		if metrics.LatestValidProposalL2Block < game.L2BlockNumber {
			metrics.LatestValidProposalL2Block = game.L2BlockNumber
		}
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

	var forecastStatus types.GameStatus
	// Games that have their block number challenged are won
	// by the challenger since the counter is proven on-chain.
	if game.BlockNumberChallenged {
		f.logger.Debug("Found game with challenged block number",
			"game", game.Proxy, "blockNum", game.L2BlockNumber, "agreement", agreement)
		// If the block number is challenged the challenger will always win
		forecastStatus = types.GameStatusChallengerWon
	} else {
		// Otherwise we go through the resolution process to determine who would win based on the current claims
		tree := transform.CreateBidirectionalTree(game.Claims)
		forecastStatus = Resolve(tree)
	}

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
