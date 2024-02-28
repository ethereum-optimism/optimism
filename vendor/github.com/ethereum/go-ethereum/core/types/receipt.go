// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type Receipt -field-override receiptMarshaling -out gen_receipt_json.go

var (
	receiptStatusFailedRLP     = []byte{}
	receiptStatusSuccessfulRLP = []byte{0x01}
)

var errShortTypedReceipt = errors.New("typed receipt too short")

const (
	// ReceiptStatusFailed is the status code of a transaction if execution failed.
	ReceiptStatusFailed = uint64(0)

	// ReceiptStatusSuccessful is the status code of a transaction if execution succeeded.
	ReceiptStatusSuccessful = uint64(1)

	// The version number for post-canyon deposit receipts.
	CanyonDepositReceiptVersion = uint64(1)
)

// Receipt represents the results of a transaction.
type Receipt struct {
	// Consensus fields: These fields are defined by the Yellow Paper
	Type              uint8  `json:"type,omitempty"`
	PostState         []byte `json:"root"`
	Status            uint64 `json:"status"`
	CumulativeGasUsed uint64 `json:"cumulativeGasUsed" gencodec:"required"`
	Bloom             Bloom  `json:"logsBloom"         gencodec:"required"`
	Logs              []*Log `json:"logs"              gencodec:"required"`

	// Implementation fields: These fields are added by geth when processing a transaction or retrieving a receipt.
	// gencodec annotated fields: these are stored in the chain database.
	TxHash            common.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress   common.Address `json:"contractAddress"`
	GasUsed           uint64         `json:"gasUsed" gencodec:"required"`
	EffectiveGasPrice *big.Int       `json:"effectiveGasPrice"` // required, but tag omitted for backwards compatibility
	BlobGasUsed       uint64         `json:"blobGasUsed,omitempty"`
	BlobGasPrice      *big.Int       `json:"blobGasPrice,omitempty"`

	// DepositNonce was introduced in Regolith to store the actual nonce used by deposit transactions
	// The state transition process ensures this is only set for Regolith deposit transactions.
	DepositNonce *uint64 `json:"depositNonce,omitempty"`
	// DepositReceiptVersion was introduced in Canyon to indicate an update to how receipt hashes
	// should be computed when set. The state transition process ensures this is only set for
	// post-Canyon deposit transactions.
	DepositReceiptVersion *uint64 `json:"depositReceiptVersion,omitempty"`

	// Inclusion information: These fields provide information about the inclusion of the
	// transaction corresponding to this receipt.
	BlockHash        common.Hash `json:"blockHash,omitempty"`
	BlockNumber      *big.Int    `json:"blockNumber,omitempty"`
	TransactionIndex uint        `json:"transactionIndex"`

	// OVM legacy: extend receipts with their L1 price (if a rollup tx)
	L1GasPrice *big.Int   `json:"l1GasPrice,omitempty"`
	L1GasUsed  *big.Int   `json:"l1GasUsed,omitempty"`
	L1Fee      *big.Int   `json:"l1Fee,omitempty"`
	FeeScalar  *big.Float `json:"l1FeeScalar,omitempty"` // always nil after Ecotone hardfork
}

type receiptMarshaling struct {
	Type              hexutil.Uint64
	PostState         hexutil.Bytes
	Status            hexutil.Uint64
	CumulativeGasUsed hexutil.Uint64
	GasUsed           hexutil.Uint64
	EffectiveGasPrice *hexutil.Big
	BlobGasUsed       hexutil.Uint64
	BlobGasPrice      *hexutil.Big
	BlockNumber       *hexutil.Big
	TransactionIndex  hexutil.Uint

	// Optimism
	L1GasPrice            *hexutil.Big
	L1GasUsed             *hexutil.Big
	L1Fee                 *hexutil.Big
	FeeScalar             *big.Float
	DepositNonce          *hexutil.Uint64
	DepositReceiptVersion *hexutil.Uint64
}

// receiptRLP is the consensus encoding of a receipt.
type receiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             Bloom
	Logs              []*Log
}

type depositReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             Bloom
	Logs              []*Log
	// DepositNonce was introduced in Regolith to store the actual nonce used by deposit transactions.
	// Must be nil for any transactions prior to Regolith or that aren't deposit transactions.
	DepositNonce *uint64 `rlp:"optional"`
	// Receipt hash post-Regolith but pre-Canyon inadvertently did not include the above
	// DepositNonce. Post Canyon, receipts will have a non-empty DepositReceiptVersion indicating
	// which post-Canyon receipt hash function to invoke.
	DepositReceiptVersion *uint64 `rlp:"optional"`
}

// storedReceiptRLP is the storage encoding of a receipt.
type storedReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Logs              []*Log
	// DepositNonce was introduced in Regolith to store the actual nonce used by deposit transactions.
	// Must be nil for any transactions prior to Regolith or that aren't deposit transactions.
	DepositNonce *uint64 `rlp:"optional"`
	// Receipt hash post-Regolith but pre-Canyon inadvertently did not include the above
	// DepositNonce. Post Canyon, receipts will have a non-empty DepositReceiptVersion indicating
	// which post-Canyon receipt hash function to invoke.
	DepositReceiptVersion *uint64 `rlp:"optional"`
}

// LegacyOptimismStoredReceiptRLP is the pre bedrock storage encoding of a
// receipt. It will only exist in the database if it was migrated using the
// migration tool. Nodes that sync using snap-sync will not have any of these
// entries.
type LegacyOptimismStoredReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Logs              []*LogForStorage
	L1GasUsed         *big.Int
	L1GasPrice        *big.Int
	L1Fee             *big.Int
	FeeScalar         string
}

// LogForStorage is a wrapper around a Log that handles
// backward compatibility with prior storage formats.
type LogForStorage Log

// EncodeRLP implements rlp.Encoder.
func (l *LogForStorage) EncodeRLP(w io.Writer) error {
	rl := Log{Address: l.Address, Topics: l.Topics, Data: l.Data}
	return rlp.Encode(w, &rl)
}

type legacyRlpStorageLog struct {
	Address     common.Address
	Topics      []common.Hash
	Data        []byte
	BlockNumber uint64
	TxHash      common.Hash
	TxIndex     uint
	BlockHash   common.Hash
	Index       uint
}

// DecodeRLP implements rlp.Decoder.
//
// Note some redundant fields(e.g. block number, tx hash etc) will be assembled later.
func (l *LogForStorage) DecodeRLP(s *rlp.Stream) error {
	blob, err := s.Raw()
	if err != nil {
		return err
	}
	var dec Log
	err = rlp.DecodeBytes(blob, &dec)
	if err == nil {
		*l = LogForStorage{
			Address: dec.Address,
			Topics:  dec.Topics,
			Data:    dec.Data,
		}
	} else {
		// Try to decode log with previous definition.
		var dec legacyRlpStorageLog
		err = rlp.DecodeBytes(blob, &dec)
		if err == nil {
			*l = LogForStorage{
				Address: dec.Address,
				Topics:  dec.Topics,
				Data:    dec.Data,
			}
		}
	}
	return err
}

// NewReceipt creates a barebone transaction receipt, copying the init fields.
// Deprecated: create receipts using a struct literal instead.
func NewReceipt(root []byte, failed bool, cumulativeGasUsed uint64) *Receipt {
	r := &Receipt{
		Type:              LegacyTxType,
		PostState:         common.CopyBytes(root),
		CumulativeGasUsed: cumulativeGasUsed,
	}
	if failed {
		r.Status = ReceiptStatusFailed
	} else {
		r.Status = ReceiptStatusSuccessful
	}
	return r
}

