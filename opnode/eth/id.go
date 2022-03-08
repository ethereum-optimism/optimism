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

type L2Node struct {
	Self     BlockID
	L2Parent BlockID
	L1Parent BlockID
}

func (id L2Node) String() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.String(), id.Self.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L2Node) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.TerminalString(), id.Self.Number)
}

type L1Node struct {
	Self   BlockID
	Parent BlockID
}

func (id L1Node) String() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.String(), id.Self.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L1Node) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Self.Hash.TerminalString(), id.Self.Number)
}
