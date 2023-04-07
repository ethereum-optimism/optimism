package l2

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

// StateOracle defines the high-level API used to retrieve L2 state data pre-images
// The returned data is always the preimage of the requested hash.
type StateOracle interface {
	// NodeByHash retrieves the merkle-patricia trie node pre-image for a given hash.
	// Trie nodes may be from the world state trie or any account storage trie.
	// Contract code is not stored as part of the trie and must be retrieved via CodeByHash
	NodeByHash(nodeHash common.Hash) []byte

	// CodeByHash retrieves the contract code pre-image for a given hash.
	// codeHash should be retrieved from the world state account for a contract.
	CodeByHash(codeHash common.Hash) []byte
}

// Oracle defines the high-level API used to retrieve L2 data.
// The returned data is always the preimage of the requested hash.
type Oracle interface {
	StateOracle

	// BlockByHash retrieves the block with the given hash.
	BlockByHash(blockHash common.Hash) *types.Block
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

func (p *PreimageOracle) BlockByHash(blockHash common.Hash) *types.Block {
	p.hint.Hint(L2BlockHint(blockHash))
	blockRlp := p.oracle.Get(preimage.Keccak256Key(blockHash))
	var block types.Block
	if err := rlp.DecodeBytes(blockRlp, &block); err != nil {
		panic(fmt.Errorf("invalid block %s: %w", blockHash, err))
	}
	return &block
}

func (p *PreimageOracle) NodeByHash(nodeHash common.Hash) []byte {
	p.hint.Hint(L2StateNodeHint(nodeHash))
	node := p.oracle.Get(preimage.Keccak256Key(nodeHash))
	return node
}

func (p *PreimageOracle) CodeByHash(codeHash common.Hash) []byte {
	p.hint.Hint(L2CodeHint(codeHash))
	code := p.oracle.Get(preimage.Keccak256Key(codeHash))
	return code
}
