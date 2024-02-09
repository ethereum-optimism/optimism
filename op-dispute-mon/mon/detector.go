package mon

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type statusBatch struct {
	inProgress, defenderWon, challengerWon int
}

func (s *statusBatch) Add(status types.GameStatus) {
	switch status {
	case types.GameStatusInProgress:
		s.inProgress++
	case types.GameStatusDefenderWon:
		s.defenderWon++
	case types.GameStatusChallengerWon:
		s.challengerWon++
	}
}

type detectionBatch struct {
	inProgress             int
	agreeDefenderWins      int
	disagreeDefenderWins   int
	agreeChallengerWins    int
	disagreeChallengerWins int
}

func (d *detectionBatch) merge(other detectionBatch) {
	d.inProgress += other.inProgress
	d.agreeDefenderWins += other.agreeDefenderWins
	d.disagreeDefenderWins += other.disagreeDefenderWins
	d.agreeChallengerWins += other.agreeChallengerWins
	d.disagreeChallengerWins += other.disagreeChallengerWins
}

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type MetadataCreator interface {
	CreateContract(game types.GameMetadata) (MetadataLoader, error)
}

type DetectorMetrics interface {
	RecordGameAgreement(status string, count int)
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
}

type detector struct {
	logger       log.Logger
	metrics      DetectorMetrics
	creator      MetadataCreator
	outputClient OutputRollupClient
}

func newDetector(logger log.Logger, metrics DetectorMetrics, creator MetadataCreator, outputClient OutputRollupClient) *detector {
	return &detector{
		logger:       logger,
		metrics:      metrics,
		creator:      creator,
		outputClient: outputClient,
	}
}

func (d *detector) Detect(ctx context.Context, games []types.GameMetadata) {
	statBatch := statusBatch{}
	detectBatch := detectionBatch{}
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
		detectBatch.merge(processed)
	}
	d.metrics.RecordGamesStatus(statBatch.inProgress, statBatch.defenderWon, statBatch.challengerWon)
	d.recordBatch(detectBatch)
	d.logger.Info("Completed updating games", "count", len(games))
}

func (d *detector) recordBatch(batch detectionBatch) {
	d.metrics.RecordGameAgreement("in_progress", batch.inProgress)
	d.metrics.RecordGameAgreement("agree_defender_wins", batch.agreeDefenderWins)
	d.metrics.RecordGameAgreement("disagree_defender_wins", batch.disagreeDefenderWins)
	d.metrics.RecordGameAgreement("agree_challenger_wins", batch.agreeChallengerWins)
	d.metrics.RecordGameAgreement("disagree_challenger_wins", batch.disagreeChallengerWins)
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

func (d *detector) checkAgreement(ctx context.Context, addr common.Address, blockNum uint64, rootClaim common.Hash, status types.GameStatus) (detectionBatch, error) {
	agree, err := d.checkRootAgreement(ctx, blockNum, rootClaim)
	if err != nil {
		return detectionBatch{}, err
	}
	batch := detectionBatch{}
	switch status {
	case types.GameStatusInProgress:
		batch.inProgress++
	case types.GameStatusDefenderWon:
		if agree {
			batch.agreeDefenderWins++
		} else {
			batch.disagreeDefenderWins++
			d.logger.Error("Defender won but root claim does not match", "gameAddr", addr, "rootClaim", rootClaim)
		}
	case types.GameStatusChallengerWon:
		if agree {
			batch.agreeChallengerWins++
		} else {
			batch.disagreeChallengerWins++
			d.logger.Error("Challenger won but root claim does not match", "gameAddr", addr, "rootClaim", rootClaim)
		}
	}
	return batch, nil
}

func (d *detector) checkRootAgreement(ctx context.Context, blockNum uint64, rootClaim common.Hash) (bool, error) {
	output, err := d.outputClient.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return false, fmt.Errorf("failed to get output at block: %w", err)
	}
	return rootClaim == common.Hash(output.OutputRoot), nil
}
