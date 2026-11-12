package dsl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

const (
	persistedBlockPollInterval = 250 * time.Millisecond
	// persistedBlockQuietLimit bounds how long the subscription may stay silent before the
	// tracker counts it as lost. A node persists nothing while it receives no blocks, so this
	// outlasts any pause in block production a test makes.
	persistedBlockQuietLimit = 15 * time.Minute
	// persistedBlockSubscribeMethod is reth's jsonrpsee subscription, whose method and
	// notification names do not follow the eth_subscribe convention go-ethereum's client expects.
	persistedBlockSubscribeMethod = "reth_subscribePersistedBlock"
)

// PersistedBlockTracker follows the newest block an op-reth node reports having written to disk.
// Reth serves the blocks above it from memory.
type PersistedBlockTracker struct {
	ctx     context.Context
	log     log.Logger
	name    string
	require *testreq.Assertions

	latest   eth.BlockID
	mu       sync.Mutex
	readErr  error
	reported bool
}

// TrackPersistedBlock subscribes to the blocks this op-reth node persists and follows the newest
// until the test ends. The subscription reports only the blocks persisted after it starts.
func (el *L2ELNode) TrackPersistedBlock() *PersistedBlockTracker {
	name := el.inner.Name()
	url := el.inner.UserRPC()
	logger := el.log.With("name", name, "url", url)
	logger.Info("Subscribing to persisted blocks")

	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	sub, err := client.SubscribeWS(ctx, client.WSConfig{Log: logger, ReadTimeout: persistedBlockQuietLimit, URL: url},
		persistedBlockSubscribeMethod)
	el.require.NoErrorf(err, "%s did not accept the persisted block subscription", name)
	el.t.Cleanup(sub.Close)

	tracker := &PersistedBlockTracker{ctx: el.ctx, log: logger, name: name, require: el.require}
	go tracker.follow(sub)
	logger.Info("Subscribed to persisted blocks")
	return tracker
}

func (tr *PersistedBlockTracker) String() string {
	return tr.name
}

// follow records every persisted block the subscription reports until the subscription ends or
// reports something that is not a block, which ends the tracker.
func (tr *PersistedBlockTracker) follow(sub *client.WSSubscription) {
	for notification := range sub.Notifications() {
		block, err := decodePersistedBlock(notification)
		if err != nil {
			sub.Close()
			tr.end(err)
			return
		}
		tr.record(block)
	}
	tr.end(sub.Err())
}

func (tr *PersistedBlockTracker) end(cause error) {
	err := fmt.Errorf("persisted block subscription on %s ended: %w", tr.name, cause)
	if errors.Is(cause, client.ErrWSSubscriptionClosed) {
		tr.log.Debug("Persisted block subscription closed", "err", err)
	} else {
		tr.log.Error("Persisted block subscription failed", "err", err)
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.readErr = err
}

func (tr *PersistedBlockTracker) record(block eth.BlockID) {
	tr.log.Debug("Node persisted block", "block", block)
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.latest = block
	tr.reported = true
}

func (tr *PersistedBlockTracker) snapshot() (eth.BlockID, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.readErr != nil {
		return eth.BlockID{}, tr.readErr
	}
	if !tr.reported {
		return eth.BlockID{}, errNoPersistedBlockReported
	}
	return tr.latest, nil
}

var errNoPersistedBlockReported = errors.New("no persisted block reported since the subscription started")

// Latest returns the newest block the node has reported persisted. It fails the test if the node
// has reported none since the tracker started, or if the subscription ended.
func (tr *PersistedBlockTracker) Latest() eth.BlockID {
	block, err := tr.snapshot()
	tr.require.NoErrorf(err, "no persisted block known for %s", tr.name)
	return block
}

// AwaitPersisted waits until the node reports block number, or a later block, persisted and
// returns the block it reported.
func (tr *PersistedBlockTracker) AwaitPersisted(number uint64, timeout time.Duration) eth.BlockID {
	tr.log.Info("Waiting for a block to be persisted", "target", number, "timeout", timeout)
	ctx, cancel := context.WithTimeout(tr.ctx, timeout)
	defer cancel()
	var persisted eth.BlockID
	err := wait.For(ctx, persistedBlockPollInterval, func() (bool, error) {
		block, err := tr.snapshot()
		if errors.Is(err, errNoPersistedBlockReported) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		persisted = block
		return block.Number >= number, nil
	})
	tr.require.NoErrorf(err, "%s did not report block %d persisted within %s; the newest it reported is %s",
		tr.name, number, timeout, persisted)
	tr.log.Info("Block persisted", "target", number, "persisted", persisted)
	return persisted
}

func decodePersistedBlock(notification json.RawMessage) (eth.BlockID, error) {
	var block *eth.BlockID
	if err := json.Unmarshal(notification, &block); err != nil {
		return eth.BlockID{}, fmt.Errorf("undecodable persisted block %q: %w", notification, err)
	}
	if block == nil {
		return eth.BlockID{}, errors.New("persisted block notification carries no block")
	}
	return *block, nil
}
