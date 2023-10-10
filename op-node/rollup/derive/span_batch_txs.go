package derive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

type spanBatchTxs struct {
	// this field must be manually set
	totalBlockTxCount uint64

	// 7 fields
	contractCreationBits *big.Int
	yParityBits          *big.Int
	txSigs               []spanBatchSignature
	txNonces             []uint64
	txGases              []uint64
	txTos                []common.Address
	txDatas              []hexutil.Bytes

	txTypes []int
}

type spanBatchSignature struct {
	v uint64
	r *uint256.Int
	s *uint256.Int
}

// contractCreationBits is bitlist right-padded to a multiple of 8 bits
func (btx *spanBatchTxs) encodeContractCreationBits(w io.Writer) error {
	contractCreationBitBufferLen := btx.totalBlockTxCount / 8
	if btx.totalBlockTxCount%8 != 0 {
		contractCreationBitBufferLen++
	}
	contractCreationBitBuffer := make([]byte, contractCreationBitBufferLen)
	for i := 0; i < int(btx.totalBlockTxCount); i += 8 {
		end := i + 8
		if end < int(btx.totalBlockTxCount) {
			end = int(btx.totalBlockTxCount)
		}
		var bits uint = 0
		for j := i; j < end; j++ {
			bits |= btx.contractCreationBits.Bit(j) << (j - i)
		}
		contractCreationBitBuffer[i/8] = byte(bits)
	}
	if _, err := w.Write(contractCreationBitBuffer); err != nil {
		return fmt.Errorf("cannot write contract creation bits: %w", err)
	}
	return nil
}

// contractCreationBits is bitlist right-padded to a multiple of 8 bits
func (btx *spanBatchTxs) decodeContractCreationBits(r *bytes.Reader) error {
	contractCreationBitBufferLen := btx.totalBlockTxCount / 8
	if btx.totalBlockTxCount%8 != 0 {
		contractCreationBitBufferLen++
	}
	// avoid out of memory before allocation
	if contractCreationBitBufferLen > MaxSpanBatchFieldSize {
		return ErrTooBigSpanBatchFieldSize
	}
	contractCreationBitBuffer := make([]byte, contractCreationBitBufferLen)
	_, err := io.ReadFull(r, contractCreationBitBuffer)
	if err != nil {
		return fmt.Errorf("failed to read contract creation bits: %w", err)
	}
	contractCreationBits := new(big.Int)
	for i := 0; i < int(btx.totalBlockTxCount); i += 8 {
		end := i + 8
		if end < int(btx.totalBlockTxCount) {
			end = int(btx.totalBlockTxCount)
		}
		bits := contractCreationBitBuffer[i/8]
		for j := i; j < end; j++ {
			bit := uint((bits >> (j - i)) & 1)
			contractCreationBits.SetBit(contractCreationBits, j, bit)
		}
	}
	btx.contractCreationBits = contractCreationBits
	return nil
}

func (btx *spanBatchTxs) contractCreationCount() uint64 {
	if btx.contractCreationBits == nil {
		panic("contract creation bits not set")
	}
	var result uint64 = 0
	for i := 0; i < int(btx.totalBlockTxCount); i++ {
		bit := btx.contractCreationBits.Bit(i)
		if bit == 1 {
			result++
		}
	}
	return result
}

// yParityBits is bitlist right-padded to a multiple of 8 bits
func (btx *spanBatchTxs) encodeYParityBits(w io.Writer) error {
	yParityBitBufferLen := btx.totalBlockTxCount / 8
	if btx.totalBlockTxCount%8 != 0 {
		yParityBitBufferLen++
	}
	yParityBitBuffer := make([]byte, yParityBitBufferLen)
	for i := 0; i < int(btx.totalBlockTxCount); i += 8 {
		end := i + 8
		if end < int(btx.totalBlockTxCount) {
			end = int(btx.totalBlockTxCount)
		}
		var bits uint = 0
		for j := i; j < end; j++ {
			bits |= btx.yParityBits.Bit(j) << (j - i)
		}
		yParityBitBuffer[i/8] = byte(bits)
	}
	if _, err := w.Write(yParityBitBuffer); err != nil {
		return fmt.Errorf("cannot write y parity bits: %w", err)
	}
	return nil
}

