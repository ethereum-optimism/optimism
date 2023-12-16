package derive

import (
	"bytes"
	"errors"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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
		"batch_type", "SingularBatch",
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

// encode writes the byte encoding of SingularBatch to Writer stream
func (b *SingularBatch) encode(w io.Writer) error {
	return rlp.Encode(w, b)
}

// decode reads the byte encoding of SingularBatch from Reader stream
func (b *SingularBatch) decode(r *bytes.Reader) error {
	return rlp.Decode(r, b)
}

// GetSingularBatch retrieves SingularBatch from batchData
func GetSingularBatch(batchData *BatchData) (*SingularBatch, error) {
	singularBatch, ok := batchData.inner.(*SingularBatch)
	if !ok {
		return nil, NewCriticalError(errors.New("failed type assertion to SingularBatch"))
	}
	return singularBatch, nil
}
