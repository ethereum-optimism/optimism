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

func (info *L1BlockInfo) MarshalBinary() ([]byte, error) {
	data := make([]byte, L1InfoLen)
	offset := 0
	copy(data[offset:4], L1InfoFuncBytes4)
	offset += 4
	binary.BigEndian.PutUint64(data[offset+24:offset+32], info.Number)
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], info.Time)
	offset += 32
	// Ensure that the baseFee is not too large.
	if info.BaseFee.BitLen() > 256 {
		return nil, fmt.Errorf("base fee exceeds 256 bits: %d", info.BaseFee)
	}
	info.BaseFee.FillBytes(data[offset : offset+32])
	offset += 32
	copy(data[offset:offset+32], info.BlockHash.Bytes())
	offset += 32
	binary.BigEndian.PutUint64(data[offset+24:offset+32], info.SequenceNumber)
	offset += 32
	copy(data[offset+12:offset+32], info.BatcherAddr[:])
	offset += 32
	copy(data[offset:offset+32], info.L1FeeOverhead[:])
	offset += 32
	copy(data[offset:offset+32], info.L1FeeScalar[:])
	return data, nil
}

func (info *L1BlockInfo) UnmarshalBinary(data []byte) error {
	if len(data) != L1InfoLen {
		return fmt.Errorf("data is unexpected length: %d", len(data))
	}
	var padding [24]byte
	offset := 4
	info.Number = binary.BigEndian.Uint64(data[offset+24 : offset+32])
	if !bytes.Equal(data[offset:offset+24], padding[:]) {
		return fmt.Errorf("l1 info number exceeds uint64 bounds: %x", data[offset:offset+32])
	}
	offset += 32
	info.Time = binary.BigEndian.Uint64(data[offset+24 : offset+32])
	if !bytes.Equal(data[offset:offset+24], padding[:]) {
		return fmt.Errorf("l1 info time exceeds uint64 bounds: %x", data[offset:offset+32])
	}
	offset += 32
	info.BaseFee = new(big.Int).SetBytes(data[offset : offset+32])
	offset += 32
	info.BlockHash.SetBytes(data[offset : offset+32])
	offset += 32
	info.SequenceNumber = binary.BigEndian.Uint64(data[offset+24 : offset+32])
	if !bytes.Equal(data[offset:offset+24], padding[:]) {
		return fmt.Errorf("l1 info sequence number exceeds uint64 bounds: %x", data[offset:offset+32])
	}
	offset += 32
	info.BatcherAddr.SetBytes(data[offset+12 : offset+32])
	offset += 32
	copy(info.L1FeeOverhead[:], data[offset:offset+32])
	offset += 32
	copy(info.L1FeeScalar[:], data[offset:offset+32])
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
func L1InfoDeposit(seqNumber uint64, block eth.BlockInfo, sysCfg eth.SystemConfig) (*types.DepositTx, error) {
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
	return &types.DepositTx{
		SourceHash:          source.SourceHash(),
		From:                L1InfoDepositerAddress,
		To:                  &L1BlockAddress,
		Mint:                nil,
		Value:               big.NewInt(0),
		Gas:                 150_000_000,
		IsSystemTransaction: true,
		Data:                data,
	}, nil
}

// L1InfoDepositBytes returns a serialized L1-info attributes transaction.
func L1InfoDepositBytes(seqNumber uint64, l1Info eth.BlockInfo, sysCfg eth.SystemConfig) ([]byte, error) {
	dep, err := L1InfoDeposit(seqNumber, l1Info, sysCfg)
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
