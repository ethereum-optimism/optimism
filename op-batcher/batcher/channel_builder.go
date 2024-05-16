package batcher

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrInvalidChannelTimeout = errors.New("channel timeout is less than the safety margin")
	ErrMaxFrameIndex         = errors.New("max frame index reached (uint16)")
	ErrMaxDurationReached    = errors.New("max channel duration reached")
	ErrChannelTimeoutClose   = errors.New("close to channel timeout")
	ErrSeqWindowClose        = errors.New("close to sequencer window timeout")
	ErrTerminated            = errors.New("channel terminated")
)

type ChannelFullError struct {
	Err error
}

func (e *ChannelFullError) Error() string {
	return "channel full: " + e.Err.Error()
}

func (e *ChannelFullError) Unwrap() error {
	return e.Err
}

type frameID struct {
	chID        derive.ChannelID
	frameNumber uint16
}

type frameData struct {
	data []byte
	id   frameID
}

// ChannelBuilder uses a ChannelOut to create a channel with output frame
// size approximation.
type ChannelBuilder struct {
	cfg       ChannelConfig
	rollupCfg rollup.Config

	// L1 block number timeout of combined
	// - channel duration timeout,
	// - consensus channel timeout,
	// - sequencing window timeout.
	// 0 if no block number timeout set yet.
	timeout uint64
	// reason for currently set timeout
	timeoutReason error

	// Reason for the channel being full. Set by setFullErr so it's always
	// guaranteed to be a ChannelFullError wrapping the specific reason.
	fullErr error
	// current channel
	co derive.ChannelOut
	// list of blocks in the channel. Saved in case the channel must be rebuilt
	blocks []*types.Block
	// latestL1Origin is the latest L1 origin of all the L2 blocks that have been added to the channel
	latestL1Origin eth.BlockID
	// frames data queue, to be send as txs
	frames []frameData
	// total frames counter
	numFrames int
	// total amount of output data of all frames created yet
	outputBytes int
}

// newChannelBuilder creates a new channel builder or returns an error if the
// channel out could not be created.
// it acts as a factory for either a span or singular channel out
func NewChannelBuilder(cfg ChannelConfig, rollupCfg rollup.Config, latestL1OriginBlockNum uint64) (*ChannelBuilder, error) {
	c, err := cfg.CompressorConfig.NewCompressor()
	if err != nil {
		return nil, err
	}
	var co derive.ChannelOut
	if cfg.BatchType == derive.SpanBatchType {
		co, err = derive.NewSpanChannelOut(rollupCfg.Genesis.L2Time, rollupCfg.L2ChainID, cfg.CompressorConfig.TargetOutputSize, cfg.CompressorConfig.CompressionAlgo)
	} else {
		co, err = derive.NewSingularChannelOut(c)
	}
	if err != nil {
		return nil, fmt.Errorf("creating channel out: %w", err)
	}

	cb := &ChannelBuilder{
		cfg:       cfg,
		rollupCfg: rollupCfg,
		co:        co,
	}

	cb.updateDurationTimeout(latestL1OriginBlockNum)

	return cb, nil
}

func (c *ChannelBuilder) ID() derive.ChannelID {
	return c.co.ID()
}

// InputBytes returns the total amount of input bytes added to the channel.
func (c *ChannelBuilder) InputBytes() int {
	return c.co.InputBytes()
}

// ReadyBytes returns the amount of bytes ready in the compression pipeline to
// output into a frame.
func (c *ChannelBuilder) ReadyBytes() int {
	return c.co.ReadyBytes()
}

func (c *ChannelBuilder) OutputBytes() int {
	return c.outputBytes
}

// Blocks returns a backup list of all blocks that were added to the channel. It
// can be used in case the channel needs to be rebuilt.
func (c *ChannelBuilder) Blocks() []*types.Block {
	return c.blocks
}

// LatestL1Origin returns the latest L1 block origin from all the L2 blocks that have been added to the channel
func (c *ChannelBuilder) LatestL1Origin() eth.BlockID {
	return c.latestL1Origin
}

