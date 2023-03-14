package derive

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// count the tagging info as 200 in terms of buffer size.
const frameOverhead = 200

// frameSize calculates the size of the frame + overhead for
// storing the frame. The sum of the frame size of each frame in
// a channel determines the channel's size. The sum of the channel
// sizes is used for pruning & compared against `MaxChannelBankSize`
func frameSize(frame Frame) uint64 {
	return uint64(len(frame.Data)) + frameOverhead
}

const DerivationVersion0 = 0

// MaxChannelBankSize is the amount of memory space, in number of bytes,
// till the bank is pruned by removing channels,
// starting with the oldest channel.
const MaxChannelBankSize = 100_000_000

// MaxRLPBytesPerChannel is the maximum amount of bytes that will be read from
// a channel. This limit is set when decoding the RLP.
const MaxRLPBytesPerChannel = 10_000_000

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

func (id ChannelID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *ChannelID) UnmarshalText(text []byte) error {
	h, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	if len(h) != ChannelIDLength {
		return errors.New("invalid length")
	}
	copy(id[:], h)
	return nil
}