func (btx *spanBatchTxs) encodeTxSigsRS(w io.Writer) error {
	for _, txSig := range btx.txSigs {
		rBuf := txSig.r.Bytes32()
		if _, err := w.Write(rBuf[:]); err != nil {
			return fmt.Errorf("cannot write tx sig r: %w", err)
		}
		sBuf := txSig.s.Bytes32()
		if _, err := w.Write(sBuf[:]); err != nil {
			return fmt.Errorf("cannot write tx sig s: %w", err)
		}
	}
	return nil
}

func (btx *spanBatchTxs) encodeTxNonces(w io.Writer) error {
	var buf [binary.MaxVarintLen64]byte
	for _, txNonce := range btx.txNonces {
		n := binary.PutUvarint(buf[:], txNonce)
		if _, err := w.Write(buf[:n]); err != nil {
			return fmt.Errorf("cannot write tx nonce: %w", err)
		}
	}
	return nil
}

func (btx *spanBatchTxs) encodeTxGases(w io.Writer) error {
	var buf [binary.MaxVarintLen64]byte
	for _, txGas := range btx.txGases {
		n := binary.PutUvarint(buf[:], txGas)
		if _, err := w.Write(buf[:n]); err != nil {
			return fmt.Errorf("cannot write tx gas: %w", err)
		}
	}
	return nil
}

func (btx *spanBatchTxs) encodeTxTos(w io.Writer) error {
	for _, txTo := range btx.txTos {
		if _, err := w.Write(txTo.Bytes()); err != nil {
			return fmt.Errorf("cannot write tx to address: %w", err)
		}
	}
	return nil
}

func (btx *spanBatchTxs) encodeTxDatas(w io.Writer) error {
	for _, txData := range btx.txDatas {
		if _, err := w.Write(txData); err != nil {
			return fmt.Errorf("cannot write tx data: %w", err)
		}
	}
	return nil
}

// yParityBits is bitlist right-padded to a multiple of 8 bits
func (btx *spanBatchTxs) decodeYParityBits(r *bytes.Reader) error {
	yParityBitBufferLen := btx.totalBlockTxCount / 8
	if btx.totalBlockTxCount%8 != 0 {
		yParityBitBufferLen++
	}
	// avoid out of memory before allocation
	if yParityBitBufferLen > MaxSpanBatchFieldSize {
		return ErrTooBigSpanBatchFieldSize
	}
	yParityBitBuffer := make([]byte, yParityBitBufferLen)
	_, err := io.ReadFull(r, yParityBitBuffer)
	if err != nil {
		return fmt.Errorf("failed to read y parity bits: %w", err)
	}
	yParityBits := new(big.Int)
	for i := 0; i < int(btx.totalBlockTxCount); i += 8 {
		end := i + 8
		if end < int(btx.totalBlockTxCount) {
			end = int(btx.totalBlockTxCount)
		}
		bits := yParityBitBuffer[i/8]
		for j := i; j < end; j++ {
			bit := uint((bits >> (j - i)) & 1)
			yParityBits.SetBit(yParityBits, j, bit)
		}
	}
	btx.yParityBits = yParityBits
	return nil
}

func (btx *spanBatchTxs) decodeTxSigsRS(r *bytes.Reader) error {
	var txSigs []spanBatchSignature
	var sigBuffer [32]byte
	for i := 0; i < int(btx.totalBlockTxCount); i++ {
		var txSig spanBatchSignature
		_, err := io.ReadFull(r, sigBuffer[:])
		if err != nil {
			return fmt.Errorf("failed to read tx sig r: %w", err)
		}
		txSig.r, _ = uint256.FromBig(new(big.Int).SetBytes(sigBuffer[:]))
		_, err = io.ReadFull(r, sigBuffer[:])
		if err != nil {
			return fmt.Errorf("failed to read tx sig s: %w", err)
		}
		txSig.s, _ = uint256.FromBig(new(big.Int).SetBytes(sigBuffer[:]))
		txSigs = append(txSigs, txSig)
	}
	btx.txSigs = txSigs
	return nil
}

