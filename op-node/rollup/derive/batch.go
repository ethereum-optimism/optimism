package derive

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// Batch format
// first byte is type followed by bytestring.
//
// An empty input is not a valid batch.
//
// Note: the type system is based on L1 typed transactions.
//
// encodeBufferPool holds temporary encoder buffers for batch encoding
var encodeBufferPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

const (
	// SingularBatchType is the first version of Batch format, representing a single L2 block.
	SingularBatchType = iota
	// SpanBatchType is the Batch version used after SpanBatch hard fork, representing a span of L2 blocks.
	SpanBatchType
)

// Batch contains information to build one or multiple L2 blocks.
// Batcher converts L2 blocks into Batch and writes encoded bytes to Channel.
// Derivation pipeline decodes Batch from Channel, and converts to one or multiple payload attributes.
type Batch interface {
	GetBatchType() int
	GetTimestamp() uint64
	LogContext(log.Logger) log.Logger
}

// BatchData is a composition type that contains raw data of each batch version.
// It has encoding & decoding methods to implement typed encoding.
type BatchData struct {
	BatchType int
	SingularBatch
	RawSpanBatch
}

// EncodeRLP implements rlp.Encoder
func (b *BatchData) EncodeRLP(w io.Writer) error {
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := b.encodeTyped(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// MarshalBinary returns the canonical encoding of the batch.
func (b *BatchData) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := b.encodeTyped(&buf)
	return buf.Bytes(), err
}

// encodeTyped encodes batch type and payload for each batch type.
func (b *BatchData) encodeTyped(buf *bytes.Buffer) error {
	switch b.BatchType {
	case SingularBatchType:
		buf.WriteByte(SingularBatchType)
		return rlp.Encode(buf, &b.SingularBatch)
	case SpanBatchType:
		buf.WriteByte(SpanBatchType)
		return b.RawSpanBatch.encode(buf)
	default:
		return fmt.Errorf("unrecognized batch type: %d", b.BatchType)
	}
}

// DecodeRLP implements rlp.Decoder
func (b *BatchData) DecodeRLP(s *rlp.Stream) error {
	if b == nil {
		return errors.New("cannot decode into nil BatchData")
	}
	v, err := s.Bytes()
	if err != nil {
		return err
	}
	return b.decodeTyped(v)
}

// UnmarshalBinary decodes the canonical encoding of batch.
func (b *BatchData) UnmarshalBinary(data []byte) error {
	if b == nil {
		return errors.New("cannot decode into nil BatchData")
	}
	return b.decodeTyped(data)
}

// decodeTyped decodes batch type and payload for each batch type.
func (b *BatchData) decodeTyped(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("batch too short")
	}
	switch data[0] {
	case SingularBatchType:
		b.BatchType = SingularBatchType
		return rlp.DecodeBytes(data[1:], &b.SingularBatch)
	case SpanBatchType:
		b.BatchType = int(data[0])
		return b.RawSpanBatch.decodeBytes(data[1:])
	default:
		return fmt.Errorf("unrecognized batch type: %d", data[0])
	}
}

// NewSingularBatchData creates new BatchData with SingularBatch
func NewSingularBatchData(singularBatch SingularBatch) *BatchData {
	return &BatchData{
		BatchType:     SingularBatchType,
		SingularBatch: singularBatch,
	}
}

// NewSpanBatchData creates new BatchData with SpanBatch
func NewSpanBatchData(spanBatch RawSpanBatch) *BatchData {
	return &BatchData{
		BatchType:    SpanBatchType,
		RawSpanBatch: spanBatch,
	}
}
