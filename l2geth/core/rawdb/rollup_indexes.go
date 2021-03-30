package rawdb

import (
	"math/big"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

func ReadHeadIndex(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(headIndexKey)
	if len(data) == 0 {
		return nil
	}
	ret := new(big.Int).SetBytes(data).Uint64()
	return &ret
}

func WriteHeadIndex(db ethdb.KeyValueWriter, index uint64) {
	value := new(big.Int).SetUint64(index).Bytes()
	if index == 0 {
		value = []byte{0}
	}
	if err := db.Put(headIndexKey, value); err != nil {
		log.Crit("Failed to store index", "err", err)
	}
}

func ReadHeadQueueIndex(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(headQueueIndexKey)
	if len(data) == 0 {
		return nil
	}
	ret := new(big.Int).SetBytes(data).Uint64()
	return &ret
}

func WriteHeadQueueIndex(db ethdb.KeyValueWriter, index uint64) {
	value := new(big.Int).SetUint64(index).Bytes()
	if index == 0 {
		value = []byte{0}
	}
	if err := db.Put(headQueueIndexKey, value); err != nil {
		log.Crit("Failed to store queue index", "err", err)
	}
}
