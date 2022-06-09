package buidl

import (
	"container/heap"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// ChannelID identifies a "channel" a stream encoding a sequence of L2 information.
// A channelID is not a perfect nonce number, but is based on time instead:
// only once the L1 block time passes the channel ID, the channel can be read.
// A channel is not read before that, and instead buffered for later consumption.
type ChannelID uint64

type TaggedData struct {
	L1Origin eth.L1BlockRef
	// ResetFirst indicates that any previous buffered data needs to be dropped before this frame can be consumed
	ResetFirst bool
	ChannelID  ChannelID
	Data       []byte
}

type Pipeline struct {
	queue    chan TaggedData
	l1Source eth.L1BlockRef
	buf      []byte
}

// Read data from the pipeline. An EOF is returned when the system closes. No errors are returned otherwise.
// The reader automatically moves to the next data sources as the current one gets exhausted.
// It's up to the caller to check CurrentSource() before reading more information.
// The CurrentSource() does not change until the first Read() after the old source has been completely exhausted.
func (p *Pipeline) Read(dest []byte) (n int, err error) {
	// if we're out of data, then rotate to the next,
	// and return an EOF to indicate that the reader should reset before trying again.
	if len(p.buf) == 0 {
		next, ok := <-p.queue
		if !ok {
			return 0, io.EOF
		}
		p.l1Source = next.L1Origin
		p.buf = next.Data
	}
	// try to consume current item
	n = copy(dest, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

// CurrentSource returns the L1 block that encodes the data that is currently being read.
// Batches should be filtered based on this source.
// Note that the source might not be canonical anymore by the time the data is processed.
func (p *Pipeline) CurrentSource() eth.L1BlockRef {
	return p.l1Source
}

type Channel struct {
	// id of the channel
	id ChannelID
	// index within the slice that backs the priority queue
	index int
	// estimated memory size, used to drop the channel if we have too much data
	size uint64

	progress uint64

	// final frame number (inclusive). Max value if we haven't seen the end yet.
	endsAt uint64

	inputs map[uint64]*TaggedData
}

var DuplicateErr = errors.New("duplicate frame")

// count the tagging info as 200 in terms of buffer size.
const frameOverhead = 200

// IngestData buffers a frame in the channel, and potentially forwards it, along with any previously buffered frames
func (ch *Channel) IngestData(ref eth.L1BlockRef, frameNum uint64, isLast bool, frameData []byte) error {
	if frameNum < ch.progress {
		// already consumed a frame with equal number, this must be a duplicate
		return DuplicateErr
	}
	if frameNum > ch.endsAt {
		return fmt.Errorf("channel already ended ingesting inputs")
	}
	// the frame is from the current or future, it will be read from the buffer later

	// create buffer if it didn't exist yet
	if ch.inputs == nil {
		ch.inputs = make(map[uint64]*TaggedData)
	}
	if _, exists := ch.inputs[frameNum]; exists {
		// already seen a frame for this channel with this frame number
		return DuplicateErr
	}
	// buffer the frame
	ch.inputs[frameNum] = &TaggedData{
		L1Origin:  ref,
		ChannelID: ch.id,
		Data:      frameData,
	}
	if isLast {
		ch.endsAt = frameNum
	}
	ch.size += uint64(len(frameData)) + frameOverhead
	return nil
}

// Read next tagged piece of data. This may return nil if there is none.
func (ch *Channel) Read() *TaggedData {
	taggedData, ok := ch.inputs[ch.progress]
	if !ok {
		return nil
	}
	ch.size -= uint64(len(taggedData.Data)) + frameOverhead
	delete(ch.inputs, ch.progress)
	ch.progress += 1
	return taggedData
}

// Closed returns if this channel has been fully read yet.
func (ch *Channel) Closed() bool {
	return ch.progress > ch.endsAt
}

type ChannelQueue []*Channel

func (pq ChannelQueue) Len() int { return len(pq) }

func (pq ChannelQueue) Less(i, j int) bool {
	// prioritize the channel with the lowest ID. Pop will give the lowest channel ID first.
	return pq[i].id < pq[j].id
}

func (pq ChannelQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push implements the heap interface. Use heap.Push if you want to add to the priority queue.
func (pq *ChannelQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Channel)
	item.index = n
	*pq = append(*pq, item)
}

// Pop implements the heap interface. Use heap.Pop if you want to pop from the priority queue
// (yes, the result is different, and the heap.Pop keeps the heap structure consistent)
func (pq *ChannelQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// Peek returns what Pop would return, without actually removing the element.
func (pq ChannelQueue) Peek() *Channel {
	if len(pq) == 0 {
		return nil
	}
	// The root of the tree has the highest priority (and the lowest channel ID)
	return pq[0]
}

// ChannelBank buffers channel frames, and emits TaggedData to an onProgress channel.
type ChannelBank struct {
	// A priority queue, lower channel IDs are prioritized.
	channels ChannelQueue

	// Current L1 origin that we have seen. Used to filter channels and continue reading.
	currentL1Origin eth.L1BlockRef
}

const ChannelVersion0 = 0

// version byte, channel ID, frame number, frame length, last frame bool
const minimumFrameSize = 1 + 1 + 1 + 1 + 1

// ChannelTimeout is the number of seconds until a channel is removed if it's not read
const ChannelTimeout = 10 * 60

// ChannelBufferTime is the number of seconds until a channel is read.
// Other data submissions have the time to front-run the reading with a lower channel ID until this.
const ChannelBufferTime = 10

// ChannelFutureMargin is the number of seconds in the future to allow a channel to be scheduled for.
const ChannelFutureMargin = 10

// MaxChannelBankSize is the amount of memory space, in number of bytes,
// till the bank is pruned and channels get pruned,
// starting with the oldest channel first.
const MaxChannelBankSize = 100_000_000

// Read returns nil if there is nothing new to Read.
func (ib *ChannelBank) Read() *TaggedData {
	// clear timed out channel(s) first
	for {
		lowestCh := ib.channels.Peek()
		if lowestCh == nil {
			return nil
		}
		if uint64(lowestCh.id)+ChannelTimeout < ib.currentL1Origin.Time {
			heap.Pop(&ib.channels)
		} else {
			break
		}
	}

	lowestCh := ib.channels.Peek()
	if lowestCh == nil {
		return nil
	}
	// if the channel is not ready yet, wait
	if lowestCh.id+ChannelBufferTime > ChannelID(ib.currentL1Origin.Time) {
		return nil
	}
	out := lowestCh.Read()
	// if this caused the channel to get closed (i.e. read all data), remove it
	if lowestCh.Closed() {
		heap.Pop(&ib.channels)
	}
	return out
}

func (ib *ChannelBank) CurrentL1() eth.L1BlockRef {
	return ib.currentL1Origin
}

// NextL1 updates the channel bank to tag new data with the next L1 reference
func (ib *ChannelBank) NextL1(ref eth.L1BlockRef) error {
	if ref.ParentHash != ib.currentL1Origin.Hash {
		return fmt.Errorf("reorg detected, cannot start consuming this L1 block without using a new channel bank: new.parent: %s, expected: %s", ref.ParentID(), ib.currentL1Origin.ParentID())
	}
	ib.currentL1Origin = ref
	return nil
}

// IngestData adds new L1 data to the channel bank.
// Read() should be called repeatedly first, until everything has been read, before adding new data.
// Then NextL1(ref) should be called to move forward to the next L1 input
func (ib *ChannelBank) IngestData(data []byte) error {
	if len(data) < minimumFrameSize {
		return fmt.Errorf("data must be at least have 1 frame")
	}
	if data[0] != ChannelVersion0 {
		return fmt.Errorf("unrecognized channel version: %d", data)
	}
	offset := 1

	// check total size
	totalSize := uint64(0)
	for _, ch := range ib.channels {
		totalSize += ch.size
	}
	// prune until it is reasonable again. The high-priority channel failed to be read, so we start pruning there.
	for totalSize > MaxChannelBankSize {
		ch := heap.Pop(&ib.channels).(*Channel)
		totalSize -= ch.size
	}

	// Iterate over all frames. They may have different channel IDs to indicate that they stream consumer should reset.
	for {
		if len(data) <= offset {
			return nil
		}

		chIDNumber, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return fmt.Errorf("failed to read frame number")
		}
		// stop reading and ignore remaining data if we encounter a zero
		if chIDNumber == 0 {
			return nil
		}

		frameNumber, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return fmt.Errorf("failed to read frame number")
		}
		frameLength, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return fmt.Errorf("failed to read frame length")
		}
		offset += n
		if remaining := uint64(len(data) - offset); remaining < frameLength {
			return fmt.Errorf("not enough data left for frame: %d < %d", remaining, frameLength)
		}
		chID := ChannelID(chIDNumber)

		frameData := data[offset : uint64(offset)+frameLength]
		offset += int(frameLength)

		if offset >= len(data) {
			return fmt.Errorf("failed to read frame end byte, no data left, offset past length %d", len(data))
		}
		isLastNum := data[offset]
		if isLastNum > 1 {
			return fmt.Errorf("invalid isLast bool value: %d", data[offset])
		}
		isLast := isLastNum == 1
		offset += 1

		// data channels must not be opened in the future, to ensure future data-transactions cannot be front-run way in advance
		if chIDNumber > ib.currentL1Origin.Time+ChannelFutureMargin {
			// TODO: log error
			//fmt.Errorf("channel ID %d cannot be higher than L1 block time %d (margin: %d)", chIDNumber, ib.currentL1Origin.Time, ChannelFutureMargin)
			continue
		}
		// if the channel is old, ignore it
		if chIDNumber+ChannelTimeout < ib.currentL1Origin.Time {
			// TODO: log error
			//fmt.Errorf("channel ID %d is too old for L1 block time %d (timeout: %d)", chIDNumber, ib.currentL1Origin.Time, ChannelTimeout)
			continue
		}

		var currentCh *Channel
		for _, ch := range ib.channels {
			if ch.id == chID {
				currentCh = ch
				break
			}
		}
		if currentCh == nil {
			// create new channel if it doesn't exist yet
			currentCh = &Channel{id: chID, endsAt: ^uint64(0)}
			heap.Push(&ib.channels, currentCh)
		}

		if err := currentCh.IngestData(ib.currentL1Origin, frameNumber, isLast, frameData); err != nil {
			// TODO: log error
			// fmt.Errorf("failed to ingest frame %d of channel %d: %w", frameNumber, chID, err)
			continue
		}
	}
}

// NewChannelBank prepares a new channel bank,
// ready to read data from starting at a point where we can be sure no channel data is missing.
// Upon a reorg, or startup, a channel bank should be constructed with the last consumed L1 block as l1Start.
// It will traverse back with the provided lookupParent to find the continuation point,
// and then replay everything with pullData to get a channel bank ready for reading from l1Start.
func NewChannelBank(ctx context.Context, l1Start eth.L1BlockRef,
	lookupParent func(ctx context.Context, id eth.BlockID) (eth.L1BlockRef, error),
	pullData func(id eth.BlockID, txIndex uint64) ([]byte, error)) (*ChannelBank, error) {
	block := l1Start
	var blocks []eth.L1BlockRef
	for block.Time+ChannelTimeout > l1Start.Time && block.Number > 0 {
		parent, err := lookupParent(ctx, block.ParentID())
		if err != nil {
			return nil, fmt.Errorf("failed to find channel bank block, failed to retrieve L1 reference: %w", err)
		}
		block = parent
		blocks = append(blocks, parent)
	}
	bank := &ChannelBank{channels: ChannelQueue{}, currentL1Origin: blocks[len(blocks)-1]}

	// now replay all the parent blocks
	for i := len(blocks) - 1; i >= 0; i-- {
		ref := blocks[i]
		if i != len(blocks)-1 {
			if err := bank.NextL1(ref); err != nil {
				return nil, fmt.Errorf("failed to continue replay at block %s: %v", ref, err)
			}
		}
		for j := uint64(0); ; j++ {
			txData, err := pullData(ref.ID(), j)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to replay data of %s: %w", blocks[i], err)
			}
			if err := bank.IngestData(txData); err != nil {
				// TODO log that tx was bad and had to be ignored, but don't stop replay.
				continue
			}
		}
		for bank.Read() != nil {
			// we drain before ingesting more, since writes affect reads this is mandatory
		}
	}
	if err := bank.NextL1(l1Start); err != nil {
		return nil, fmt.Errorf("failed to move bank origin to final %s: %v", l1Start, err)
	}
	return bank, nil
}
