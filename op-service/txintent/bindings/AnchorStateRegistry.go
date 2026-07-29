package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type AnchorStateRegistry struct {
	AnchorGame    func() TypedCall[common.Address] `sol:"anchorGame"`
	GetAnchorRoot func() TypedCall[struct {
		Root             common.Hash
		L2SequenceNumber *big.Int
	}] `sol:"getAnchorRoot"`
	SetRespectedGameType func(gameType uint32) TypedCall[any] `sol:"setRespectedGameType"`
}
