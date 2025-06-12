package bindings

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var InvalidLogSignature = errors.New("invalid log signature")

type GameSearchResult struct {
	Index     *big.Int
	Metadata  common.Hash
	Timestamp uint64
	RootClaim common.Hash
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
	Games     func(gameType uint32, rootClaim common.Hash, extraData []byte) TypedCall[struct {
		Proxy     common.Address
		Timestamp uint64
	}] `sol:"games"`
	GetGameUUID     func(gameType uint32, rootClaim common.Hash, extraData []byte) TypedCall[common.Hash] `sol:"getGameUUID"`
	InitBonds       func(gameType uint32) TypedCall[*big.Int]                                             `sol:"initBonds"`
	Owner           func() TypedCall[common.Address]                                                      `sol:"owner"`
	Version         func() TypedCall[string]                                                              `sol:"version"`
	FindLatestGames func(gameType uint32, start *big.Int, n *big.Int) TypedCall[[]GameSearchResult]       `sol:"findLatestGames"`

	// Write functions
	Create            func(gameType uint32, rootClaim common.Hash, extraData []byte) TypedCall[common.Address] `sol:"create"`
	Initialize        func(owner common.Address) TypedCall[any]                                                `sol:"initialize"`
	RenounceOwnership func() TypedCall[any]                                                                    `sol:"renounceOwnership"`
	SetImplementation func(gameType uint32, impl common.Address) TypedCall[any]                                `sol:"setImplementation"`
	SetInitBond       func(gameType uint32, initBond *big.Int) TypedCall[any]                                  `sol:"setInitBond"`
	TransferOwnership func(newOwner common.Address) TypedCall[any]                                             `sol:"transferOwnership"`
}

func NewDisputeGameFactory(opts ...CallFactoryOption) *DisputeGameFactory {
	dgf := NewBindings[DisputeGameFactory](opts...)
	return &dgf
}

type DisputeGameCreated struct {
	DisputeProxy common.Address
	GameType     uint32
	RootClaim    common.Hash
}

var disputeGameCreatedSignature = crypto.Keccak256Hash([]byte("DisputeGameCreated(address,uint32,bytes32)"))

func (d *DisputeGameFactory) ParseDisputeGameCreated(log *types.Log) (*DisputeGameCreated, error) {
	if log.Topics[0] != disputeGameCreatedSignature {
		return nil, InvalidLogSignature
	}
	parsed := &DisputeGameCreated{
		DisputeProxy: common.HexToAddress(log.Topics[1].Hex()),
		GameType:     uint32(log.Topics[2].Big().Uint64()),
		RootClaim:    log.Topics[3],
	}

	return parsed, nil
}
