package batcher

import (
	"github.com/ethereum/go-ethereum/common"

	opfees "github.com/ethereum-optimism/optimism/op-core/fees"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SizedBlock wraps an L2 execution payload with cached size estimates of its
// batch contribution. Deposit transactions (0x7E) are excluded — they are
// derived from L1, not batch-submitted.
type SizedBlock struct {
	*eth.ExecutionPayload
	rawSize          uint64
	estimatedDABytes uint64
}

func ToSizedBlock(payload *eth.ExecutionPayload) SizedBlock {
	b := SizedBlock{ExecutionPayload: payload}
	// populate caches
	b.RawSize()
	b.EstimatedDABytes()
	return b
}

func (b SizedBlock) Hash() common.Hash { return b.BlockHash }

func (b SizedBlock) NumberU64() uint64 { return uint64(b.BlockNumber) }

func (b *SizedBlock) RawSize() uint64 {
	if b.rawSize == 0 {
		b.rawSize = uint64(70)
		for _, tx := range b.Transactions {
			// Deposit transactions are not included in batches
			if len(tx) > 0 && tx[0] == optypes.DepositTxType {
				continue
			}
			// Add 2 for the overhead of encoding the tx bytes in a RLP list
			b.rawSize += uint64(len(tx)) + 2
		}
	}
	return b.rawSize
}

func (b *SizedBlock) EstimatedDABytes() uint64 {
	if b.estimatedDABytes == 0 {
		daSize := uint64(70) // estimated overhead of batch metadata
		for _, tx := range b.Transactions {
			// Deposit transactions are not included in batches
			if len(tx) > 0 && tx[0] == optypes.DepositTxType {
				continue
			}
			// It is safe to assume that the estimated DA size is always a uint64
			daSize += bigs.Uint64Strict(opfees.NewRollupCostData(tx).EstimatedDASize())
		}
		b.estimatedDABytes = daSize
	}
	return b.estimatedDABytes
}
