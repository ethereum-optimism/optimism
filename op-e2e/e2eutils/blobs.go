package e2eutils

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BlobsStore is a simple in-memory store of blobs, for testing purposes
type BlobsStore struct {
	// block timestamp -> blob versioned hash -> blob
	blobs map[uint64]map[common.Hash]*eth.Blob
}

func NewBlobStore() *BlobsStore {
	return &BlobsStore{blobs: make(map[uint64]map[common.Hash]*eth.Blob)}
}

func (store *BlobsStore) StoreBlob(blockTime uint64, versionedHash common.Hash, blob *eth.Blob) {
	m, ok := store.blobs[blockTime]
	if !ok {
		m = make(map[common.Hash]*eth.Blob)
		store.blobs[blockTime] = m
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
		b, ok := m[h.Hash]
		if !ok {
			return nil, fmt.Errorf("blob %d %s is not in store: %w", h.Index, h.Hash, ethereum.NotFound)
		}
		out = append(out, b)
	}
	return out, nil
}

func (store *BlobsStore) GetBlobSidecars(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.BlobSidecar, error) {
	out := make([]*eth.BlobSidecar, 0, len(hashes))
	m, ok := store.blobs[ref.Time]
	if !ok {
		return nil, fmt.Errorf("no blobs known with given time: %w", ethereum.NotFound)
	}
	for _, h := range hashes {
		b, ok := m[h.Hash]
		if !ok {
			return nil, fmt.Errorf("blob %d %s is not in store: %w", h.Index, h.Hash, ethereum.NotFound)
		}
		if b == nil {
			return nil, fmt.Errorf("blob %d %s is nil, cannot copy: %w", h.Index, h.Hash, ethereum.NotFound)
		}

		commitment, err := kzg4844.BlobToCommitment(b.KZGBlob())
		if err != nil {
			return nil, fmt.Errorf("failed to convert blob to commitment: %w", err)
		}
		out = append(out, &eth.BlobSidecar{
			Index:         eth.Uint64String(h.Index),
			Blob:          *b,
			KZGCommitment: eth.Bytes48(commitment),
		})
	}
	return out, nil
}

var (
	_ derive.L1BlobsFetcher = (*BlobsStore)(nil)
)
