package derive

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

// Batch format
// first byte is type followed by bytestring.
//
// BatchV1Type := 0
// batchV1 := BatchV1Type ++ RLP([epoch, timestamp, transaction_list]
//
// An empty input is not a valid batch.
//
// Batch-bundle format
// first byte is type followed by bytestring
//
// payload := RLP([batch_0, batch_1, ..., batch_N])
// bundleV1 := BatchBundleV1Type ++ payload
// bundleV2 := BatchBundleV2Type ++ compress(payload)  # TODO: compressed bundle of batches
//
// An empty input is not a valid bundle.
//
// Note: the type system is based on L1 typed transactions.

// encodeBufferPool holds temporary encoder buffers for batch encoding
var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

const (
	BatchV1Type = iota
)

const (
	BatchBundleV1Type = iota
	BatchBundleV2Type
)

type BatchV1 struct {
	Epoch     rollup.Epoch // aka l1 num
	Timestamp uint64
	// no feeRecipient address input, all fees go to a L2 contract
	Transactions []hexutil.Bytes
}

type BatchData struct {
	BatchV1
	// batches may contain additional data with new upgrades
}

func DecodeBatches(config *rollup.Config, r io.Reader) ([]*BatchData, error) {
	var typeData [1]byte
	if _, err := io.ReadFull(r, typeData[:]); err != nil {
		return nil, fmt.Errorf("failed to read batch bundle type byte: %v", err)
	}
	switch typeData[0] {
	case BatchBundleV1Type:
		var out []*BatchData
		if err := rlp.Decode(r, &out); err != nil {
			return nil, fmt.Errorf("failed to decode v1 batches list: %v", err)
		}
		return out, nil
	case BatchBundleV2Type:
		// TODO: implement compression of a bundle of batches
		return nil, errors.New("bundle v2 not supported yet")
	default:
		return nil, fmt.Errorf("unrecognized batch bundle type: %d", typeData[0])
	}
}

func EncodeBatches(config *rollup.Config, batches []*BatchData, w io.Writer) error {
	// default to encode as v1 (no compression). Config may change this in the future.
	bundleType := byte(BatchBundleV1Type)

	if _, err := w.Write([]byte{bundleType}); err != nil {
		return fmt.Errorf("failed to encode batch type")
	}
	switch bundleType {
	case BatchBundleV1Type:
		if err := rlp.Encode(w, batches); err != nil {
			return fmt.Errorf("failed to encode RLP-list payload of v1 bundle: %v", err)
		}
		return nil
	case BatchBundleV2Type:
		return errors.New("bundle v2 not supported yet")
	default:
		return fmt.Errorf("unrecognized batch bundle type: %d", bundleType)
	}
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

func (b *BatchData) encodeTyped(buf *bytes.Buffer) error {
	buf.WriteByte(BatchV1Type)
	return rlp.Encode(buf, &b.BatchV1)
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

func (b *BatchData) decodeTyped(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("batch too short")
	}
	switch data[0] {
	case BatchV1Type:
		return rlp.DecodeBytes(data[1:], &b.BatchV1)
	default:
		return fmt.Errorf("unrecognized batch type: %d", data[0])
	}
}
