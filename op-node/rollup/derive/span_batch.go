package derive

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Batch format
//
// SpanBatchType := 1
// spanBatch := SpanBatchType ++ prefix ++ payload
// prefix := rel_timestamp ++ l1_origin_num ++ parent_check ++ l1_origin_check
// payload := block_count ++ origin_bits ++ block_tx_counts ++ txs
// txs := contract_creation_bits ++ y_parity_bits ++ tx_sigs ++ tx_tos ++ tx_datas ++ tx_nonces ++ tx_gases ++ protected_bits

var ErrTooBigSpanBatchSize = errors.New("span batch size limit reached")

var ErrEmptySpanBatch = errors.New("span-batch must not be empty")

type spanBatchPrefix struct {
	relTimestamp  uint64   // Relative timestamp of the first block
	l1OriginNum   uint64   // L1 origin number
	parentCheck   [20]byte // First 20 bytes of the first block's parent hash
	l1OriginCheck [20]byte // First 20 bytes of the last block's L1 origin hash
}

type spanBatchPayload struct {
	blockCount    uint64        // Number of L2 block in the span
	originBits    *big.Int      // Standard span-batch bitlist of blockCount bits. Each bit indicates if the L1 origin is changed at the L2 block.
	blockTxCounts []uint64      // List of transaction counts for each L2 block
	txs           *spanBatchTxs // Transactions encoded in SpanBatch specs
}

// RawSpanBatch is another representation of SpanBatch, that encodes data according to SpanBatch specs.
type RawSpanBatch struct {
	spanBatchPrefix
	spanBatchPayload
}

// GetBatchType returns its batch type (batch_version)
func (b *RawSpanBatch) GetBatchType() int {
	return SpanBatchType
}

// decodeOriginBits parses data into bp.originBits
func (bp *spanBatchPayload) decodeOriginBits(r *bytes.Reader) error {
	if bp.blockCount > MaxSpanBatchElementCount {
		return ErrTooBigSpanBatchSize
	}
	bits, err := decodeSpanBatchBits(r, bp.blockCount)
	if err != nil {
		return fmt.Errorf("failed to decode origin bits: %w", err)
	}
	bp.originBits = bits
	return nil
}

// decodeRelTimestamp parses data into bp.relTimestamp
func (bp *spanBatchPrefix) decodeRelTimestamp(r *bytes.Reader) error {
	relTimestamp, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("failed to read rel timestamp: %w", err)
	}
	bp.relTimestamp = relTimestamp
	return nil
}

// decodeL1OriginNum parses data into bp.l1OriginNum
func (bp *spanBatchPrefix) decodeL1OriginNum(r *bytes.Reader) error {
	L1OriginNum, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("failed to read l1 origin num: %w", err)
	}
	bp.l1OriginNum = L1OriginNum
	return nil
}

// decodeParentCheck parses data into bp.parentCheck
func (bp *spanBatchPrefix) decodeParentCheck(r *bytes.Reader) error {
	_, err := io.ReadFull(r, bp.parentCheck[:])
	if err != nil {
		return fmt.Errorf("failed to read parent check: %w", err)
	}
	return nil
}

// decodeL1OriginCheck parses data into bp.decodeL1OriginCheck
func (bp *spanBatchPrefix) decodeL1OriginCheck(r *bytes.Reader) error {
	_, err := io.ReadFull(r, bp.l1OriginCheck[:])
	if err != nil {
		return fmt.Errorf("failed to read l1 origin check: %w", err)
	}
	return nil
}

// decodePrefix parses data into bp.spanBatchPrefix
func (bp *spanBatchPrefix) decodePrefix(r *bytes.Reader) error {
	if err := bp.decodeRelTimestamp(r); err != nil {
		return err
	}
	if err := bp.decodeL1OriginNum(r); err != nil {
		return err
	}
	if err := bp.decodeParentCheck(r); err != nil {
		return err
	}
	if err := bp.decodeL1OriginCheck(r); err != nil {
		return err
	}
	return nil
}

