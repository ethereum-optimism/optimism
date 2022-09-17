package derive

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type BlobsSidecar struct {
	BeaconBlockRoot common.Hash    `json:"beacon_block_root"`
	BeaconBlockSlot common.Hash    `json:"beacon_block_slot"`
	Blobs           []types.Blob   `json:"blobs"`
	AggregatedProof types.KZGProof `json:"kzg_aggregated_proof"`
}

type BlobsSidecarFetcher interface {
	FetchSidecar(ctx context.Context, slot uint64) (*BlobsSidecar, error)
}

type BlobdataSource struct {
	log          log.Logger
	cfg          *rollup.Config
	txFetcher    L1TransactionFetcher
	blobsFetcher BlobsSidecarFetcher
}

func NewBlobdataSource(log log.Logger, cfg *rollup.Config, txFetcher L1TransactionFetcher, blobsFetcher BlobsSidecarFetcher) *BlobdataSource {
	return &BlobdataSource{log: log, cfg: cfg, txFetcher: txFetcher, blobsFetcher: blobsFetcher}
}

func (cs *BlobdataSource) OpenData(ctx context.Context, id eth.BlockID) (DataIter, error) {
	info, txs, err := cs.txFetcher.InfoAndTxsByHash(ctx, id.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Based on the timestamp we check which slot is being used
	l1GenesisTime := uint64(12434)
	blockTime := uint64(12)
	slot := (info.Time() - l1GenesisTime) / blockTime

	sidecar, err := cs.blobsFetcher.FetchSidecar(ctx, slot)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blobs sidecar for slot %d: %w", slot, err)
	}
	// compute kzg commitments and versioned hashes of each blob in the block
	var computedHashes []common.Hash
	for i, bl := range sidecar.Blobs {
		commitment, ok := bl.ComputeCommitment()
		if !ok {
			return nil, fmt.Errorf("failed compute blob commitment for blob %d", i)
		}
		computedHashes = append(computedHashes, commitment.ComputeVersionedHash())
	}
	// check which blobs are sent into our inbox

	var expectedHashes []common.Hash
	for _, tx := range txs {
		if tx.Type() == types.BlobTxType {
			expectedHashes = append(expectedHashes, tx.BlobVersionedHashes()...)
		}
	}
	if len(computedHashes) != len(expectedHashes) {
		return nil, fmt.Errorf("got %d data hashes for beacon block %s (%d), but expected %d hashes from blob txs in execution block %s",
			len(computedHashes), sidecar.BeaconBlockRoot, sidecar.BeaconBlockSlot, len(expectedHashes), id)
	}
	for i, h := range computedHashes {
		if h != expectedHashes[i] {
			return nil, fmt.Errorf("data hash mismatch %d (beacon block %s, slot %d): hash %d does not match expected hash from el block %s: %s",
				i, sidecar.BeaconBlockRoot, sidecar.BeaconBlockSlot, h, id, expectedHashes[i])
		}
	}

	l1Signer := cs.cfg.L1Signer()

	allBlobs := sidecar.Blobs
	outBlobs := allBlobs[:]
	for txi, tx := range txs {
		if tx.Type() != types.BlobTxType {
			continue
		}
		blobsInTx := len(tx.BlobVersionedHashes())
		txBlobs := allBlobs[:blobsInTx]
		allBlobs = allBlobs[blobsInTx:]
		// check if the tx is to our inbox, and sent by the correct batch submitter.
		// if yes, then add the blob to the output
		if to := tx.To(); to != nil && *to == cs.cfg.BatchInboxAddress {
			seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				log.Warn("tx in inbox with invalid signature", "index", txi, "err", err)
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != cs.cfg.BatchSenderAddress {
				log.Warn("tx in inbox with unauthorized submitter", "index", txi, "err", err)
				continue // not an authorized batch submitter, ignore
			}
			outBlobs = append(outBlobs, txBlobs...)
		}
	}
	var out []eth.Data
	for _, b := range outBlobs {
		out = append(out, blobToData(&b))
	}
	return (*DataSlice)(&out), nil
}

func blobToData(blob *types.Blob) eth.Data {
	var out eth.Data
	// can be more optimized, packing bits etc. for a marginal amount of more data
	for _, p := range blob {
		out = append(out, p[:31]...) // strip the last byte (little endian, last byte is largest). Since BLS scalars don't cover full 32 bytes, but do 31 bytes.
	}
	return out
}
