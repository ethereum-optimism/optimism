package silhouette

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

// channelCompressor adapts derive.ChannelCompressor to the derive.Compressor interface a ChannelOut
// consumes. The only thing missing is FullErr, which is a batcher concern: a batcher stops adding
// blocks when a channel has reached the size it wants to post, whereas this source already knows
// exactly which blocks go in — one proof batch's worth — and posts them all. There is nothing for a
// fullness signal to decide, so it never reports full.
//
// Written here rather than borrowed from op-batcher/compressor so that op-supernode does not depend
// on the batcher for a fifteen-line adapter.
type channelCompressor struct {
	derive.ChannelCompressor
}

func newChannelCompressor() (*channelCompressor, error) {
	inner, err := derive.NewChannelCompressor(derive.Zlib)
	if err != nil {
		return nil, err
	}
	return &channelCompressor{ChannelCompressor: inner}, nil
}

// FullErr never reports full: see the type comment.
func (c *channelCompressor) FullErr() error { return nil }
