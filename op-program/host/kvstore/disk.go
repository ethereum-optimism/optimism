package kvstore

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
)

// read/write mode for user/group/other, not executable.
const diskPermission = 0666

// DiskKV is a disk-backed key-value store, every key-value pair is a hex-encoded .txt file, with the value as content.
// DiskKV is safe for concurrent use; Puts may conflict, but write the exact same data anyway.
type DiskKV struct {
	path string
}

// NewDiskKV creates a DiskKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func NewDiskKV(path string) *DiskKV {
	return &DiskKV{path: path}
}

func (d *DiskKV) pathKey(k common.Hash) string {
	return path.Join(d.path, k.String()+".txt")
}

func (d *DiskKV) Put(k common.Hash, v []byte) error {
	f, err := os.OpenFile(d.pathKey(k), os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, diskPermission)
	if err != nil {
		if os.IsExist(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed to open new pre-image file %s: %w", k, err)
	}
	if _, err := f.Write([]byte(hex.EncodeToString(v))); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write pre-image %s to disk: %w", k, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close pre-image %s file: %w", k, err)
	}
	return nil
}

func (d *DiskKV) Get(k common.Hash) ([]byte, error) {
	f, err := os.OpenFile(d.pathKey(k), os.O_RDONLY, diskPermission)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to open pre-image file %s: %w", k, err)
	}
	defer f.Close() // fine to ignore closing error here
	dat, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read pre-image from file %s: %w", k, err)
	}
	return hex.DecodeString(string(dat))
}

var _ KV = (*DiskKV)(nil)
