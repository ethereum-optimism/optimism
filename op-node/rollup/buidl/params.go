package buidl

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

// count the tagging info as 200 in terms of buffer size.
const frameOverhead = 200

const DerivationVersion0 = 0

// channel ID, frame number, frame length, last frame bool
const minimumFrameSize = ChannelIDSize + 1 + 1 + 1

// ChannelTimeout is the number of seconds until a channel is removed if it's not read
const ChannelTimeout = 10 * 60

// MaxChannelBankSize is the amount of memory space, in number of bytes,
// till the bank is pruned by removing channels,
// starting with the oldest channel.
const MaxChannelBankSize = 100_000_000

// DuplicateErr is returned when a newly read frame is already known
var DuplicateErr = errors.New("duplicate frame")

// ChannelIDSize defines the length of a channel ID
const ChannelIDSize = 32 // TODO: we can maybe use smaller IDs. As long as we don't get random collisions

// ChannelID identifies a "channel" a stream encoding a sequence of L2 information.
// A channelID is not a perfect nonce number, but is based on time instead:
// only once the L1 block time passes the channel ID, the channel can be read.
// A channel is not read before that, and instead buffered for later consumption.
type ChannelID [ChannelIDSize]byte

type TaggedData struct {
	L1Origin  eth.L1BlockRef
	ChannelID ChannelID
	Data      []byte
}
