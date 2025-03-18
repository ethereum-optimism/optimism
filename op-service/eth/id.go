package eth

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
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

func ReceiptBlockID(r *types.Receipt) BlockID {
	return BlockID{Number: r.BlockNumber.Uint64(), Hash: r.BlockHash}
}

func HeaderBlockID(h *types.Header) BlockID {
	return BlockID{Number: h.Number.Uint64(), Hash: h.Hash()}
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

func (id L2BlockRef) BlockRef() BlockRef {
	return BlockRef{
		Hash:       id.Hash,
		Number:     id.Number,
		ParentHash: id.ParentHash,
		Time:       id.Time,
	}
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

// BlockRef is a Block Ref indepdendent of L1 or L2
// Because L1BlockRefs are strict subsets of L2BlockRefs, BlockRef is a direct alias of L1BlockRef
type BlockRef = L1BlockRef

func (b *BlockRef) UnmarshalJSON(data []byte) error {
	type BlockRefAlias BlockRef
	aux := &struct {
		Number json.RawMessage `json:"number"`
		Time   json.RawMessage `json:"timestamp"`
		*BlockRefAlias
	}{
		BlockRefAlias: (*BlockRefAlias)(b),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if err := parseHexOrUint64Field(aux.Number, &b.Number); err != nil {
		return fmt.Errorf("failed to parse block number: %w", err)
	}

	if err := parseHexOrUint64Field(aux.Time, &b.Time); err != nil {
		return fmt.Errorf("failed to parse block timestamp: %w", err)
	}

	return nil
}

func parseHexOrUint64Field(data json.RawMessage, target *uint64) error {
	var hexVal hexutil.Uint64
	if err := json.Unmarshal(data, &hexVal); err == nil {
		*target = uint64(hexVal)
		return nil
	}

	return json.Unmarshal(data, target)
}

func BlockRefFromHeader(h *types.Header) *BlockRef {
	return &BlockRef{
		Hash:       h.Hash(),
		Number:     h.Number.Uint64(),
		ParentHash: h.ParentHash,
		Time:       h.Time,
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

// IndexedBlobHash represents a blob hash that commits to a single blob confirmed in a block.  The
// index helps us avoid unnecessary blob to blob hash conversions to find the right content in a
// sidecar.
type IndexedBlobHash struct {
	Index uint64      // absolute index in the block, a.k.a. position in sidecar blobs array
	Hash  common.Hash // hash of the blob, used for consistency checks
}
