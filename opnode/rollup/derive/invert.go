package derive

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// L1InfoDepositTxData is the inverse of L1InfoDeposit, to see where the L2 chain is derived from
func L1InfoDepositTxData(data []byte) (nr uint64, time uint64, baseFee *big.Int, blockHash common.Hash, err error) {
	if len(data) != 4+8+8+32+32 {
		err = fmt.Errorf("data is unexpected length: %d", len(data))
		return
	}
	offset := 4
	nr = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	time = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	baseFee = new(big.Int).SetBytes(data[offset : offset+32])
	offset += 32
	blockHash.SetBytes(data[offset : offset+32])
	return
}

type Block interface {
	Hash() common.Hash
	NumberU64() uint64
	ParentHash() common.Hash
	Transactions() types.Transactions
	Time() uint64
}

// BlockReferences takes a L2 block and determines which L1 block it was derived from, its L2 parent id, and its own id.
func BlockReferences(l2Block Block, genesis *rollup.Genesis) (eth.L2BlockRef, error) {
	id := eth.L2BlockRef{
		Hash:       l2Block.Hash(),
		Number:     l2Block.NumberU64(),
		ParentHash: l2Block.ParentHash(),
		Time:       l2Block.Time(),
	}

	if id.Number <= genesis.L2.Number {
		if id.Hash != genesis.L2.Hash {
			return eth.L2BlockRef{}, fmt.Errorf("unexpected L2 genesis block: %s:%d, expected %s", id.Hash, id.Number, genesis.L2)
		}
		id.L1Origin = genesis.L1
		return id, nil
	}

	txs := l2Block.Transactions()
	if len(txs) == 0 || txs[0].Type() != types.DepositTxType {
		return eth.L2BlockRef{}, fmt.Errorf("l2 block is missing L1 info deposit tx, block hash: %s", l2Block.Hash())
	}
	l1Number, _, _, l1Hash, err := L1InfoDepositTxData(txs[0].Data())
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("failed to parse L1 info deposit tx from L2 block: %v", err)
	}
	id.L1Origin = eth.BlockID{Hash: l1Hash, Number: l1Number}
	return id, nil
}
