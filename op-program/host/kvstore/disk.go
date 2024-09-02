package kvstore

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/ethereum/go-ethereum/common"
)

// DiskKV is a disk-backed key-value store, with PebbleDB as the underlying DBMS.
// DiskKV is safe for concurrent use with a single DiskKV instance.
type DiskKV struct {
	sync.RWMutex
	db *pebble.DB
}

// NewDiskKV creates a DiskKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func NewDiskKV(path string, readOnly bool) *DiskKV {
	opts := &pebble.Options{
		Cache:                    pebble.NewCache(int64(32 * 1024 * 1024)),
		MaxConcurrentCompactions: runtime.NumCPU,
		Levels: []pebble.LevelOptions{
			{Compression: pebble.SnappyCompression},
		},
	}

	// Check if the database exists. If it does not, create it.
	desc, err := pebble.Peek(path, vfs.Default)
	if err != nil || !desc.Exists {
		// Attempt to create the database if it does not exist.
		// We ignore the error; if the database cannot be created, the subsequent Open will fail.
		db, _ := pebble.Open(path, opts)
		db.Close()
	}

	opts.ReadOnly = readOnly
	db, err := pebble.Open(path, opts)
	if err != nil {
		panic(fmt.Errorf("failed to open pebbledb at %s: %w", path, err))
	}

	return &DiskKV{db: db}
}

func (d *DiskKV) Put(k common.Hash, v []byte) error {
	d.Lock()
	defer d.Unlock()
	return d.db.Set(k.Bytes(), v, pebble.NoSync)
}

func (d *DiskKV) Get(k common.Hash) ([]byte, error) {
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

func (d *DiskKV) Close() error {
	d.Lock()
	defer d.Unlock()

	return d.db.Close()
}

var _ KV = (*DiskKV)(nil)