// decodeBlockCount parses data into bp.blockCount
func (bp *spanBatchPayload) decodeBlockCount(r *bytes.Reader) error {
	blockCount, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("failed to read block count: %w", err)
	}
	// number of L2 block in span batch cannot be greater than MaxSpanBatchElementCount
	if blockCount > MaxSpanBatchElementCount {
		return ErrTooBigSpanBatchSize
	}
	if blockCount == 0 {
		return ErrEmptySpanBatch
	}
	bp.blockCount = blockCount
	return nil
}

// decodeBlockTxCounts parses data into bp.blockTxCounts
// and sets bp.txs.totalBlockTxCount as sum(bp.blockTxCounts)
func (bp *spanBatchPayload) decodeBlockTxCounts(r *bytes.Reader) error {
	var blockTxCounts []uint64
	for i := 0; i < int(bp.blockCount); i++ {
		blockTxCount, err := binary.ReadUvarint(r)
		if err != nil {
			return fmt.Errorf("failed to read block tx count: %w", err)
		}
		// number of txs in single L2 block cannot be greater than MaxSpanBatchElementCount
		// every tx will take at least single byte
		if blockTxCount > MaxSpanBatchElementCount {
			return ErrTooBigSpanBatchSize
		}
		blockTxCounts = append(blockTxCounts, blockTxCount)
	}
	bp.blockTxCounts = blockTxCounts
	return nil
}

// decodeTxs parses data into bp.txs
func (bp *spanBatchPayload) decodeTxs(r *bytes.Reader) error {
	if bp.txs == nil {
		bp.txs = &spanBatchTxs{}
	}
	if bp.blockTxCounts == nil {
		return errors.New("failed to read txs: blockTxCounts not set")
	}
	totalBlockTxCount := uint64(0)
	for i := 0; i < len(bp.blockTxCounts); i++ {
		total, overflow := math.SafeAdd(totalBlockTxCount, bp.blockTxCounts[i])
		if overflow {
			return ErrTooBigSpanBatchSize
		}
		totalBlockTxCount = total
	}
	// total number of txs in span batch cannot be greater than MaxSpanBatchElementCount
	if totalBlockTxCount > MaxSpanBatchElementCount {
		return ErrTooBigSpanBatchSize
	}
	bp.txs.totalBlockTxCount = totalBlockTxCount
	if err := bp.txs.decode(r); err != nil {
		return err
	}
	return nil
}

// decodePayload parses data into bp.spanBatchPayload
func (bp *spanBatchPayload) decodePayload(r *bytes.Reader) error {
	if err := bp.decodeBlockCount(r); err != nil {
		return err
	}
	if err := bp.decodeOriginBits(r); err != nil {
		return err
	}
	if err := bp.decodeBlockTxCounts(r); err != nil {
		return err
	}
	if err := bp.decodeTxs(r); err != nil {
		return err
	}
	return nil
}

// decode reads the byte encoding of SpanBatch from Reader stream
func (b *RawSpanBatch) decode(r *bytes.Reader) error {
	if err := b.decodePrefix(r); err != nil {
		return fmt.Errorf("failed to decode span batch prefix: %w", err)
	}
	if err := b.decodePayload(r); err != nil {
		return fmt.Errorf("failed to decode span batch payload: %w", err)
	}
	return nil
}

// encodeRelTimestamp encodes bp.relTimestamp
func (bp *spanBatchPrefix) encodeRelTimestamp(w io.Writer) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], bp.relTimestamp)
	if _, err := w.Write(buf[:n]); err != nil {
		return fmt.Errorf("cannot write rel timestamp: %w", err)
	}
	return nil
}

// encodeL1OriginNum encodes bp.l1OriginNum
func (bp *spanBatchPrefix) encodeL1OriginNum(w io.Writer) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], bp.l1OriginNum)
	if _, err := w.Write(buf[:n]); err != nil {
		return fmt.Errorf("cannot write l1 origin number: %w", err)
	}
	return nil
}

// encodeParentCheck encodes bp.parentCheck
func (bp *spanBatchPrefix) encodeParentCheck(w io.Writer) error {
	if _, err := w.Write(bp.parentCheck[:]); err != nil {
		return fmt.Errorf("cannot write parent check: %w", err)
	}
	return nil
}

