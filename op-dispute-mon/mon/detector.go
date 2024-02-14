package mon

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type OutputValidator interface {
	CheckRootAgreement(ctx context.Context, blockNum uint64, root common.Hash) (bool, common.Hash, error)
}

type GameCallerCreator interface {
	CreateContract(game types.GameMetadata) (GameCaller, error)
}

type DetectorMetrics interface {
	RecordGameAgreement(status string, count int)
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type detector struct {
	logger    log.Logger
	metrics   DetectorMetrics
	creator   GameCallerCreator
	validator OutputValidator
}

func newDetector(logger log.Logger, metrics DetectorMetrics, creator GameCallerCreator, validator OutputValidator) *detector {
	return &detector{
		logger:    logger,
		metrics:   metrics,
		creator:   creator,
		validator: validator,
	}
}

func (d *detector) Detect(ctx context.Context, games []types.GameMetadata) {
	statBatch := monTypes.StatusBatch{}
	detectBatch := monTypes.DetectionBatch{}
	for _, game := range games {
		// Fetch the game metadata to ensure the game status is recorded
		// regardless of whether the game agreement is checked.
		l2BlockNum, rootClaim, status, err := d.fetchGameMetadata(ctx, game)
		if err != nil {
			d.logger.Error("Failed to fetch game metadata", "err", err)
			continue
		}
		statBatch.Add(status)
		processed, err := d.checkAgreement(ctx, game.Proxy, l2BlockNum, rootClaim, status)
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

func (d *detector) fetchGameMetadata(ctx context.Context, game types.GameMetadata) (uint64, common.Hash, types.GameStatus, error) {
	loader, err := d.creator.CreateContract(game)
	if err != nil {
		return 0, common.Hash{}, 0, fmt.Errorf("failed to create contract: %w", err)
	}
	blockNum, rootClaim, status, err := loader.GetGameMetadata(ctx)
	if err != nil {
		return 0, common.Hash{}, 0, fmt.Errorf("failed to fetch game metadata: %w", err)
	}
	return blockNum, rootClaim, status, nil
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
			d.logger.Error("Unexpected game result", "gameAddr", addr, "expectedResult", expectedResult, "actualResult", status, "rootClaim", rootClaim, "correctClaim", expectedClaim)
		}
	}
	return batch, nil
}
