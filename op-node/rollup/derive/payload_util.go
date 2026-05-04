package derive

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BlockRefFromHeaderAndDeposit extracts the L2BlockRef from a block header and
// its first transaction. The first transaction is expected to be the L1 info
// deposit (except at genesis, where deposit may be nil).
func BlockRefFromHeaderAndDeposit(rollupCfg *rollup.Config, info eth.BlockInfo, deposit *types.Transaction) (eth.L2BlockRef, error) {
	genesis := &rollupCfg.Genesis
	var l1Origin eth.BlockID
	var sequenceNumber uint64
	if info.NumberU64() == genesis.L2.Number {
		if info.Hash() != genesis.L2.Hash {
			return eth.L2BlockRef{}, fmt.Errorf("expected L2 genesis hash to match L2 block at genesis block number %d: %s <> %s", genesis.L2.Number, info.Hash(), genesis.L2.Hash)
		}
		l1Origin = genesis.L1
		sequenceNumber = 0
	} else {
		if deposit == nil {
			return eth.L2BlockRef{}, fmt.Errorf("l2 block is missing L1 info deposit tx, block hash: %s", info.Hash())
		}
		if deposit.Type() != types.DepositTxType {
			return eth.L2BlockRef{}, fmt.Errorf("first payload tx has unexpected tx type: %d", deposit.Type())
		}
		l1Info, err := L1BlockInfoFromBytes(rollupCfg, info.Time(), deposit.Data())
		if err != nil {
			return eth.L2BlockRef{}, fmt.Errorf("failed to parse L1 info deposit tx from L2 block: %w", err)
		}
		l1Origin = eth.BlockID{Hash: l1Info.BlockHash, Number: l1Info.Number}
		sequenceNumber = l1Info.SequenceNumber
	}

	return eth.L2BlockRef{
		Hash:           info.Hash(),
		Number:         info.NumberU64(),
		ParentHash:     info.ParentHash(),
		Time:           info.Time(),
		L1Origin:       l1Origin,
		SequenceNumber: sequenceNumber,
	}, nil
}

// SystemConfigFromHeaderAndDeposit extracts the SystemConfig from a block
// header and its first transaction (the L1 info deposit). At genesis, deposit
// may be nil.
func SystemConfigFromHeaderAndDeposit(rollupCfg *rollup.Config, info eth.BlockInfo, deposit *types.Transaction) (eth.SystemConfig, error) {
	if info.NumberU64() == rollupCfg.Genesis.L2.Number {
		if info.Hash() != rollupCfg.Genesis.L2.Hash {
			return eth.SystemConfig{}, fmt.Errorf(
				"expected L2 genesis hash to match L2 block at genesis block number %d: %s <> %s",
				rollupCfg.Genesis.L2.Number, info.Hash(), rollupCfg.Genesis.L2.Hash)
		}
		return rollupCfg.Genesis.SystemConfig, nil
	}

	if deposit == nil {
		return eth.SystemConfig{}, fmt.Errorf("l2 block is missing L1 info deposit tx, block hash: %s", info.Hash())
	}
	if deposit.Type() != types.DepositTxType {
		return eth.SystemConfig{}, fmt.Errorf("first payload tx has unexpected tx type: %d", deposit.Type())
	}
	l1Info, err := L1BlockInfoFromBytes(rollupCfg, info.Time(), deposit.Data())
	if err != nil {
		return eth.SystemConfig{}, fmt.Errorf("failed to parse L1 info deposit tx from L2 block: %w", err)
	}
	if isEcotoneButNotFirstBlock(rollupCfg, info.Time()) {
		// Translate Ecotone values back into encoded scalar if needed.
		// We do not know if it was derived from a v0 or v1 scalar,
		// but v1 is fine, a 0 blob base fee has the same effect.
		l1Info.L1FeeScalar[0] = 1
		binary.BigEndian.PutUint32(l1Info.L1FeeScalar[24:28], l1Info.BlobBaseFeeScalar)
		binary.BigEndian.PutUint32(l1Info.L1FeeScalar[28:32], l1Info.BaseFeeScalar)
	}
	r := eth.SystemConfig{
		BatcherAddr: l1Info.BatcherAddr,
		Overhead:    l1Info.L1FeeOverhead,
		Scalar:      l1Info.L1FeeScalar,
		GasLimit:    info.GasLimit(),
	}
	err = eip1559.ValidateOptimismExtraData(rollupCfg, info.Time(), info.Extra())
	if err != nil {
		return eth.SystemConfig{}, err
	}
	d, e, m := eip1559.DecodeOptimismExtraData(rollupCfg, info.Time(), info.Extra())
	copy(r.EIP1559Params[:], eip1559.EncodeHolocene1559Params(d, e))

	if rollupCfg.IsIsthmus(info.Time()) {
		r.OperatorFeeParams = eth.EncodeOperatorFeeParams(eth.OperatorFeeParams{
			Scalar:   l1Info.OperatorFeeScalar,
			Constant: l1Info.OperatorFeeConstant,
		})
	}

	if rollupCfg.IsJovian(info.Time()) {
		// ValidateOptimismExtraData returning a nil error guarantees that m is not nil
		r.MinBaseFee = *m
		r.DAFootprintGasScalar = l1Info.DAFootprintGasScalar
	}
	return r, nil
}

// PayloadToBlockRef extracts the essential L2BlockRef information from an execution payload,
// falling back to genesis information if necessary.
func PayloadToBlockRef(rollupCfg *rollup.Config, payload *eth.ExecutionPayload) (eth.L2BlockRef, error) {
	deposit, err := firstTxAsDeposit(payload)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	return BlockRefFromHeaderAndDeposit(rollupCfg, eth.PayloadToInfo(payload), deposit)
}

func PayloadToSystemConfig(rollupCfg *rollup.Config, payload *eth.ExecutionPayload) (eth.SystemConfig, error) {
	deposit, err := firstTxAsDeposit(payload)
	if err != nil {
		return eth.SystemConfig{}, err
	}
	return SystemConfigFromHeaderAndDeposit(rollupCfg, eth.PayloadToInfo(payload), deposit)
}

// firstTxAsDeposit decodes the first transaction of a payload. Returns nil
// without error when the payload has no transactions, so the genesis branch
// in the inner helpers can pass through.
func firstTxAsDeposit(payload *eth.ExecutionPayload) (*types.Transaction, error) {
	if len(payload.Transactions) == 0 {
		return nil, nil
	}
	var tx types.Transaction
	if err := tx.UnmarshalBinary(payload.Transactions[0]); err != nil {
		return nil, fmt.Errorf("failed to decode first tx to read l1 info from: %w", err)
	}
	return &tx, nil
}
