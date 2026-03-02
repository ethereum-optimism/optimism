package derive

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opderive "github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Deriver is an iterator that produces one payload attributes at a time from
// incrementally-added L1 blocks. It replaces the batch-mode PureDerive function
// with an API that matches how derivation works in practice: derive one block,
// execute on engine, verify, then derive the next.
type Deriver struct {
	cfg           *rollup.Config
	l1ChainConfig *params.ChainConfig
	lgr           log.Logger
	spec          *rollup.ChainSpec

	// L1 data — appended via AddL1Block
	l1Blocks   []L1Input
	l1Origins  []eth.L1BlockRef
	firstL1Num uint64

	// System config — evolves with config logs
	sysConfig eth.SystemConfig

	// L1 processing position — next L1 block index to scan for frames
	l1Pos int

	// Channel assembly
	assembler *channelAssembler

	// Batch buffer from completed channels
	pendingBatches      []*opderive.SingularBatch
	batchInclusionBlock eth.L1BlockRef

	// Derivation cursor
	cursor l2Cursor
}

// NewDeriver creates a new iterator-style deriver starting from the given safe
// head. The caller must then call AddL1Block to provide L1 data and Next to
// consume derived payload attributes.
func NewDeriver(
	cfg *rollup.Config,
	l1ChainConfig *params.ChainConfig,
	lgr log.Logger,
	safeHead eth.L2BlockRef,
	sysConfig eth.SystemConfig,
) (*Deriver, error) {
	if !cfg.IsKarst(safeHead.Time) {
		return nil, fmt.Errorf("derivation requires Karst fork (no overlapping span batches), safe head time %d is pre-Karst", safeHead.Time)
	}

	return &Deriver{
		cfg:           cfg,
		l1ChainConfig: l1ChainConfig,
		lgr:           lgr,
		spec:          rollup.NewChainSpec(cfg),
		sysConfig:     sysConfig,
		assembler:     newChannelAssembler(),
		cursor:        newCursor(safeHead),
	}, nil
}

// AddL1Block appends one or more L1 blocks. Blocks must be contiguous with
// previously added blocks. Returns ErrReorg if a block's parent hash doesn't
// match the tip of the already-added chain.
func (d *Deriver) AddL1Block(blocks ...L1Input) error {
	for i := range blocks {
		ref := blocks[i].BlockRef()

		if len(d.l1Origins) > 0 {
			tip := d.l1Origins[len(d.l1Origins)-1]
			if ref.ParentHash != tip.Hash {
				return fmt.Errorf("%w: block %d parent %s != tip %s", ErrReorg, ref.Number, ref.ParentHash, tip.Hash)
			}
		}

		if len(d.l1Blocks) == 0 {
			d.firstL1Num = ref.Number
		}

		d.l1Blocks = append(d.l1Blocks, blocks[i])
		d.l1Origins = append(d.l1Origins, ref)
	}
	return nil
}

// Next returns the next derived payload attributes and the L1 block they were
// derived from. safeHead provides the current L2 safe head (including Hash)
// for parent hash validation via upstream CheckBatch.
//
// Returns ErrNeedL1Data when more L1 blocks are needed.
func (d *Deriver) Next(safeHead eth.L2BlockRef) (*eth.PayloadAttributes, eth.L1BlockRef, error) {
	// Step 1: Try consuming from pending batches first.
	if attrs, l1Ref, err := d.tryPendingBatch(safeHead); err != nil {
		return nil, eth.L1BlockRef{}, err
	} else if attrs != nil {
		return attrs, l1Ref, nil
	}

	// Step 2: Process more L1 blocks to find new channels/batches.
	for d.l1Pos < len(d.l1Blocks) {
		if err := d.processNextL1Block(); err != nil {
			return nil, eth.L1BlockRef{}, err
		}

		// If we got new pending batches, try them.
		if attrs, l1Ref, err := d.tryPendingBatch(safeHead); err != nil {
			return nil, eth.L1BlockRef{}, err
		} else if attrs != nil {
			return attrs, l1Ref, nil
		}

		// After each L1 block, check if the seq window expired → empty batch.
		if attrs, l1Ref, err := d.tryEmptyBatch(safeHead); err != nil {
			return nil, eth.L1BlockRef{}, err
		} else if attrs != nil {
			return attrs, l1Ref, nil
		}
	}

	// Step 3: Nothing to do — need more L1 data.
	return nil, eth.L1BlockRef{}, ErrNeedL1Data
}

// Reset clears all internal state back to the given safe head + system config.
// Used after L1 reorgs. The caller must re-add L1 blocks from the new chain.
func (d *Deriver) Reset(safeHead eth.L2BlockRef, sysConfig eth.SystemConfig) {
	d.l1Blocks = nil
	d.l1Origins = nil
	d.firstL1Num = 0
	d.l1Pos = 0
	d.sysConfig = sysConfig
	d.assembler.reset()
	d.pendingBatches = nil
	d.batchInclusionBlock = eth.L1BlockRef{}
	d.cursor = newCursor(safeHead)
}

// findL1 does O(1) lookup by block number into the l1Blocks slice.
func (d *Deriver) findL1(number uint64) *L1Input {
	if len(d.l1Blocks) == 0 {
		return nil
	}
	idx := int(number - d.firstL1Num)
	if idx >= 0 && idx < len(d.l1Blocks) {
		return &d.l1Blocks[idx]
	}
	return nil
}

