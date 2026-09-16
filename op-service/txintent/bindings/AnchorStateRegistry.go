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
	RespectedGameType    func() TypedCall[uint32]                  `sol:"respectedGameType"`
	IsGameFinalized      func(game common.Address) TypedCall[bool] `sol:"isGameFinalized"`
	SetAnchorState       func(game common.Address) TypedCall[any]  `sol:"setAnchorState"`
	SetRespectedGameType func(gameType uint32) TypedCall[any]      `sol:"setRespectedGameType"`
}
