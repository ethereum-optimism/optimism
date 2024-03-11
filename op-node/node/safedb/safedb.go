package safedb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidEntry = errors.New("invalid db entry")
)

const (
	// Keys are prefixed with a constant byte to allow us to differentiate different "columns" within the data
	keyPrefixSafeByL1BlockNum byte = 0
)

var (
	safeByL1BlockNumKey = uint64Key{prefix: keyPrefixSafeByL1BlockNum}
)

type uint64Key struct {
	prefix byte
}

func (c uint64Key) Of(num uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, c.prefix)
	key = binary.BigEndian.AppendUint64(key, num)
	return key
}
func (c uint64Key) Max() []byte {
	return c.Of(math.MaxUint64)
}

func (c uint64Key) IterRange() *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: c.Of(0),
		UpperBound: c.Max(),
	}
}

type SafeDB struct {
	// m ensures all read iterators are closed before closing the database by preventing concurrent read and write
	// operations (with close considered a write operation).
	m   sync.RWMutex
	log log.Logger
	db  *pebble.DB

	writeOpts *pebble.WriteOptions

	closed bool
}

func safeByL1BlockNumValue(l1 eth.BlockID, l2 eth.BlockID) []byte {
	val := make([]byte, 0, 72)
	val = append(val, l1.Hash.Bytes()...)
	val = append(val, l2.Hash.Bytes()...)
	val = binary.BigEndian.AppendUint64(val, l2.Number)
	return val
}

func decodeSafeByL1BlockNum(key []byte, val []byte) (l1 eth.BlockID, l2 eth.BlockID, err error) {
	if len(key) != 9 || len(val) != 72 || key[0] != keyPrefixSafeByL1BlockNum {
		err = ErrInvalidEntry
		return
	}
	copy(l1.Hash[:], val[:32])
	l1.Number = binary.BigEndian.Uint64(key[1:])
	copy(l2.Hash[:], val[32:64])
	l2.Number = binary.BigEndian.Uint64(val[64:])
	return
}

func NewSafeDB(logger log.Logger, path string) (*SafeDB, error) {
	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &SafeDB{
		log:       logger,
		db:        db,
		writeOpts: &pebble.WriteOptions{Sync: true},
	}, nil
}

func (d *SafeDB) Enabled() bool {
	return true
}

func (d *SafeDB) SafeHeadUpdated(safeHead eth.L2BlockRef, l1Head eth.BlockID) error {
	d.m.Lock()
	defer d.m.Unlock()
	d.log.Info("Record safe head", "l2", safeHead.ID(), "l1", l1Head)
	batch := d.db.NewBatch()
	defer batch.Close()
	if err := batch.Set(safeByL1BlockNumKey.Of(l1Head.Number), safeByL1BlockNumValue(l1Head, safeHead.ID()), d.writeOpts); err != nil {
		return fmt.Errorf("failed to record safe head update: %w", err)
	}
	if err := batch.Commit(d.writeOpts); err != nil {
		return fmt.Errorf("failed to commit safe head update: %w", err)
	}
	return nil
}

func (d *SafeDB) SafeHeadReset(safeHead eth.L2BlockRef) error {
	d.m.Lock()
	defer d.m.Unlock()
	iter, err := d.db.NewIter(safeByL1BlockNumKey.IterRange())
	if err != nil {
		return fmt.Errorf("reset failed to create iterator: %w", err)
	}
	defer iter.Close()
	if valid := iter.SeekGE(safeByL1BlockNumKey.Of(safeHead.L1Origin.Number)); !valid {
		// Reached end of column without finding any entries to delete
		return nil
	}
	for {
		val, err := iter.ValueAndErr()
		if err != nil {
			return fmt.Errorf("reset failed to read entry: %w", err)
		}
		l1Block, l2Block, err := decodeSafeByL1BlockNum(iter.Key(), val)
		if err != nil {
			return fmt.Errorf("reset encountered invalid entry: %w", err)
		}
		if l2Block.Number >= safeHead.Number {
			// Keep a copy of this key - it may be modified when calling Prev()
			l1HeadKey := slices.Clone(iter.Key())
			hasPrevEntry := iter.Prev()
			// Found the first entry that made the new safe head safe.
			batch := d.db.NewBatch()
			if err := batch.DeleteRange(l1HeadKey, safeByL1BlockNumKey.Max(), d.writeOpts); err != nil {
				return fmt.Errorf("reset failed to delete entries after %v: %w", l1HeadKey, err)
			}

			// If we reset to a safe head before the first entry, we don't know if the new safe head actually became
			// safe in that L1 block or if it was just before our records start, so don't record it as safe at the
			// specified L1 block.
			if hasPrevEntry {
				if err := batch.Set(l1HeadKey, safeByL1BlockNumValue(l1Block, safeHead.ID()), d.writeOpts); err != nil {
					return fmt.Errorf("reset failed to record safe head update: %w", err)
				}
			}
			if err := batch.Commit(d.writeOpts); err != nil {
				return fmt.Errorf("reset failed to commit batch: %w", err)
			}
			return nil
		}
		if valid := iter.Next(); !valid {
			// Reached end of column
			return nil
		}
	}
}

func (d *SafeDB) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (l1Block eth.BlockID, safeHead eth.BlockID, err error) {
	d.m.RLock()
	defer d.m.RUnlock()
	iter, err := d.db.NewIterWithContext(ctx, safeByL1BlockNumKey.IterRange())
	if err != nil {
		return
	}
	defer iter.Close()
	if valid := iter.SeekLT(safeByL1BlockNumKey.Of(l1BlockNum + 1)); !valid {
		err = ErrNotFound
		return
	}
	// Found an entry at or before the requested L1 block
	val, err := iter.ValueAndErr()
	if err != nil {
		return
	}
	l1Block, safeHead, err = decodeSafeByL1BlockNum(iter.Key(), val)
	return
}

func (d *SafeDB) Close() error {
	d.m.Lock()
	defer d.m.Unlock()
	if d.closed {
		// Already closed
		return nil
	}
	d.closed = true
	return d.db.Close()
}
