package eth

import (
	"context"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

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

// CombinedL1Source balances multiple L1 sources, to shred concurrent requests to multiple endpoints
type CombinedL1Source struct {
	i       uint64
	sources []L1Source
}

func NewCombinedL1Source(sources []L1Source) L1Source {
	if len(sources) == 0 {
		panic("need at least 1 source")
	}
	return &CombinedL1Source{i: 0, sources: sources}
}

func (cs *CombinedL1Source) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return cs.sources[atomic.AddUint64(&cs.i, 1)%uint64(len(cs.sources))].HeaderByHash(ctx, hash)
}

func (cs *CombinedL1Source) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return cs.sources[atomic.AddUint64(&cs.i, 1)%uint64(len(cs.sources))].HeaderByNumber(ctx, number)
}

func (cs *CombinedL1Source) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	// TODO: can't use multiple sources as consensus, or head may be conflicting too much
	return cs.sources[0].SubscribeNewHead(ctx, ch)
}

func (cs *CombinedL1Source) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return cs.sources[atomic.AddUint64(&cs.i, 1)%uint64(len(cs.sources))].TransactionReceipt(ctx, txHash)
}

func (cs *CombinedL1Source) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return cs.sources[atomic.AddUint64(&cs.i, 1)%uint64(len(cs.sources))].BlockByHash(ctx, hash)
}

func (cs *CombinedL1Source) Close() {
	for _, src := range cs.sources {
		src.Close()
	}
}
