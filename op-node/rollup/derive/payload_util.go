package derive

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// PayloadToBlockRef extracts the essential L2BlockRef information from an execution payload,
// falling back to genesis information if necessary.
func PayloadToBlockRef(payload *eth.ExecutionPayload, genesis *rollup.Genesis) (eth.L2BlockRef, error) {
	var l1Origin eth.BlockID
	var sequenceNumber uint64
	if uint64(payload.BlockNumber) == genesis.L2.Number {
		if payload.BlockHash != genesis.L2.Hash {
			return eth.L2BlockRef{}, fmt.Errorf("expected L2 genesis hash to match L2 block at genesis block number %d: %s <> %s", genesis.L2.Number, payload.BlockHash, genesis.L2.Hash)
		}
		l1Origin = genesis.L1
		sequenceNumber = 0
	} else {
		if len(payload.Transactions) == 0 {
			return eth.L2BlockRef{}, fmt.Errorf("l2 block is missing L1 info deposit tx, block hash: %s", payload.BlockHash)
		}
		var tx types.Transaction
		if err := tx.UnmarshalBinary(payload.Transactions[0]); err != nil {
			return eth.L2BlockRef{}, fmt.Errorf("failed to decode first tx to read l1 info from: %w", err)
		}
		if tx.Type() != types.DepositTxType {
			return eth.L2BlockRef{}, fmt.Errorf("first payload tx has unexpected tx type: %d", tx.Type())
		}
		info, err := L1InfoDepositTxData(tx.Data())
		if err != nil {
			return eth.L2BlockRef{}, fmt.Errorf("failed to parse L1 info deposit tx from L2 block: %w", err)
		}
		l1Origin = eth.BlockID{Hash: info.BlockHash, Number: info.Number}
		sequenceNumber = info.SequenceNumber
	}

	return eth.L2BlockRef{
		Hash:           payload.BlockHash,
		Number:         uint64(payload.BlockNumber),
		ParentHash:     payload.ParentHash,
		Time:           uint64(payload.Timestamp),
		L1Origin:       l1Origin,
		SequenceNumber: sequenceNumber,
	}, nil
}

func PayloadToSystemConfig(payload *eth.ExecutionPayload, cfg *rollup.Config) (eth.SystemConfig, error) {
	if uint64(payload.BlockNumber) == cfg.Genesis.L2.Number {
		if payload.BlockHash != cfg.Genesis.L2.Hash {
			return eth.SystemConfig{}, fmt.Errorf("expected L2 genesis hash to match L2 block at genesis block number %d: %s <> %s", cfg.Genesis.L2.Number, payload.BlockHash, cfg.Genesis.L2.Hash)
		}
		return cfg.Genesis.SystemConfig, nil
	} else {
		if len(payload.Transactions) == 0 {
			return eth.SystemConfig{}, fmt.Errorf("l2 block is missing L1 info deposit tx, block hash: %s", payload.BlockHash)
		}
		var tx types.Transaction
		if err := tx.UnmarshalBinary(payload.Transactions[0]); err != nil {
			return eth.SystemConfig{}, fmt.Errorf("failed to decode first tx to read l1 info from: %w", err)
		}
		if tx.Type() != types.DepositTxType {
			return eth.SystemConfig{}, fmt.Errorf("first payload tx has unexpected tx type: %d", tx.Type())
		}
		info, err := L1InfoDepositTxData(tx.Data())
		if err != nil {
			return eth.SystemConfig{}, fmt.Errorf("failed to parse L1 info deposit tx from L2 block: %w", err)
		}
		return eth.SystemConfig{
			BatcherAddr: info.BatcherAddr,
			Overhead:    info.L1FeeOverhead,
			Scalar:      info.L1FeeScalar,
			GasLimit:    uint64(payload.GasLimit),
		}, err
	}
}
