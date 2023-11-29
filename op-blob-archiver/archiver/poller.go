package archiver

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-blob-archiver/storage"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type PollerConfig struct {
	PollInterval   time.Duration
	NetworkTimeout time.Duration
}

// TODO - use these interfaces instead of the particular clients themselves
// type L1TransactionFetcher interface {
// 	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
// }

// type L1BlobsFetcher interface {
// 	// BlobsByRefAndIndexedDataHashes fetches blobs that were confirmed in the given L1 block with the given indexed hashes.
// 	BlobsByRefAndIndexedDataHashes(ctx context.Context, ref eth.L1BlockRef, dataHashes []eth.IndexedDataHash) ([]*eth.Blob, error)
// }

// capitalize or not?
type Poller struct {
	PollerConfig
	storage      storage.Storage
	beaconClient *sources.L1BeaconClient
	batcherAddr  common.Address
	L1Client     *ethclient.Client
	fetcher      derive.L1TransactionFetcher
	RollupConfig rollup.Config
	Log          log.Logger
}

// not sure if this needs ctx - are there other poller examples?
func NewPoller(
	log log.Logger,
	s storage.Storage,
	beaconClient *sources.L1BeaconClient,
	batcherAddr common.Address,
	l1Client *ethclient.Client,
	fetcher derive.L1TransactionFetcher,
	rollupConfig rollup.Config,
	pollInterval time.Duration,
) *Poller {
	// what about the config that checks if we're saving all blobs or just some
	return &Poller{
		Log:          log,
		storage:      s,
		beaconClient: beaconClient,
		batcherAddr:  batcherAddr,
		L1Client:     l1Client,
		fetcher:      fetcher,
		RollupConfig: rollupConfig,
		PollerConfig: PollerConfig{
			PollInterval: pollInterval,
		},
	}
}

/*
	pair
	- service setup
		- context
			- timeouts
		- aws
	- error handling

	todo:
	- logging? vs return error? return error
	- config set up

	done
	- frequency of running
	- implement calls to the eth client
	- superchain branching
	- include kzg proofs to storage as json
*/

func (p *Poller) Run() error {
	go func() {
		for {
			p.processBlocks(ctx)
			time.Sleep(p.PollerConfig.PollInterval)
		}
	}()
}

func (p *Poller) processBlocks(ctx context.Context) error {
	// get the hash of the block that was saved last
	latestSavedBlockHash, err := p.storage.GetLatestSavedBlockHash()
	if err != nil {
		// handle case where it's the service starting up - take a block to start from?
		return fmt.Errorf("failed to get latest saved block hash from storage")
	}
	latestSavedBlock, err := p.L1Client.BlockByHash(context.Background(), common.HexToHash(latestSavedBlockHash))
	if err != nil {
		return err
	}
	latestSavedBlockNumber := latestSavedBlock.Number().Uint64()

	// should these be wrapped with a timeout? like in func (l *BatchSubmitter) l1Tip(ctx context.Context) (eth.L1BlockRef, error) {
	currentBlockNumber, err := p.L1Client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number")
	}

	// 	for every unarchived block, i.e. those between the last saved block number and the current block number
	for i := latestSavedBlockNumber + 1; i <= currentBlockNumber; i++ {
		header, err := p.L1Client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(i))
		if err != nil {
			return err
		}

		blockRef := eth.InfoToL1BlockRef(eth.HeaderBlockInfo(header))

		_, txs, err := p.fetcher.InfoAndTxsByHash(ctx, blockRef.Hash)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				return nil, NewResetError(fmt.Errorf("failed to open blob-data source: %w", err))
			} else {
				return nil, NewTemporaryError(fmt.Errorf("failed to open blob-data source: %w", err))
			}
		}

		blobDataHashes := BlobDataFromEVMTransactions(&p.RollupConfig, p.batcherAddr, txs, p.Log)

		blobs, err := p.beaconClient.BlobsAndProofsByRefAndIndexedDataHashes(ctx, blockRef, blobDataHashes)
		// extract blobs[i].

		err = p.storage.SaveBlobs(blockRef.Hash, blobs)
		if err != nil {
			return fmt.Errorf("failed to save blobs to storage for block with hash %v", blockRef.Hash)
		}
	}
	return nil
}

// BlobDataFromEVMTransactions filters all of the transactions and returns the blob data-hashes
// from transactions. It optionally filters for tx sent to the batch inbox address from the batch sender address.
// This will return an empty array if no valid transactions are found.
// TODO - and remove rollup.Config to just get the l1 signer
func BlobDataFromEVMTransactions(config *rollup.Config, batcherAddr common.Address, txs types.Transactions, log log.Logger) []eth.IndexedDataHash {
	var indexedDataHashes []eth.IndexedDataHash
	blobIndex := uint64(0)
	l1Signer := config.L1Signer()
	for j, tx := range txs {
		to := tx.To()
		if to == nil || batcherAddr.String() != "" && *to != batcherAddr {
			blobIndex += uint64(len(tx.BlobHashes()))
			continue
		}

		if batcherAddr.String() != "" {
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
		}

		for _, h := range tx.BlobHashes() {
			indexedDataHashes = append(indexedDataHashes, eth.IndexedDataHash{
				Index:    blobIndex,
				DataHash: h,
			})
			blobIndex += 1
		}
	}
	return indexedDataHashes
}
