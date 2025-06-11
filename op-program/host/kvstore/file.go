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

// read/write mode for user/group/other, not executable.
const filePermission = 0666

// fileKV is a disk-backed key-value store, every key-value pair is a hex-encoded .txt file, with the value as content.
// fileKV is safe for concurrent use with a single fileKV instance.
// fileKV is safe for concurrent use between different fileKV instances of the same disk directory as long as the
// file system supports atomic renames.
type fileKV struct {
	sync.RWMutex
	path string
}

// newFileKV creates a fileKV that puts/gets pre-images as files in the given directory path.
// The path must exist, or subsequent Put/Get calls will error when it does not.
func newFileKV(path string) *fileKV {
	return &fileKV{path: path}
}

func (d *fileKV) pathKey(k common.Hash) string {
	return path.Join(d.path, k.String()+".txt")
}

func (d *fileKV) Put(k common.Hash, v []byte) error {
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
	if err := os.Rename(f.Name(), targetFile); err != nil {
		return fmt.Errorf("failed to move temp dir %v to final destination %v: %w", f.Name(), targetFile, err)
	}
	return nil
}

func openTempFile(dir string, nameTemplate string) (*os.File, error) {
	f, err := os.CreateTemp(dir, nameTemplate)
	// Directory has been deleted out from underneath us. Recreate it.
	if errors.Is(err, os.ErrNotExist) {
		if mkdirErr := os.MkdirAll(dir, 0777); mkdirErr != nil {
			return nil, errors.Join(fmt.Errorf("failed to create directory %v: %w", dir, mkdirErr), err)
		}
		f, err = os.CreateTemp(dir, nameTemplate)
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (d *fileKV) Get(k common.Hash) ([]byte, error) {
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

func (d *fileKV) Close() error {
	return nil
}

var _ KV = (*fileKV)(nil)
