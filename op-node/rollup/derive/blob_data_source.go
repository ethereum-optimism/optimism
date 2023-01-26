package derive

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BlobDataSource fetches both call-data (backup) and blobs and transforms them into usable rollup data.
type BlobDataSource struct {
	open           bool
	callData       []eth.Data
	blobDataHashes []eth.IndexedDataHash

	blobs []*eth.Blob

	ref         eth.L1BlockRef
	batcherAddr common.Address

	cfg *rollup.Config

	fetcher      L1TransactionFetcher
	blobsFetcher L1BlobsFetcher
	log          log.Logger
}

// NewBlobDataSource creates a new blob-data source. It suppresses errors in fetching the L1 block if they occur.
// If there is an error, it will attempt to fetch the result on the next call to `Next`.
func NewBlobDataSource(ctx context.Context, log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher, blobsFetcher L1BlobsFetcher, ref eth.L1BlockRef, batcherAddr common.Address) DataIter {
	return &BlobDataSource{
		open:         false,
		ref:          ref,
		cfg:          cfg,
		fetcher:      fetcher,
		log:          log.New("origin", ref),
		batcherAddr:  batcherAddr,
		blobsFetcher: blobsFetcher,
	}
}

// Next returns the next piece of data if it has it. If the constructor failed, this
// will attempt to reinitialize itself. If it cannot find the block it returns a ResetError
// otherwise it returns a temporary error if fetching the block returns an error.
func (ds *BlobDataSource) Next(ctx context.Context) (eth.Data, error) {
	if !ds.open {
		if _, txs, err := ds.fetcher.InfoAndTxsByHash(ctx, ds.ref.Hash); err == nil {
			ds.open = true
			ds.callData, ds.blobDataHashes = BlobDataFromEVMTransactions(ds.cfg, ds.batcherAddr, txs, ds.log)
		} else if errors.Is(err, ethereum.NotFound) {
			return nil, NewResetError(fmt.Errorf("failed to open blob-data source: %w", err))
		} else {
			return nil, NewTemporaryError(fmt.Errorf("failed to open blob-data source: %w", err))
		}
	}
	// prioritize call-data
	if len(ds.callData) != 0 {
		data := ds.callData[0]
		ds.callData = ds.callData[1:]
		return data, nil
	}
	if len(ds.blobDataHashes) > 0 { // check if there is any blob data in this block we have opened.
		if ds.blobs == nil { // fetch blobs if we haven't already
			blobs, err := ds.blobsFetcher.BlobsByRefAndIndexedDataHashes(ctx, ds.ref, ds.blobDataHashes)
			if errors.Is(err, ethereum.NotFound) { // if the L1 block was seen to be available, then the blobs should also be available
				return nil, NewResetError(fmt.Errorf("failed to find blobs: %w", err))
			} else if err != nil {
				return nil, NewTemporaryError(fmt.Errorf("failed to fetch blobs: %w", err))
			}
			ds.blobs = blobs
		}
		if len(ds.blobs) > 0 { // parse the next blob, if any
			// TODO: can optimize this to get a bit more data from the blobs by using the top byte partially
			blob := ds.blobs[0]
			data := make(eth.Data, 4096*31)
			// field elems are big endian uint256 in BLS modulus range, thus can't full use the first byte each
			for i := 0; i < 4096; i++ {
				// All field elements are encoded big-endian, and the low-end byte cannot be fully used,
				// since the field-element doesn't cover the full 256 bit range.
				copy(data[i*31:i*31+31], blob[i*32+1:i*32+32])
			}
			// TODO better versioning of blob data extraction method
			datLen := binary.LittleEndian.Uint32(data[:4])
			data = data[4:]
			if datLen < uint32(len(data)) {
				data = data[:datLen]
			}
			ds.blobs = ds.blobs[1:]
			return data, nil
		}
	}
	return nil, io.EOF
}

// BlobDataFromEVMTransactions filters all of the transactions and returns the call-data and blob data-hashes
// from transactions that are sent to the batch inbox address from the batch sender address.
// This will return an empty array if no valid transactions are found.
// Call-data can be used as fallback in case blobs are overpriced or unstable.
func BlobDataFromEVMTransactions(config *rollup.Config, batcherAddr common.Address, txs types.Transactions, log log.Logger) ([]eth.Data, []eth.IndexedDataHash) {
	var callData []eth.Data
	var indexedDataHashes []eth.IndexedDataHash
	blobIndex := uint64(0)
	l1Signer := config.L1Signer()
	for j, tx := range txs {
		if to := tx.To(); to != nil && *to == config.BatchInboxAddress {
			seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				log.Warn("tx in inbox with invalid signature", "index", j, "err", err)
				blobIndex += uint64(len(tx.BlobHashes()))
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != batcherAddr {
				log.Warn("tx in inbox with unauthorized submitter", "index", j, "err", err)
				blobIndex += uint64(len(tx.BlobHashes()))
				continue // not an authorized batch submitter, ignore
			}
			if len(tx.Data()) > 0 { // ignore empty calldata
				callData = append(callData, tx.Data())
			}
			for _, h := range tx.BlobHashes() {
				indexedDataHashes = append(indexedDataHashes, eth.IndexedDataHash{
					Index:    blobIndex,
					DataHash: h,
				})
				blobIndex += 1
			}
		} else {
			blobIndex += uint64(len(tx.BlobHashes()))
		}
	}
	return callData, indexedDataHashes
}
