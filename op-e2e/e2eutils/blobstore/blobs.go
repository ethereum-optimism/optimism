package blobstore

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"

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

var _ derive.L1BlobsFetcher = (*Store)(nil)
