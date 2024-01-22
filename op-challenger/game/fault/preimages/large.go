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
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var errNotSupported = errors.New("not supported")

var _ PreimageUploader = (*LargePreimageUploader)(nil)

// MaxLeafsPerChunk is the maximum number of leafs per chunk.
const MaxLeafsPerChunk = 300

// MaxChunkSize is the maximum size of a preimage chunk in bytes.
// Notice, the max chunk size must be a multiple of the leaf size.
// The max chunk size is roughly 0.04MB to avoid memory expansion.
const MaxChunkSize = MaxLeafsPerChunk * matrix.LeafSize

// Chunk is a contigous segment of preimage data.
type Chunk struct {
	// Input is the preimage data.
	Input []byte
	// Commitments are the keccak commitments for each leaf in the chunk.
	Commitments [][32]byte
	// Finalize indicates whether the chunk is the final chunk.
	Finalize bool
}

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
	chunks, err := p.splitChunks(data)
	if err != nil {
		return fmt.Errorf("failed to split preimage into chunks for data with oracle offset %d: %w", data.OracleOffset, err)
	}

	uuid := p.newUUID(data)

	// Fetch the current metadata for this preimage data, if it exists.
	ident := gameTypes.LargePreimageIdent{Claimant: p.txMgr.From(), UUID: uuid}
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
		chunks = chunks[numSkip:]
		// If the timestamp is non-zero, the preimage has been finalized.
		if metadata[0].Timestamp != 0 {
			chunks = chunks[len(chunks):]
		}
	}

	err = p.addLargePreimageLeafs(ctx, uuid, chunks)
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
func (p *LargePreimageUploader) splitChunks(data *types.PreimageOracleData) ([]Chunk, error) {
	stateMatrix := matrix.NewStateMatrix()
	chunk := make([]byte, 0, MaxChunkSize)
	chunks := []Chunk{}
	commitments := make([][32]byte, 0, MaxLeafsPerChunk)
	in := bytes.NewReader(data.OracleData)
	for i := 0; ; i++ {
		// Absorb the next preimage chunk leaf and run the keccak permutation.
		leaf, err := stateMatrix.AbsorbNextLeaf(in)
		chunk = append(chunk, leaf...)
		commitments = append(commitments, stateMatrix.StateCommitment())
		// SAFETY: the last leaf will always return an [io.EOF] error from [AbsorbNextLeaf].
		if errors.Is(err, io.EOF) {
			chunks = append(chunks, Chunk{chunk, commitments[:], true})
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to absorb leaf: %w", err)
		}

		// Only create a call if the chunk is full.
		if len(chunk) >= MaxChunkSize {
			chunks = append(chunks, Chunk{chunk, commitments[:], false})
			chunk = make([]byte, 0, MaxChunkSize)
			commitments = make([][32]byte, 0, MaxLeafsPerChunk)
		}
	}
	return chunks, nil
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

// addLargePreimageLeafs adds leafs to the large preimage proposal.
// This method **must** be called after calling [initLargePreimage].
// SAFETY: submits transactions in a [Queue] for latency while preserving submission order.
func (p *LargePreimageUploader) addLargePreimageLeafs(ctx context.Context, uuid *big.Int, chunks []Chunk) error {
	queue := txmgr.NewQueue[int](ctx, p.txMgr, 10)
	receiptChs := make([]chan txmgr.TxReceipt[int], len(chunks))
	for i, chunk := range chunks {
		tx, err := p.contract.AddLeaves(uuid, chunk.Input, chunk.Commitments, chunk.Finalize)
		if err != nil {
			return fmt.Errorf("failed to create pre-image oracle tx: %w", err)
		}
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