// encodeL1OriginCheck encodes bp.l1OriginCheck
func (bp *spanBatchPrefix) encodeL1OriginCheck(w io.Writer) error {
	if _, err := w.Write(bp.l1OriginCheck[:]); err != nil {
		return fmt.Errorf("cannot write l1 origin check: %w", err)
	}
	return nil
}

// encodePrefix encodes spanBatchPrefix
func (bp *spanBatchPrefix) encodePrefix(w io.Writer) error {
	if err := bp.encodeRelTimestamp(w); err != nil {
		return err
	}
	if err := bp.encodeL1OriginNum(w); err != nil {
		return err
	}
	if err := bp.encodeParentCheck(w); err != nil {
		return err
	}
	if err := bp.encodeL1OriginCheck(w); err != nil {
		return err
	}
	return nil
}

// encodeOriginBits encodes bp.originBits
func (bp *spanBatchPayload) encodeOriginBits(w io.Writer) error {
	if err := encodeSpanBatchBits(w, bp.blockCount, bp.originBits); err != nil {
		return fmt.Errorf("failed to encode origin bits: %w", err)
	}
	return nil
}

// encodeBlockCount encodes bp.blockCount
func (bp *spanBatchPayload) encodeBlockCount(w io.Writer) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], bp.blockCount)
	if _, err := w.Write(buf[:n]); err != nil {
		return fmt.Errorf("cannot write block count: %w", err)
	}
	return nil
}

// encodeBlockTxCounts encodes bp.blockTxCounts
func (bp *spanBatchPayload) encodeBlockTxCounts(w io.Writer) error {
	var buf [binary.MaxVarintLen64]byte
	for _, blockTxCount := range bp.blockTxCounts {
		n := binary.PutUvarint(buf[:], blockTxCount)
		if _, err := w.Write(buf[:n]); err != nil {
			return fmt.Errorf("cannot write block tx count: %w", err)
		}
	}
	return nil
}

// encodeTxs encodes bp.txs
func (bp *spanBatchPayload) encodeTxs(w io.Writer) error {
	if bp.txs == nil {
		return errors.New("cannot write txs: txs not set")
	}
	if err := bp.txs.encode(w); err != nil {
		return err
	}
	return nil
}

// encodePayload encodes spanBatchPayload
func (bp *spanBatchPayload) encodePayload(w io.Writer) error {
	if err := bp.encodeBlockCount(w); err != nil {
		return err
	}
	if err := bp.encodeOriginBits(w); err != nil {
		return err
	}
	if err := bp.encodeBlockTxCounts(w); err != nil {
		return err
	}
	if err := bp.encodeTxs(w); err != nil {
		return err
	}
	return nil
}

// encode writes the byte encoding of SpanBatch to Writer stream
func (b *RawSpanBatch) encode(w io.Writer) error {
	if err := b.encodePrefix(w); err != nil {
		return err
	}
	if err := b.encodePayload(w); err != nil {
		return err
	}
	return nil
}

// derive converts RawSpanBatch into SpanBatch, which has a list of SpanBatchElement.
// We need chain config constants to derive values for making payload attributes.
func (b *RawSpanBatch) derive(blockTime, genesisTimestamp uint64, chainID *big.Int) (*SpanBatch, error) {
	if b.blockCount == 0 {
		return nil, ErrEmptySpanBatch
	}
	blockOriginNums := make([]uint64, b.blockCount)
	l1OriginBlockNumber := b.l1OriginNum
	for i := int(b.blockCount) - 1; i >= 0; i-- {
		blockOriginNums[i] = l1OriginBlockNumber
		if b.originBits.Bit(i) == 1 && i > 0 {
			l1OriginBlockNumber--
		}
	}

	if err := b.txs.recoverV(chainID); err != nil {
		return nil, err
	}
	fullTxs, err := b.txs.fullTxs(chainID)
	if err != nil {
		return nil, err
	}

	spanBatch := SpanBatch{
		ParentCheck:   b.parentCheck,
		L1OriginCheck: b.l1OriginCheck,
	}
	txIdx := 0
	for i := 0; i < int(b.blockCount); i++ {
		batch := SpanBatchElement{}
		batch.Timestamp = genesisTimestamp + b.relTimestamp + blockTime*uint64(i)
		batch.EpochNum = rollup.Epoch(blockOriginNums[i])
		for j := 0; j < int(b.blockTxCounts[i]); j++ {
			batch.Transactions = append(batch.Transactions, fullTxs[txIdx])
			txIdx++
		}
		spanBatch.Batches = append(spanBatch.Batches, &batch)
	}
	return &spanBatch, nil
}

