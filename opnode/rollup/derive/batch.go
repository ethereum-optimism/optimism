package derive

import (
	"bytes"
	"errors"

	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

// Batch format
// first byte is type followed by bytstring.
//
// BatchV1Type := 0
// BatchV1Type ++ RLP([epoch, timestamp, transaction_list]

const (
	BatchV1Type = iota
)

type BatchV1 struct {
	Epoch        uint64
	Timestamp    uint64
	Transactions []hexutil.Bytes
}

type BatchData struct {
	Epoch     rollup.Epoch // aka l1 num
	Timestamp uint64
	// no feeRecipient address input, all fees go to a L2 contract
	Transactions []l2.Data
}

func ParseBatch(data []byte) (BatchData, error) {
	var v1 BatchV1
	if err := v1.UnmarshalBinary(data); err != nil {
		return BatchData{}, err
	}
	return BatchData{Epoch: rollup.Epoch(v1.Epoch), Timestamp: v1.Timestamp, Transactions: v1.Transactions}, nil
}

// // TODO: Is this needed?
// // EncodeRLP implements rlp.Encoder
// func (b *BatchV1) EncodeRLP(w io.Writer) error {
// 	return rlp.Encode(w, b)
// }

// MarshalBinary returns the canonical encoding of the batch.
func (b *BatchV1) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(BatchV1Type)
	err := rlp.Encode(&buf, b)
	return buf.Bytes(), err
}

// // TODO: Is this needed?
// // DecodeRLP implements rlp.Decoder
// func (b *BatchV1) DecodeRLP(s *rlp.Stream) error {
// 	return s.Decode(b)
// }

// UnmarshalBinary decodes the canonical encoding of batch.
func (batch *BatchV1) UnmarshalBinary(b []byte) error {
	if len(b) == 0 {
		return errors.New("Batch too short")
	}
	return rlp.DecodeBytes(b[1:], batch)
}
