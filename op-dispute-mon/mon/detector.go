package mon

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/extract"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type OutputValidator interface {
	CheckRootAgreement(ctx context.Context, blockNum uint64, root common.Hash) (bool, common.Hash, error)
}

type GameCallerCreator interface {
	CreateContract(game types.GameMetadata) (extract.GameCaller, error)
}

type DetectorMetrics interface {
	RecordGameAgreement(status metrics.GameAgreementStatus, count int)
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type detector struct {
	logger    log.Logger
	metrics   DetectorMetrics
	validator OutputValidator
}

func newDetector(logger log.Logger, metrics DetectorMetrics, validator OutputValidator) *detector {
	return &detector{
		logger:    logger,
		metrics:   metrics,
		validator: validator,
	}
}

func (d *detector) Detect(ctx context.Context, games []*monTypes.EnrichedGameData) {
	statBatch := monTypes.StatusBatch{}
	detectBatch := monTypes.DetectionBatch{}
	for _, game := range games {
		statBatch.Add(game.Status)
		processed, err := d.checkAgreement(ctx, game.Proxy, game.L2BlockNumber, game.RootClaim, game.Status)
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
	d.metrics.RecordGameAgreement(metrics.AgreeDefenderWins, batch.AgreeDefenderWins)
	d.metrics.RecordGameAgreement(metrics.DisagreeDefenderWins, batch.DisagreeDefenderWins)
	d.metrics.RecordGameAgreement(metrics.AgreeChallengerWins, batch.AgreeChallengerWins)
	d.metrics.RecordGameAgreement(metrics.DisagreeChallengerWins, batch.DisagreeChallengerWins)
}

func (d *detector) checkAgreement(ctx context.Context, addr common.Address, blockNum uint64, rootClaim common.Hash, status types.GameStatus) (monTypes.DetectionBatch, error) {
	agree, expectedClaim, err := d.validator.CheckRootAgreement(ctx, blockNum, rootClaim)
	if err != nil {
		return monTypes.DetectionBatch{}, err
	}
	batch := monTypes.DetectionBatch{}
	batch.Update(status, agree)
	if status != types.GameStatusInProgress {
		expectedResult := types.GameStatusDefenderWon
		if !agree {
			expectedResult = types.GameStatusChallengerWon
		}
		if status != expectedResult {
			d.logger.Error("Unexpected game result",
				"gameAddr", addr, "blockNum", blockNum,
				"expectedResult", expectedResult, "actualResult", status,
				"rootClaim", rootClaim, "correctClaim", expectedClaim)
		}
	}
	return batch, nil
}