// ToSpanBatch converts RawSpanBatch to SpanBatch,
// which implements a wrapper of derive method of RawSpanBatch
func (b *RawSpanBatch) ToSpanBatch(blockTime, genesisTimestamp uint64, chainID *big.Int) (*SpanBatch, error) {
	spanBatch, err := b.derive(blockTime, genesisTimestamp, chainID)
	if err != nil {
		return nil, err
	}
	return spanBatch, nil
}

// SpanBatchElement is a derived form of input to build a L2 block.
// similar to SingularBatch, but does not have ParentHash and EpochHash
// because Span batch spec does not contain parent hash and epoch hash of every block in the span.
type SpanBatchElement struct {
	EpochNum     rollup.Epoch // aka l1 num
	Timestamp    uint64
	Transactions []hexutil.Bytes
}

// singularBatchToElement converts a SingularBatch to a SpanBatchElement
func singularBatchToElement(singularBatch *SingularBatch) *SpanBatchElement {
	return &SpanBatchElement{
		EpochNum:     singularBatch.EpochNum,
		Timestamp:    singularBatch.Timestamp,
		Transactions: singularBatch.Transactions,
	}
}

// SpanBatch is an implementation of Batch interface,
// containing the input to build a span of L2 blocks in derived form (SpanBatchElement)
type SpanBatch struct {
	ParentCheck      [20]byte // First 20 bytes of the first block's parent hash
	L1OriginCheck    [20]byte // First 20 bytes of the last block's L1 origin hash
	GenesisTimestamp uint64
	ChainID          *big.Int
	Batches          []*SpanBatchElement // List of block input in derived form

	// caching
	originBits    *big.Int
	blockTxCounts []uint64
	sbtxs         *spanBatchTxs
}

func (b *SpanBatch) AsSingularBatch() (*SingularBatch, bool) { return nil, false }
func (b *SpanBatch) AsSpanBatch() (*SpanBatch, bool)         { return b, true }

// spanBatchMarshaling is a helper type used for JSON marshaling.
type spanBatchMarshaling struct {
	ParentCheck   []hexutil.Bytes     `json:"parent_check"`
	L1OriginCheck []hexutil.Bytes     `json:"l1_origin_check"`
	Batches       []*SpanBatchElement `json:"span_batch_elements"`
}

func (b *SpanBatch) MarshalJSON() ([]byte, error) {
	spanBatch := spanBatchMarshaling{
		ParentCheck:   []hexutil.Bytes{b.ParentCheck[:]},
		L1OriginCheck: []hexutil.Bytes{b.L1OriginCheck[:]},
		Batches:       b.Batches,
	}
	return json.Marshal(spanBatch)
}

// GetBatchType returns its batch type (batch_version)
func (b *SpanBatch) GetBatchType() int {
	return SpanBatchType
}

// GetTimestamp returns timestamp of the first block in the span
func (b *SpanBatch) GetTimestamp() uint64 {
	return b.Batches[0].Timestamp
}

// TxCount returns the tx count for the batch
func (b *SpanBatch) TxCount() (count uint64) {
	for _, txCount := range b.blockTxCounts {
		count += txCount
	}
	return
}

// LogContext creates a new log context that contains information of the batch
func (b *SpanBatch) LogContext(log log.Logger) log.Logger {
	if len(b.Batches) == 0 {
		return log.New("block_count", 0)
	}
	return log.New(
		"batch_type", "SpanBatch",
		"batch_timestamp", b.Batches[0].Timestamp,
		"parent_check", hexutil.Encode(b.ParentCheck[:]),
		"origin_check", hexutil.Encode(b.L1OriginCheck[:]),
		"start_epoch_number", b.GetStartEpochNum(),
		"end_epoch_number", b.GetBlockEpochNum(len(b.Batches)-1),
		"block_count", len(b.Batches),
		"txs", b.TxCount(),
	)
}

