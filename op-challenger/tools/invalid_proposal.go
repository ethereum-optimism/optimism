package tools

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type RollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

type InvalidityReason int

const (
	UnsafeBlock InvalidityReason = iota
	IncorrectOutputRoot
	LowerBlockNumber
	HigherBlockNumber

	// invalidityReasonCount must always be last and is used as the marker of how many reasons to select from
	invalidityReasonCount
)

type InvalidProposer struct {
	log          log.Logger
	gameCreator  *GameCreator
	client       RollupClient
	traceType    uint64
	selectReason func() InvalidityReason
}

func NewInvalidProposer(logger log.Logger, gameCreator *GameCreator, client RollupClient, traceType uint64) *InvalidProposer {
	return &InvalidProposer{
		log:         logger,
		gameCreator: gameCreator,
		client:      client,
		traceType:   traceType,
		selectReason: func() InvalidityReason {
			// Select a random invalidity reason
			return InvalidityReason(rand.Intn(int(invalidityReasonCount)))
		},
	}
}

func (p *InvalidProposer) Propose(ctx context.Context) error {
	status, err := p.client.SyncStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to load sync status: %w", err)
	}
	reason := p.selectReason()
	outputRoot, l2BlockNum, err := p.proposalData(ctx, reason, status)
	if err != nil {
		return err
	}

	game, err := p.gameCreator.CreateGame(ctx, outputRoot, p.traceType, l2BlockNum)
	if err != nil {
		return fmt.Errorf("failed to propose: %w", err)
	} else {
		p.log.Info("Proposed invalid output root", "output", outputRoot, "block", l2BlockNum, "type", p.traceType, "game", game)
	}
	return nil
}

func (p *InvalidProposer) proposalData(ctx context.Context, reason InvalidityReason, status *eth.SyncStatus) (common.Hash, uint64, error) {
	var outputRoot common.Hash
	var l2BlockNum uint64
	var err error

	switch reason {
	case UnsafeBlock:
		log.Info("Proposing with unsafe block")
		outputRoot, err = p.outputAtBlock(ctx, status.UnsafeL2.Number)
		if err != nil {
			return common.Hash{}, 0, err
		}
		l2BlockNum = status.UnsafeL2.Number
	case IncorrectOutputRoot:
		log.Info("Proposing with incorrect output root")
		outputRoot = common.Hash{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		l2BlockNum = status.SafeL2.Number
	case LowerBlockNumber:
		log.Info("Proposing with lower block number")
		outputRoot, err = p.outputAtBlock(ctx, status.SafeL2.Number)
		if err != nil {
			return common.Hash{}, 0, err
		}
		l2BlockNum = status.SafeL2.Number - 1
	case HigherBlockNumber:
		log.Info("Proposing with higher block number")
		outputRoot, err = p.outputAtBlock(ctx, status.SafeL2.Number)
		if err != nil {
			return common.Hash{}, 0, err
		}
		l2BlockNum = status.SafeL2.Number + 1
	default:
		return common.Hash{}, 0, fmt.Errorf("invalid proposer selected: %v", reason)
	}
	return outputRoot, l2BlockNum, nil
}

func (p *InvalidProposer) outputAtBlock(ctx context.Context, l2BlockNum uint64) (common.Hash, error) {
	outputResponse, err := p.client.OutputAtBlock(ctx, l2BlockNum)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to retrieve correct output at block %v: %w", l2BlockNum, err)
	}
	return common.Hash(outputResponse.OutputRoot), nil
}
