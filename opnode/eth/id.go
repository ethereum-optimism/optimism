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
