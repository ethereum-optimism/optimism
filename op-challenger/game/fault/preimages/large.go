package preimages

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var errNotSupported = errors.New("not supported")

var _ PreimageUploader = (*LargePreimageUploader)(nil)

// MaxBlocksPerChunk is the maximum number of keccak blocks per chunk.
const MaxBlocksPerChunk = 300

// MaxChunkSize is the maximum size of a preimage chunk in bytes.
// Notice, the max chunk size must be a multiple of the leaf size.
// The max chunk size is roughly 0.04MB to avoid memory expansion.
const MaxChunkSize = MaxBlocksPerChunk * keccakTypes.BlockSize

// LargePreimageUploader handles uploading large preimages by
// streaming the merkleized preimage to the PreimageOracle contract,
// tightly packed across multiple transactions.
type LargePreimageUploader struct {
	log log.Logger

	txMgr    txmgr.TxManager
	contract PreimageOracleContract
}

func NewLargePreimageUploader(logger log.Logger, txMgr txmgr.TxManager, contract PreimageOracleContract) *LargePreimageUploader {
	return &LargePreimageUploader{logger, txMgr, contract}
}

func (p *LargePreimageUploader) UploadPreimage(ctx context.Context, parent uint64, data *types.PreimageOracleData) error {
	calls, err := p.splitCalls(data)
	if err != nil {
		return fmt.Errorf("failed to split preimage into chunks for data with oracle offset %d: %w", data.OracleOffset, err)
	}

	uuid := p.newUUID(data)

	// Fetch the current metadata for this preimage data, if it exists.
	ident := keccakTypes.LargePreimageIdent{Claimant: p.txMgr.From(), UUID: uuid}
	metadata, err := p.contract.GetProposalMetadata(ctx, batching.BlockLatest, ident)
	if err != nil {
		return fmt.Errorf("failed to get pre-image oracle metadata: %w", err)
	}

	// The proposal is not initialized if the queried metadata has a claimed size of 0.
	if len(metadata) == 1 && metadata[0].ClaimedSize == 0 {
		err = p.initLargePreimage(ctx, uuid, data.OracleOffset, uint32(len(data.OracleData)))
		if err != nil {
			return fmt.Errorf("failed to initialize large preimage with uuid: %s: %w", uuid, err)
		}
	}

	// Filter out any chunks that have already been uploaded to the Preimage Oracle.
	if len(metadata) > 0 {
		numSkip := metadata[0].BytesProcessed / MaxChunkSize
		calls = calls[numSkip:]
		// If the timestamp is non-zero, the preimage has been finalized.
		if metadata[0].Timestamp != 0 {
			calls = calls[len(calls):]
		}
	}

	err = p.addLargePreimageData(ctx, uuid, calls)
	if err != nil {
		return fmt.Errorf("failed to add leaves to large preimage with uuid: %s: %w", uuid, err)
	}

	// todo(proofs#467): track the challenge period starting once the full preimage is posted.
	// todo(proofs#467): once the challenge period is over, call `squeezeLPP` on the preimage oracle contract.

	return errNotSupported
}

// newUUID generates a new unique identifier for the preimage by hashing the
// concatenated preimage data, preimage offset, and sender address.
func (p *LargePreimageUploader) newUUID(data *types.PreimageOracleData) *big.Int {
	sender := p.txMgr.From()
	offset := make([]byte, 4)
	binary.LittleEndian.PutUint32(offset, data.OracleOffset)
	concatenated := append(data.OracleData, offset...)
	concatenated = append(concatenated, sender.Bytes()...)
	hash := crypto.Keccak256Hash(concatenated)
	return hash.Big()
}

// splitChunks splits the preimage data into chunks of size [MaxChunkSize] (except the last chunk).
func (p *LargePreimageUploader) splitCalls(data *types.PreimageOracleData) ([]keccakTypes.InputData, error) {
	// Split the preimage data into chunks of size [MaxChunkSize] (except the last chunk).
	stateMatrix := matrix.NewStateMatrix()
	var calls []keccakTypes.InputData
	in := bytes.NewReader(data.OracleData)
	for {
		call, err := stateMatrix.AbsorbUpTo(in, MaxChunkSize)
		if errors.Is(err, io.EOF) {
			calls = append(calls, call)
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to absorb data: %w", err)
		}
		calls = append(calls, call)
	}
	return calls, nil
}

// initLargePreimage initializes the large preimage proposal.
// This method *must* be called before adding any leaves.
func (p *LargePreimageUploader) initLargePreimage(ctx context.Context, uuid *big.Int, partOffset uint32, claimedSize uint32) error {
	candidate, err := p.contract.InitLargePreimage(uuid, partOffset, claimedSize)
	if err != nil {
		return fmt.Errorf("failed to create pre-image oracle tx: %w", err)
	}
	if err := p.sendTxAndWait(ctx, candidate); err != nil {
		return fmt.Errorf("failed to populate pre-image oracle: %w", err)
	}
	return nil
}

// addLargePreimageData adds data to the large preimage proposal.
// This method **must** be called after calling [initLargePreimage].
// SAFETY: submits transactions in a [Queue] for latency while preserving submission order.
func (p *LargePreimageUploader) addLargePreimageData(ctx context.Context, uuid *big.Int, chunks []keccakTypes.InputData) error {
	queue := txmgr.NewQueue[int](ctx, p.txMgr, 10)
	receiptChs := make([]chan txmgr.TxReceipt[int], len(chunks))
	blocksProcessed := int64(0)
	for i, chunk := range chunks {
		tx, err := p.contract.AddLeaves(uuid, big.NewInt(blocksProcessed), chunk.Input, chunk.Commitments, chunk.Finalize)
		if err != nil {
			return fmt.Errorf("failed to create pre-image oracle tx: %w", err)
		}
		blocksProcessed += int64(len(chunk.Input) / keccakTypes.BlockSize)
		receiptChs[i] = make(chan txmgr.TxReceipt[int], 1)
		queue.Send(i, tx, receiptChs[i])
	}
	for _, receiptCh := range receiptChs {
		receipt := <-receiptCh
		if receipt.Err != nil {
			return receipt.Err
		}
		if receipt.Receipt.Status == ethtypes.ReceiptStatusFailed {
			p.log.Error("LargePreimageUploader add leafs tx successfully published but reverted", "tx_hash", receipt.Receipt.TxHash)
		} else {
			p.log.Debug("LargePreimageUploader add leafs tx successfully published", "tx_hash", receipt.Receipt.TxHash)
		}
	}
	return nil
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (p *LargePreimageUploader) sendTxAndWait(ctx context.Context, candidate txmgr.TxCandidate) error {
	receipt, err := p.txMgr.Send(ctx, candidate)
	if err != nil {
		return err
	}
	if receipt.Status == ethtypes.ReceiptStatusFailed {
		p.log.Error("LargePreimageUploader tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		p.log.Debug("LargePreimageUploader tx successfully published", "tx_hash", receipt.TxHash)
	}
	return nil
}
