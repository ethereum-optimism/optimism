package rawdb

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/l2geth/ethdb"
	"github.com/ethereum-optimism/optimism/l2geth/log"
)

// ReadHeadIndex will read the known tip of the CTC
func ReadHeadIndex(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(headIndexKey)
	if len(data) == 0 {
		return nil
	}
	ret := new(big.Int).SetBytes(data).Uint64()
	return &ret
}

// WriteHeadIndex will write the known tip of the CTC
func WriteHeadIndex(db ethdb.KeyValueWriter, index uint64) {
	value := new(big.Int).SetUint64(index).Bytes()
	if index == 0 {
		value = []byte{0}
	}
	if err := db.Put(headIndexKey, value); err != nil {
		log.Crit("Failed to store index", "err", err)
	}
}

// ReadHeadQueueIndex will read the known tip of the queue
func ReadHeadQueueIndex(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(headQueueIndexKey)
	if len(data) == 0 {
		return nil
	}
	ret := new(big.Int).SetBytes(data).Uint64()
	return &ret
}

// WriteHeadQueueIndex will write the known tip of the queue
func WriteHeadQueueIndex(db ethdb.KeyValueWriter, index uint64) {
	value := new(big.Int).SetUint64(index).Bytes()
	if index == 0 {
		value = []byte{0}
	}
	if err := db.Put(headQueueIndexKey, value); err != nil {
		log.Crit("Failed to store queue index", "err", err)
	}
}

// ReadHeadVerifiedIndex will read the known tip of the batched transactions
func ReadHeadVerifiedIndex(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(headVerifiedIndexKey)
	if len(data) == 0 {
		return nil
	}
	ret := new(big.Int).SetBytes(data).Uint64()
	return &ret
}

// WriteHeadVerifiedIndex will write the known tip of the batched transactions
func WriteHeadVerifiedIndex(db ethdb.KeyValueWriter, index uint64) {
	value := new(big.Int).SetUint64(index).Bytes()
	if index == 0 {
		value = []byte{0}
	}
	if err := db.Put(headVerifiedIndexKey, value); err != nil {
		log.Crit("Failed to store verifier index", "err", err)
	}
}

// ReadHeadBatchIndex will read the known tip of the processed batches
func ReadHeadBatchIndex(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(headBatchKey)
	if len(data) == 0 {
		return nil
	}
	ret := new(big.Int).SetBytes(data).Uint64()
	return &ret
}

// WriteHeadBatchIndex will write the known tip of the processed batches
func WriteHeadBatchIndex(db ethdb.KeyValueWriter, index uint64) {
	value := new(big.Int).SetUint64(index).Bytes()
	if index == 0 {
		value = []byte{0}
	}
	if err := db.Put(headBatchKey, value); err != nil {
		log.Crit("Failed to store head batch index", "err", err)
	}
}
