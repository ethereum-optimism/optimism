package derive

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

// Batch format
//
// SingularBatchType := 0
// singularBatch := SingularBatchType ++ RLP([parent_hash, epoch_number, epoch_hash, timestamp, transaction_list])

// SingularBatch is an implementation of Batch interface, containing the input to build one L2 block.
type SingularBatch struct {
	ParentHash   common.Hash  // parent L2 block hash
	EpochNum     rollup.Epoch // aka l1 num
	EpochHash    common.Hash  // l1 block hash
	Timestamp    uint64
	Transactions []hexutil.Bytes
}

// GetBatchType returns its batch type (batch_version)
func (b *SingularBatch) GetBatchType() int {
	return SingularBatchType
}

// GetTimestamp returns its block timestamp
func (b *SingularBatch) GetTimestamp() uint64 {
	return b.Timestamp
}

// GetEpochNum returns its epoch number (L1 origin block number)
func (b *SingularBatch) GetEpochNum() rollup.Epoch {
	return b.EpochNum
}

// LogContext creates a new log context that contains information of the batch
func (b *SingularBatch) LogContext(log log.Logger) log.Logger {
	return log.New(
		"batch_timestamp", b.Timestamp,
		"parent_hash", b.ParentHash,
		"batch_epoch", b.Epoch(),
		"txs", len(b.Transactions),
	)
}

// Epoch returns a BlockID of its L1 origin.
func (b *SingularBatch) Epoch() eth.BlockID {
	return eth.BlockID{Hash: b.EpochHash, Number: uint64(b.EpochNum)}
}
