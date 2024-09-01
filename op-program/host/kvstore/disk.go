package kvstore

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
)

// DiskKV is a disk-backed key-value store, with PebbleDB as the underlying DBMS.
// DiskKV is safe for concurrent use with a single DiskKV instance.
// DiskKV is safe for concurrent use between different DiskKV instances of the same disk directory as long as the
// file system supports atomic renames.
type DiskKV struct {
	sync.RWMutex
	db *pebble.DB
}

// NewDiskKV creates a DiskKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func NewDiskKV(path string) *DiskKV {
	levels := make([]pebble.LevelOptions, 1)
	levels[0].Compression = pebble.ZstdCompression
	opts := &pebble.Options{
		Cache:                    pebble.NewCache(int64(32 * 1024 * 1024)),
		MaxConcurrentCompactions: runtime.NumCPU,
		Levels: []pebble.LevelOptions{
			{Compression: pebble.ZstdCompression},
		},
	}
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
		if err == pebble.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	ret := make([]byte, len(dat))
	copy(ret, dat)
	closer.Close()
	return ret, nil
}

var _ KV = (*DiskKV)(nil)
