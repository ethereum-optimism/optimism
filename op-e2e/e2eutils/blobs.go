package e2eutils

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BlobsStore is a simple in-memory store of blobs, for testing purposes
type BlobsStore struct {
	blobs map[uint64]map[common.Hash]*eth.Blob
}

func NewBlobStore() *BlobsStore {
	return &BlobsStore{blobs: make(map[uint64]map[common.Hash]*eth.Blob)}
}

func (store *BlobsStore) StoreBlob(time uint64, versionedHash common.Hash, blob *eth.Blob) {
	m, ok := store.blobs[time]
	if !ok {
		m = make(map[common.Hash]*eth.Blob)
		store.blobs[time] = m
	}
	m[versionedHash] = blob
}

func (store *BlobsStore) GetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error) {
	out := make([]*eth.Blob, 0, len(hashes))
	m, ok := store.blobs[ref.Time]
	if !ok {
		return nil, fmt.Errorf("no blobs known with given time: %w", ethereum.NotFound)
	}
	for _, h := range hashes {
		if b, ok := m[h.Hash]; !ok {
			return nil, fmt.Errorf("blob %d %s is not in store: %w", h.Index, h.Hash, ethereum.NotFound)
		} else {
			out = append(out, b)
		}
	}
	return out, nil
}

var _ derive.L1BlobsFetcher = (*BlobsStore)(nil)
