package derive

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
)

// count the tagging info as 200 in terms of buffer size.
const frameOverhead = 200

const DerivationVersion0 = 0

// channel ID (data + time), frame number, frame length, last frame bool
const minimumFrameSize = (ChannelIDDataSize + 1) + 1 + 1 + 1

// MaxChannelBankSize is the amount of memory space, in number of bytes,
// till the bank is pruned by removing channels,
// starting with the oldest channel.
const MaxChannelBankSize = 100_000_000

// DuplicateErr is returned when a newly read frame is already known
var DuplicateErr = errors.New("duplicate frame")

// ChannelIDDataSize defines the length of the channel ID data part
const ChannelIDDataSize = 32

// ChannelID identifies a "channel" a stream encoding a sequence of L2 information.
// A channelID is part random data (this may become a hash commitment to restrict who opens which channel),
// and part timestamp. The timestamp invalidates the ID,
// to ensure channels cannot be re-opened after timeout, or opened too soon.
//
// The ChannelID type is flat and can be used as map key.
type ChannelID struct {
	Data [ChannelIDDataSize]byte
	Time uint64
}

func (id ChannelID) String() string {
	return fmt.Sprintf("%x:%d", id.Data[:], id.Time)
}

func (id ChannelID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *ChannelID) UnmarshalText(text []byte) error {
	if id == nil {
		return errors.New("cannot unmarshal text into nil Channel ID")
	}
	if len(text) < ChannelIDDataSize+1 {
		return fmt.Errorf("channel ID too short: %d", len(text))
	}
	if _, err := hex.Decode(id.Data[:], text[:ChannelIDDataSize]); err != nil {
		return fmt.Errorf("failed to unmarshal hex data part of channel ID: %v", err)
	}
	if c := text[ChannelIDDataSize*2]; c != ':' {
		return fmt.Errorf("expected : separator in channel ID, but got %d", c)
	}
	v, err := strconv.ParseUint(string(text[ChannelIDDataSize*2+1:]), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to unmarshal decimal time part of channel ID: %v", err)
	}
	id.Time = v
	return nil
}

// TerminalString implements log.TerminalStringer, formatting a string for console output during logging.
func (id ChannelID) TerminalString() string {
	return fmt.Sprintf("%x..%x-%d", id.Data[:3], id.Data[29:], id.Time)
}
