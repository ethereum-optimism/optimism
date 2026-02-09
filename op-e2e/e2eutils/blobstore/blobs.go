package blobstore

import (
	"context"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Store is a simple in-memory store of blobs, for testing purposes
type Store struct {
	// block timestamp -> blob versioned hash -> blob
	blobs map[uint64]map[eth.IndexedBlobHash]*eth.Blob
}

func New() *Store {
	return &Store{blobs: make(map[uint64]map[eth.IndexedBlobHash]*eth.Blob)}
}

func (store *Store) StoreBlob(blockTime uint64, indexedHash eth.IndexedBlobHash, blob *eth.Blob) {
	m, ok := store.blobs[blockTime]
	if !ok {
		m = make(map[eth.IndexedBlobHash]*eth.Blob)
		store.blobs[blockTime] = m
	}
	m[indexedHash] = blob
}

// GetBlobsByHash returns a slice of blobs in the slot at the given timestamp,
// corresponding to the supplied versioned hashes.
// If the provided hashes is empty, all blobs in the store at the supplied timestamp are returned.
// Blobs are ordered by their index in the block.
func (store *Store) GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	blobMap, ok := store.blobs[time]
	if !ok {
		return nil, fmt.Errorf("no blobs known with given time: %w", ethereum.NotFound)
	}

	// Case of empty hashes
	if len(hashes) == 0 {
		out := make([]*eth.Blob, len(blobMap))
		for k, v := range blobMap {
			out[k.Index] = v
		}
		return out, nil
	}

	// When hashes is not empty,
	type indexedBlob struct {
		Index uint64
		Blob  *eth.Blob
	}

	// find the blob for each hash
	indexedBlobSlice := make([]indexedBlob, 0, len(hashes))
	for _, h := range hashes {
		for k, v := range blobMap {
			if h == k.Hash {
				indexedBlobSlice = append(indexedBlobSlice, indexedBlob{Index: k.Index, Blob: v})
			}
		}
	}

	if len(indexedBlobSlice) != len(hashes) {
		return nil, fmt.Errorf("not all blobs found")
	}

	// sort by index
	slices.SortFunc(indexedBlobSlice, func(a, b indexedBlob) int {
		return int(a.Index) - int(b.Index)
	})

	// extract blobs
	blobSlice := make([]*eth.Blob, len(indexedBlobSlice))
	for i, blob := range indexedBlobSlice {
		blobSlice[i] = blob.Blob
	}

	return blobSlice, nil
}

var _ derive.L1BlobsFetcher = (*Store)(nil)
