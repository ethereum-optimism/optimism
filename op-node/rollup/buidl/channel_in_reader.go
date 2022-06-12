package buidl

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"github.com/ethereum-optimism/optimism/l2geth/rlp"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"io"

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
	// Returns the next frame to process
	// blocks until there is new data to consume
	// returns an error when the source is broken
	source   func() (*TaggedData, error)
	readZlib zlibReader
	readRLP  *rlp.Stream

	l1Origin eth.L1BlockRef
	channel  ChannelID
	buf      *bytes.Buffer
}

func NewChannelInReader(source func() (*TaggedData, error)) (*ChannelInReader, error) {
	cr := &ChannelInReader{
		source: source,
		buf:    bytes.NewBuffer(make([]byte, 1000)),
	}
	err := cr.Reset()
	return cr, err
}

// ReadBatch returns a decoded rollup batch, or an error:
// - any other error (e.g. invalid compression or batch data):
//   the caller should ChannelInReader.Reset() before continuing reading the next batch.
//
// The reader automatically moves to the next data sources as the current one gets exhausted.
// It's up to the caller to check CurrentSource() before reading more information.
// The CurrentSource() does not change until the first ReadBatch() after the old source has been completely exhausted.
func (cr *ChannelInReader) ReadBatch(dest *derive.BatchData) error {
	return cr.readRLP.Decode(dest)
}

func (cr *ChannelInReader) readChannel(p []byte) (n int, err error) {
	bufN, err := cr.buf.Read(p)
	if err != nil { // *bytes.Buffer.Read() only returns io.EOF errors, and only if the buffer is empty.
		// if we're out of data, then rotate to the next frame
		next, err := cr.source()
		if err != nil {
			return 0, fmt.Errorf("channel reader source failed: %w", err)
		}
		// always keep L1 origin up to date: it may change per frame
		cr.l1Origin = next.L1Origin
		// reset if we switched to a new channel, append frame data otherwise
		if cr.channel != next.ChannelID {
			if err := cr.reset(next.Data, next.ChannelID); err != nil {
				return 0, fmt.Errorf("failed to reset ChannelInReader for next channel %s: %w", next.ChannelID, err)
			}
		} else {
			cr.buf.Write(next.Data)
		}
		return 0, nil
	}
	return bufN, nil
}

// Reset forces the next read to continue with the next channel,
// resetting any decoding/decompression state to a fresh start.
func (cr *ChannelInReader) Reset() error {
	// empty channel ID, always different from the next thing that is read, since 0 is not a valid ID
	return cr.reset(nil, ChannelID{})
}

func (cr *ChannelInReader) reset(data []byte, chID ChannelID) error {
	cr.buf.Reset()
	cr.buf.Write(data)
	cr.channel = chID

	if err := cr.readZlib.Reset(readFn(cr.readChannel), nil); err != nil {
		return nil
	}

	// Set input limit for ZLIB as a whole:
	// we don't want to decode a crazy large batch (zip bomb).
	// but we also don't want to decode the same tiny batch 1000x
	cr.readRLP.Reset(cr.readZlib, 10_000_000) // TODO: define a max number of bytes per channel, or per batch (and then be more careful about reading batches)
	return nil
}

// CurrentSource returns the L1 block that encodes the data that is currently being read.
// Batches should be filtered based on this source.
// Note that the source might not be canonical anymore by the time the data is processed.
func (cr *ChannelInReader) CurrentSource() eth.L1BlockRef {
	return cr.l1Origin
}
