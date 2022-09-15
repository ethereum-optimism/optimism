package derive

import (
	"errors"
	"fmt"
)

// count the tagging info as 200 in terms of buffer size.
const frameOverhead = 200

const DerivationVersion0 = 0

// MaxChannelBankSize is the amount of memory space, in number of bytes,
// till the bank is pruned by removing channels,
// starting with the oldest channel.
const MaxChannelBankSize = 100_000_000

// DuplicateErr is returned when a newly read frame is already known
var DuplicateErr = errors.New("duplicate frame")

// ChannelIDLength defines the length of the channel IDs
const ChannelIDLength = 16

// ChannelID is an opaque identifier for a channel. It is 128 bits to be globally unique.
type ChannelID [ChannelIDLength]byte

func (id ChannelID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// TerminalString implements log.TerminalStringer, formatting a string for console output during logging.
func (id ChannelID) TerminalString() string {
	return fmt.Sprintf("%x..%x", id[:3], id[13:])
}
