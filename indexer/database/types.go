package database

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Wrapper over types.Header such that we can get an RLP
// encoder over it via a `types.Block` wrapper

type RLPHeader types.Header

func (h *RLPHeader) EncodeRLP(w io.Writer) error {
	return types.NewBlockWithHeader((*types.Header)(h)).EncodeRLP(w)
}

func (h *RLPHeader) DecodeRLP(s *rlp.Stream) error {
	block := new(types.Block)
	err := block.DecodeRLP(s)
	if err != nil {
		return err
	}

	header := block.Header()
	*h = (RLPHeader)(*header)
	return nil
}

func (h *RLPHeader) Header() *types.Header {
	return (*types.Header)(h)
}

func (h *RLPHeader) Hash() common.Hash {
	return h.Header().Hash()
}

// Type definition for []byte to conform to the
// interface expected by the `bytes` serializer

type Bytes []byte

func (b Bytes) Bytes() []byte {
	return b[:]
}
func (b *Bytes) SetBytes(bytes []byte) {
	*b = bytes
}
