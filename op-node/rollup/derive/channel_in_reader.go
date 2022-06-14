package derive

import (
	"bytes"
	"compress/zlib"
	"io"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// zlib returns an io.ReadCloser but explicitly documents it is also a zlib.Resetter, and we want to use it as such.
type zlibReader interface {
	io.ReadCloser
	zlib.Resetter
}

// read function is an util to expose a function as io.Reader, e.g. to not expose the reading to public API.
type readFn func(p []byte) (n int, err error)

func (fn readFn) Read(p []byte) (n int, err error) {
	return fn(p)
}

type ChannelInReader struct {
	// Returns the next frame to process blocks until there is new data to consume.
	// Returns nil when no new data from source is available currently.
	source func() *TaggedData

	buf      *bytes.Buffer
	readZlib zlibReader
	readRLP  *rlp.Stream

	l1Origin eth.L1BlockRef
	channel  ChannelID
	frameNr  uint64
}

func NewChannelInReader(source func() *TaggedData) *ChannelInReader {
	cr := &ChannelInReader{
		source: source,
		buf:    bytes.NewBuffer(make([]byte, 1000)),
	}
	cr.Reset()
	return cr
}

// ReadBatch returns a decoded rollup batch, or an error:
// - io.EOF, if the ChannelInReader source stops returning data then it's interpreted as EOF of the underlying stream,
//   and wrapped readers will eventually EOF too, or error if the EOF was unexpected, after reading any remaining data.
// - any other error (e.g. invalid compression or batch data):
//   the caller should ChannelInReader.Reset() before continuing reading the next batch.
//
// The reader blocks on retrieving additional data from the source, and closes itself when the source closes (by returning nil).
//
// The source may return nil for other reasons (timeout, reorg, etc.), and cause the stream to fail with an early EOF, even though new data is available later.
// The reader will be closed after this, and CurrentL1Origin(), CurrentChannel() and CurrentFrameNumber() can be checked to get the state of where the reader failed.
//
// The reader automatically moves to the next data sources as the current one gets exhausted.
// It's up to the caller to check CurrentL1Origin() before reading more information.
// The CurrentL1Origin() does not change until the first ReadBatch() after the old source has been completely exhausted.
func (cr *ChannelInReader) ReadBatch(dest *BatchData) error {
	return cr.readRLP.Decode(dest)
}

func (cr *ChannelInReader) readChannel(p []byte) (n int, err error) {
	bufN, err := cr.buf.Read(p)
	if err != nil { // *bytes.Buffer.Read() only returns io.EOF errors, and only if the buffer is empty.
		if cr.source == nil {
			return 0, io.EOF
		}
		// if we're out of data, then rotate to the next frame
		next := cr.source()
		if next == nil {
			cr.source = nil // close source
			return 0, io.EOF
		}
		// always keep L1 origin up to date: it may change per frame
		cr.l1Origin = next.L1Origin
		cr.frameNr = next.FrameNumber

		// reset if we switched to a new channel, append frame data otherwise
		if cr.channel != next.ChannelID {
			cr.reset(next.Data, next.ChannelID)
		} else {
			cr.buf.Write(next.Data)
		}
		// We don't immediately read from the buffer; we have to handle an empty buffer
		// (if frame was empty after transformations etc.) and not return
		// an EOF (default for empty bytes.Buffer read) because of only that.
		return 0, nil
	}
	return bufN, nil
}

// Reset forces the next read to continue with the next channel,
// resetting any decoding/decompression state to a fresh start.
func (cr *ChannelInReader) Reset() {
	// empty channel ID, always different from the next thing that is read, since 0 is not a valid ID
	cr.reset(nil, ChannelID{})
}

func (cr *ChannelInReader) reset(data []byte, chID ChannelID) {
	cr.buf.Reset()
	cr.buf.Write(data)
	cr.channel = chID

	if err := cr.readZlib.Reset(readFn(cr.readChannel), nil); err != nil {
		// TODO: this only errors because it pro-actively reads from the channel. It shouldn't do this. We can switch to the zlib algo without the header etc. data.
		panic(err)
	}

	// Set input limit for ZLIB as a whole:
	// we don't want to decode a crazy large batch (zip bomb).
	// but we also don't want to decode the same tiny batch 1000x
	cr.readRLP.Reset(cr.readZlib, 10_000_000) // TODO: define a max number of bytes per channel, or per batch (and then be more careful about reading batches)
}

// CurrentL1Origin returns the L1 block that encodes the data that is currently being read.
// Batches should be filtered based on this source.
// Note that the source might not be canonical anymore by the time the data is processed.
func (cr *ChannelInReader) CurrentL1Origin() eth.L1BlockRef {
	return cr.l1Origin
}

// CurrentChannel returns the channel that is being read from.
func (cr *ChannelInReader) CurrentChannel() ChannelID {
	return cr.channel
}

// CurrentFrameNumber returns the frame number of the frame that is being read from in the current channel.
func (cr *ChannelInReader) CurrentFrameNumber() uint64 {
	return cr.frameNr
}

// Closed returns true when no additional data may be read from the underlying stream.
// The ChannelInReader itself may still be read till EOF or error, since some data may have been buffered.
func (cr *ChannelInReader) Closed() bool {
	return cr.source == nil
}
