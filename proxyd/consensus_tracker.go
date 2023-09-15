package proxyd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
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

// DTO to hold the current consensus state
type ConsensusTrackerState struct {
	Latest    hexutil.Uint64 `json:"latest"`
	Safe      hexutil.Uint64 `json:"safe"`
	Finalized hexutil.Uint64 `json:"finalized"`
}

func (ct *InMemoryConsensusTracker) update(o *ConsensusTrackerState) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	ct.state.Latest = o.Latest
	ct.state.Safe = o.Safe
	ct.state.Finalized = o.Finalized
}

// InMemoryConsensusTracker store and retrieve in memory, async-safe
type InMemoryConsensusTracker struct {
	mutex sync.Mutex
	state *ConsensusTrackerState
}

func NewInMemoryConsensusTracker() ConsensusTracker {
	return &InMemoryConsensusTracker{
		mutex: sync.Mutex{},
		state: &ConsensusTrackerState{},
	}
}

func (ct *InMemoryConsensusTracker) Valid() bool {
	return ct.GetLatestBlockNumber() > 0 &&
		ct.GetSafeBlockNumber() > 0 &&
		ct.GetFinalizedBlockNumber() > 0
}

func (ct *InMemoryConsensusTracker) Behind(other *InMemoryConsensusTracker) bool {
	return ct.GetLatestBlockNumber() < other.GetLatestBlockNumber() ||
		ct.GetSafeBlockNumber() < other.GetSafeBlockNumber() ||
		ct.GetFinalizedBlockNumber() < other.GetFinalizedBlockNumber()
}

func (ct *InMemoryConsensusTracker) GetLatestBlockNumber() hexutil.Uint64 {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	return ct.state.Latest
}

func (ct *InMemoryConsensusTracker) SetLatestBlockNumber(blockNumber hexutil.Uint64) {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	ct.state.Latest = blockNumber
}

func (ct *InMemoryConsensusTracker) GetSafeBlockNumber() hexutil.Uint64 {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	return ct.state.Safe
}

func (ct *InMemoryConsensusTracker) SetSafeBlockNumber(blockNumber hexutil.Uint64) {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	ct.state.Safe = blockNumber
}

func (ct *InMemoryConsensusTracker) GetFinalizedBlockNumber() hexutil.Uint64 {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	return ct.state.Finalized
}

func (ct *InMemoryConsensusTracker) SetFinalizedBlockNumber(blockNumber hexutil.Uint64) {
	defer ct.mutex.Unlock()
	ct.mutex.Lock()

	ct.state.Finalized = blockNumber
}

// RedisConsensusTracker store and retrieve in a shared Redis cluster, with leader election
type RedisConsensusTracker struct {
	ctx          context.Context
	client       *redis.Client
	namespace    string
	backendGroup *BackendGroup

	redlock           *redsync.Mutex
	lockPeriod        time.Duration
	heartbeatInterval time.Duration

	leader     bool
	leaderName string

	// holds the state collected by local pollers
	local *InMemoryConsensusTracker

	// holds a copy of the remote shared state
	// when leader, updates the remote with the local state
	remote *InMemoryConsensusTracker
}

type RedisConsensusTrackerOpt func(cp *RedisConsensusTracker)

func WithLockPeriod(lockPeriod time.Duration) RedisConsensusTrackerOpt {
	return func(ct *RedisConsensusTracker) {
		ct.lockPeriod = lockPeriod
	}
}

func WithHeartbeatInterval(heartbeatInterval time.Duration) RedisConsensusTrackerOpt {
	return func(ct *RedisConsensusTracker) {
		ct.heartbeatInterval = heartbeatInterval
	}
}
func NewRedisConsensusTracker(ctx context.Context,
	redisClient *redis.Client,
	bg *BackendGroup,
	namespace string,
	opts ...RedisConsensusTrackerOpt) ConsensusTracker {

	tracker := &RedisConsensusTracker{
		ctx:          ctx,
		client:       redisClient,
		backendGroup: bg,
		namespace:    namespace,

		lockPeriod:        30 * time.Second,
		heartbeatInterval: 2 * time.Second,
		local:             NewInMemoryConsensusTracker().(*InMemoryConsensusTracker),
		remote:            NewInMemoryConsensusTracker().(*InMemoryConsensusTracker),
	}

	for _, opt := range opts {
		opt(tracker)
	}

	return tracker
}

