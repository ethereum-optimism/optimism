package derive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// Batch format
//
// SpanBatchType := 1
// spanBatch := SpanBatchType ++ prefix ++ payload
// prefix := rel_timestamp ++ l1_origin_num ++ parent_check ++ l1_origin_check
// payload := block_count ++ origin_bits ++ block_tx_counts ++ txs
// txs := contract_creation_bits ++ y_parity_bits ++ tx_sigs ++ tx_tos ++ tx_datas ++ tx_nonces ++ tx_gases

var ErrTooBigSpanBatchFieldSize = errors.New("batch would cause field bytes to go over limit")

type spanBatchPrefix struct {
	relTimestamp  uint64 // Relative timestamp of the first block
	l1OriginNum   uint64 // L1 origin number
	parentCheck   []byte // First 20 bytes of the first block's parent hash
	l1OriginCheck []byte // First 20 bytes of the last block's L1 origin hash
}

type spanBatchPayload struct {
	blockCount    uint64        // Number of L2 block in the span
	originBits    *big.Int      // Bitlist of blockCount bits. Each bit indicates if the L1 origin is changed at the L2 block.
	blockTxCounts []uint64      // List of transaction counts for each L2 block
	txs           *spanBatchTxs // Transactions encoded in SpanBatch specs
}

// RawSpanBatch is another representation of SpanBatch, that encodes data according to SpanBatch specs.
type RawSpanBatch struct {
	spanBatchPrefix
	spanBatchPayload
}

// decodeOriginBits parses data into bp.originBits
// originBits is bitlist right-padded to a multiple of 8 bits
func (bp *spanBatchPayload) decodeOriginBits(r *bytes.Reader) error {
	originBitBufferLen := bp.blockCount / 8
	if bp.blockCount%8 != 0 {
		originBitBufferLen++
	}
	// avoid out of memory before allocation
	if originBitBufferLen > MaxSpanBatchFieldSize {
		return ErrTooBigSpanBatchFieldSize
	}
	originBitBuffer := make([]byte, originBitBufferLen)
	_, err := io.ReadFull(r, originBitBuffer)
	if err != nil {
		return fmt.Errorf("failed to read origin bits: %w", err)
	}
	originBits := new(big.Int)
	for i := 0; i < int(bp.blockCount); i += 8 {
		end := i + 8
		if end < int(bp.blockCount) {
			end = int(bp.blockCount)
		}
		bits := originBitBuffer[i/8]
		for j := i; j < end; j++ {
			bit := uint((bits >> (j - i)) & 1)
			originBits.SetBit(originBits, j, bit)
		}
	}
	bp.originBits = originBits
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
	bp.parentCheck = make([]byte, 20)
	_, err := io.ReadFull(r, bp.parentCheck)
	if err != nil {
		return fmt.Errorf("failed to read parent check: %w", err)
	}
	return nil
}

