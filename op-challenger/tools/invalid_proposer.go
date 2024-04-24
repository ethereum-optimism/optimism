package tools

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
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
	interval     time.Duration
	txMgr        txmgr.TxManager
	selectReason func() InvalidityReason

	cancelFunc context.CancelFunc
	stopped    atomic.Bool
}

func NewInvalidProposer(logger log.Logger, gameCreator *GameCreator, client RollupClient, traceType uint64, interval time.Duration, txMgr txmgr.TxManager) *InvalidProposer {
	return &InvalidProposer{
		log:         logger,
		gameCreator: gameCreator,
		client:      client,
		traceType:   traceType,
		interval:    interval,
		txMgr:       txMgr,
		selectReason: func() InvalidityReason {
			// Select a random invalidity reason
			return InvalidityReason(rand.Intn(int(invalidityReasonCount)))
		},
	}
}

func (p *InvalidProposer) Start(ctx context.Context) error {
	p.log.Info("Starting invalid proposer")
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	p.cancelFunc = cancelFunc
	go p.loop(cancelCtx)
	return nil
}

func (p *InvalidProposer) Stop(_ context.Context) error {
	p.log.Info("Stopping invalid proposer")
	p.txMgr.Close()
	p.cancelFunc()
	p.stopped.Store(true)
	return nil
}

func (p *InvalidProposer) Stopped() bool {
	return p.stopped.Load()
}

func (p *InvalidProposer) loop(ctx context.Context) {
	// Propose immediately at startup
	if err := p.propose(ctx); err != nil {
		p.log.Error("Failed to propose invalid output", "err", err)
	}

	// Then wait for the next instance
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			p.log.Info("Exiting invalid proposer loop")
			return
		case <-ticker.C:
			if err := p.propose(ctx); err != nil {
				p.log.Error("Failed to propose invalid output", "err", err)
			}
		}
	}
}

func (p *InvalidProposer) propose(ctx context.Context) error {
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
