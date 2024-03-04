package plasma

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
)

// MockDAClient mocks a DA storage provider to avoid running an HTTP DA server
// in unit tests.
type MockDAClient struct {
	store ethdb.KeyValueStore
	log   log.Logger
}

func NewMockDAClient(log log.Logger) *MockDAClient {
	return &MockDAClient{
		store: memorydb.New(),
		log:   log,
	}
}

func (c *MockDAClient) GetInput(ctx context.Context, key []byte) ([]byte, error) {
	bytes, err := c.store.Get(key)
	if err != nil {
		return nil, ErrNotFound
	}
	return bytes, nil
}

func (c *MockDAClient) SetInput(ctx context.Context, data []byte) ([]byte, error) {
	key := crypto.Keccak256(data)
	return key, c.store.Put(key, data)
}

func (c *MockDAClient) DeleteData(key []byte) error {
	return c.store.Delete(key)
}

type DAErrFaker struct {
	Client *MockDAClient

	getInputErr error
	setInputErr error
}

func (f *DAErrFaker) GetInput(ctx context.Context, key []byte) ([]byte, error) {
	if err := f.getInputErr; err != nil {
		f.getInputErr = nil
		return nil, err
	}
	return f.Client.GetInput(ctx, key)
}

func (f *DAErrFaker) SetPreImage(ctx context.Context, data []byte) ([]byte, error) {
	if err := f.setInputErr; err != nil {
		f.setInputErr = nil
		return nil, err
	}
	return f.Client.SetInput(ctx, data)
}

func (f *DAErrFaker) ActGetPreImageFail() {
	f.getInputErr = errors.New("get input failed")
}

func (f *DAErrFaker) ActSetPreImageFail() {
	f.setInputErr = errors.New("set input failed")
}
