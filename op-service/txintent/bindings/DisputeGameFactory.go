package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type GameSearchResult struct {
	Index     *big.Int
	Metadata  [32]byte
	Timestamp uint64
	RootClaim [32]byte
	ExtraData []byte
}

type DisputeGameFactory struct {
	// Read-only functions
	GameCount   func() TypedCall[*big.Int] `sol:"gameCount"`
	GameAtIndex func(index *big.Int) TypedCall[struct {
		GameType  uint32
		Timestamp uint64
		Proxy     common.Address
	}] `sol:"gameAtIndex"`
	GameImpls func(gameType uint32) TypedCall[common.Address] `sol:"gameImpls"`
	Games     func(gameType uint32, rootClaim [32]byte, extraData []byte) TypedCall[struct {
		Proxy     common.Address
		Timestamp uint64
	}] `sol:"games"`
	GetGameUUID     func(gameType uint32, rootClaim [32]byte, extraData []byte) TypedCall[[32]byte] `sol:"getGameUUID"`
	InitBonds       func(gameType uint32) TypedCall[*big.Int]                                       `sol:"initBonds"`
	Owner           func() TypedCall[common.Address]                                                `sol:"owner"`
	Version         func() TypedCall[string]                                                        `sol:"version"`
	FindLatestGames func(gameType uint32, start *big.Int, n *big.Int) TypedCall[[]GameSearchResult] `sol:"findLatestGames"`

	// Write functions
	Create            func(gameType uint32, rootClaim [32]byte, extraData []byte) TypedCall[common.Address] `sol:"create"`
	Initialize        func(owner common.Address) TypedCall[any]                                             `sol:"initialize"`
	RenounceOwnership func() TypedCall[any]                                                                 `sol:"renounceOwnership"`
	SetImplementation func(gameType uint32, impl common.Address) TypedCall[any]                             `sol:"setImplementation"`
	SetInitBond       func(gameType uint32, initBond *big.Int) TypedCall[any]                               `sol:"setInitBond"`
	TransferOwnership func(newOwner common.Address) TypedCall[any]                                          `sol:"transferOwnership"`
}
