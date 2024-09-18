package kvstore

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
)

// pebbleKV is a disk-backed key-value store, with PebbleDB as the underlying DBMS.
// pebbleKV is safe for concurrent use with a single pebbleKV instance.
type pebbleKV struct {
	sync.RWMutex
	db *pebble.DB
}

// newPebbleKV creates a pebbleKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func newPebbleKV(path string) *pebbleKV {
	opts := &pebble.Options{
		Cache:                    pebble.NewCache(int64(32 * 1024 * 1024)),
		MaxConcurrentCompactions: runtime.NumCPU,
		Levels: []pebble.LevelOptions{
			{Compression: pebble.SnappyCompression},
		},
	}
	db, err := pebble.Open(path, opts)
	if err != nil {
		panic(fmt.Errorf("failed to open pebbledb at %s: %w", path, err))
	}

	return &pebbleKV{db: db}
}

func (d *pebbleKV) Put(k common.Hash, v []byte) error {
	d.Lock()
	defer d.Unlock()
	return d.db.Set(k.Bytes(), v, pebble.NoSync)
}

func (d *pebbleKV) Get(k common.Hash) ([]byte, error) {
	d.RLock()
	defer d.RUnlock()

	dat, closer, err := d.db.Get(k.Bytes())
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	ret := make([]byte, len(dat))
	copy(ret, dat)
	closer.Close()
	return ret, nil
}

func (d *pebbleKV) Close() error {
	d.Lock()
	defer d.Unlock()

	return d.db.Close()
}

var _ KV = (*pebbleKV)(nil)
