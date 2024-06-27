package mtcannon

import (
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

// TODO(client-pod#942): replace with actual mipsevm MT state implementation
type MTState interface {
	EncodeWitness() ([]byte, common.Hash)
}

func parseState(string) (MTState, error) {
	return nil, errors.New("unimplemented")
}

func parseStateFromReader(in io.ReadCloser) (MTState, error) {
	return nil, errors.New("unimplemented")
}