// GetStartEpochNum returns epoch number(L1 origin block number) of the first block in the span
func (b *SpanBatch) GetStartEpochNum() rollup.Epoch {
	return b.Batches[0].EpochNum
}

// CheckOriginHash checks if the l1OriginCheck matches the first 20 bytes of given hash, probably L1 block hash from the current canonical L1 chain.
func (b *SpanBatch) CheckOriginHash(hash common.Hash) bool {
	return bytes.Equal(b.L1OriginCheck[:], hash.Bytes()[:20])
}

// CheckParentHash checks if the parentCheck matches the first 20 bytes of given hash, probably the current L2 safe head.
func (b *SpanBatch) CheckParentHash(hash common.Hash) bool {
	return bytes.Equal(b.ParentCheck[:], hash.Bytes()[:20])
}

// GetBlockEpochNum returns the epoch number(L1 origin block number) of the block at the given index in the span.
func (b *SpanBatch) GetBlockEpochNum(i int) uint64 {
	return uint64(b.Batches[i].EpochNum)
}

// GetBlockTimestamp returns the timestamp of the block at the given index in the span.
func (b *SpanBatch) GetBlockTimestamp(i int) uint64 {
	return b.Batches[i].Timestamp
}

// GetBlockTransactions returns the encoded transactions of the block at the given index in the span.
func (b *SpanBatch) GetBlockTransactions(i int) []hexutil.Bytes {
	return b.Batches[i].Transactions
}

// GetBlockCount returns the number of blocks in the span
func (b *SpanBatch) GetBlockCount() int {
	return len(b.Batches)
}

func (b *SpanBatch) peek(n int) *SpanBatchElement { return b.Batches[len(b.Batches)-1-n] }

// AppendSingularBatch appends a SingularBatch into the span batch
// updates l1OriginCheck or parentCheck if needed.
func (b *SpanBatch) AppendSingularBatch(singularBatch *SingularBatch, seqNum uint64) error {
	// if this new element is not ordered with respect to the last element, panic
	if len(b.Batches) > 0 && b.peek(0).Timestamp > singularBatch.Timestamp {
		panic("span batch is not ordered")
	}

	// always append the new batch and set the L1 origin check
	b.Batches = append(b.Batches, singularBatchToElement(singularBatch))

	// always update the L1 origin check
	copy(b.L1OriginCheck[:], singularBatch.EpochHash.Bytes()[:20])
	// if there is only one batch, initialize the ParentCheck
	// and set the epochBit based on the seqNum
	epochBit := uint(0)
	if len(b.Batches) == 1 {
		if seqNum == 0 {
			epochBit = 1
		}
		copy(b.ParentCheck[:], singularBatch.ParentHash.Bytes()[:20])
	} else {
		// if there is more than one batch, set the epochBit based on the last two batches
		if b.peek(1).EpochNum < b.peek(0).EpochNum {
			epochBit = 1
		}
	}
	// set the respective bit in the originBits
	b.originBits.SetBit(b.originBits, len(b.Batches)-1, epochBit)

	// update the blockTxCounts cache with the latest batch's tx count
	b.blockTxCounts = append(b.blockTxCounts, uint64(len(b.peek(0).Transactions)))

	// add the new txs to the sbtxs
	newTxs := make([][]byte, 0, len(b.peek(0).Transactions))
	for i := 0; i < len(b.peek(0).Transactions); i++ {
		newTxs = append(newTxs, b.peek(0).Transactions[i])
	}
	// add the new txs to the sbtxs
	// this is the only place where we can get an error
	return b.sbtxs.AddTxs(newTxs, b.ChainID)
}

// ToRawSpanBatch merges SingularBatch List and initialize single RawSpanBatch
func (b *SpanBatch) ToRawSpanBatch() (*RawSpanBatch, error) {
	if len(b.Batches) == 0 {
		return nil, errors.New("cannot merge empty singularBatch list")
	}
	span_start := b.Batches[0]
	span_end := b.Batches[len(b.Batches)-1]

	return &RawSpanBatch{
		spanBatchPrefix: spanBatchPrefix{
			relTimestamp:  span_start.Timestamp - b.GenesisTimestamp,
			l1OriginNum:   uint64(span_end.EpochNum),
			parentCheck:   b.ParentCheck,
			l1OriginCheck: b.L1OriginCheck,
		},
		spanBatchPayload: spanBatchPayload{
			blockCount:    uint64(len(b.Batches)),
			originBits:    b.originBits,
			blockTxCounts: b.blockTxCounts,
			txs:           b.sbtxs,
		},
	}, nil
}

