package l1

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

// Oracle retrieves the requested data in any way.
type Oracle interface {

	// BlockByHash retrieves the block with the given hash.
	// Returns an error if the block is not available.
	BlockByHash(blockHash common.Hash) (*types.Block, error)
}

// PreimageOracle implements Oracle using by interfacing with the pure preimage.Oracle
// to fetch pre-images to decode into the requested data.
type PreimageOracle struct {
	oracle preimage.Oracle
	hint   preimage.Hinter
}

var _ Oracle = (*PreimageOracle)(nil)

func NewPreimageOracle(raw preimage.Oracle, hint preimage.Hinter) *PreimageOracle {
	return &PreimageOracle{
		oracle: raw,
		hint:   hint,
	}
}

func (p *PreimageOracle) BlockByHash(blockHash common.Hash) (*types.Block, error) {
	p.hint.Hint(L1BlockHint(blockHash))
	blockRlp := p.oracle.Get(preimage.Keccak256Key(blockHash))
	var block types.Block
	if err := rlp.DecodeBytes(blockRlp, &block); err != nil {
		panic(fmt.Errorf("invalid block %s: %w", blockHash, err))
	}
	return &block, nil
}
