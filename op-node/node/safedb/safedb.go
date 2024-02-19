package safedb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
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

	writeOpts *pebble.WriteOptions

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
		log:       logger,
		db:        db,
		writeOpts: &pebble.WriteOptions{Sync: true},
	}, nil
}

func (d *SafeDB) SafeHeadUpdated(safeHead eth.L2BlockRef, l1Head eth.BlockID) error {
	d.log.Debug("Update safe head", "l2", safeHead.ID(), "l1", l1Head)
	// Delete any entries after this L1 block. Normally the l1Head continuously increases and this does nothing
	// However when the pipeline resets the L1 head may drop back and we need to remove later entries and allow them
	// to be repopulated as derivation progresses again. The resulting data may be different if L1 reorged.
	if err := d.db.DeleteRange(KeyL1BlockNum(l1Head.Number+1), KeyL1BlockNum(math.MaxUint64), d.writeOpts); err != nil {
		return fmt.Errorf("failed to truncate safe head entries: %w", err)
	}
	if err := d.db.Set(KeyL1BlockNum(l1Head.Number), ValueL1BlockNum(l1Head.Hash, safeHead.Hash), d.writeOpts); err != nil {
		// TODO(client-pod#593): Add tests to ensure we don't lose data here
		// We do in fact lose this update here. Even if we didn't the correct behaviour is to retry the exact same write
		// so maybe we should just keep retrying here instead of returning an error?
		return fmt.Errorf("failed to record safe head update: %w", err)
	}
	return nil
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
