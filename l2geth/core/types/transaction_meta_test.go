package types

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var (
	addr          = common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	l1BlockNumber = big.NewInt(0)

	txMetaSerializationTests = []struct {
		l1BlockNumber  *big.Int
		l1Timestamp    uint64
		msgSender      *common.Address
		sighashType    SignatureHashType
		queueOrigin    QueueOrigin
		rawTransaction []byte
	}{
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    100,
			msgSender:      &addr,
			sighashType:    SighashEthSign,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{255, 255, 255, 255},
		},
		{
			l1BlockNumber:  nil,
			l1Timestamp:    45,
			msgSender:      &addr,
			sighashType:    SighashEthSign,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{42, 69, 42, 69},
		},
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    0,
			msgSender:      nil,
			sighashType:    SighashEthSign,
			queueOrigin:    QueueOriginSequencer,
			rawTransaction: []byte{0, 0, 0, 0},
		},
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    0,
			msgSender:      &addr,
			sighashType:    SighashEthSign,
			queueOrigin:    QueueOriginSequencer,
			rawTransaction: []byte{0, 0, 0, 0},
		},
		{
			l1BlockNumber:  nil,
			l1Timestamp:    0,
			msgSender:      nil,
			sighashType:    SighashEthSign,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{0, 0, 0, 0},
		},
		{
			l1BlockNumber:  l1BlockNumber,
			l1Timestamp:    0,
			msgSender:      &addr,
			sighashType:    SighashEthSign,
			queueOrigin:    QueueOriginL1ToL2,
			rawTransaction: []byte{0, 0, 0, 0},
		},
	}

	txMetaSighashEncodeTests = []struct {
		input  SignatureHashType
		output SignatureHashType
	}{
		{
			input:  SighashEIP155,
			output: SighashEIP155,
		},
		{
			input:  SighashEthSign,
			output: SighashEthSign,
		},
	}
)

func TestTransactionMetaEncode(t *testing.T) {
	for _, test := range txMetaSerializationTests {
		txmeta := NewTransactionMeta(test.l1BlockNumber, test.l1Timestamp, test.msgSender, test.sighashType, test.queueOrigin, nil, nil, test.rawTransaction)

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

func TestTransactionSighashEncode(t *testing.T) {
	for _, test := range txMetaSighashEncodeTests {
		txmeta := NewTransactionMeta(l1BlockNumber, 0, &addr, test.input, QueueOriginSequencer, nil, nil, nil)
		encoded := TxMetaEncode(txmeta)
		decoded, err := TxMetaDecode(encoded)

		if err != nil {
			t.Fatal(err)
		}

		if decoded.SignatureHashType != test.output {
			t.Fatal("SighashTypes do not match")
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

	if meta1.SignatureHashType != meta2.SignatureHashType {
		return false
	}

	if meta1.QueueOrigin == nil || meta2.QueueOrigin == nil {
		// Note: this only works because it is the final comparison
		if meta1.QueueOrigin == nil && meta2.QueueOrigin == nil {
			return true
		}
	}

	if !bytes.Equal(meta1.QueueOrigin.Bytes(), meta2.QueueOrigin.Bytes()) {
		return false
	}

	return true
}
