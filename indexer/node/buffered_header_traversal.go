package node

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrBufferedHeaderTraversalInvalidAdvance = errors.New("invalid advancement based on the BufferedHeaderTraversal's internal state")
)

// BufferedHeaderTraversal is a wrapper over HeaderTraversal which buffers traversed headers and only
// expands the buffer on a requested size increase or advancements are made which reduces the buffer
type BufferedHeaderTraversal struct {
	*HeaderTraversal
	bufferedHeaders []*types.Header
}

// NewBufferedHeaderTraversal creates a new instance of BufferedHeaderTraversal
func NewBufferedHeaderTraversal(ethClient EthClient, fromHeader *types.Header) *BufferedHeaderTraversal {
	return &BufferedHeaderTraversal{HeaderTraversal: NewHeaderTraversal(ethClient, fromHeader)}
}

// NextFinalizedHeaders returns the buffered set of headers bounded by the supplied size
func (bf *BufferedHeaderTraversal) NextFinalizedHeaders(maxSize uint64) ([]*types.Header, error) {
	numBuffered := uint64(len(bf.bufferedHeaders))
	if maxSize <= numBuffered {
		return bf.bufferedHeaders[:maxSize], nil
	}

	headers, err := bf.HeaderTraversal.NextFinalizedHeaders(maxSize - uint64(numBuffered))
	if err != nil {
		if numBuffered == 0 {
			return nil, err
		}

		// swallow the error and return existing buffered headers
		return bf.bufferedHeaders, nil
	}

	// No need to check the integrity of this new batch since the underlying HeaderTraversal ensures this
	bf.bufferedHeaders = append(bf.bufferedHeaders, headers...)
	return bf.bufferedHeaders, nil
}

// Advance reduces the internal buffer by marking the supplied header as the new base for the buffer
func (bf *BufferedHeaderTraversal) Advance(header *types.Header) error {
	numBuffered := uint64(len(bf.bufferedHeaders))
	if numBuffered == 0 {
		return ErrBufferedHeaderTraversalInvalidAdvance
	}

	firstBuffered := bf.bufferedHeaders[0]
	if firstBuffered.Number.Cmp(header.Number) > 0 {
		return ErrBufferedHeaderTraversalInvalidAdvance
	}

	step := new(big.Int).Sub(header.Number, firstBuffered.Number).Uint64()
	if step > numBuffered-1 || header.Hash() != bf.bufferedHeaders[step].Hash() {
		// too large a step or the supplied header does not match the buffered header
		return ErrBufferedHeaderTraversalInvalidAdvance
	}

	if step < numBuffered-1 {
		// partial advancement
		bf.bufferedHeaders = bf.bufferedHeaders[step+1:]
	} else {
		// throw away the entire buffer
		bf.bufferedHeaders = nil
	}

	return nil
}
