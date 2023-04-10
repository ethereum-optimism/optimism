package derive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/eth"
)

const (
	L1InfoFuncSignature = "setL1BlockValues(uint64,uint64,uint256,bytes32,uint64,bytes32,uint256,uint256)"
	L1InfoArguments     = 8
	L1InfoLen           = 4 + 32*L1InfoArguments
)

var (
	L1InfoFuncBytes4       = crypto.Keccak256([]byte(L1InfoFuncSignature))[:4]
	L1InfoDepositerAddress = common.HexToAddress("0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001")
	L1BlockAddress         = predeploys.L1BlockAddr
)

const (
	RegolithSystemTxGas = 1_000_000
)

// L1BlockInfo presents the information stored in a L1Block.setL1BlockValues call
type L1BlockInfo struct {
	Number    uint64
	Time      uint64
	BaseFee   *big.Int
	BlockHash common.Hash
	// Not strictly a piece of L1 information. Represents the number of L2 blocks since the start of the epoch,
	// i.e. when the actual L1 info was first introduced.
	SequenceNumber uint64
	// BatcherHash version 0 is just the address with 0 padding to the left.
	BatcherAddr   common.Address
	L1FeeOverhead eth.Bytes32
	L1FeeScalar   eth.Bytes32
}

//+---------+--------------------------+
//| Bytes   | Field                    |
//+---------+--------------------------+
//| 4       | Function signature       |
//| 24      | Padding for Number       |
//| 8       | Number                   |
//| 24      | Padding for Time         |
//| 8       | Time                     |
//| 32      | BaseFee                  |
//| 32      | BlockHash                |
//| 24      | Padding for SequenceNumber|
//| 8       | SequenceNumber           |
//| 12      | Padding for BatcherAddr  |
//| 20      | BatcherAddr              |
//| 32      | L1FeeOverhead            |
//| 32      | L1FeeScalar              |
//+---------+--------------------------+

func (info *L1BlockInfo) MarshalBinary() ([]byte, error) {
	writer := bytes.NewBuffer(make([]byte, 0, L1InfoLen))
	writer.Write(L1InfoFuncBytes4)

	var padding [24]byte
	writer.Write(padding[:])
	binary.Write(writer, binary.BigEndian, info.Number)
	writer.Write(padding[:])
	binary.Write(writer, binary.BigEndian, info.Time)
	// Ensure that the baseFee is not too large.
	if info.BaseFee.BitLen() > 256 {
		return nil, fmt.Errorf("base fee exceeds 256 bits: %d", info.BaseFee)
	}
	var baseFeeBuf [32]byte
	info.BaseFee.FillBytes(baseFeeBuf[:])
	writer.Write(baseFeeBuf[:])
	writer.Write(info.BlockHash.Bytes())
	writer.Write(padding[:])
	binary.Write(writer, binary.BigEndian, info.SequenceNumber)

	var addrPadding [12]byte
	writer.Write(addrPadding[:])
	writer.Write(info.BatcherAddr.Bytes())
	writer.Write(info.L1FeeOverhead[:])
	writer.Write(info.L1FeeScalar[:])
	return writer.Bytes(), nil
}

