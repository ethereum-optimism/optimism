package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru/v2"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-base32"
)

const (
	scoreDataV0       = "0"
	scoreCacheSize    = 100
	expiryPeriod      = 24 * time.Hour
	maxPruneBatchSize = 20
)

var scoresBase = ds.NewKey("/peers/scores")

type scoreRecord struct {
	PeerScores PeerScores `json:"peerScores"`
	LastUpdate int64      `json:"lastUpdate"` // unix timestamp in seconds
}

type scoreBook struct {
	ctx      context.Context
	cancelFn context.CancelFunc
	clock    clock.Clock
	log      log.Logger
	bgTasks  sync.WaitGroup
	store    ds.Batching
	cache    *lru.Cache[peer.ID, scoreRecord]
	sync.RWMutex
}

func newScoreBook(ctx context.Context, logger log.Logger, clock clock.Clock, store ds.Batching) (*scoreBook, error) {
	cache, err := lru.New[peer.ID, scoreRecord](scoreCacheSize)
	if err != nil {
		return nil, fmt.Errorf("creating cache: %w", err)
	}

	ctx, cancelFn := context.WithCancel(ctx)
	book := scoreBook{
		ctx:      ctx,
		cancelFn: cancelFn,
		clock:    clock,
		log:      logger,
		store:    store,
		cache:    cache,
	}
	return &book, nil
}

func (d *scoreBook) startGC() {
	startGc(d.ctx, d.log, d.clock, &d.bgTasks, d.prune)
}

func (d *scoreBook) GetPeerScores(id peer.ID) (PeerScores, error) {
	d.RLock()
	defer d.RUnlock()
	record, err := d.getRecord(id)
	if err != nil {
		return PeerScores{}, nil
	}
	return record.PeerScores, err
}

func (d *scoreBook) getRecord(id peer.ID) (scoreRecord, error) {
	if scores, ok := d.cache.Get(id); ok {
		return scores, nil
	}
	data, err := d.store.Get(d.ctx, scoreKey(id))
	if errors.Is(err, ds.ErrNotFound) {
		return scoreRecord{}, nil
	} else if err != nil {
		return scoreRecord{}, fmt.Errorf("load scores for peer %v: %w", id, err)
	}
	record, err := deserializeScoresV0(data)
	if err != nil {
		return scoreRecord{}, fmt.Errorf("invalid score data for peer %v: %w", id, err)
	}
	d.cache.Add(id, record)
	return record, nil
}

func (d *scoreBook) SetScore(id peer.ID, diff ScoreDiff) error {
	d.Lock()
	defer d.Unlock()
	scores, err := d.getRecord(id)
	if err != nil {
		return err
	}
	scores.LastUpdate = d.clock.Now().Unix()
	diff.Apply(&scores)
	data, err := serializeScoresV0(scores)
	if err != nil {
		return fmt.Errorf("encode scores for peer %v: %w", id, err)
	}
	err = d.store.Put(d.ctx, scoreKey(id), data)
	if err != nil {
		return fmt.Errorf("storing updated scores for peer %v: %w", id, err)
	}
	d.cache.Add(id, scores)
	return nil
}

// prune deletes entries from the store that are older than expiryPeriod.
// Note that the expiry period is not a strict TTL. Entries that are eligible for deletion may still be present
// either because the prune function hasn't yet run or because they are still preserved in the in-memory cache after
// having been deleted from the database.
func (d *scoreBook) prune() error {
	results, err := d.store.Query(d.ctx, query.Query{
		Prefix: scoresBase.String(),
	})
	if err != nil {
		return err
	}
	pending := 0
	batch, err := d.store.Batch(d.ctx)
	if err != nil {
		return err
	}
	for result := range results.Next() {
		// Bail out if the context is done
		select {
		case <-d.ctx.Done():
			return d.ctx.Err()
		default:
		}
		record, err := deserializeScoresV0(result.Value)
		if err != nil {
			return err
		}
		if time.Unix(record.LastUpdate, 0).Add(expiryPeriod).Before(d.clock.Now()) {
			if pending > maxPruneBatchSize {
				if err := batch.Commit(d.ctx); err != nil {
					return err
				}
				batch, err = d.store.Batch(d.ctx)
				if err != nil {
					return err
				}
				pending = 0
			}
			pending++
			if err := batch.Delete(d.ctx, ds.NewKey(result.Key)); err != nil {
				return err
			}
		}
	}
	if err := batch.Commit(d.ctx); err != nil {
		return err
	}
	return nil
}

func (d *scoreBook) Close() {
	d.cancelFn()
	d.bgTasks.Wait()
}

func scoreKey(id peer.ID) ds.Key {
	return scoresBase.ChildString(base32.RawStdEncoding.EncodeToString([]byte(id))).ChildString(scoreDataV0)
}