func (ct *RedisConsensusTracker) Init() {
	go func() {
		for {
			timer := time.NewTimer(ct.heartbeatInterval)
			ct.stateHeartbeat()

			select {
			case <-timer.C:
				continue
			case <-ct.ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}

func (ct *RedisConsensusTracker) stateHeartbeat() {
	pool := goredis.NewPool(ct.client)
	rs := redsync.New(pool)
	key := ct.key("mutex")

	val, err := ct.client.Get(ct.ctx, key).Result()
	if err != nil && err != redis.Nil {
		log.Error("failed to read the lock", "err", err)
		if ct.leader {
			ok, err := ct.redlock.Unlock()
			if err != nil || !ok {
				log.Error("failed to release the lock after error", "err", err)
				return
			}
			ct.leader = false
		}
		return
	}
	if val != "" {
		if ct.leader {
			log.Debug("extending lock")
			ok, err := ct.redlock.Extend()
			if err != nil || !ok {
				log.Error("failed to extend lock", "err", err, "mutex", ct.redlock.Name(), "val", ct.redlock.Value())
				ok, err := ct.redlock.Unlock()
				if err != nil || !ok {
					log.Error("failed to release the lock after error", "err", err)
					return
				}
				ct.leader = false
				return
			}
			ct.postPayload(val)
		} else {
			// retrieve current leader
			leaderName, err := ct.client.Get(ct.ctx, ct.key(fmt.Sprintf("leader:%s", val))).Result()
			if err != nil && err != redis.Nil {
				log.Error("failed to read the remote leader", "err", err)
				return
			}
			ct.leaderName = leaderName
			log.Debug("following", "val", val, "leader", leaderName)
			// retrieve payload
			val, err := ct.client.Get(ct.ctx, ct.key(fmt.Sprintf("state:%s", val))).Result()
			if err != nil && err != redis.Nil {
				log.Error("failed to read the remote state", "err", err)
				return
			}
			if val == "" {
				log.Error("remote state is missing (recent leader election maybe?)")
				return
			}
			state := &ConsensusTrackerState{}
			err = json.Unmarshal([]byte(val), state)
			if err != nil {
				log.Error("failed to unmarshal the remote state", "err", err)
				return
			}

			ct.remote.update(state)
			log.Debug("updated state from remote", "state", val, "leader", leaderName)

			RecordGroupConsensusHALatestBlock(ct.backendGroup, leaderName, ct.remote.state.Latest)
			RecordGroupConsensusHASafeBlock(ct.backendGroup, leaderName, ct.remote.state.Safe)
			RecordGroupConsensusHAFinalizedBlock(ct.backendGroup, leaderName, ct.remote.state.Finalized)
		}
	} else {
		if !ct.local.Valid() {
			log.Warn("local state is not valid or behind remote, skipping")
			return
		}
		if ct.remote.Valid() && ct.local.Behind(ct.remote) {
			log.Warn("local state is behind remote, skipping")
			return
		}

		log.Info("lock not found, creating a new one")

		mutex := rs.NewMutex(key,
			redsync.WithExpiry(ct.lockPeriod),
			redsync.WithFailFast(true),
			redsync.WithTries(1))

		// nosemgrep: missing-unlock-before-return
		// this lock is hold indefinitely, and it is extended until the leader dies
		if err := mutex.Lock(); err != nil {
			log.Debug("failed to obtain lock", "err", err)
			ct.leader = false
			return
		}

		log.Info("lock acquired", "mutex", mutex.Name(), "val", mutex.Value())
		ct.redlock = mutex
		ct.leader = true
		ct.postPayload(mutex.Value())
	}
}

func (ct *RedisConsensusTracker) key(tag string) string {
	return fmt.Sprintf("consensus:%s:%s", ct.namespace, tag)
}

func (ct *RedisConsensusTracker) GetLatestBlockNumber() hexutil.Uint64 {
	return ct.remote.GetLatestBlockNumber()
}

func (ct *RedisConsensusTracker) SetLatestBlockNumber(blockNumber hexutil.Uint64) {
	ct.local.SetLatestBlockNumber(blockNumber)
}

func (ct *RedisConsensusTracker) GetSafeBlockNumber() hexutil.Uint64 {
	return ct.remote.GetSafeBlockNumber()
}

func (ct *RedisConsensusTracker) SetSafeBlockNumber(blockNumber hexutil.Uint64) {
	ct.local.SetSafeBlockNumber(blockNumber)
}

func (ct *RedisConsensusTracker) GetFinalizedBlockNumber() hexutil.Uint64 {
	return ct.remote.GetFinalizedBlockNumber()
}

func (ct *RedisConsensusTracker) SetFinalizedBlockNumber(blockNumber hexutil.Uint64) {
	ct.local.SetFinalizedBlockNumber(blockNumber)
}

func (ct *RedisConsensusTracker) postPayload(mutexVal string) {
	jsonState, err := json.Marshal(ct.local.state)
	if err != nil {
		log.Error("failed to marshal local", "err", err)
		ct.leader = false
		return
	}
	ct.client.Set(ct.ctx, ct.key(fmt.Sprintf("state:%s", mutexVal)), jsonState, ct.lockPeriod)

	leader, _ := os.LookupEnv("HOSTNAME")
	ct.client.Set(ct.ctx, ct.key(fmt.Sprintf("leader:%s", mutexVal)), leader, ct.lockPeriod)

	log.Debug("posted state", "state", string(jsonState), "leader", leader)

	ct.leaderName = leader
	ct.remote.update(ct.local.state)

	RecordGroupConsensusHALatestBlock(ct.backendGroup, leader, ct.remote.state.Latest)
	RecordGroupConsensusHASafeBlock(ct.backendGroup, leader, ct.remote.state.Safe)
	RecordGroupConsensusHAFinalizedBlock(ct.backendGroup, leader, ct.remote.state.Finalized)
}