func (btx *spanBatchTxs) decodeTxNonces(r *bytes.Reader) error {
	var txNonces []uint64
	for i := 0; i < int(btx.totalBlockTxCount); i++ {
		txNonce, err := binary.ReadUvarint(r)
		if err != nil {
			return fmt.Errorf("failed to read tx nonce: %w", err)
		}
		txNonces = append(txNonces, txNonce)
	}
	btx.txNonces = txNonces
	return nil
}

func (btx *spanBatchTxs) decodeTxGases(r *bytes.Reader) error {
	var txGases []uint64
	for i := 0; i < int(btx.totalBlockTxCount); i++ {
		txGas, err := binary.ReadUvarint(r)
		if err != nil {
			return fmt.Errorf("failed to read tx gas: %w", err)
		}
		txGases = append(txGases, txGas)
	}
	btx.txGases = txGases
	return nil
}

func (btx *spanBatchTxs) decodeTxTos(r *bytes.Reader) error {
	var txTos []common.Address
	txToBuffer := make([]byte, common.AddressLength)
	contractCreationCount := btx.contractCreationCount()
	for i := 0; i < int(btx.totalBlockTxCount-contractCreationCount); i++ {
		_, err := io.ReadFull(r, txToBuffer)
		if err != nil {
			return fmt.Errorf("failed to read tx to address: %w", err)
		}
		txTos = append(txTos, common.BytesToAddress(txToBuffer))
	}
	btx.txTos = txTos
	return nil
}

func (btx *spanBatchTxs) decodeTxDatas(r *bytes.Reader) error {
	var txDatas []hexutil.Bytes
	var txTypes []int
	// Do not need txDataHeader because RLP byte stream already includes length info
	for i := 0; i < int(btx.totalBlockTxCount); i++ {
		txData, txType, err := ReadTxData(r)
		if err != nil {
			return err
		}
		txDatas = append(txDatas, txData)
		txTypes = append(txTypes, txType)
	}
	btx.txDatas = txDatas
	btx.txTypes = txTypes
	return nil
}

func (btx *spanBatchTxs) recoverV(chainID *big.Int) {
	if len(btx.txTypes) != len(btx.txSigs) {
		panic("tx type length and tx sigs length mismatch")
	}
	for idx, txType := range btx.txTypes {
		bit := uint64(btx.yParityBits.Bit(idx))
		var v uint64
		switch txType {
		case types.LegacyTxType:
			// EIP155
			v = chainID.Uint64()*2 + 35 + bit
		case types.AccessListTxType:
			v = bit
		case types.DynamicFeeTxType:
			v = bit
		default:
			panic(fmt.Sprintf("invalid tx type: %d", txType))
		}
		btx.txSigs[idx].v = v
	}
}

func (btx *spanBatchTxs) encode(w io.Writer) error {
	if err := btx.encodeContractCreationBits(w); err != nil {
		return err
	}
	if err := btx.encodeYParityBits(w); err != nil {
		return err
	}
	if err := btx.encodeTxSigsRS(w); err != nil {
		return err
	}
	if err := btx.encodeTxTos(w); err != nil {
		return err
	}
	if err := btx.encodeTxDatas(w); err != nil {
		return err
	}
	if err := btx.encodeTxNonces(w); err != nil {
		return err
	}
	if err := btx.encodeTxGases(w); err != nil {
		return err
	}
	return nil
}

func (btx *spanBatchTxs) decode(r *bytes.Reader) error {
	if err := btx.decodeContractCreationBits(r); err != nil {
		return err
	}
	if err := btx.decodeYParityBits(r); err != nil {
		return err
	}
	if err := btx.decodeTxSigsRS(r); err != nil {
		return err
	}
	if err := btx.decodeTxTos(r); err != nil {
		return err
	}
	if err := btx.decodeTxDatas(r); err != nil {
		return err
	}
	if err := btx.decodeTxNonces(r); err != nil {
		return err
	}
	if err := btx.decodeTxGases(r); err != nil {
		return err
	}
	return nil
}

