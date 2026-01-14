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

func (store *Store) GetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error) {
	out := make([]*eth.Blob, 0, len(hashes))
	m, ok := store.blobs[ref.Time]
	if !ok {
		return nil, fmt.Errorf("no blobs known with given time: %w", ethereum.NotFound)
	}
	for _, h := range hashes {
		b, ok := m[h]
		if !ok {
			return nil, fmt.Errorf("blob %d %s is not in store: %w", h.Index, h.Hash, ethereum.NotFound)
		}
		out = append(out, b)
	}
	return out, nil
}

// GetBlobsByHash returns a slice of blobs in the slot at the given timestamp,
// corresponding to the supplied versioned hashes.
// If the provided hashes is empty, all blobs in the store at the supplied timestamp are returned.
// Blobs are ordered by their index in the block.
func (store *Store) GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	m, ok := store.blobs[time]
	if !ok {
		return nil, fmt.Errorf("no blobs known with given time: %w", ethereum.NotFound)
	}

	if len(hashes) == 0 {
		out := make([]*eth.Blob, len(m))
		for k, v := range m {
			out[k.Index] = v
		}
		return out, nil
	}

	out := make([]*eth.Blob, len(hashes))
	numBlobsFound := 0
	for k, v := range m {
		if slices.Contains(hashes, k.Hash) {
			out[k.Index] = v
			numBlobsFound++
		}
	}
	if numBlobsFound != len(hashes) {
		return nil, fmt.Errorf("not all blobs found")
	}

	return out, nil
}

var _ derive.L1BlobsFetcher = (*Store)(nil)
