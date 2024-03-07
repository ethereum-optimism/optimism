package plasma

import (
	"context"
	"errors"
	"io"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	// Validate the commitment to make sure we only pass encoded types.
	_, err := DecodeKeccak256(key)
	if err != nil {
		return nil, err
	}
	bytes, err := c.store.Get(key)
	if err != nil {
		return nil, ErrNotFound
	}
	return bytes, nil
}

func (c *MockDAClient) SetInput(ctx context.Context, data []byte) ([]byte, error) {
	key := Keccak256(data).Encode()
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

func (f *DAErrFaker) SetInput(ctx context.Context, data []byte) ([]byte, error) {
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

var Disabled = &PlasmaDisabled{}

var ErrNotEnabled = errors.New("plasma not enabled")

// PlasmaDisabled is a noop plasma DA implementation for stubbing.
type PlasmaDisabled struct{}

func (d *PlasmaDisabled) GetInput(ctx context.Context, commitment []byte, blockId eth.BlockID) (eth.Data, error) {
	return nil, ErrNotEnabled
}

func (d *PlasmaDisabled) Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error {
	return io.EOF
}

func (d *PlasmaDisabled) FinalizeL1(ref eth.L1BlockRef) {
}

func (d *PlasmaDisabled) OnFinalizedHeadSignal(f eth.HeadSignalFn) {
}

func (d *PlasmaDisabled) AdvanceL1Origin(ctx context.Context, blockId eth.BlockID) error {
	return ErrNotEnabled
}