// AddBlock adds a block to the channel compression pipeline. IsFull should be
// called afterwards to test whether the channel is full. If full, a new channel
// must be started.
//
// AddBlock returns a ChannelFullError if called even though the channel is
// already full. See description of FullErr for details.
//
// AddBlock also returns the L1BlockInfo that got extracted from the block's
// first transaction for subsequent use by the caller.
//
// Call OutputFrames() afterwards to create frames.
func (c *ChannelBuilder) AddBlock(block *types.Block) (*derive.L1BlockInfo, error) {
	if c.IsFull() {
		return nil, c.FullErr()
	}

	batch, l1info, err := derive.BlockToSingularBatch(&c.rollupCfg, block)
	if err != nil {
		return l1info, fmt.Errorf("converting block to batch: %w", err)
	}

	if err = c.co.AddSingularBatch(batch, l1info.SequenceNumber); errors.Is(err, derive.ErrTooManyRLPBytes) || errors.Is(err, derive.ErrCompressorFull) {
		c.setFullErr(err)
		return l1info, c.FullErr()
	} else if err != nil {
		return l1info, fmt.Errorf("adding block to channel out: %w", err)
	}

	c.blocks = append(c.blocks, block)
	c.updateSwTimeout(batch)

	if l1info.Number > c.latestL1Origin.Number {
		c.latestL1Origin = eth.BlockID{
			Hash:   l1info.BlockHash,
			Number: l1info.Number,
		}
	}

	if err = c.co.FullErr(); err != nil {
		c.setFullErr(err)
		// Adding this block still worked, so don't return error, just mark as full
	}

	return l1info, nil
}

// Timeout management

// Timeout returns the block number of the channel timeout. If no timeout is set it returns 0
func (c *ChannelBuilder) Timeout() uint64 {
	return c.timeout
}

// FramePublished should be called whenever a frame of this channel got
// published with the L1-block number of the block that the frame got included
// in.
func (c *ChannelBuilder) FramePublished(l1BlockNum uint64) {
	timeout := l1BlockNum + c.cfg.ChannelTimeout - c.cfg.SubSafetyMargin
	c.updateTimeout(timeout, ErrChannelTimeoutClose)
}

// updateDurationTimeout updates the block timeout with the channel duration
// timeout derived from the given L1-block number. The timeout is only moved
// forward if the derived timeout is earlier than the currently set timeout.
//
// It does nothing if the max channel duration is set to 0.
func (c *ChannelBuilder) updateDurationTimeout(l1BlockNum uint64) {
	if c.cfg.MaxChannelDuration == 0 {
		return
	}
	timeout := l1BlockNum + c.cfg.MaxChannelDuration
	c.updateTimeout(timeout, ErrMaxDurationReached)
}

// updateSwTimeout updates the block timeout with the sequencer window timeout
// derived from the batch's origin L1 block. The timeout is only moved forward
// if the derived sequencer window timeout is earlier than the currently set
// timeout.
func (c *ChannelBuilder) updateSwTimeout(batch *derive.SingularBatch) {
	timeout := uint64(batch.EpochNum) + c.cfg.SeqWindowSize - c.cfg.SubSafetyMargin
	c.updateTimeout(timeout, ErrSeqWindowClose)
}

// updateTimeout updates the timeout block to the given block number if it is
// earlier than the current block timeout, or if it still unset.
//
// If the timeout is updated, the provided reason will be set as the channel
// full error reason in case the timeout is hit in the future.
func (c *ChannelBuilder) updateTimeout(timeoutBlockNum uint64, reason error) {
	if c.timeout == 0 || c.timeout > timeoutBlockNum {
		c.timeout = timeoutBlockNum
		c.timeoutReason = reason
	}
}

// CheckTimeout checks if the channel is timed out at the given block number and
// in this case marks the channel as full, if it wasn't full already.
func (c *ChannelBuilder) CheckTimeout(l1BlockNum uint64) {
	if !c.IsFull() && c.TimedOut(l1BlockNum) {
		c.setFullErr(c.timeoutReason)
	}
}

// TimedOut returns whether the passed block number is after the timeout block
// number. If no block timeout is set yet, it returns false.
func (c *ChannelBuilder) TimedOut(blockNum uint64) bool {
	return c.timeout != 0 && blockNum >= c.timeout
}

// IsFull returns whether the channel is full.
// FullErr returns the reason for the channel being full.
func (c *ChannelBuilder) IsFull() bool {
	return c.fullErr != nil
}

// FullErr returns the reason why the channel is full. If not full yet, it
// returns nil.
//
// It returns a ChannelFullError wrapping one of the following possible reasons
// for the channel being full:
//   - derive.ErrCompressorFull if the compressor target has been reached,
//   - derive.MaxRLPBytesPerChannel if the general maximum amount of input data
//     would have been exceeded by the latest AddBlock call,
//   - ErrMaxFrameIndex if the maximum number of frames has been generated
//     (uint16),
//   - ErrMaxDurationReached if the max channel duration got reached,
//   - ErrChannelTimeoutClose if the consensus channel timeout got too close,
//   - ErrSeqWindowClose if the end of the sequencer window got too close,
//   - ErrTerminated if the channel was explicitly terminated.
func (c *ChannelBuilder) FullErr() error {
	return c.fullErr
}

