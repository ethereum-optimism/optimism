package eth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type BlockID struct {
	Hash   common.Hash
	Number uint64
}

func (id BlockID) String() string {
	return fmt.Sprintf("%s:%d", id.Hash.String(), id.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id BlockID) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Hash.TerminalString(), id.Number)
}

type L2BlockRef struct {
	Self     BlockID
	Parent   BlockID
	L1Origin BlockID
}

func (id L2BlockRef) String() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.String(), id.Self.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L2BlockRef) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.TerminalString(), id.Self.Number)
}

type L1BlockRef struct {
	Self   BlockID
	Parent BlockID
}

func (id L1BlockRef) String() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.String(), id.Self.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L1BlockRef) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.TerminalString(), id.Self.Number)
}