// tryPendingBatch validates the next pending batch via CheckBatch and builds
// attributes if accepted.
func (d *Deriver) tryPendingBatch(safeHead eth.L2BlockRef) (*eth.PayloadAttributes, eth.L1BlockRef, error) {
	for len(d.pendingBatches) > 0 {
		batch := d.pendingBatches[0]

		// Build l1Blocks slice for CheckBatch: must start at safeHead.L1Origin.
		startIdx := int(safeHead.L1Origin.Number - d.firstL1Num)
		if startIdx < 0 || startIdx >= len(d.l1Origins) {
			return nil, eth.L1BlockRef{}, ErrNeedL1Data
		}
		l1BlocksForCheck := d.l1Origins[startIdx:]

		batchWithInclusion := &opderive.BatchWithL1InclusionBlock{
			Batch:            batch,
			L1InclusionBlock: d.batchInclusionBlock,
		}

		validity := opderive.CheckBatch(
			context.Background(), d.cfg, d.lgr,
			l1BlocksForCheck, safeHead, batchWithInclusion, nil,
		)

		switch validity {
		case opderive.BatchAccept:
			d.pendingBatches = d.pendingBatches[1:]

			epochL1 := d.findL1(uint64(batch.EpochNum))
			if epochL1 == nil {
				return nil, eth.L1BlockRef{}, fmt.Errorf("missing L1 block %d for batch epoch", batch.EpochNum)
			}

			attrs, err := buildAttributes(batch, epochL1, d.cursor, d.sysConfig, d.cfg, d.l1ChainConfig)
			if err != nil {
				return nil, eth.L1BlockRef{}, fmt.Errorf("building attributes: %w", err)
			}

			epochID := eth.BlockID{Number: uint64(batch.EpochNum), Hash: batch.EpochHash}
			var seqNum uint64
			if epochID.Number != d.cursor.L1Origin.Number {
				seqNum = 0
			} else {
				seqNum = d.cursor.SequenceNumber + 1
			}
			d.cursor.advance(batch.Timestamp, epochID, seqNum)

			return attrs, d.batchInclusionBlock, nil

		case opderive.BatchPast:
			d.pendingBatches = d.pendingBatches[1:]
			continue

		case opderive.BatchUndecided:
			return nil, eth.L1BlockRef{}, ErrNeedL1Data

		default: // BatchDrop, BatchFuture, etc.
			d.lgr.Warn("invalid batch, discarding remaining channel batches",
				"timestamp", batch.Timestamp, "epoch", batch.EpochNum, "validity", validity)
			d.pendingBatches = nil
			return nil, eth.L1BlockRef{}, nil
		}
	}
	return nil, eth.L1BlockRef{}, nil
}

// processNextL1Block processes the L1 block at l1Pos: applies config logs,
// checks channel timeout, parses frames, and decodes any completed channels
// into pendingBatches.
func (d *Deriver) processNextL1Block() error {
	l1 := d.l1Blocks[d.l1Pos]
	l1Ref := d.l1Origins[d.l1Pos]
	d.l1Pos++

	for _, configLog := range l1.ConfigLogs {
		if err := opderive.ProcessSystemConfigUpdateLogEvent(&d.sysConfig, configLog, d.cfg, l1.Header.Time); err != nil {
			return fmt.Errorf("processing system config update at L1 block %d: %w", l1Ref.Number, err)
		}
	}

	d.assembler.checkTimeout(l1Ref, d.spec.ChannelTimeout(l1Ref.Time))

	for _, txData := range l1.BatcherData {
		frames, err := opderive.ParseFrames(txData)
		if err != nil {
			d.lgr.Warn("failed to parse frames", "l1_block", l1Ref.Number, "err", err)
			continue
		}

		for _, frame := range frames {
			ready := d.assembler.addFrame(frame, l1Ref)
			if ready == nil {
				continue
			}

			d.lgr.Debug("channel ready", "channel", ready.id, "l1_block", l1Ref.Number)

			batches := decodeBatches(d.lgr, ready.channel.Reader(), d.cfg, d.l1Origins, d.cursor, l1Ref)
			if len(batches) > 0 {
				d.pendingBatches = batches
				d.batchInclusionBlock = l1Ref
			}
		}
	}

	return nil
}

// tryEmptyBatch generates an empty batch if the sequencing window has expired
// for the most recently processed L1 block.
func (d *Deriver) tryEmptyBatch(safeHead eth.L2BlockRef) (*eth.PayloadAttributes, eth.L1BlockRef, error) {
	if d.l1Pos == 0 {
		return nil, eth.L1BlockRef{}, nil
	}
	currentL1 := d.l1Origins[d.l1Pos-1]

	if !d.cursor.needsEmptyBatch(currentL1, d.cfg) {
		return nil, eth.L1BlockRef{}, nil
	}

	batch, epochL1, newOrigin := makeEmptyBatch(d.cursor, d.findL1, d.cfg)
	if batch == nil {
		return nil, eth.L1BlockRef{}, nil
	}

	attrs, err := buildAttributes(batch, epochL1, d.cursor, d.sysConfig, d.cfg, d.l1ChainConfig)
	if err != nil {
		return nil, eth.L1BlockRef{}, fmt.Errorf("building empty batch attributes: %w", err)
	}

	var seqNum uint64
	if newOrigin.Number != d.cursor.L1Origin.Number {
		seqNum = 0
	} else {
		seqNum = d.cursor.SequenceNumber + 1
	}
	d.cursor.advance(batch.Timestamp, newOrigin, seqNum)

	return attrs, currentL1, nil
}
