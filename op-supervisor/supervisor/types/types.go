package types

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	interopmsgs "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	errNilSafetyLevel          = errors.New("nil safety level")
	errUnrecognizedSafetyLevel = errors.New("unrecognized safety level")
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

type SafetyLevel string

func (lvl SafetyLevel) String() string {
	return string(lvl)
}

// Validate returns true if the SafetyLevel is one of the recognized levels
func (lvl SafetyLevel) Validate() bool {
	switch lvl {
	case Invalid, Finalized, CrossSafe, LocalSafe, CrossUnsafe, LocalUnsafe:
		return true
	default:
		return false
	}
}

func (lvl SafetyLevel) MarshalText() ([]byte, error) {
	return []byte(lvl), nil
}

func (lvl *SafetyLevel) UnmarshalText(text []byte) error {
	if lvl == nil {
		return errNilSafetyLevel
	}
	x := SafetyLevel(text)
	if !x.Validate() {
		return fmt.Errorf("%w: %q", errUnrecognizedSafetyLevel, text)
	}
	*lvl = x
	return nil
}

const (
	// Finalized is CrossSafe, with the additional constraint that every
	// dependency is derived only from finalized L1 input data.
	// This matches RPC label "finalized".
	Finalized SafetyLevel = "finalized"
	// CrossSafe is as safe as LocalSafe, with all its dependencies
	// also fully verified to be reproducible from L1.
	// This matches RPC label "safe".
	CrossSafe SafetyLevel = "safe"
	// LocalSafe is verified to be reproducible from L1,
	// without any verified cross-L2 dependencies.
	// This does not have an RPC label.
	LocalSafe SafetyLevel = "local-safe"
	// CrossUnsafe is as safe as LocalUnsafe,
	// but with verified cross-L2 dependencies that are at least CrossUnsafe.
	// This does not have an RPC label.
	CrossUnsafe SafetyLevel = "cross-unsafe"
	// LocalUnsafe is the safety of the tip of the chain. This matches RPC label "unsafe".
	LocalUnsafe SafetyLevel = "unsafe"
	// Invalid is the safety of when the message or block is not matching the expected data.
	Invalid SafetyLevel = "invalid"
)

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
