package proxyd

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
)

// ConsensusTracker abstracts how we store and retrieve the current consensus
// allowing it to be stored locally in-memory or in a shared Redis cluster
type ConsensusTracker interface {
	GetConsensusBlockNumber() string
	SetConsensusBlockNumber(blockNumber string)
}

// InMemoryConsensusTracker store and retrieve in memory, async-safe
type InMemoryConsensusTracker struct {
	consensusBlockNumber string
	mutex                sync.Mutex
}

func NewInMemoryConsensusTracker() ConsensusTracker {
	return &InMemoryConsensusTracker{
		consensusBlockNumber: "", // empty string semantics means unknown
		mutex:                sync.Mutex{},
	}
}

func (ct *InMemoryConsensusTracker) GetConsensusBlockNumber() string {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	return ct.consensusBlockNumber
}

func (ct *InMemoryConsensusTracker) SetConsensusBlockNumber(blockNumber string) {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	ct.consensusBlockNumber = blockNumber
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

func (ct *RedisConsensusTracker) key() string {
	return fmt.Sprintf("consensus_latest_block:%s", ct.backendGroup)
}

func (ct *RedisConsensusTracker) GetConsensusBlockNumber() string {
	return ct.client.Get(ct.ctx, ct.key()).Val()
}

func (ct *RedisConsensusTracker) SetConsensusBlockNumber(blockNumber string) {
	ct.client.Set(ct.ctx, ct.key(), blockNumber, 0)
}
