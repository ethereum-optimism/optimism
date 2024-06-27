package source

import "github.com/ethereum-optimism/optimism/op-service/eth"

// UnsafeHeadEvent indicates the unsafe head has updated. This may skip ahead multiple blocks or switch to a different
// fork as part of a reorg.
type UnsafeHeadEvent struct {
	Block eth.L1BlockRef
}

// UnsafeBlockEvent indicates an unsafe block to be processed. Unlike UnsafeHeadEvent the stream of events provide
// contiguous blocks except for when a reorg occurs.
type UnsafeBlockEvent struct {
	Block eth.L1BlockRef
}
