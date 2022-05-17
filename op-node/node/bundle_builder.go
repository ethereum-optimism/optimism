package node

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BundleCandidate is a struct holding the BlockID of an L2 block and the
// derived batch.
type BundleCandidate struct {
	// ID is the block ID of an L2 block.
	ID eth.BlockID

	// Batch is batch data drived from the L2 Block. If Batch is nil, the block
	// is considered to be empty. Empty blocks do not contribute to the size of
	// a bundle.
	Batch *derive.BatchData
}

// BundleBuilder is a helper struct used to construct BatchBundleResponses. This
// struct helps to provide efficient operations to modify a set of
// BundleCandidates that are need to craft bundles.
type BundleBuilder struct {
	prevBlockID eth.BlockID
	candidates  []BundleCandidate
	numNonEmpty int
}

// NewBundleBuilder creates a new instance of a BundleBuilder, where prevBlockID
// is the latest, canonical block that was chosen as the common fork ancestor.
func NewBundleBuilder(prevBlockID eth.BlockID) *BundleBuilder {
	return &BundleBuilder{
		prevBlockID: prevBlockID,
		candidates:  nil,
		numNonEmpty: 0,
	}
}

// AddCandidate appends a candidate block to the BundleBuilder.
func (b *BundleBuilder) AddCandidate(candidate BundleCandidate) {
	b.candidates = append(b.candidates, candidate)
	if candidate.Batch != nil {
		b.numNonEmpty++
	}
}

// HasNonEmptyCandidate returns true if there are a non-zero number of
// non-empty bundle candidates.
func (b *BundleBuilder) HasNonEmptyCandidate() bool {
	return b.numNonEmpty > 0
}

// PruneLastNonEmpty removes the latest non-empty candidate block and all empty
// blocks follow it. This method is used to reduce the size of the encoded
// bundle in order to satisfy the desired size constraints.
func (b *BundleBuilder) PruneLastNonEmpty() {
	if b.numNonEmpty == 0 {
		return
	}

	for i := len(b.candidates) - 1; i >= 0; i-- {
		candidate := b.candidates[i]
		if candidate.Batch != nil {
			b.candidates = b.candidates[:i]
			b.numNonEmpty--
			return
		}
	}
}

// Batches returns a slice of all non-nil batches contained within the candidate
// blocks.
func (b *BundleBuilder) Batches() []*derive.BatchData {
	var batches = make([]*derive.BatchData, 0, b.numNonEmpty)
	for _, candidate := range b.candidates {
		if candidate.Batch != nil {
			batches = append(batches, candidate.Batch)
		}
	}

	return batches
}

// Response returns the BatchBundleResponse given the current state of the
// BundleBuilder. The method accepts the encoded bundle as an argument, and
// fills in the correct metadata in the response.
func (b *BundleBuilder) Response(bundle []byte) *BatchBundleResponse {
	lastBlockID := b.prevBlockID
	if len(b.candidates) > 0 {
		lastBlockID = b.candidates[len(b.candidates)-1].ID
	}

	return &BatchBundleResponse{
		PrevL2BlockHash: b.prevBlockID.Hash,
		PrevL2BlockNum:  hexutil.Uint64(b.prevBlockID.Number),
		LastL2BlockHash: lastBlockID.Hash,
		LastL2BlockNum:  hexutil.Uint64(lastBlockID.Number),
		Bundle:          hexutil.Bytes(bundle),
	}
}
