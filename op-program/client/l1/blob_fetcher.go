package l1

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var InvalidHashesLengthError = errors.New("invalid hashes length")

type BlobFetcher struct {
	logger log.Logger
	oracle Oracle
}

var _ derive.L1BlobsFetcher = (*BlobFetcher)(nil)

func NewBlobFetcher(logger log.Logger, oracle Oracle) *BlobFetcher {
	return &BlobFetcher{
		logger: logger,
		oracle: oracle,
	}
}

// GetBlobsByHash fetches blobs that were confirmed at the given timestamp with the given versioned hashes.
func (b *BlobFetcher) GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	blobs := make([]*eth.Blob, len(hashes))
	ref := eth.L1BlockRef{Time: time}
	for i := 0; i < len(hashes); i++ {
		b.logger.Info("Fetching blob", "time", time, "blob_versioned_hash", hashes[i])
		blobs[i] = b.oracle.GetBlob(ref, hashes[i])
	}
	return blobs, nil
}
