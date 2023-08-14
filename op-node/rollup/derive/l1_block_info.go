package derive

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
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

// Binary Format
// +---------+--------------------------+
// | Bytes   | Field                    |
// +---------+--------------------------+
// | 4       | Function signature       |
// | 32      | Number                   |
// | 32      | Time                     |
// | 32      | BaseFee                  |
// | 32      | BlockHash                |
// | 32      | SequenceNumber           |
// | 32      | BatcherAddr              |
// | 32      | L1FeeOverhead            |
// | 32      | L1FeeScalar              |
// +---------+--------------------------+

func (info *L1BlockInfo) MarshalBinary() ([]byte, error) {
	w := bytes.NewBuffer(make([]byte, 0, L1InfoLen))
	if err := solabi.WriteSignature(w, L1InfoFuncBytes4); err != nil {
		return nil, err
	}
	if err := solabi.WriteUint64(w, info.Number); err != nil {
		return nil, err
	}
	if err := solabi.WriteUint64(w, info.Time); err != nil {
		return nil, err
	}
	if err := solabi.WriteUint256(w, info.BaseFee); err != nil {
		return nil, err
	}
	if err := solabi.WriteHash(w, info.BlockHash); err != nil {
		return nil, err
	}
	if err := solabi.WriteUint64(w, info.SequenceNumber); err != nil {
		return nil, err
	}
	if err := solabi.WriteAddress(w, info.BatcherAddr); err != nil {
		return nil, err
	}
	if err := solabi.WriteEthBytes32(w, info.L1FeeOverhead); err != nil {
		return nil, err
	}
	if err := solabi.WriteEthBytes32(w, info.L1FeeScalar); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (info *L1BlockInfo) UnmarshalBinary(data []byte) error {
	if len(data) != L1InfoLen {
		return fmt.Errorf("data is unexpected length: %d", len(data))
	}
	reader := bytes.NewReader(data)

	var err error
	if _, err := solabi.ReadAndValidateSignature(reader, L1InfoFuncBytes4); err != nil {
		return err
	}
	if info.Number, err = solabi.ReadUint64(reader); err != nil {
		return err
	}
	if info.Time, err = solabi.ReadUint64(reader); err != nil {
		return err
	}
	if info.BaseFee, err = solabi.ReadUint256(reader); err != nil {
		return err
	}
	if info.BlockHash, err = solabi.ReadHash(reader); err != nil {
		return err
	}
	if info.SequenceNumber, err = solabi.ReadUint64(reader); err != nil {
		return err
	}
	if info.BatcherAddr, err = solabi.ReadAddress(reader); err != nil {
		return err
	}
	if info.L1FeeOverhead, err = solabi.ReadEthBytes32(reader); err != nil {
		return err
	}
	if info.L1FeeScalar, err = solabi.ReadEthBytes32(reader); err != nil {
		return err
	}
	if !solabi.EmptyReader(reader) {
		return errors.New("too many bytes")
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
