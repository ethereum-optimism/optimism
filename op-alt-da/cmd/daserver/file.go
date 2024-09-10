package main

import (
	"context"
	"encoding/hex"
	"os"
	"path"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
)

type FileStore struct {
	directory string
}

func NewFileStore(directory string) *FileStore {
	return &FileStore{
		directory: directory,
	}
}

func (s *FileStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	data, err := os.ReadFile(s.fileName(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, altda.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (s *FileStore) Put(ctx context.Context, key []byte, value []byte) error {
	return os.WriteFile(s.fileName(key), value, 0600)
}

func (s *FileStore) fileName(key []byte) string {
	return path.Join(s.directory, hex.EncodeToString(key))
}
