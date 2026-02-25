package pure

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// readyChannel is a completed channel ready for batch decoding.
type readyChannel struct {
	id        derive.ChannelID
	openBlock eth.L1BlockRef
	channel   *derive.Channel
}

// channelAssembler implements Holocene single-channel strict-order assembly.
// Only one channel is active at a time. Frames must arrive in order.
// A frame for a new channel ID discards the current in-progress channel.
type channelAssembler struct {
	current   *derive.Channel
	currentID derive.ChannelID
	openBlock eth.L1BlockRef
	nextFrame uint16
}

func newChannelAssembler() *channelAssembler {
	return &channelAssembler{}
}

// addFrame processes a single frame. Returns a readyChannel if the channel is complete.
func (ca *channelAssembler) addFrame(frame derive.Frame, l1Ref eth.L1BlockRef) *readyChannel {
	if ca.current == nil || frame.ID != ca.currentID {
		ca.current = derive.NewChannel(frame.ID, l1Ref, true)
		ca.currentID = frame.ID
		ca.openBlock = l1Ref
		ca.nextFrame = 0
	}

	if frame.FrameNumber != ca.nextFrame {
		return nil
	}

	if err := ca.current.AddFrame(frame, l1Ref); err != nil {
		return nil
	}
	ca.nextFrame++

	if ca.current.IsReady() {
		ready := &readyChannel{
			id:        ca.currentID,
			openBlock: ca.openBlock,
			channel:   ca.current,
		}
		ca.current = nil
		return ready
	}
	return nil
}

// checkTimeout returns true and discards the current channel if it has timed out.
func (ca *channelAssembler) checkTimeout(current eth.L1BlockRef, channelTimeout uint64) bool {
	if ca.current == nil {
		return false
	}
	if current.Number > ca.openBlock.Number+channelTimeout {
		ca.current = nil
		return true
	}
	return false
}
