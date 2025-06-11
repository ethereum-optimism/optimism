package monitor

import "github.com/ethereum-optimism/optimism/op-service/eth"

// BlockBuffer is a circular buffer of seen blocks.
// It can be used as a fix-sized stack of blocks to ensure
// a canonical and contiguous view of the block history.
type BlockBuffer struct {
	buffer []eth.BlockInfo
	idx    int
	total  int
}

// NewBlockBuffer creates a new block buffer
func NewBlockBuffer(size int) *BlockBuffer {
	return &BlockBuffer{
		buffer: make([]eth.BlockInfo, size),
		idx:    0,
		total:  0,
	}
}

// Add adds a block to the buffer
func (r *BlockBuffer) Add(block eth.BlockInfo) {
	r.buffer[r.idx] = block
	r.idx++
	r.idx %= len(r.buffer)
	r.total++
}

// Peek returns the last added block to the buffer
// if the buffer is empty, it returns nil
// if the buffer is not empty, it returns the last added block
func (r *BlockBuffer) Peek() eth.BlockInfo {
	// if the buffer is empty, return nil
	if r.total == 0 {
		return nil
	}
	// get the previous index, wrap around if necessary
	prevIndex := (r.idx + len(r.buffer) - 1) % len(r.buffer)
	block := r.buffer[prevIndex]
	// if the block is nil, the buffer is empty
	if block == nil {
		return nil
	}
	return block
}

// Reset resets the buffer to empty
func (r *BlockBuffer) Reset() {
	r.idx = 0
	r.total = 0
	for i := range r.buffer {
		r.buffer[i] = nil
	}
}