// EncodeRLP implements rlp.Encoder, and flattens the consensus fields of a receipt
// into an RLP stream. If no post state is present, byzantium fork is assumed.
func (r *Receipt) EncodeRLP(w io.Writer) error {
	data := &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs}
	if r.Type == LegacyTxType {
		return rlp.Encode(w, data)
	}
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := r.encodeTyped(data, buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// encodeTyped writes the canonical encoding of a typed receipt to w.
func (r *Receipt) encodeTyped(data *receiptRLP, w *bytes.Buffer) error {
	w.WriteByte(r.Type)
	switch r.Type {
	case DepositTxType:
		withNonce := &depositReceiptRLP{data.PostStateOrStatus, data.CumulativeGasUsed, data.Bloom, data.Logs, r.DepositNonce, r.DepositReceiptVersion}
		return rlp.Encode(w, withNonce)
	default:
		return rlp.Encode(w, data)
	}
}

// MarshalBinary returns the consensus encoding of the receipt.
func (r *Receipt) MarshalBinary() ([]byte, error) {
	if r.Type == LegacyTxType {
		return rlp.EncodeToBytes(r)
	}
	data := &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs}
	var buf bytes.Buffer
	err := r.encodeTyped(data, &buf)
	return buf.Bytes(), err
}

// DecodeRLP implements rlp.Decoder, and loads the consensus fields of a receipt
// from an RLP stream.
func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		// It's a legacy receipt.
		var dec receiptRLP
		if err := s.Decode(&dec); err != nil {
			return err
		}
		r.Type = LegacyTxType
		return r.setFromRLP(dec)
	case kind == rlp.Byte:
		return errShortTypedReceipt
	default:
		// It's an EIP-2718 typed tx receipt.
		b, buf, err := getPooledBuffer(size)
		if err != nil {
			return err
		}
		defer encodeBufferPool.Put(buf)
		if err := s.ReadBytes(b); err != nil {
			return err
		}
		return r.decodeTyped(b)
	}
}

// UnmarshalBinary decodes the consensus encoding of receipts.
// It supports legacy RLP receipts and EIP-2718 typed receipts.
func (r *Receipt) UnmarshalBinary(b []byte) error {
	if len(b) > 0 && b[0] > 0x7f {
		// It's a legacy receipt decode the RLP
		var data receiptRLP
		err := rlp.DecodeBytes(b, &data)
		if err != nil {
			return err
		}
		r.Type = LegacyTxType
		return r.setFromRLP(data)
	}
	// It's an EIP2718 typed transaction envelope.
	return r.decodeTyped(b)
}

// decodeTyped decodes a typed receipt from the canonical format.
func (r *Receipt) decodeTyped(b []byte) error {
	if len(b) <= 1 {
		return errShortTypedReceipt
	}
	switch b[0] {
	case DynamicFeeTxType, AccessListTxType, BlobTxType:
		var data receiptRLP
		err := rlp.DecodeBytes(b[1:], &data)
		if err != nil {
			return err
		}
		r.Type = b[0]
		return r.setFromRLP(data)
	case DepositTxType:
		var data depositReceiptRLP
		err := rlp.DecodeBytes(b[1:], &data)
		if err != nil {
			return err
		}
		r.Type = b[0]
		r.DepositNonce = data.DepositNonce
		r.DepositReceiptVersion = data.DepositReceiptVersion
		return r.setFromRLP(receiptRLP{data.PostStateOrStatus, data.CumulativeGasUsed, data.Bloom, data.Logs})
	default:
		return ErrTxTypeNotSupported
	}
}

func (r *Receipt) setFromRLP(data receiptRLP) error {
	r.CumulativeGasUsed, r.Bloom, r.Logs = data.CumulativeGasUsed, data.Bloom, data.Logs
	return r.setStatus(data.PostStateOrStatus)
}

func (r *Receipt) setStatus(postStateOrStatus []byte) error {
	switch {
	case bytes.Equal(postStateOrStatus, receiptStatusSuccessfulRLP):
		r.Status = ReceiptStatusSuccessful
	case bytes.Equal(postStateOrStatus, receiptStatusFailedRLP):
		r.Status = ReceiptStatusFailed
	case len(postStateOrStatus) == len(common.Hash{}):
		r.PostState = postStateOrStatus
	default:
		return fmt.Errorf("invalid receipt status %x", postStateOrStatus)
	}
	return nil
}

