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
	Hash           common.Hash `json:"hash"`
	Number         uint64      `json:"number"`
	ParentHash     common.Hash `json:"parentHash"`
	Time           uint64      `json:"timestamp"`
	L1Origin       BlockID     `json:"l1origin"`
	SequenceNumber uint64      `json:"sequenceNumber"` // distance to first block of epoch
}

func (id L2BlockRef) String() string {
	return fmt.Sprintf("%s:%d", id.Hash.String(), id.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L2BlockRef) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Hash.TerminalString(), id.Number)
}

type L1BlockRef struct {
	Hash       common.Hash `json:"hash"`
	Number     uint64      `json:"number"`
	ParentHash common.Hash `json:"parentHash"`
	Time       uint64      `json:"timestamp"`
}

func (id L1BlockRef) String() string {
	return fmt.Sprintf("%s:%d", id.Hash.String(), id.Number)
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (id L1BlockRef) TerminalString() string {
	return fmt.Sprintf("%s:%d", id.Hash.TerminalString(), id.Number)
}

func (id L1BlockRef) ID() BlockID {
	return BlockID{
		Hash:   id.Hash,
		Number: id.Number,
	}
}

func (id L1BlockRef) ParentID() BlockID {
	n := id.ID().Number
	// Saturate at 0 with subtraction
	if n > 0 {
		n -= 1
	}
	return BlockID{
		Hash:   id.ParentHash,
		Number: n,
	}
}

func (id L2BlockRef) ID() BlockID {
	return BlockID{
		Hash:   id.Hash,
		Number: id.Number,
	}
}

func (id L2BlockRef) ParentID() BlockID {
	n := id.ID().Number
	// Saturate at 0 with subtraction
	if n > 0 {
		n -= 1
	}
	return BlockID{
		Hash:   id.ParentHash,
		Number: n,
	}
}
