package downloader

import (
	"bytes"
	"embed"
	"encoding/gob"
	"fmt"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// userDepositNonces is a struct to hold the reference data for user deposits
// The reference data is used to correct the deposit nonce in the receipts
type userDepositNonces struct {
	ChainID uint64
	First   uint64
	Last    uint64 // non inclusive
	Results map[uint64][]uint64
}

var (
	systemAddress        = common.HexToAddress("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001")
	receiptReferencePath = "userDepositData"
	//go:embed userDepositData/*.gob
	receiptReference                 embed.FS
	userDepositNoncesAlreadySearched = map[uint64]bool{}
	userDepositNoncesReference       = map[uint64]userDepositNonces{}
)

// lazy load the reference data for the requested chain
// if this chain data was already requested, returns early
func initReceiptReferences(chainID uint64) {
	// if already loaded, return
	if userDepositNoncesAlreadySearched[chainID] {
		return
	}
	// look for a file prefixed by the chainID
	fPrefix := fmt.Sprintf("%d.", chainID)
	files, err := receiptReference.ReadDir(receiptReferencePath)
	if err != nil {
		log.Warn("Receipt Correction: Failed to read reference directory", "err", err)
		return
	}
	// mark as loaded so we don't try again, even if no files match
	userDepositNoncesAlreadySearched[chainID] = true
	for _, file := range files {
		// skip files which don't match the prefix
		if !strings.HasPrefix(file.Name(), fPrefix) {
			continue
		}
		fpath := path.Join(receiptReferencePath, file.Name())
		bs, err := receiptReference.ReadFile(fpath)
		if err != nil {
			log.Warn("Receipt Correction: Failed to read reference data", "err", err)
			continue
		}
		udns := userDepositNonces{}
		err = gob.NewDecoder(bytes.NewReader(bs)).Decode(&udns)
		if err != nil {
			log.Warn("Receipt Correction: Failed to decode reference data", "err", err)
			continue
		}
		userDepositNoncesReference[udns.ChainID] = udns
		return
	}
}

// correctReceipts corrects the deposit nonce in the receipts using the reference data
// prior to Canyon Hard Fork, DepositNonces were not cryptographically verifiable.
// As a consequence, the deposit nonces found during Snap Sync may be incorrect.
// This function inspects transaction data for user deposits, and if it is found to be incorrect, it is corrected.
// The data used to correct the deposit nonce is stored in the userDepositData directory,
// and was generated with the receipt reference tool in the optimism monorepo.
func correctReceipts(receipts types.Receipts, transactions types.Transactions, blockNumber uint64, chainID uint64) types.Receipts {
	initReceiptReferences(chainID)
	// if there is no data even after initialization, return the receipts as is
	depositNoncesForChain, ok := userDepositNoncesReference[chainID]
	if !ok {
		log.Trace("Receipt Correction: No data source for chain", "chainID", chainID)
		return receipts
	}

	// check that the block number being examined is within the range of the reference data
	if blockNumber < depositNoncesForChain.First || blockNumber > depositNoncesForChain.Last {
		log.Trace("Receipt Correction: Block is out of range for receipt reference",
			"blockNumber", blockNumber,
			"start", depositNoncesForChain.First,
			"end", depositNoncesForChain.Last)
		return receipts
	}

	// get the block nonces
	blockNonces, ok := depositNoncesForChain.Results[blockNumber]
	if !ok {
		log.Trace("Receipt Correction: Block does not contain user deposits", "blockNumber", blockNumber)
		return receipts
	}

	// iterate through the receipts and transactions to correct the deposit nonce
	// user deposits should always be at the front of the block, but we will check all transactions to be sure
	udCount := 0
	for i := 0; i < len(receipts); i++ {
		r := receipts[i]
		tx := transactions[i]
		from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		if err != nil {
			log.Warn("Receipt Correction: Failed to determine sender", "err", err)
			continue
		}
		// break as soon as a non deposit is found
		if r.Type != types.DepositTxType {
			break
		}
		// ignore any transactions from the system address
		if from != systemAddress {
			// prevent index out of range (indicates a problem with the reference data or the block data)
			if udCount >= len(blockNonces) {
				log.Warn("Receipt Correction: More user deposits in block than included in reference data", "in_reference", len(blockNonces))
				break
			}
			nonce := blockNonces[udCount]
			udCount++
			log.Trace("Receipt Correction: User Deposit detected", "address", from, "nonce", nonce)
			if nonce != *r.DepositNonce {
				// correct the deposit nonce
				// warn because this should not happen unless the data was modified by corruption or a malicious peer
				// by correcting the nonce, the entire block is still valid for use
				log.Warn("Receipt Correction: Corrected deposit nonce", "nonce", *r.DepositNonce, "corrected", nonce)
				r.DepositNonce = &nonce
			}
		}
	}
	// check for unused reference data (indicates a problem with the reference data or the block data)
	if udCount < len(blockNonces) {
		log.Warn("Receipt Correction: More user deposits in reference data than found in block", "in_reference", len(blockNonces), "in_block", udCount)
	}

	log.Trace("Receipt Correction: Completed", "blockNumber", blockNumber, "userDeposits", udCount, "receipts", len(receipts), "transactions", len(transactions))
	return receipts
}
