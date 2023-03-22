package util

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// maxSlot is the maximum possible storage slot.
	maxSlot = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
)

type DBFactory func() (*state.StateDB, error)

type StateCallback func(db *state.StateDB, key, value common.Hash) error

func IterateState(dbFactory DBFactory, address common.Address, cb StateCallback, workers int) error {
	if workers <= 0 {
		panic("workers must be greater than 0")
	}

	// WaitGroup to wait for all workers to finish.
	var wg sync.WaitGroup

	// Channel to receive errors from each iteration job.
	errCh := make(chan error, workers)
	// Channel to cancel all iteration jobs.
	cancelCh := make(chan struct{})

	worker := func(start, end common.Hash) {
		// Decrement the WaitGroup when the function returns.
		defer wg.Done()

		db, err := dbFactory()
		if err != nil {
			// Should never happen, so explode if it does.
			log.Crit("cannot create state db", "err", err)
		}
		st, err := db.StorageTrie(address)
		if err != nil {
			// Should never happen, so explode if it does.
			log.Crit("cannot get storage trie", "address", address, "err", err)
		}
		// st can be nil if the account doesn't exist.
		if st == nil {
			errCh <- fmt.Errorf("account does not exist: %s", address.Hex())
			return
		}

		it := trie.NewIterator(st.NodeIterator(start.Bytes()))

		// Below code is largely based on db.ForEachStorage. We can't use that
		// because it doesn't allow us to specify a start and end key.
		for it.Next() {
			select {
			case <-cancelCh:
				// If one of the workers encounters an error, cancel all of them.
				return
			default:
				break
			}

			// Use the raw (i.e., secure hashed) key to check if we've reached
			// the end of the partition. Use > rather than >= here to account for
			// the fact that the values returned by PartitionKeys are inclusive.
			// Duplicate addresses that may be returned by this iteration are
			// filtered out in the collector.
			if new(big.Int).SetBytes(it.Key).Cmp(end.Big()) > 0 {
				return
			}

			// Skip if the value is empty.
			rawValue := it.Value
			if len(rawValue) == 0 {
				continue
			}

			// Get the preimage.
			rawKey := st.GetKey(it.Key)
			if rawKey == nil {
				// Should never happen, so explode if it does.
				log.Crit("cannot get preimage for storage key", "key", it.Key)
			}
			key := common.BytesToHash(rawKey)

			// Parse the raw value.
			_, content, _, err := rlp.Split(rawValue)
			if err != nil {
				// Should never happen, so explode if it does.
				log.Crit("mal-formed data in state: %v", err)
			}

			value := common.BytesToHash(content)

			// Call the callback with the DB, key, and value. Errors get
			// bubbled up to the errCh.
			if err := cb(db, key, value); err != nil {
				errCh <- err
				return
			}
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)

		// Partition the keyspace per worker.
		start, end := PartitionKeyspace(i, workers)

		// Kick off our worker.
		go worker(start, end)
	}

	wg.Wait()

	for len(errCh) > 0 {
		err := <-errCh
		if err != nil {
			return err
		}
	}

	return nil
}

// PartitionKeyspace divides the key space into partitions by dividing the maximum keyspace
// by count then multiplying by i. This will leave some slots left over, which we handle below. It
// returns the start and end keys for the partition as a common.Hash. Note that the returned range
// of keys is inclusive, i.e., [start, end] NOT [start, end).
func PartitionKeyspace(i int, count int) (common.Hash, common.Hash) {
	if i < 0 || count < 0 {
		panic("i and count must be greater than 0")
	}

	if i > count-1 {
		panic("i must be less than count - 1")
	}

	// Divide the key space into partitions by dividing the key space by the number
	// of jobs. This will leave some slots left over, which we handle below.
	partSize := new(big.Int).Div(maxSlot.Big(), big.NewInt(int64(count)))

	start := common.BigToHash(new(big.Int).Mul(big.NewInt(int64(i)), partSize))
	var end common.Hash
	if i < count-1 {
		// If this is not the last partition, use the next partition's start key as the end.
		end = common.BigToHash(new(big.Int).Mul(big.NewInt(int64(i+1)), partSize))
	} else {
		// If this is the last partition, use the max slot as the end.
		end = maxSlot
	}

	return start, end
}
