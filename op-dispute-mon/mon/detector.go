package mon

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"

	"github.com/ethereum/go-ethereum/log"
)

type DetectorMetrics interface {
	RecordGameAgreement(status string, count int)
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type detector struct {
	logger  log.Logger
	metrics DetectorMetrics
}

func newDetector(logger log.Logger, metrics DetectorMetrics) *detector {
	return &detector{
		logger:  logger,
		metrics: metrics,
	}
}

func (d *detector) Detect(ctx context.Context, games []monTypes.EnrichedGameData) {
	statBatch := monTypes.StatusBatch{}
	detectBatch := monTypes.DetectionBatch{}
	for _, game := range games {
		statBatch.Add(game.Status)
		processed, err := d.checkAgreement(ctx, game)
		if err != nil {
			d.logger.Error("Failed to process game", "err", err)
			continue
		}
		detectBatch.Merge(processed)
	}
	d.metrics.RecordGamesStatus(statBatch.InProgress, statBatch.DefenderWon, statBatch.ChallengerWon)
	d.recordBatch(detectBatch)
	d.logger.Info("Completed updating games", "count", len(games))
}

func (d *detector) recordBatch(batch monTypes.DetectionBatch) {
	d.metrics.RecordGameAgreement("in_progress", batch.InProgress)
	d.metrics.RecordGameAgreement("agree_defender_wins", batch.AgreeDefenderWins)
	d.metrics.RecordGameAgreement("disagree_defender_wins", batch.DisagreeDefenderWins)
	d.metrics.RecordGameAgreement("agree_challenger_wins", batch.AgreeChallengerWins)
	d.metrics.RecordGameAgreement("disagree_challenger_wins", batch.DisagreeChallengerWins)
}

func (d *detector) checkAgreement(ctx context.Context, game monTypes.EnrichedGameData) (monTypes.DetectionBatch, error) {
	agree := game.RootClaim == game.ExpectedRoot
	batch := monTypes.DetectionBatch{}
	batch.Update(game.Status, agree)
	if game.Status != types.GameStatusInProgress {
		expectedResult := types.GameStatusDefenderWon
		if !agree {
			expectedResult = types.GameStatusChallengerWon
		}
		if game.Status != expectedResult {
			d.logger.Error("Unexpected game result", "gameAddr", game.Proxy, "expectedResult", expectedResult, "actualResult", game.Status, "rootClaim", game.RootClaim, "correctClaim", game.ExpectedRoot)
		}
	}
	return batch, nil
}
