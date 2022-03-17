package eth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type BlockID struct {
	Hash   common.Hash `json:"hash"`
	Number uint64      `json:"number"`
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
	Self     BlockID `json:"self"`
	Parent   BlockID `json:"parent"`
	L1Origin BlockID `json:"l1_origin"`
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
	Self   BlockID `json:"self"`
	Parent BlockID `json:"parent"`
}

func (id L1BlockRef) String() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.String(), id.Self.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L1BlockRef) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.TerminalString(), id.Self.Number)
}
