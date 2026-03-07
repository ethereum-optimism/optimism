package state

import (
	"os"
	"path/filepath"
)

// LocalStore implements Store using the local filesystem.
// Each key maps to a file in the configured directory.
type LocalStore struct {
	dir string
}

// NewLocalStore creates a filesystem-backed state store.
func NewLocalStore(dir string) *LocalStore {
	os.MkdirAll(dir, 0o755)
	return &LocalStore{dir: dir}
}

func (s *LocalStore) Load(key string) ([]byte, error) {
	data, err := os.ReadFile(s.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (s *LocalStore) Save(key string, data []byte) error {
	return os.WriteFile(s.path(key), data, 0o644)
}

func (s *LocalStore) path(key string) string {
	return filepath.Join(s.dir, key+".json")
}
