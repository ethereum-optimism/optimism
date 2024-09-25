package kvstore

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// directoryKV is a disk-backed key-value store, every key-value pair is a hex-encoded .txt file, with the value as content.
// directoryKV is safe for concurrent use with a single directoryKV instance.
// directoryKV is safe for concurrent use between different directoryKV instances of the same disk directory as long as the
// file system supports atomic renames.
type directoryKV struct {
	sync.RWMutex
	path string
}

// newDirectoryKV creates a directoryKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func newDirectoryKV(path string) *directoryKV {
	return &directoryKV{path: path}
}

// pathKey returns the file path for the given key.
// This is composed of the first characters of the non-0x-prefixed hex key as a directory, and the rest as the file name.
func (d *directoryKV) pathKey(k common.Hash) string {
	key := k.String()
	dir, name := key[2:6], key[6:]
	return path.Join(d.path, dir, name+".txt")
}

func (d *directoryKV) Put(k common.Hash, v []byte) error {
	d.Lock()
	defer d.Unlock()
	f, err := openTempFile(d.path, k.String()+".txt.*")
	if err != nil {
		return fmt.Errorf("failed to open temp file for pre-image %s: %w", k, err)
	}
	defer os.Remove(f.Name()) // Clean up the temp file if it doesn't actually get moved into place
	if _, err := f.Write([]byte(hex.EncodeToString(v))); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write pre-image %s to disk: %w", k, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp pre-image %s file: %w", k, err)
	}

	targetFile := d.pathKey(k)
	if err := os.MkdirAll(path.Dir(targetFile), 0777); err != nil {
		return fmt.Errorf("failed to create parent directory for pre-image %s: %w", f.Name(), err)
	}
	if err := os.Rename(f.Name(), targetFile); err != nil {
		return fmt.Errorf("failed to move temp file %v to final destination %v: %w", f.Name(), targetFile, err)
	}
	return nil
}

func (d *directoryKV) Get(k common.Hash) ([]byte, error) {
	d.RLock()
	defer d.RUnlock()
	f, err := os.OpenFile(d.pathKey(k), os.O_RDONLY, filePermission)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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

func (d *directoryKV) Close() error {
	return nil
}

var _ KV = (*directoryKV)(nil)