func (c *ChannelBuilder) setFullErr(err error) {
	c.fullErr = &ChannelFullError{Err: err}
}

// OutputFrames creates new frames with the channel out. It should be called
// after AddBlock and before iterating over available frames with HasFrame and
// NextFrame.
//
// If the channel isn't full yet, it will conservatively only
// pull readily available frames from the compression output.
// If it is full, the channel is closed and all remaining
// frames will be created, possibly with a small leftover frame.
func (c *ChannelBuilder) OutputFrames() error {
	if c.IsFull() {
		err := c.closeAndOutputAllFrames()
		if err != nil {
			return fmt.Errorf("error while closing full channel (full reason: %w): %w", c.FullErr(), err)
		}
		return nil
	}
	return c.outputReadyFrames()
}

// outputReadyFrames creates new frames as long as there's enough data ready in
// the channel out compression pipeline.
//
// This is part of an optimization to already generate frames and send them off
// as txs while still collecting blocks in the channel builder.
func (c *ChannelBuilder) outputReadyFrames() error {
	// When creating a frame from the ready compression data, the frame overhead
	// will be added to the total output size, so we can add it in the condition.
	for c.co.ReadyBytes()+derive.FrameV0OverHeadSize >= int(c.cfg.MaxFrameSize) {
		if err := c.outputFrame(); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (c *ChannelBuilder) closeAndOutputAllFrames() error {
	if err := c.co.Close(); err != nil {
		return fmt.Errorf("closing channel out: %w", err)
	}

	for {
		if err := c.outputFrame(); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// outputFrame creates one new frame and adds it to the frames queue.
// Note that compressed output data must be available on the underlying
// ChannelOut, or an empty frame will be produced.
func (c *ChannelBuilder) outputFrame() error {
	var buf bytes.Buffer
	fn, err := c.co.OutputFrame(&buf, c.cfg.MaxFrameSize)
	if err != io.EOF && err != nil {
		return fmt.Errorf("writing frame[%d]: %w", fn, err)
	}

	// Mark as full if max index reached
	// TODO: If there's still data in the compression pipeline of the channel out,
	// we would miss it and the whole channel would be broken because the last
	// frames would never be generated...
	// Hitting the max index is impossible with current parameters, so ignore for
	// now. Note that in order to properly catch this, we'd need to call Flush
	// after every block addition to estimate how many more frames are coming.
	if fn == math.MaxUint16 {
		c.setFullErr(ErrMaxFrameIndex)
	}

	frame := frameData{
		id:   frameID{chID: c.co.ID(), frameNumber: fn},
		data: buf.Bytes(),
	}
	c.frames = append(c.frames, frame)
	c.numFrames++
	c.outputBytes += len(frame.data)
	return err // possibly io.EOF (last frame)
}

// Close immediately marks the channel as full with an ErrTerminated
// if the channel is not already full.
func (c *ChannelBuilder) Close() {
	if !c.IsFull() {
		c.setFullErr(ErrTerminated)
	}
}

// TotalFrames returns the total number of frames that were created in this channel so far.
// It does not decrease when the frames queue is being emptied.
func (c *ChannelBuilder) TotalFrames() int {
	return c.numFrames
}

// HasFrame returns whether there's any available frame. If true, it can be
// popped using NextFrame().
//
// Call OutputFrames before to create new frames from the channel out
// compression pipeline.
func (c *ChannelBuilder) HasFrame() bool {
	return len(c.frames) > 0
}

// PendingFrames returns the number of pending frames in the frames queue.
// It is larger zero iff HasFrames() returns true.
func (c *ChannelBuilder) PendingFrames() int {
	return len(c.frames)
}

// NextFrame returns the next available frame.
// HasFrame must be called prior to check if there's a next frame available.
// Panics if called when there's no next frame.
func (c *ChannelBuilder) NextFrame() frameData {
	if len(c.frames) == 0 {
		panic("no next frame")
	}

	f := c.frames[0]
	c.frames = c.frames[1:]
	return f
}

// PushFrames adds the frames back to the internal frames queue. Panics if not of
// the same channel.
func (c *ChannelBuilder) PushFrames(frames ...frameData) {
	for _, f := range frames {
		if f.id.chID != c.ID() {
			panic("wrong channel")
		}
		c.frames = append(c.frames, f)
	}
}
