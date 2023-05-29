package proxyd

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/go-redis/redis/v8"
)

// ConsensusTracker abstracts how we store and retrieve the current consensus
// allowing it to be stored locally in-memory or in a shared Redis cluster
type ConsensusTracker interface {
	GetLatestBlockNumber() hexutil.Uint64
	SetLatestBlockNumber(blockNumber hexutil.Uint64)
	GetSafeBlockNumber() hexutil.Uint64
	SetSafeBlockNumber(blockNumber hexutil.Uint64)
	GetFinalizedBlockNumber() hexutil.Uint64
	SetFinalizedBlockNumber(blockNumber hexutil.Uint64)
}

// InMemoryConsensusTracker store and retrieve in memory, async-safe
type InMemoryConsensusTracker struct {
	latestBlockNumber    hexutil.Uint64
	safeBlockNumber      hexutil.Uint64
	finalizedBlockNumber hexutil.Uint64
	mutex                sync.Mutex
}

func NewInMemoryConsensusTracker() ConsensusTracker {
	return &InMemoryConsensusTracker{
		mutex: sync.Mutex{},
	}
}

func (ct *InMemoryConsensusTracker) GetLatestBlockNumber() hexutil.Uint64 {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	return ct.latestBlockNumber
}

func (ct *InMemoryConsensusTracker) SetLatestBlockNumber(blockNumber hexutil.Uint64) {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	ct.latestBlockNumber = blockNumber
}

func (ct *InMemoryConsensusTracker) GetSafeBlockNumber() hexutil.Uint64 {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	return ct.safeBlockNumber
}

func (ct *InMemoryConsensusTracker) SetSafeBlockNumber(blockNumber hexutil.Uint64) {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	ct.safeBlockNumber = blockNumber
}

func (ct *InMemoryConsensusTracker) GetFinalizedBlockNumber() hexutil.Uint64 {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	return ct.finalizedBlockNumber
}

func (ct *InMemoryConsensusTracker) SetFinalizedBlockNumber(blockNumber hexutil.Uint64) {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	ct.finalizedBlockNumber = blockNumber
}

// RedisConsensusTracker uses a Redis `client` to store and retrieve consensus, async-safe
type RedisConsensusTracker struct {
	ctx          context.Context
	client       *redis.Client
	backendGroup string
}

func NewRedisConsensusTracker(ctx context.Context, r *redis.Client, namespace string) ConsensusTracker {
	return &RedisConsensusTracker{
		ctx:          ctx,
		client:       r,
		backendGroup: namespace,
	}
}

func (ct *RedisConsensusTracker) key(tag string) string {
	return fmt.Sprintf("consensus:%s:%s", ct.backendGroup, tag)
}

func (ct *RedisConsensusTracker) GetLatestBlockNumber() hexutil.Uint64 {
	return hexutil.Uint64(hexutil.MustDecodeUint64(ct.client.Get(ct.ctx, ct.key("latest")).Val()))
}

func (ct *RedisConsensusTracker) SetLatestBlockNumber(blockNumber hexutil.Uint64) {
	ct.client.Set(ct.ctx, ct.key("latest"), blockNumber, 0)
}

func (ct *RedisConsensusTracker) GetSafeBlockNumber() hexutil.Uint64 {
	return hexutil.Uint64(hexutil.MustDecodeUint64(ct.client.Get(ct.ctx, ct.key("safe")).Val()))
}

func (ct *RedisConsensusTracker) SetSafeBlockNumber(blockNumber hexutil.Uint64) {
	ct.client.Set(ct.ctx, ct.key("safe"), blockNumber, 0)
}

func (ct *RedisConsensusTracker) GetFinalizedBlockNumber() hexutil.Uint64 {
	return hexutil.Uint64(hexutil.MustDecodeUint64(ct.client.Get(ct.ctx, ct.key("finalized")).Val()))
}

func (ct *RedisConsensusTracker) SetFinalizedBlockNumber(blockNumber hexutil.Uint64) {
	ct.client.Set(ct.ctx, ct.key("finalized"), blockNumber, 0)
}