func (info *L1BlockInfo) UnmarshalBinary(data []byte) error {
	if len(data) != L1InfoLen {
		return fmt.Errorf("data is unexpected length: %d", len(data))
	}
	reader := bytes.NewReader(data)

	funcSignature := make([]byte, 4)
	if _, err := reader.Read(funcSignature); err != nil || !bytes.Equal(funcSignature, L1InfoFuncBytes4) {
		return fmt.Errorf("data does not match L1 info function signature: 0x%x", funcSignature)
	}

	var padding, readPadding [24]byte

	if _, err := reader.Read(readPadding[:]); err != nil || !bytes.Equal(readPadding[:], padding[:]) {
		return fmt.Errorf("l1 info number exceeds uint64 bounds: %x", readPadding[:])
	}
	if err := binary.Read(reader, binary.BigEndian, &info.Number); err != nil {
		return err
	}

	if _, err := reader.Read(readPadding[:]); err != nil || !bytes.Equal(readPadding[:], padding[:]) {
		return fmt.Errorf("l1 info time exceeds uint64 bounds: %x", readPadding[:])
	}
	if err := binary.Read(reader, binary.BigEndian, &info.Time); err != nil {
		return err
	}

	var baseFeeBytes [32]byte
	if _, err := reader.Read(baseFeeBytes[:]); err != nil {
		return err
	}
	info.BaseFee = new(big.Int).SetBytes(baseFeeBytes[:])

	var blockHashBytes [32]byte
	if _, err := reader.Read(blockHashBytes[:]); err != nil {
		return err
	}
	info.BlockHash.SetBytes(blockHashBytes[:])

	if _, err := reader.Read(readPadding[:]); err != nil || !bytes.Equal(readPadding[:], padding[:]) {
		return fmt.Errorf("l1 info sequence number exceeds uint64 bounds: %x", readPadding[:])
	}
	if err := binary.Read(reader, binary.BigEndian, &info.SequenceNumber); err != nil {
		return err
	}

	var addrPadding [12]byte
	if _, err := reader.Read(addrPadding[:]); err != nil {
		return err
	}
	if _, err := reader.Read(info.BatcherAddr[:]); err != nil {
		return err
	}
	if _, err := reader.Read(info.L1FeeOverhead[:]); err != nil {
		return err
	}
	if _, err := reader.Read(info.L1FeeScalar[:]); err != nil {
		return err
	}

	return nil
}

// L1InfoDepositTxData is the inverse of L1InfoDeposit, to see where the L2 chain is derived from
func L1InfoDepositTxData(data []byte) (L1BlockInfo, error) {
	var info L1BlockInfo
	err := info.UnmarshalBinary(data)
	return info, err
}

// L1InfoDeposit creates a L1 Info deposit transaction based on the L1 block,
// and the L2 block-height difference with the start of the epoch.
func L1InfoDeposit(seqNumber uint64, block eth.BlockInfo, sysCfg eth.SystemConfig, regolith bool) (*types.DepositTx, error) {
	infoDat := L1BlockInfo{
		Number:         block.NumberU64(),
		Time:           block.Time(),
		BaseFee:        block.BaseFee(),
		BlockHash:      block.Hash(),
		SequenceNumber: seqNumber,
		BatcherAddr:    sysCfg.BatcherAddr,
		L1FeeOverhead:  sysCfg.Overhead,
		L1FeeScalar:    sysCfg.Scalar,
	}
	data, err := infoDat.MarshalBinary()
	if err != nil {
		return nil, err
	}

	source := L1InfoDepositSource{
		L1BlockHash: block.Hash(),
		SeqNumber:   seqNumber,
	}
	// Set a very large gas limit with `IsSystemTransaction` to ensure
	// that the L1 Attributes Transaction does not run out of gas.
	out := &types.DepositTx{
		SourceHash:          source.SourceHash(),
		From:                L1InfoDepositerAddress,
		To:                  &L1BlockAddress,
		Mint:                nil,
		Value:               big.NewInt(0),
		Gas:                 150_000_000,
		IsSystemTransaction: true,
		Data:                data,
	}
	// With the regolith fork we disable the IsSystemTx functionality, and allocate real gas
	if regolith {
		out.IsSystemTransaction = false
		out.Gas = RegolithSystemTxGas
	}
	return out, nil
}

// L1InfoDepositBytes returns a serialized L1-info attributes transaction.
func L1InfoDepositBytes(seqNumber uint64, l1Info eth.BlockInfo, sysCfg eth.SystemConfig, regolith bool) ([]byte, error) {
	dep, err := L1InfoDeposit(seqNumber, l1Info, sysCfg, regolith)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 info tx: %w", err)
	}
	l1Tx := types.NewTx(dep)
	opaqueL1Tx, err := l1Tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to encode L1 info tx: %w", err)
	}
	return opaqueL1Tx, nil
}
