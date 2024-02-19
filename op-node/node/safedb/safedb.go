package safedb

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidEntry = errors.New("invalid db entry")
)

type SafeDB struct {
	log log.Logger
	db  *pebble.DB

	closed atomic.Bool
}

func KeyL1BlockNum(num uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, 0)
	key = binary.LittleEndian.AppendUint64(key, num)
	return key
}

func ValueL1BlockNum(l1Hash common.Hash, l2Hash common.Hash) []byte {
	val := make([]byte, 0, 64)
	val = append(val, l1Hash.Bytes()...)
	val = append(val, l2Hash.Bytes()...)
	return val
}

func DecodeValueL1BlockNum(val []byte) (l1Hash common.Hash, l2Hash common.Hash, err error) {
	if len(val) != 64 {
		err = ErrInvalidEntry
		return
	}
	copy(l1Hash[:], val[:32])
	copy(l2Hash[:], val[32:])
	return
}

func NewSafeDB(logger log.Logger, path string) (*SafeDB, error) {
	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &SafeDB{
		log: logger,
		db:  db,
	}, nil
}

func (d *SafeDB) SafeHeadUpdated(safeHead eth.L2BlockRef, l1Head eth.BlockID) {
	d.log.Debug("Update safe head", "l2", safeHead.ID(), "l1", l1Head)
	if err := d.db.Set(KeyL1BlockNum(l1Head.Number), ValueL1BlockNum(l1Head.Hash, safeHead.Hash), &pebble.WriteOptions{Sync: true}); err != nil {
		// TODO(client-pod#593): Need to work out how to not drop the update and lose data here.
		d.log.Error("Failed to record safe head update", "err", err)
	}
}

func (d *SafeDB) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (l1Hash common.Hash, l2Hash common.Hash, err error) {
	iter, err := d.db.NewIterWithContext(ctx, &pebble.IterOptions{
		LowerBound: KeyL1BlockNum(0),
		UpperBound: KeyL1BlockNum(math.MaxUint64),
	})
	if err != nil {
		return
	}
	defer iter.Close()
	if valid := iter.SeekLT(KeyL1BlockNum(l1BlockNum + 1)); !valid {
		err = ErrNotFound
		return
	}
	// Found an entry at or before the requested L1 block
	val, err := iter.ValueAndErr()
	if err != nil {
		return
	}
	l1Hash, l2Hash, err = DecodeValueL1BlockNum(val)
	return
}

func (d *SafeDB) Close() error {
	if !d.closed.CompareAndSwap(false, true) {
		// Already closed
		return nil
	}
	return d.db.Close()
}