// decodeL1OriginCheck parses data into bp.decodeL1OriginCheck
func (bp *spanBatchPrefix) decodeL1OriginCheck(r *bytes.Reader) error {
	bp.l1OriginCheck = make([]byte, 20)
	_, err := io.ReadFull(r, bp.l1OriginCheck)
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
	bp.blockCount = blockCount
	if err != nil {
		return fmt.Errorf("failed to read block count: %w", err)
	}
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
		totalBlockTxCount += bp.blockTxCounts[i]
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

// decodeBytes parses data into b from data
func (b *RawSpanBatch) decodeBytes(data []byte) error {
	r := bytes.NewReader(data)
	if err := b.decodePrefix(r); err != nil {
		return err
	}
	if err := b.decodePayload(r); err != nil {
		return err
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
	if _, err := w.Write(bp.parentCheck); err != nil {
		return fmt.Errorf("cannot write parent check: %w", err)
	}
	return nil
}

// encodeL1OriginCheck encodes bp.l1OriginCheck
func (bp *spanBatchPrefix) encodeL1OriginCheck(w io.Writer) error {
	if _, err := w.Write(bp.l1OriginCheck); err != nil {
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
// originBits is bitlist right-padded to a multiple of 8 bits
func (bp *spanBatchPayload) encodeOriginBits(w io.Writer) error {
	originBitBufferLen := bp.blockCount / 8
	if bp.blockCount%8 != 0 {
		originBitBufferLen++
	}
	originBitBuffer := make([]byte, originBitBufferLen)
	for i := 0; i < int(bp.blockCount); i += 8 {
		end := i + 8
		if end < int(bp.blockCount) {
			end = int(bp.blockCount)
		}
		var bits uint = 0
		for j := i; j < end; j++ {
			bits |= bp.originBits.Bit(j) << (j - i)
		}
		originBitBuffer[i/8] = byte(bits)
	}
	if _, err := w.Write(originBitBuffer); err != nil {
		return fmt.Errorf("cannot write origin bits: %w", err)
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

// encodeBytes returns the byte encoding of SpanBatch
func (b *RawSpanBatch) encodeBytes() ([]byte, error) {
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := b.encode(buf); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

// derive converts RawSpanBatch into SpanBatch, which has a list of spanBatchElement.
// We need chain config constants to derive values for making payload attributes.
func (b *RawSpanBatch) derive(blockTime, genesisTimestamp uint64, chainID *big.Int) (*SpanBatch, error) {
	blockOriginNums := make([]uint64, b.blockCount)
	l1OriginBlockNumber := b.l1OriginNum
	for i := int(b.blockCount) - 1; i >= 0; i-- {
		blockOriginNums[i] = l1OriginBlockNumber
		if b.originBits.Bit(i) == 1 && i > 0 {
			l1OriginBlockNumber--
		}
	}

	b.txs.recoverV(chainID)
	fullTxs, err := b.txs.fullTxs(chainID)
	if err != nil {
		return nil, err
	}

	spanBatch := SpanBatch{
		parentCheck:   b.parentCheck,
		l1OriginCheck: b.l1OriginCheck,
	}
	txIdx := 0
	for i := 0; i < int(b.blockCount); i++ {
		batch := spanBatchElement{}
		batch.Timestamp = genesisTimestamp + b.relTimestamp + blockTime*uint64(i)
		batch.EpochNum = rollup.Epoch(blockOriginNums[i])
		for j := 0; j < int(b.blockTxCounts[i]); j++ {
			batch.Transactions = append(batch.Transactions, fullTxs[txIdx])
			txIdx++
		}
		spanBatch.batches = append(spanBatch.batches, &batch)
	}
	return &spanBatch, nil
}

// spanBatchElement is a derived form of input to build a L2 block.
// similar to SingularBatch, but does not have ParentHash and EpochHash
// because Span batch spec does not contain parent hash and epoch hash of every block in the span.
type spanBatchElement struct {
	EpochNum     rollup.Epoch // aka l1 num
	Timestamp    uint64
	Transactions []hexutil.Bytes
}

// singularBatchToElement converts a SingularBatch to a spanBatchElement
func singularBatchToElement(singularBatch *SingularBatch) *spanBatchElement {
	return &spanBatchElement{
		EpochNum:     singularBatch.EpochNum,
		Timestamp:    singularBatch.Timestamp,
		Transactions: singularBatch.Transactions,
	}
}

// SpanBatch is an implementation of Batch interface,
// containing the input to build a span of L2 blocks in derived form (spanBatchElement)
type SpanBatch struct {
	parentCheck   []byte              // First 20 bytes of the first block's parent hash
	l1OriginCheck []byte              // First 20 bytes of the last block's L1 origin hash
	batches       []*spanBatchElement // List of block input in derived form
}

// GetBatchType returns its batch type (batch_version)
func (b *SpanBatch) GetBatchType() int {
	return SpanBatchType
}

// GetTimestamp returns timestamp of the first block in the span
func (b *SpanBatch) GetTimestamp() uint64 {
	return b.batches[0].Timestamp
}

// LogContext creates a new log context that contains information of the batch
func (b *SpanBatch) LogContext(log log.Logger) log.Logger {
	if len(b.batches) == 0 {
		return log.New("block_count", 0)
	}
	return log.New(
		"batch_timestamp", b.batches[0].Timestamp,
		"parent_check", hexutil.Encode(b.parentCheck),
		"origin_check", hexutil.Encode(b.l1OriginCheck),
		"start_epoch_number", b.GetStartEpochNum(),
		"end_epoch_number", b.GetBlockEpochNum(len(b.batches)-1),
		"block_count", len(b.batches),
	)
}

// GetStartEpochNum returns epoch number(L1 origin block number) of the first block in the span
func (b *SpanBatch) GetStartEpochNum() rollup.Epoch {
	return b.batches[0].EpochNum
}

// CheckOriginHash checks if the l1OriginCheck matches the first 20 bytes of given hash, probably L1 block hash from the current canonical L1 chain.
func (b *SpanBatch) CheckOriginHash(hash common.Hash) bool {
	return bytes.Equal(b.l1OriginCheck, hash.Bytes()[:20])
}

// CheckParentHash checks if the parentCheck matches the first 20 bytes of given hash, probably the current L2 safe head.
func (b *SpanBatch) CheckParentHash(hash common.Hash) bool {
	return bytes.Equal(b.parentCheck, hash.Bytes()[:20])
}

// GetBlockEpochNum returns the epoch number(L1 origin block number) of the block at the given index in the span.
func (b *SpanBatch) GetBlockEpochNum(i int) uint64 {
	return uint64(b.batches[i].EpochNum)
}

// GetBlockTimestamp returns the timestamp of the block at the given index in the span.
func (b *SpanBatch) GetBlockTimestamp(i int) uint64 {
	return b.batches[i].Timestamp
}

// GetBlockTransactions returns the encoded transactions of the block at the given index in the span.
func (b *SpanBatch) GetBlockTransactions(i int) []hexutil.Bytes {
	return b.batches[i].Transactions
}

// GetBlockCount returns the number of blocks in the span
func (b *SpanBatch) GetBlockCount() int {
	return len(b.batches)
}

// AppendSingularBatch appends a SingularBatch into the span batch
// updates l1OriginCheck or parentCheck if needed.
func (b *SpanBatch) AppendSingularBatch(singularBatch *SingularBatch) {
	if len(b.batches) == 0 {
		b.parentCheck = singularBatch.ParentHash.Bytes()[:20]
	}
	b.batches = append(b.batches, singularBatchToElement(singularBatch))
	b.l1OriginCheck = singularBatch.EpochHash.Bytes()[:20]
}

// ToRawSpanBatch merges SingularBatch List and initialize single RawSpanBatch
func (b *SpanBatch) ToRawSpanBatch(originChangedBit uint, genesisTimestamp uint64, chainID *big.Int) (*RawSpanBatch, error) {
	if len(b.batches) == 0 {
		return nil, errors.New("cannot merge empty singularBatch list")
	}
	raw := RawSpanBatch{}
	// Sort by timestamp of L2 block
	sort.Slice(b.batches, func(i, j int) bool {
		return b.batches[i].Timestamp < b.batches[j].Timestamp
	})
	// spanBatchPrefix
	span_start := b.batches[0]
	span_end := b.batches[len(b.batches)-1]
	raw.relTimestamp = span_start.Timestamp - genesisTimestamp
	raw.l1OriginNum = uint64(span_end.EpochNum)
	raw.parentCheck = make([]byte, 20)
	copy(raw.parentCheck, b.parentCheck)
	raw.l1OriginCheck = make([]byte, 20)
	copy(raw.l1OriginCheck, b.l1OriginCheck)
	// spanBatchPayload
	raw.blockCount = uint64(len(b.batches))
	raw.originBits = new(big.Int)
	raw.originBits.SetBit(raw.originBits, 0, originChangedBit)
	for i := 1; i < len(b.batches); i++ {
		bit := uint(0)
		if b.batches[i-1].EpochNum < b.batches[i].EpochNum {
			bit = 1
		}
		raw.originBits.SetBit(raw.originBits, i, bit)
	}
	var blockTxCounts []uint64
	var txs [][]byte
	for _, batch := range b.batches {
		blockTxCount := uint64(len(batch.Transactions))
		blockTxCounts = append(blockTxCounts, blockTxCount)
		for _, rawTx := range batch.Transactions {
			txs = append(txs, rawTx)
		}
	}
	raw.blockTxCounts = blockTxCounts
	stxs, err := newSpanBatchTxs(txs, chainID)
	if err != nil {
		return nil, err
	}
	raw.txs = stxs
	return &raw, nil
}

// GetSingularBatches converts spanBatchElements after L2 safe head to SingularBatches.
// Since spanBatchElement does not contain EpochHash, set EpochHash from the given L1 blocks.
// The result SingularBatches do not contain ParentHash yet. It must be set by BatchQueue.
func (b *SpanBatch) GetSingularBatches(l1Origins []eth.L1BlockRef, l2SafeHead eth.L2BlockRef) ([]*SingularBatch, error) {
	var singularBatches []*SingularBatch
	originIdx := 0
	for _, batch := range b.batches {
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

// NewSpanBatch converts given singularBatches into spanBatchElements, and creates a new SpanBatch.
func NewSpanBatch(singularBatches []*SingularBatch) *SpanBatch {
	if len(singularBatches) == 0 {
		return &SpanBatch{}
	}
	spanBatch := SpanBatch{
		parentCheck:   singularBatches[0].ParentHash.Bytes()[:20],
		l1OriginCheck: singularBatches[len(singularBatches)-1].EpochHash.Bytes()[:20],
	}
	for _, singularBatch := range singularBatches {
		spanBatch.batches = append(spanBatch.batches, singularBatchToElement(singularBatch))
	}
	return &spanBatch
}

// SpanBatchBuilder is a utility type to build a SpanBatch by adding a SingularBatch one by one.
// makes easier to stack SingularBatches and convert to RawSpanBatch for encoding.
type SpanBatchBuilder struct {
	parentEpoch      uint64
	genesisTimestamp uint64
	chainID          *big.Int
	spanBatch        *SpanBatch
}

func NewSpanBatchBuilder(parentEpoch uint64, genesisTimestamp uint64, chainID *big.Int) *SpanBatchBuilder {
	return &SpanBatchBuilder{
		parentEpoch:      parentEpoch,
		genesisTimestamp: genesisTimestamp,
		chainID:          chainID,
		spanBatch:        &SpanBatch{},
	}
}

func (b *SpanBatchBuilder) AppendSingularBatch(singularBatch *SingularBatch) {
	b.spanBatch.AppendSingularBatch(singularBatch)
}

func (b *SpanBatchBuilder) GetRawSpanBatch() (*RawSpanBatch, error) {
	originChangedBit := 0
	if uint64(b.spanBatch.GetStartEpochNum()) != b.parentEpoch {
		originChangedBit = 1
	}
	raw, err := b.spanBatch.ToRawSpanBatch(uint(originChangedBit), b.genesisTimestamp, b.chainID)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (b *SpanBatchBuilder) GetBlockCount() int {
	return len(b.spanBatch.batches)
}

func (b *SpanBatchBuilder) Reset() {
	b.spanBatch = &SpanBatch{}
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
	s := rlp.NewStream(r, MaxSpanBatchFieldSize)
	var txPayload []byte
	kind, _, err := s.Kind()
	switch {
	case err != nil:
		if errors.Is(err, rlp.ErrValueTooLarge) {
			return nil, 0, ErrTooBigSpanBatchFieldSize
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
