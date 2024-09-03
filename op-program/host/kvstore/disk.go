package kvstore

import (
	"os"
	"sync"

	"github.com/bmatsuo/lmdb-go/lmdb"
	"github.com/ethereum/go-ethereum/common"
)

// read/write mode for user/group/other, executable only for user.
const diskPermission = 0766

// DiskKV is a disk-backed key-value store, every key-value pair is a hex-encoded .txt file, with the value as content.
// DiskKV is safe for concurrent use with a single DiskKV instance.
// DiskKV is safe for concurrent use between different DiskKV instances of the same disk directory as long as the
// file system supports atomic renames.
type DiskKV struct {
	sync.RWMutex
	env *lmdb.Env
}

// NewDiskKV creates a DiskKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func NewDiskKV(path string) (*DiskKV, error) {
	env, err := lmdb.NewEnv()
	if err != nil {
		return nil, err
	}
	// Only allow one database in the environment
	if err := env.SetMaxDBs(1); err != nil {
		return nil, err
	}
	// Set a 1GB map size
	if err = env.SetMapSize(1 << 30); err != nil {
		return nil, err
	}

	if err = os.MkdirAll(path, diskPermission); err != nil {
		return nil, err
	}
	if err = env.Open(path, lmdb.Create, diskPermission); err != nil {
		return nil, err
	}

	return &DiskKV{env: env}, nil
}

func (d *DiskKV) Put(k common.Hash, v []byte) error {
	d.Lock()
	defer d.Unlock()

	return d.env.Update(func(txn *lmdb.Txn) error {
		// Open the kvstore dbi, creating it if it doesn't exist.
		dbi, err := txn.OpenDBI("kvstore", lmdb.Create)
		if err != nil {
			return err
		}

		return txn.Put(dbi, k.Bytes(), v, 0)
	})
}

func (d *DiskKV) Get(k common.Hash) ([]byte, error) {
	d.RLock()
	defer d.RUnlock()

	var v []byte
	err := d.env.Update(func(txn *lmdb.Txn) error {
		// Open the kvstore dbi, creating it if it doesn't exist.
		dbi, err := txn.OpenDBI("kvstore", lmdb.Create)
		if err != nil {
			return err
		}

		v, err = txn.Get(dbi, k.Bytes())
		return err
	})
	if err != nil {
		if lmdb.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return v, nil
}

func (d *DiskKV) Close() error {
	return d.env.Close()
}

var _ KV = (*DiskKV)(nil)