func (r *Receipt) statusEncoding() []byte {
	if len(r.PostState) == 0 {
		if r.Status == ReceiptStatusFailed {
			return receiptStatusFailedRLP
		}
		return receiptStatusSuccessfulRLP
	}
	return r.PostState
}

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (r *Receipt) Size() common.StorageSize {
	size := common.StorageSize(unsafe.Sizeof(*r)) + common.StorageSize(len(r.PostState))
	size += common.StorageSize(len(r.Logs)) * common.StorageSize(unsafe.Sizeof(Log{}))
	for _, log := range r.Logs {
		size += common.StorageSize(len(log.Topics)*common.HashLength + len(log.Data))
	}
	return size
}

// ReceiptForStorage is a wrapper around a Receipt with RLP serialization
// that omits the Bloom field and deserialization that re-computes it.
type ReceiptForStorage Receipt

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt
// into an RLP stream.
func (r *ReceiptForStorage) EncodeRLP(_w io.Writer) error {
	w := rlp.NewEncoderBuffer(_w)
	outerList := w.List()
	w.WriteBytes((*Receipt)(r).statusEncoding())
	w.WriteUint64(r.CumulativeGasUsed)
	logList := w.List()
	for _, log := range r.Logs {
		if err := log.EncodeRLP(w); err != nil {
			return err
		}
	}
	w.ListEnd(logList)
	if r.DepositNonce != nil {
		w.WriteUint64(*r.DepositNonce)
		if r.DepositReceiptVersion != nil {
			w.WriteUint64(*r.DepositReceiptVersion)
		}
	}
	w.ListEnd(outerList)
	return w.Flush()
}

// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
// fields of a receipt from an RLP stream.
func (r *ReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	// Retrieve the entire receipt blob as we need to try multiple decoders
	blob, err := s.Raw()
	if err != nil {
		return err
	}
	// First try to decode the latest receipt database format, try the pre-bedrock Optimism legacy format otherwise.
	if err := decodeStoredReceiptRLP(r, blob); err == nil {
		return nil
	}
	return decodeLegacyOptimismReceiptRLP(r, blob)
}

func decodeLegacyOptimismReceiptRLP(r *ReceiptForStorage, blob []byte) error {
	var stored LegacyOptimismStoredReceiptRLP
	if err := rlp.DecodeBytes(blob, &stored); err != nil {
		return err
	}
	if err := (*Receipt)(r).setStatus(stored.PostStateOrStatus); err != nil {
		return err
	}
	r.CumulativeGasUsed = stored.CumulativeGasUsed
	r.Logs = make([]*Log, len(stored.Logs))
	for i, log := range stored.Logs {
		r.Logs[i] = (*Log)(log)
	}
	r.Bloom = CreateBloom(Receipts{(*Receipt)(r)})
	// UsingOVM
	scalar := new(big.Float)
	if stored.FeeScalar != "" {
		var ok bool
		scalar, ok = scalar.SetString(stored.FeeScalar)
		if !ok {
			return errors.New("cannot parse fee scalar")
		}
	}
	r.L1GasUsed = stored.L1GasUsed
	r.L1GasPrice = stored.L1GasPrice
	r.L1Fee = stored.L1Fee
	r.FeeScalar = scalar
	return nil
}

func decodeStoredReceiptRLP(r *ReceiptForStorage, blob []byte) error {
	var stored storedReceiptRLP
	if err := rlp.DecodeBytes(blob, &stored); err != nil {
		return err
	}
	if err := (*Receipt)(r).setStatus(stored.PostStateOrStatus); err != nil {
		return err
	}
	r.CumulativeGasUsed = stored.CumulativeGasUsed
	r.Logs = stored.Logs
	r.Bloom = CreateBloom(Receipts{(*Receipt)(r)})
	if stored.DepositNonce != nil {
		r.DepositNonce = stored.DepositNonce
		r.DepositReceiptVersion = stored.DepositReceiptVersion
	}
	return nil
}

// Receipts implements DerivableList for receipts.
type Receipts []*Receipt

// Len returns the number of receipts in this list.
func (rs Receipts) Len() int { return len(rs) }

