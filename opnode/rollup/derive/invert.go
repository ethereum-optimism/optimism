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
}

// BlockReferences takes a L2 block and determines which L1 block it was derived from, and the L2 self and parent id.
func BlockReferences(refL2Block Block, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error) {
	refL2 = eth.BlockID{Hash: refL2Block.Hash(), Number: refL2Block.NumberU64()}
	if refL2.Number <= genesis.L2.Number {
		if refL2.Hash != genesis.L2.Hash {
			err = fmt.Errorf("unexpected L2 genesis block: %s, expected %s", refL2, genesis.L2)
			return
		}
		refL1 = genesis.L1
		refL2 = genesis.L2
		parentL2 = common.Hash{}
		return
	}

	parentL2 = refL2Block.ParentHash()
	txs := refL2Block.Transactions()
	if len(txs) == 0 || txs[0].Type() != types.DepositTxType {
		err = fmt.Errorf("l2 block is missing L1 info deposit tx, block hash: %s", refL2Block.Hash())
		return
	}
	refL1Nr, _, _, refL1Hash, err := L1InfoDepositTxData(txs[0].Data())
	if err != nil {
		err = fmt.Errorf("failed to parse L1 info deposit tx from L2 block: %v", err)
		return
	}
	refL1 = eth.BlockID{Hash: refL1Hash, Number: refL1Nr}
	return
}