// GetSingularBatches converts SpanBatchElements after L2 safe head to SingularBatches.
// Since SpanBatchElement does not contain EpochHash, set EpochHash from the given L1 blocks.
// The result SingularBatches do not contain ParentHash yet. It must be set by BatchQueue.
func (b *SpanBatch) GetSingularBatches(l1Origins []eth.L1BlockRef, l2SafeHead eth.L2BlockRef) ([]*SingularBatch, error) {
	var singularBatches []*SingularBatch
	originIdx := 0
	for _, batch := range b.Batches {
		if batch.Timestamp <= l2SafeHead.Time {
			continue
		}
		singularBatch := SingularBatch{
			EpochNum:     batch.EpochNum,
			Timestamp:    batch.Timestamp,
			Transactions: batch.Transactions,
		}
		originFound := false
		for i := originIdx; i < len(l1Origins); i++ {
			if l1Origins[i].Number == uint64(batch.EpochNum) {
				originIdx = i
				singularBatch.EpochHash = l1Origins[i].Hash
				originFound = true
				break
			}
		}
		if !originFound {
			return nil, fmt.Errorf("unable to find L1 origin for the epoch number: %d", batch.EpochNum)
		}
		singularBatches = append(singularBatches, &singularBatch)
	}
	return singularBatches, nil
}

// NewSpanBatch converts given singularBatches into SpanBatchElements, and creates a new SpanBatch.
func NewSpanBatch(genesisTimestamp uint64, chainID *big.Int) *SpanBatch {
	// newSpanBatchTxs can't fail with empty txs
	sbtxs, _ := newSpanBatchTxs([][]byte{}, chainID)
	return &SpanBatch{
		GenesisTimestamp: genesisTimestamp,
		ChainID:          chainID,
		originBits:       big.NewInt(0),
		sbtxs:            sbtxs,
	}
}

// DeriveSpanBatch derives SpanBatch from BatchData.
func DeriveSpanBatch(batchData *BatchData, blockTime, genesisTimestamp uint64, chainID *big.Int) (*SpanBatch, error) {
	rawSpanBatch, ok := batchData.inner.(*RawSpanBatch)
	if !ok {
		return nil, NewCriticalError(errors.New("failed type assertion to SpanBatch"))
	}
	// If the batch type is Span batch, derive block inputs from RawSpanBatch.
	return rawSpanBatch.ToSpanBatch(blockTime, genesisTimestamp, chainID)
}

// ReadTxData reads raw RLP tx data from reader and returns txData and txType
func ReadTxData(r *bytes.Reader) ([]byte, int, error) {
	var txData []byte
	offset, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to seek tx reader: %w", err)
	}
	b, err := r.ReadByte()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read tx initial byte: %w", err)
	}
	txType := byte(0)
	if int(b) <= 0x7F {
		// EIP-2718: non legacy tx so write tx type
		txType = byte(b)
		txData = append(txData, txType)
	} else {
		// legacy tx: seek back single byte to read prefix again
		_, err = r.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to seek tx reader: %w", err)
		}
	}
	// avoid out of memory before allocation
	s := rlp.NewStream(r, MaxSpanBatchElementCount)
	var txPayload []byte
	kind, _, err := s.Kind()
	switch {
	case err != nil:
		if errors.Is(err, rlp.ErrValueTooLarge) {
			return nil, 0, ErrTooBigSpanBatchSize
		}
		return nil, 0, fmt.Errorf("failed to read tx RLP prefix: %w", err)
	case kind == rlp.List:
		if txPayload, err = s.Raw(); err != nil {
			return nil, 0, fmt.Errorf("failed to read tx RLP payload: %w", err)
		}
	default:
		return nil, 0, errors.New("tx RLP prefix type must be list")
	}
	txData = append(txData, txPayload...)
	return txData, int(txType), nil
}
