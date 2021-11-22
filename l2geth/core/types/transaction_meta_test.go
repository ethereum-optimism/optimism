package types

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/l2geth/common"
)

var (
	addr          = common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	l1BlockNumber = big.NewInt(0)

	txMetaSerializationTests = []struct {
		l1BlockNumber  *big.Int
		l1Timestamp    uint64
		msgSender      *common.Address
		queueOrigin    QueueOrigin
		rawTransaction []byte
	}{
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    100,
			msgSender:      &addr,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{255, 255, 255, 255},
		},
		{
			l1BlockNumber:  nil,
			l1Timestamp:    45,
			msgSender:      &addr,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{42, 69, 42, 69},
		},
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    0,
			msgSender:      nil,
			queueOrigin:    QueueOriginSequencer,
			rawTransaction: []byte{0, 0, 0, 0},
		},
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    0,
			msgSender:      &addr,
			queueOrigin:    QueueOriginSequencer,
			rawTransaction: []byte{0, 0, 0, 0},
		},
		{
			l1BlockNumber:  nil,
			l1Timestamp:    0,
			msgSender:      nil,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{0, 0, 0, 0},
		},
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    0,
			msgSender:      &addr,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{0, 0, 0, 0},
		},
	}
)

func TestTransactionMetaEncode(t *testing.T) {
	for _, test := range txMetaSerializationTests {
		txmeta := NewTransactionMeta(test.l1BlockNumber, test.l1Timestamp, test.msgSender, test.queueOrigin, nil, nil, test.rawTransaction)

		encoded := TxMetaEncode(txmeta)
		decoded, err := TxMetaDecode(encoded)

		if err != nil {
			t.Fatal(err)
		}

		if !isTxMetaEqual(txmeta, decoded) {
			t.Fatal("Encoding/decoding mismatch")
		}
	}
}

func isTxMetaEqual(meta1 *TransactionMeta, meta2 *TransactionMeta) bool {
	// Maybe can just return this
	if !reflect.DeepEqual(meta1, meta2) {
		return false
	}

	if meta1.L1Timestamp != meta2.L1Timestamp {
		return false
	}

	if meta1.L1MessageSender == nil || meta2.L1MessageSender == nil {
		if meta1.L1MessageSender != meta2.L1MessageSender {
			return false
		}
	} else {
		if !bytes.Equal(meta1.L1MessageSender.Bytes(), meta2.L1MessageSender.Bytes()) {
			return false
		}
	}

	if meta1.L1BlockNumber == nil || meta2.L1BlockNumber == nil {
		if meta1.L1BlockNumber != meta2.L1BlockNumber {
			return false
		}
	} else {
		if !bytes.Equal(meta1.L1BlockNumber.Bytes(), meta2.L1BlockNumber.Bytes()) {
			return false
		}
	}

	if meta1.QueueOrigin != meta2.QueueOrigin {
		return false
	}

	return true
}
