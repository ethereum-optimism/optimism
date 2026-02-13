package prestates

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type SinglePrestateSource struct {
	path string
}

func NewSinglePrestateSource(path string) *SinglePrestateSource {
	return &SinglePrestateSource{path: path}
}

func (s *SinglePrestateSource) PrestatePath(_ context.Context, _ common.Hash) (string, error) {
	return s.path, nil
}
