package kvstore

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	_ "modernc.org/sqlite"
)

// read/write mode for user/group/other, executable only for user.
const diskPermission = 0766

// The name of the driver for the DiskKV.
const driverName = "sqlite"

// DiskKV is a disk-backed key-value store, every key-value pair is a hex-encoded .txt file, with the value as content.
// DiskKV is safe for concurrent use with a single DiskKV instance.
// DiskKV is safe for concurrent use between different DiskKV instances of the same disk directory as long as the
// file system supports atomic renames.
type DiskKV struct {
	sync.RWMutex
	db *sql.DB
}

// NewDiskKV creates a DiskKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func NewDiskKV(path string) (*DiskKV, error) {
	if err := os.MkdirAll(path, diskPermission); err != nil {
		return nil, err
	}
	db, err := sql.Open(driverName, filepath.Join(path, "kvstore.db"))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS kv_store (key BLOB PRIMARY KEY, value TSQLRawBlob)")
	if err != nil {
		return nil, err
	}

	return &DiskKV{db: db}, nil
}

func (d *DiskKV) Put(k common.Hash, v []byte) error {
	d.Lock()
	defer d.Unlock()

	s, err := d.db.Prepare("INSERT OR REPLACE INTO kv_store (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = s.Exec(k.Bytes(), v)
	if err != nil {
		return err
	}

	return nil
}

func (d *DiskKV) Get(k common.Hash) ([]byte, error) {
	d.RLock()
	defer d.RUnlock()

	s, err := d.db.Prepare("SELECT value FROM kv_store WHERE key = ?")
	if err != nil {
		return nil, err
	}
	defer s.Close()

	var v []byte
	row := s.QueryRow(k.Bytes())
	if err := row.Scan(&v); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return v, nil
}

func (d *DiskKV) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

var _ KV = (*DiskKV)(nil)
