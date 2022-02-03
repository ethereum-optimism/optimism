package eth

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Source interfaces isolate individual ethereum.ChainReader methods,
// and enable anonymous functions to implement them.

type NewHeadSource interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

type HeaderByHashSource interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
}

type HeaderByNumberSource interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type ReceiptSource interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type BlockByHashSource interface {
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
}

type BlockByNumberSource interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

type L1Source interface {
	NewHeadSource
	HeaderByHashSource
	HeaderByNumberSource
	ReceiptSource
	BlockByHashSource
	Close()
}

type BlockSource interface {
	BlockByHashSource
	BlockByNumberSource
}

// For test instances, composition etc. we implement the interfaces with equivalent function types

type NewHeadFn func(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)

func (fn NewHeadFn) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return fn(ctx, ch)
}

type HeaderByHashFn func(ctx context.Context, hash common.Hash) (*types.Header, error)

func (fn HeaderByHashFn) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return fn(ctx, hash)
}

type HeaderByNumberFn func(ctx context.Context, number *big.Int) (*types.Header, error)

func (fn HeaderByNumberFn) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return fn(ctx, number)
}

type ReceiptFn func(ctx context.Context, txHash common.Hash) (*types.Receipt, error)

func (fn ReceiptFn) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return fn(ctx, txHash)
}

type BlockByHashFn func(ctx context.Context, hash common.Hash) (*types.Block, error)

func (fn BlockByHashFn) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return fn(ctx, hash)
}

type BlockByNumFn func(ctx context.Context, number *big.Int) (*types.Block, error)

func (fn BlockByNumFn) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return fn(ctx, number)
}
