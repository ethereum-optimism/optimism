package mon

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

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

type MetadataLoader interface {
	GetGameMetadata(context.Context, common.Address) (uint64, common.Hash, types.GameStatus, error)
}

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type detector struct {
	logger       log.Logger
	metrics      MonitorMetricer
	loader       MetadataLoader
	outputClient OutputRollupClient
}

func newDetector(logger log.Logger, metrics MonitorMetricer, loader MetadataLoader, outputClient OutputRollupClient) *detector {
	return &detector{
		logger:       logger,
		metrics:      metrics,
		loader:       loader,
		outputClient: outputClient,
	}
}

func (d *detector) Detect(ctx context.Context, games []types.GameMetadata) {
	batch := detectionBatch{}
	for _, game := range games {
		processed, err := d.processGame(ctx, game)
		if err != nil {
			d.logger.Error("Failed to process game", "err", err)
			continue
		}
		batch.merge(processed)
	}
	d.recordBatch(batch)
}

func (d *detector) recordBatch(batch detectionBatch) {
	d.metrics.RecordGameAgreement("in_progress", batch.inProgress)
	d.metrics.RecordGameAgreement("agree_defender_wins", batch.agreeDefenderWins)
	d.metrics.RecordGameAgreement("disagree_defender_wins", batch.disagreeDefenderWins)
	d.metrics.RecordGameAgreement("agree_challenger_wins", batch.agreeChallengerWins)
	d.metrics.RecordGameAgreement("disagree_challenger_wins", batch.disagreeChallengerWins)
}

func (d *detector) recordGameStatus(ctx context.Context, status types.GameStatus) {
	switch status {
	case types.GameStatusInProgress:
		d.metrics.RecordGamesStatus(1, 0, 0)
	case types.GameStatusDefenderWon:
		d.metrics.RecordGamesStatus(0, 1, 0)
	case types.GameStatusChallengerWon:
		d.metrics.RecordGamesStatus(0, 0, 1)
	}
}

func (d *detector) processGame(ctx context.Context, game types.GameMetadata) (detectionBatch, error) {
	blockNum, rootClaim, status, err := d.loader.GetGameMetadata(ctx, game.Proxy)
	if err != nil {
		return detectionBatch{}, err
	}
	agree, err := d.checkRootAgreement(ctx, blockNum, rootClaim)
	if err != nil {
		return detectionBatch{}, err
	}
	d.recordGameStatus(ctx, status)
	batch := detectionBatch{}
	switch status {
	case types.GameStatusInProgress:
		batch.inProgress++
	case types.GameStatusDefenderWon:
		if agree {
			batch.agreeDefenderWins++
		} else {
			batch.disagreeDefenderWins++
			d.logger.Error("Defender won but root claim does not match", "game", game, "rootClaim", rootClaim)
		}
	case types.GameStatusChallengerWon:
		if agree {
			batch.agreeChallengerWins++
		} else {
			batch.disagreeChallengerWins++
			d.logger.Error("Challenger won but root claim does not match", "game", game, "rootClaim", rootClaim)
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
