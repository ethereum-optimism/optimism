package proxyd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
)

const MaxRPSScript = `
local current
current = redis.call("incr", KEYS[1])
if current == 1 then
    redis.call("expire", KEYS[1], 1)
end
return current
`

const MaxConcurrentWSConnsScript = `
redis.call("sadd", KEYS[1], KEYS[2])
local total = 0
local scanres = redis.call("sscan", KEYS[1], 0)
for _, k in ipairs(scanres[2]) do
	local value = redis.call("get", k)
	if value then
		total = total + value
	end
end

if total < tonumber(ARGV[1]) then
	redis.call("incr", KEYS[2])
	redis.call("expire", KEYS[2], 300)
	return true
end

return false
`

type BackendRateLimiter interface {
	IsBackendOnline(name string) (bool, error)
	SetBackendOffline(name string, duration time.Duration) error
	IncBackendRPS(name string) (int, error)
	IncBackendWSConns(name string, max int) (bool, error)
	DecBackendWSConns(name string) error
	FlushBackendWSConns(names []string) error
}

type RedisBackendRateLimiter struct {
	rdb       *redis.Client
	randID    string
	touchKeys map[string]time.Duration
	tkMtx     sync.Mutex
}

func NewRedisRateLimiter(rdb *redis.Client) BackendRateLimiter {
	out := &RedisBackendRateLimiter{
		rdb:       rdb,
		randID:    randStr(20),
		touchKeys: make(map[string]time.Duration),
	}
	go out.touch()
	return out
}

func (r *RedisBackendRateLimiter) IsBackendOnline(name string) (bool, error) {
	exists, err := r.rdb.Exists(context.Background(), fmt.Sprintf("backend:%s:offline", name)).Result()
	if err != nil {
		RecordRedisError("IsBackendOnline")
		return false, wrapErr(err, "error getting backend availability")
	}

	return exists == 0, nil
}

func (r *RedisBackendRateLimiter) SetBackendOffline(name string, duration time.Duration) error {
	if duration == 0 {
		return nil
	}
	err := r.rdb.SetEX(
		context.Background(),
		fmt.Sprintf("backend:%s:offline", name),
		1,
		duration,
	).Err()
	if err != nil {
		RecordRedisError("SetBackendOffline")
		return wrapErr(err, "error setting backend unavailable")
	}
	return nil
}

func (r *RedisBackendRateLimiter) IncBackendRPS(name string) (int, error) {
	cmd := r.rdb.Eval(
		context.Background(),
		MaxRPSScript,
		[]string{fmt.Sprintf("backend:%s:ratelimit", name)},
	)
	rps, err := cmd.Int()
	if err != nil {
		RecordRedisError("IncBackendRPS")
		return -1, wrapErr(err, "error upserting backend rate limit")
	}
	return rps, nil
}

func (r *RedisBackendRateLimiter) IncBackendWSConns(name string, max int) (bool, error) {
	connsKey := fmt.Sprintf("proxy:%s:wsconns:%s", r.randID, name)
	r.tkMtx.Lock()
	r.touchKeys[connsKey] = 5 * time.Minute
	r.tkMtx.Unlock()
	cmd := r.rdb.Eval(
		context.Background(),
		MaxConcurrentWSConnsScript,
		[]string{
			fmt.Sprintf("backend:%s:proxies", name),
			connsKey,
		},
		max,
	)
	incremented, err := cmd.Bool()
	// false gets coerced to redis.nil, see https://redis.io/commands/eval#conversion-between-lua-and-redis-data-types
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		RecordRedisError("IncBackendWSConns")
		return false, wrapErr(err, "error incrementing backend ws conns")
	}
	return incremented, nil
}

func (r *RedisBackendRateLimiter) DecBackendWSConns(name string) error {
	connsKey := fmt.Sprintf("proxy:%s:wsconns:%s", r.randID, name)
	err := r.rdb.Decr(context.Background(), connsKey).Err()
	if err != nil {
		RecordRedisError("DecBackendWSConns")
		return wrapErr(err, "error decrementing backend ws conns")
	}
	return nil
}

func (r *RedisBackendRateLimiter) FlushBackendWSConns(names []string) error {
	ctx := context.Background()
	for _, name := range names {
		connsKey := fmt.Sprintf("proxy:%s:wsconns:%s", r.randID, name)
		err := r.rdb.SRem(
			ctx,
			fmt.Sprintf("backend:%s:proxies", name),
			connsKey,
		).Err()
		if err != nil {
			return wrapErr(err, "error flushing backend ws conns")
		}
		err = r.rdb.Del(ctx, connsKey).Err()
		if err != nil {
			return wrapErr(err, "error flushing backend ws conns")
		}
	}
	return nil
}

func (r *RedisBackendRateLimiter) touch() {
	for {
		r.tkMtx.Lock()
		for key, dur := range r.touchKeys {
			if err := r.rdb.Expire(context.Background(), key, dur).Err(); err != nil {
				RecordRedisError("touch")
				log.Error("error touching redis key", "key", key, "err", err)
			}
		}
		r.tkMtx.Unlock()
		time.Sleep(5 * time.Second)
	}
}

type LocalBackendRateLimiter struct {
	deadBackends   map[string]time.Time
	backendRPS     map[string]int
	backendWSConns map[string]int
	mtx            sync.RWMutex
}

func NewLocalBackendRateLimiter() *LocalBackendRateLimiter {
	out := &LocalBackendRateLimiter{
		deadBackends:   make(map[string]time.Time),
		backendRPS:     make(map[string]int),
		backendWSConns: make(map[string]int),
	}
	go out.clear()
	return out
}

func (l *LocalBackendRateLimiter) IsBackendOnline(name string) (bool, error) {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.deadBackends[name].Before(time.Now()), nil
}

func (l *LocalBackendRateLimiter) SetBackendOffline(name string, duration time.Duration) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.deadBackends[name] = time.Now().Add(duration)
	return nil
}

func (l *LocalBackendRateLimiter) IncBackendRPS(name string) (int, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.backendRPS[name] += 1
	return l.backendRPS[name], nil
}

func (l *LocalBackendRateLimiter) IncBackendWSConns(name string, max int) (bool, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.backendWSConns[name] == max {
		return false, nil
	}
	l.backendWSConns[name] += 1
	return true, nil
}

func (l *LocalBackendRateLimiter) DecBackendWSConns(name string) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.backendWSConns[name] == 0 {
		return nil
	}
	l.backendWSConns[name] -= 1
	return nil
}

func (l *LocalBackendRateLimiter) FlushBackendWSConns(names []string) error {
	return nil
}

func (l *LocalBackendRateLimiter) clear() {
	for {
		time.Sleep(time.Second)
		l.mtx.Lock()
		l.backendRPS = make(map[string]int)
		l.mtx.Unlock()
	}
}

func randStr(l int) string {
	b := make([]byte, l)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type ServerRateLimiter struct {
}