func (btx *spanBatchTxs) fullTxs(chainID *big.Int) ([][]byte, error) {
	var txs [][]byte
	toIdx := 0
	for idx := 0; idx < int(btx.totalBlockTxCount); idx++ {
		var stx spanBatchTx
		if err := stx.UnmarshalBinary(btx.txDatas[idx]); err != nil {
			return nil, err
		}
		nonce := btx.txNonces[idx]
		gas := btx.txGases[idx]
		var to *common.Address = nil
		bit := btx.contractCreationBits.Bit(idx)
		if bit == 0 {
			if len(btx.txTos) <= toIdx {
				return nil, errors.New("tx to not enough")
			}
			to = &btx.txTos[toIdx]
			toIdx++
		}
		v := new(big.Int).SetUint64(btx.txSigs[idx].v)
		r := btx.txSigs[idx].r.ToBig()
		s := btx.txSigs[idx].s.ToBig()
		tx, err := stx.convertToFullTx(nonce, gas, to, chainID, v, r, s)
		if err != nil {
			return nil, err
		}
		encodedTx, err := tx.MarshalBinary()
		if err != nil {
			return nil, err
		}
		txs = append(txs, encodedTx)
	}
	return txs, nil
}

func convertVToYParity(v uint64, txType int) uint {
	var yParityBit uint
	switch txType {
	case types.LegacyTxType:
		// EIP155: v = 2 * chainID + 35 + yParity
		// v - 35 = yParity (mod 2)
		yParityBit = uint((v - 35) & 1)
	case types.AccessListTxType:
		yParityBit = uint(v)
	case types.DynamicFeeTxType:
		yParityBit = uint(v)
	default:
		panic(fmt.Sprintf("invalid tx type: %d", txType))
	}
	return yParityBit
}

func newSpanBatchTxs(txs [][]byte, chainID *big.Int) (*spanBatchTxs, error) {
	totalBlockTxCount := uint64(len(txs))
	var txSigs []spanBatchSignature
	var txTos []common.Address
	var txNonces []uint64
	var txGases []uint64
	var txDatas []hexutil.Bytes
	var txTypes []int
	contractCreationBits := new(big.Int)
	yParityBits := new(big.Int)
	for idx := 0; idx < int(totalBlockTxCount); idx++ {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(txs[idx]); err != nil {
			return nil, errors.New("failed to decode tx")
		}
		if tx.Protected() && tx.ChainId().Cmp(chainID) != 0 {
			return nil, fmt.Errorf("protected tx has chain ID %d, but expected chain ID %d", tx.ChainId(), chainID)
		}
		var txSig spanBatchSignature
		v, r, s := tx.RawSignatureValues()
		R, _ := uint256.FromBig(r)
		S, _ := uint256.FromBig(s)
		txSig.v = v.Uint64()
		txSig.r = R
		txSig.s = S
		txSigs = append(txSigs, txSig)
		contractCreationBit := uint(1)
		if tx.To() != nil {
			txTos = append(txTos, *tx.To())
			contractCreationBit = uint(0)
		}
		contractCreationBits.SetBit(contractCreationBits, idx, contractCreationBit)
		yParityBit := convertVToYParity(txSig.v, int(tx.Type()))
		yParityBits.SetBit(yParityBits, idx, yParityBit)
		txNonces = append(txNonces, tx.Nonce())
		txGases = append(txGases, tx.Gas())
		stx, err := newSpanBatchTx(tx)
		if err != nil {
			return nil, err
		}
		txData, err := stx.MarshalBinary()
		if err != nil {
			return nil, err
		}
		txDatas = append(txDatas, txData)
		txTypes = append(txTypes, int(tx.Type()))
	}
	return &spanBatchTxs{
		totalBlockTxCount:    totalBlockTxCount,
		contractCreationBits: contractCreationBits,
		yParityBits:          yParityBits,
		txSigs:               txSigs,
		txNonces:             txNonces,
		txGases:              txGases,
		txTos:                txTos,
		txDatas:              txDatas,
		txTypes:              txTypes,
	}, nil
}
