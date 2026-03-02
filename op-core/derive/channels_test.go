package derive

import (
	"testing"

	opderive "github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/stretchr/testify/require"
)

func testChannelID(b byte) opderive.ChannelID {
	var id opderive.ChannelID
	id[0] = b
	return id
}

func TestChannelAssembler_SingleFrameChannel(t *testing.T) {
	ca := newChannelAssembler()
	l1 := testL1Ref(1)

	ready := ca.addFrame(opderive.Frame{
		ID:          testChannelID(0xAA),
		FrameNumber: 0,
		Data:        []byte("hello"),
		IsLast:      true,
	}, l1)

	require.NotNil(t, ready, "single-frame channel should be ready immediately")
	require.Equal(t, testChannelID(0xAA), ready.id)
	require.Equal(t, l1, ready.openBlock)
	require.True(t, ready.channel.IsReady())
}

func TestChannelAssembler_MultiFrameChannel(t *testing.T) {
	ca := newChannelAssembler()
	chID := testChannelID(0xBB)
	l1 := testL1Ref(1)

	ready := ca.addFrame(opderive.Frame{
		ID:          chID,
		FrameNumber: 0,
		Data:        []byte("part1"),
		IsLast:      false,
	}, l1)
	require.Nil(t, ready, "channel should not be ready after first frame")

	l1b := testL1Ref(2)
	ready = ca.addFrame(opderive.Frame{
		ID:          chID,
		FrameNumber: 1,
		Data:        []byte("part2"),
		IsLast:      true,
	}, l1b)

	require.NotNil(t, ready, "channel should be ready after last frame")
	require.Equal(t, chID, ready.id)
	require.Equal(t, l1, ready.openBlock, "openBlock should be from the first frame")
	require.True(t, ready.channel.IsReady())
}

func TestChannelAssembler_NewChannelDiscardsOld(t *testing.T) {
	ca := newChannelAssembler()
	chA := testChannelID(0xAA)
	chB := testChannelID(0xBB)
	l1 := testL1Ref(1)

	ready := ca.addFrame(opderive.Frame{
		ID:          chA,
		FrameNumber: 0,
		Data:        []byte("A-frame0"),
		IsLast:      false,
	}, l1)
	require.Nil(t, ready)
	require.Equal(t, chA, ca.currentID)

	l1b := testL1Ref(2)
	ready = ca.addFrame(opderive.Frame{
		ID:          chB,
		FrameNumber: 0,
		Data:        []byte("B-frame0"),
		IsLast:      true,
	}, l1b)

	require.NotNil(t, ready, "new channel B should complete")
	require.Equal(t, chB, ready.id)
	require.Equal(t, l1b, ready.openBlock, "openBlock should be from channel B's first frame")
}

func TestChannelAssembler_Timeout(t *testing.T) {
	ca := newChannelAssembler()
	chID := testChannelID(0xCC)
	l1Open := testL1Ref(10)

	ca.addFrame(opderive.Frame{
		ID:          chID,
		FrameNumber: 0,
		Data:        []byte("data"),
		IsLast:      false,
	}, l1Open)
	require.NotNil(t, ca.current, "channel should be in progress")

	channelTimeout := uint64(50)

	notTimedOut := testL1Ref(10 + channelTimeout)
	require.False(t, ca.checkTimeout(notTimedOut, channelTimeout),
		"should not timeout at exactly openBlock + channelTimeout")
	require.NotNil(t, ca.current)

	timedOut := testL1Ref(10 + channelTimeout + 1)
	require.True(t, ca.checkTimeout(timedOut, channelTimeout),
		"should timeout when current.Number > openBlock.Number + channelTimeout")
	require.Nil(t, ca.current, "channel should be discarded after timeout")
}

func TestChannelAssembler_OutOfOrderFrame(t *testing.T) {
	ca := newChannelAssembler()
	chID := testChannelID(0xDD)
	l1 := testL1Ref(1)

	ready := ca.addFrame(opderive.Frame{
		ID:          chID,
		FrameNumber: 0,
		Data:        []byte("frame0"),
		IsLast:      false,
	}, l1)
	require.Nil(t, ready)

	ready = ca.addFrame(opderive.Frame{
		ID:          chID,
		FrameNumber: 2, // skip frame 1
		Data:        []byte("frame2"),
		IsLast:      true,
	}, l1)
	require.Nil(t, ready, "out-of-order frame should be dropped")

	require.NotNil(t, ca.current, "channel should still be in progress")
	require.Equal(t, uint16(1), ca.nextFrame, "nextFrame should still expect frame 1")

	ready = ca.addFrame(opderive.Frame{
		ID:          chID,
		FrameNumber: 1,
		Data:        []byte("frame1"),
		IsLast:      true,
	}, l1)
	require.NotNil(t, ready, "channel should complete once gap is filled")
}
