package loadtest

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// OptimizedRoundRobin is a load balancer that attempts to reduce contention
// by using random selection with health tracking instead of pure round-robin.
type OptimizedRoundRobin[T any] struct {
	items       []T
	healthStats []healthStat
	mu          sync.RWMutex
	rng         *rand.Rand
}

type healthStat struct {
	failures  atomic.Uint64
	successes atomic.Uint64
	lastUsed  atomic.Int64 // Unix timestamp
}

// NewOptimizedRoundRobin creates a new optimized round robin selector.
func NewOptimizedRoundRobin[T any](items []T) *OptimizedRoundRobin[T] {
	return &OptimizedRoundRobin[T]{
		items:       items,
		healthStats: make([]healthStat, len(items)),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Get returns an item using weighted random selection based on health and usage.
func (p *OptimizedRoundRobin[T]) Get() T {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.items) == 1 {
		return p.items[0]
	}

	// Calculate weights based on health and last usage
	weights := make([]float64, len(p.items))
	totalWeight := 0.0
	now := time.Now().Unix()

	for i := range p.items {
		stat := &p.healthStats[i]
		failures := stat.failures.Load()
		successes := stat.successes.Load()
		lastUsed := stat.lastUsed.Load()

		// Base weight (higher is better)
		weight := 1.0

		// Reduce weight for items with high failure rate
		if total := failures + successes; total > 0 {
			failureRate := float64(failures) / float64(total)
			weight *= (1.0 - failureRate*0.5) // Reduce by up to 50%
		}

		// Increase weight for items not used recently
		timeSinceUsed := now - lastUsed
		if timeSinceUsed > 1 { // More than 1 second
			weight *= 1.0 + float64(timeSinceUsed)*0.1
		}

		weights[i] = weight
		totalWeight += weight
	}

	// Weighted random selection
	target := p.rng.Float64() * totalWeight
	current := 0.0
	for i, weight := range weights {
		current += weight
		if current >= target {
			p.healthStats[i].lastUsed.Store(now)
			return p.items[i]
		}
	}

	// Fallback (shouldn't happen)
	return p.items[0]
}

// RecordSuccess records a successful operation for the item at the given index.
func (p *OptimizedRoundRobin[T]) RecordSuccess(item T) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for i, candidate := range p.items {
		// Use pointer comparison or type assertion as needed
		if &candidate == &item {
			p.healthStats[i].successes.Add(1)
			break
		}
	}
}

// RecordFailure records a failed operation for the item at the given index.
func (p *OptimizedRoundRobin[T]) RecordFailure(item T) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for i, candidate := range p.items {
		// Use pointer comparison or type assertion as needed
		if &candidate == &item {
			p.healthStats[i].failures.Add(1)
			break
		}
	}
}
