package derive

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

type DataIter interface {
	Next(ctx context.Context) (eth.Data, error)
}

type L1TransactionFetcher interface {
	// InfoAndRawTxsByHash returns the block's transactions in their EIP-2718 encoding, in block
	// order. The L1 of a chain can itself be an OP Stack chain, whose blocks hold deposit and
	// post-exec transactions that upstream go-ethereum cannot decode.
	InfoAndRawTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, []hexutil.Bytes, error)
}

type L1BlobsFetcher interface {
	// GetBlobsByHash fetches blobs that were confirmed at the given timestamp with the given versioned hashes.
	GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error)
}

type AltDAInputFetcher interface {
	// GetInput fetches the input for the given commitment at the given block number from the DA storage service.
	GetInput(ctx context.Context, l1 altda.L1Fetcher, c altda.CommitmentData, blockId eth.L1BlockRef) (eth.Data, error)
	// AdvanceL1Origin advances the L1 origin to the given block number, syncing the DA challenge events.
	AdvanceL1Origin(ctx context.Context, l1 altda.L1Fetcher, blockId eth.BlockID) error
	// Reset the challenge origin in case of L1 reorg
	Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error
}

// DataSourceFactory reads raw transactions from a given block & then filters for
// batch submitter transactions.
// This is not a stage in the pipeline, but a wrapper for another stage in the pipeline
type DataSourceFactory struct {
	log               log.Logger
	dsCfg             DataSourceConfig
	fetcher           L1Fetcher
	blobsFetcher      L1BlobsFetcher
	altDAFetcher      AltDAInputFetcher
	altDAMaxInputSize uint64
	ecotoneTime       *uint64
}

func NewDataSourceFactory(log log.Logger, cfg *rollup.Config, fetcher L1Fetcher, blobsFetcher L1BlobsFetcher, altDAFetcher AltDAInputFetcher) *DataSourceFactory {
	config := DataSourceConfig{
		l1Signer:          cfg.L1Signer(),
		batchInboxAddress: cfg.BatchInboxAddress,
		altDAEnabled:      cfg.AltDAEnabled(),
	}
	return &DataSourceFactory{
		log:               log,
		dsCfg:             config,
		fetcher:           fetcher,
		blobsFetcher:      blobsFetcher,
		altDAFetcher:      altDAFetcher,
		altDAMaxInputSize: cfg.AltDAConfig.MaxInputSizeOrDefault(),
		ecotoneTime:       cfg.EcotoneTime,
	}
}

// OpenData returns the appropriate data source for the L1 block `ref`.
func (ds *DataSourceFactory) OpenData(ctx context.Context, ref eth.L1BlockRef, batcherAddr common.Address) (DataIter, error) {
	// Creates a data iterator from blob or calldata source so we can forward it to the altDA source
	// if enabled as it still requires an L1 data source for fetching input commmitments.
	var src DataIter
	if ds.ecotoneTime != nil && ref.Time >= *ds.ecotoneTime {
		if ds.blobsFetcher == nil {
			return nil, NewCriticalError(fmt.Errorf("ecotone upgrade active but beacon endpoint not configured"))
		}
		src = NewBlobDataSource(ctx, ds.log, ds.dsCfg, ds.fetcher, ds.blobsFetcher, ref, batcherAddr)
	} else {
		src = NewCalldataSource(ctx, ds.log, ds.dsCfg, ds.fetcher, ref, batcherAddr)
	}
	if ds.dsCfg.altDAEnabled {
		// altDA([calldata | blobdata](l1Ref)) -> data
		return NewAltDADataSource(ds.log, src, ds.fetcher, ds.altDAFetcher, ds.altDAMaxInputSize, ref), nil
	}
	return src, nil
}

// DataSourceConfig regroups the mandatory rollup.Config fields needed for DataFromEVMTransactions.
type DataSourceConfig struct {
	l1Signer          types.Signer
	batchInboxAddress common.Address
	altDAEnabled      bool
}

// batcherTx is a valid batcher transaction of an L1 block.
type batcherTx struct {
	// tx is the transaction if it is a standard Ethereum transaction, and nil for a deposit.
	tx *types.Transaction
	// calldata is the transaction's input data.
	calldata eth.Data
}

// batcherTxs decodes the transactions of an L1 block, given in their EIP-2718 encoding, and
// returns its valid batcher transactions in block order. A standard Ethereum transaction is
// decoded with go-ethereum and checked by isValidBatchTx. When the L1 is an OP Stack chain, it
// also holds deposits, which batch submission can use to force-include a batcher transaction, and
// post-exec transactions, which are never batcher transactions. A transaction that does not decode
// fails the whole block.
func batcherTxs(rawTxs []hexutil.Bytes, dsCfg *DataSourceConfig, batcherAddr common.Address, logger log.Logger) ([]batcherTx, error) {
	var out []batcherTx
	for i, raw := range rawTxs {
		switch {
		case optypes.IsDepositTxBytes(raw):
			dep, err := optypes.UnmarshalDepositTx(raw)
			if err != nil {
				return nil, fmt.Errorf("failed to decode deposit tx %d: %w", i, err)
			}
			if isValidDepositBatchTx(dep, raw, dsCfg.batchInboxAddress, batcherAddr, logger) {
				out = append(out, batcherTx{calldata: dep.Data})
			}
		case optypes.IsPostExecTxBytes(raw):
			if _, err := optypes.UnmarshalPostExecTx(raw); err != nil {
				return nil, fmt.Errorf("failed to decode post-exec tx %d: %w", i, err)
			}
		default:
			tx := new(types.Transaction)
			if err := tx.UnmarshalBinary(raw); err != nil {
				return nil, fmt.Errorf("failed to decode tx %d: %w", i, err)
			}
			if isValidBatchTx(tx, dsCfg.l1Signer, dsCfg.batchInboxAddress, batcherAddr, logger) {
				out = append(out, batcherTx{tx: tx, calldata: tx.Data()})
			}
		}
	}
	return out, nil
}

// isValidBatchTx returns true if:
//  1. the transaction type is any of Legacy, ACL, DynamicFee, or Blob,
//  2. the transaction has a To() address that matches the batch inbox address, and
//  3. the transaction has a valid signature from the batcher address
func isValidBatchTx(tx *types.Transaction, l1Signer types.Signer, batchInboxAddr, batcherAddr common.Address, logger log.Logger) bool {
	// For now, we want to disallow the SetCodeTx type or any future types.
	if tx.Type() > types.BlobTxType {
		return false
	}

	to := tx.To()
	if to == nil || *to != batchInboxAddr {
		return false
	}
	seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
	if err != nil {
		logger.Warn("tx in inbox with invalid signature", "hash", tx.Hash(), "err", err)
		return false
	}
	// some random L1 user might have sent a transaction to our batch inbox, ignore them
	if seqDataSubmitter != batcherAddr {
		logger.Warn("tx in inbox with unauthorized submitter", "addr", seqDataSubmitter, "hash", tx.Hash(), "err", err)
		return false
	}
	return true
}

// isValidDepositBatchTx returns true if the deposit, with EIP-2718 encoding raw, is sent to the
// batch inbox address from the batcher address. A deposit carries its sender instead of a
// signature.
func isValidDepositBatchTx(dep *optypes.DepositTx, raw []byte, batchInboxAddr, batcherAddr common.Address, logger log.Logger) bool {
	if dep.To == nil || *dep.To != batchInboxAddr {
		return false
	}
	if dep.From != batcherAddr {
		logger.Warn("tx in inbox with unauthorized submitter", "addr", dep.From, "hash", crypto.Keccak256Hash(raw))
		return false
	}
	return true
}
