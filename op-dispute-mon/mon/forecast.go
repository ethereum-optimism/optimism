package mon

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/transform"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type ForecastMetrics interface {
	RecordGameAgreements(counts map[metrics.GameAgreementStatus]int)
	RecordLatestValidProposalL2Block(validL2Block uint64)
	RecordLatestProposals(validTimestamp, invalidTimestamp uint64)
	RecordIgnoredGames(count int)
	RecordFailedGames(count int)
}

type forecastBatch struct {
	GameAgreements map[metrics.GameAgreementStatus]int

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

func (f *Forecast) Forecast(games []monTypes.EnrichedGame, ignoredCount, failedCount int) {
	batch := forecastBatch{GameAgreements: make(map[metrics.GameAgreementStatus]int)}
	for _, game := range games {
		f.forecastGame(game, &batch)
	}
	f.recordBatch(batch, ignoredCount, failedCount)
}

func (f *Forecast) recordBatch(batch forecastBatch, ignoredCount, failedCount int) {
	f.metrics.RecordGameAgreements(batch.GameAgreements)

	f.metrics.RecordLatestValidProposalL2Block(batch.LatestValidProposalL2Block)
	f.metrics.RecordLatestProposals(batch.LatestValidProposal, batch.LatestInvalidProposal)

	f.metrics.RecordIgnoredGames(ignoredCount)
	f.metrics.RecordFailedGames(failedCount)
}

func (f *Forecast) forecastGame(game monTypes.EnrichedGame, batch *forecastBatch) {
	gameData := game.Common()
	agreement := gameData.AgreeWithClaim
	expectedResult := types.GameStatusDefenderWon

	if !agreement {
		expectedResult = types.GameStatusChallengerWon
		if batch.LatestInvalidProposal < gameData.Timestamp {
			batch.LatestInvalidProposal = gameData.Timestamp
		}
	} else {
		if batch.LatestValidProposal < gameData.Timestamp {
			batch.LatestValidProposal = gameData.Timestamp
		}
		if gameData.UsesOutputRoots() && batch.LatestValidProposalL2Block < gameData.L2SequenceNumber {
			batch.LatestValidProposalL2Block = gameData.L2SequenceNumber
		}
	}

	actualResult := gameData.Status
	inProgress := actualResult == types.GameStatusInProgress
	if inProgress {
		actualResult = f.forecastInProgressGame(game)
	} else if actualResult != expectedResult {
		f.logger.Error("Unexpected game result",
			"game", gameData.Proxy, "l2SequenceNumber", gameData.L2SequenceNumber,
			"expectedResult", expectedResult, "actualResult", gameData.Status,
			"rootClaim", gameData.RootClaim, "correctClaim", gameData.ExpectedRootClaim)
	}

	status := agreementStatus(agreement, actualResult, inProgress)
	batch.GameAgreements[status]++

	if inProgress {
		if actualResult != expectedResult {
			f.logger.Warn("Forecasting unexpected game result", "status", actualResult,
				"game", gameData.Proxy, "l2SequenceNumber", gameData.L2SequenceNumber,
				"rootClaim", gameData.RootClaim, "expected", gameData.ExpectedRootClaim)
		} else {
			f.logger.Debug("Forecasting expected game result", "status", actualResult,
				"game", gameData.Proxy, "l2SequenceNumber", gameData.L2SequenceNumber,
				"rootClaim", gameData.RootClaim, "expected", gameData.ExpectedRootClaim)
		}
	}
}

func (f *Forecast) forecastInProgressGame(game monTypes.EnrichedGame) types.GameStatus {
	gameData := game.Common()
	switch game := game.(type) {
	case *monTypes.SuperPermissionedGameData:
		// Unreachable since super permissioned games resolve immediately, unless the game was misconfigured!
		f.logger.Error("Found super permissioned game still in progress, this should be impossible, check game configuration", "game", gameData.Proxy)
		// Since we don't know how an in-progress super permissioned game would resolve, assume the
		// opposite of the expected root result so existing monitoring alerts.
		return pessimisticForecast(gameData)
	case *monTypes.FaultGameData:
		if game.BlockNumberChallenged {
			// Games that have their block number challenged are won
			// by the challenger since the counter is proven on-chain.
			f.logger.Debug("Found game with challenged block number",
				"game", gameData.Proxy, "l2SequenceNumber", gameData.L2SequenceNumber, "agreement", gameData.AgreeWithClaim)
			return types.GameStatusChallengerWon
		}
		// Otherwise we go through the resolution process to determine who would win based on the current claims
		tree := transform.CreateBidirectionalTree(game.Claims)
		return Resolve(tree)
	case *monTypes.ZKGameData:
		if game.ParentStatus != nil && *game.ParentStatus == types.GameStatusChallengerWon {
			return types.GameStatusChallengerWon
		}
		switch game.ProposalStatus {
		case contracts.ProposalStatusUnchallenged,
			contracts.ProposalStatusUnchallengedAndValidProofProvided,
			contracts.ProposalStatusChallengedAndValidProofProvided:
			return types.GameStatusDefenderWon
		case contracts.ProposalStatusChallenged:
			return types.GameStatusChallengerWon
		default:
			f.logger.Error("Unable to forecast ZK game with incompatible proposal and global status",
				"game", gameData.Proxy, "proposalStatus", game.ProposalStatus, "globalStatus", gameData.Status)
			return pessimisticForecast(gameData)
		}
	default:
		f.logger.Error("Unable to forecast unknown enriched game variant", "game", gameData.Proxy, "type", gameData.GameType)
		return pessimisticForecast(gameData)
	}
}

func pessimisticForecast(game *monTypes.CommonGameData) types.GameStatus {
	if game.AgreeWithClaim {
		return types.GameStatusChallengerWon
	}
	return types.GameStatusDefenderWon
}

func agreementStatus(agreement bool, result types.GameStatus, inProgress bool) metrics.GameAgreementStatus {
	if inProgress {
		if agreement {
			if result == types.GameStatusChallengerWon {
				return metrics.AgreeChallengerAhead
			}
			return metrics.AgreeDefenderAhead
		}
		if result == types.GameStatusChallengerWon {
			return metrics.DisagreeChallengerAhead
		}
		return metrics.DisagreeDefenderAhead
	}
	if agreement {
		if result == types.GameStatusChallengerWon {
			return metrics.AgreeChallengerWins
		}
		return metrics.AgreeDefenderWins
	}
	if result == types.GameStatusChallengerWon {
		return metrics.DisagreeChallengerWins
	}
	return metrics.DisagreeDefenderWins
}