// EncodeIndex encodes the i'th receipt to w. For DepositTxType receipts with non-nil DepositNonce
// but nil DepositReceiptVersion, the output will differ than calling r.MarshalBinary(); this
// behavior difference should not be changed to preserve backwards compatibility of receipt-root
// hash computation.
func (rs Receipts) EncodeIndex(i int, w *bytes.Buffer) {
	r := rs[i]
	data := &receiptRLP{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs}
	if r.Type == LegacyTxType {
		rlp.Encode(w, data)
		return
	}
	w.WriteByte(r.Type)
	switch r.Type {
	case AccessListTxType, DynamicFeeTxType, BlobTxType:
		rlp.Encode(w, data)
	case DepositTxType:
		if r.DepositReceiptVersion != nil {
			// post-canyon receipt hash computation update
			depositData := &depositReceiptRLP{data.PostStateOrStatus, data.CumulativeGasUsed, r.Bloom, r.Logs, r.DepositNonce, r.DepositReceiptVersion}
			rlp.Encode(w, depositData)
		} else {
			rlp.Encode(w, data)
		}
	default:
		// For unsupported types, write nothing. Since this is for
		// DeriveSha, the error will be caught matching the derived hash
		// to the block.
	}
}

// DeriveFields fills the receipts with their computed fields based on consensus
// data and contextual infos like containing block and transactions.
func (rs Receipts) DeriveFields(config *params.ChainConfig, hash common.Hash, number uint64, time uint64, baseFee *big.Int, blobGasPrice *big.Int, txs []*Transaction) error {
	signer := MakeSigner(config, new(big.Int).SetUint64(number), time)

	logIndex := uint(0)
	if len(txs) != len(rs) {
		return errors.New("transaction and receipt count mismatch")
	}
	for i := 0; i < len(rs); i++ {
		// The transaction type and hash can be retrieved from the transaction itself
		rs[i].Type = txs[i].Type()
		rs[i].TxHash = txs[i].Hash()
		rs[i].EffectiveGasPrice = txs[i].inner.effectiveGasPrice(new(big.Int), baseFee)

		// EIP-4844 blob transaction fields
		if txs[i].Type() == BlobTxType {
			rs[i].BlobGasUsed = txs[i].BlobGas()
			rs[i].BlobGasPrice = blobGasPrice
		}

		// block location fields
		rs[i].BlockHash = hash
		rs[i].BlockNumber = new(big.Int).SetUint64(number)
		rs[i].TransactionIndex = uint(i)

		// The contract address can be derived from the transaction itself
		if txs[i].To() == nil {
			// Deriving the signer is expensive, only do if it's actually needed
			from, _ := Sender(signer, txs[i])
			nonce := txs[i].Nonce()
			if rs[i].DepositNonce != nil {
				nonce = *rs[i].DepositNonce
			}
			rs[i].ContractAddress = crypto.CreateAddress(from, nonce)
		} else {
			rs[i].ContractAddress = common.Address{}
		}

		// The used gas can be calculated based on previous r
		if i == 0 {
			rs[i].GasUsed = rs[i].CumulativeGasUsed
		} else {
			rs[i].GasUsed = rs[i].CumulativeGasUsed - rs[i-1].CumulativeGasUsed
		}

		// The derived log fields can simply be set from the block and transaction
		for j := 0; j < len(rs[i].Logs); j++ {
			rs[i].Logs[j].BlockNumber = number
			rs[i].Logs[j].BlockHash = hash
			rs[i].Logs[j].TxHash = rs[i].TxHash
			rs[i].Logs[j].TxIndex = uint(i)
			rs[i].Logs[j].Index = logIndex
			logIndex++
		}
	}
	if config.Optimism != nil && len(txs) >= 2 && config.IsBedrock(new(big.Int).SetUint64(number)) { // need at least an info tx and a non-info tx
		l1BaseFee, costFunc, feeScalar, err := extractL1GasParams(config, time, txs[0].Data())
		if err != nil {
			return err
		}
		for i := 0; i < len(rs); i++ {
			if txs[i].IsDepositTx() {
				continue
			}
			rs[i].L1GasPrice = l1BaseFee
			rs[i].L1Fee, rs[i].L1GasUsed = costFunc(txs[i].RollupCostData())
			rs[i].FeeScalar = feeScalar
		}
	}
	return nil
}
