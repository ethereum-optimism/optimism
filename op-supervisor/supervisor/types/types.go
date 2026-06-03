package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	interopmsgs "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Revision uint64

// RevisionAny is used as indicator to ignore the revision during lookups.
// This is used in the cross-safe queries,
// where there will only ever be a single derived block per derived block number,
// but where the revision is still tracked to match the local-safe DB block replacements.
// We use the max-uint64 value, since this is reserved, and will not be allowed to decode/encode.
const RevisionAny = ^Revision(0)

func (r Revision) Any() bool {
	return r == RevisionAny
}

// Number returns the block-number, where the revision started (i.e. the invalidated/replacement block height)
func (r Revision) Number() uint64 {
	return uint64(r) &^ uint64(1<<63)
}

func (r Revision) String() string {
	if r.Any() {
		return "Rev(any)"
	}
	return fmt.Sprintf("Rev(%d)", r.Number())
}

// Cmp returns:
// 0 if the revision matches any block number
// 1 if the revision is higher than the given number
// 0 if the revision is equal than the given number
// -1 if the revision is lower than the given number
func (r Revision) Cmp(blockNum uint64) int {
	if r.Any() {
		return 0
	}
	if r.Number() > blockNum {
		return 1
	}
	if r.Number() == blockNum {
		return 0
	}
	return -1
}

// DerivedBlockRefPair is a pair of block refs, where Derived (L2) is derived from Source (L1).
type DerivedBlockRefPair struct {
	Source  eth.BlockRef `json:"source"`
	Derived eth.BlockRef `json:"derived"`
}

func (refs *DerivedBlockRefPair) IDs() DerivedIDPair {
	return DerivedIDPair{
		Source:  refs.Source.ID(),
		Derived: refs.Derived.ID(),
	}
}

func (refs *DerivedBlockRefPair) Seals() DerivedBlockSealPair {
	return DerivedBlockSealPair{
		Source:  interopmsgs.BlockSealFromRef(refs.Source),
		Derived: interopmsgs.BlockSealFromRef(refs.Derived),
	}
}

func (refs DerivedBlockRefPair) String() string {
	return fmt.Sprintf("refPair(source: %s, derived: %s)", refs.Source, refs.Derived)
}

// DerivedBlockSealPair is a pair of block seals, where Derived (L2) is derived from Source (L1).
type DerivedBlockSealPair struct {
	Source  interopmsgs.BlockSeal `json:"source"`
	Derived interopmsgs.BlockSeal `json:"derived"`
}

func (seals *DerivedBlockSealPair) IDs() DerivedIDPair {
	return DerivedIDPair{
		Source:  seals.Source.ID(),
		Derived: seals.Derived.ID(),
	}
}

func (seals DerivedBlockSealPair) String() string {
	return fmt.Sprintf("sealPair(source: %s, derived: %s)", seals.Source, seals.Derived)
}

// DerivedIDPair is a pair of block IDs, where Derived (L2) is derived from Source (L1).
type DerivedIDPair struct {
	Source  eth.BlockID `json:"source"`
	Derived eth.BlockID `json:"derived"`
}

func (ids DerivedIDPair) String() string {
	return fmt.Sprintf("idPair(source: %s, derived: %s)", ids.Source, ids.Derived)
}

type BlockReplacement struct {
	Replacement eth.BlockRef `json:"replacement"`
	Invalidated common.Hash  `json:"invalidated"`
}

// IndexingEvent is an event sent by the indexing node to the supervisor,
// to share an update. One of the fields will be non-null; different kinds of updates may be sent.
type IndexingEvent struct {
	Reset                  *string              `json:"reset,omitempty"`
	UnsafeBlock            *eth.BlockRef        `json:"unsafeBlock,omitempty"`
	DerivationUpdate       *DerivedBlockRefPair `json:"derivationUpdate,omitempty"`
	ExhaustL1              *DerivedBlockRefPair `json:"exhaustL1,omitempty"`
	ReplaceBlock           *BlockReplacement    `json:"replaceBlock,omitempty"`
	DerivationOriginUpdate *eth.BlockRef        `json:"derivationOriginUpdate,omitempty"`
}
