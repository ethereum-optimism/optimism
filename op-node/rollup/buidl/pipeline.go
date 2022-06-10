package buidl

import (
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// TODO: maybe rename to ChannelInReader ?
type Pipeline struct {
	// Returns the next frame to process
	// blocks until there is new data to consume
	// returns nil when the source is closed
	source   func() *TaggedData
	l1Origin eth.L1BlockRef
	channel  ChannelID
	buf      []byte
}

// Read data from the pipeline.
// An EOF is returned when the reader should reset before reading more data again.
// The returned data may not be canonical anymore when read. The CurrentSource should be checked.
// No errors are returned otherwise.
// The reader automatically moves to the next data sources as the current one gets exhausted.
// It's up to the caller to check CurrentSource() before reading more information.
// The CurrentSource() does not change until the first Read() after the old source has been completely exhausted.
func (p *Pipeline) Read(dest []byte) (n int, err error) {
	// if we're out of data, then rotate to the next frame
	if len(p.buf) == 0 {
		next := p.source()
		if next == nil {
			return 0, io.EOF
		}
		p.l1Origin = next.L1Origin
		p.buf = next.Data
		if p.channel != next.ChannelID {
			p.channel = next.ChannelID
			return 0, io.EOF // reset the stream before we start reading from the next channel
		}
	}

	// try to consume current item
	n = copy(dest, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

// Reset forces the next read to continue with the next item
func (p *Pipeline) Reset() {
	p.buf = p.buf[:0]
	// empty channel ID, always different from the next thing that is read, since 0 is not a valid ID
	p.channel = ChannelID{}
}

// CurrentSource returns the L1 block that encodes the data that is currently being read.
// Batches should be filtered based on this source.
// Note that the source might not be canonical anymore by the time the data is processed.
func (p *Pipeline) CurrentSource() eth.L1BlockRef {
	return p.l1Origin
}
