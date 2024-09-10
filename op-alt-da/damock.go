package altda

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
	CommitmentType CommitmentType
	store          ethdb.KeyValueStore
	log            log.Logger
}

func NewMockDAClient(log log.Logger) *MockDAClient {
	return &MockDAClient{
		CommitmentType: Keccak256CommitmentType,
		store:          memorydb.New(),
		log:            log,
	}
}

func (c *MockDAClient) GetInput(ctx context.Context, key CommitmentData) ([]byte, error) {
	bytes, err := c.store.Get(key.Encode())
	if err != nil {
		return nil, ErrNotFound
	}
	return bytes, nil
}

func (c *MockDAClient) SetInput(ctx context.Context, data []byte) (CommitmentData, error) {
	key := NewCommitmentData(c.CommitmentType, data)
	return key, c.store.Put(key.Encode(), data)
}

func (c *MockDAClient) DeleteData(key []byte) error {
	return c.store.Delete(key)
}

type DAErrFaker struct {
	Client *MockDAClient

	getInputErr error
	setInputErr error
}

func (f *DAErrFaker) GetInput(ctx context.Context, key CommitmentData) ([]byte, error) {
	if err := f.getInputErr; err != nil {
		f.getInputErr = nil
		return nil, err
	}
	return f.Client.GetInput(ctx, key)
}

func (f *DAErrFaker) SetInput(ctx context.Context, data []byte) (CommitmentData, error) {
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

var Disabled = &AltDADisabled{}

var ErrNotEnabled = errors.New("altDA not enabled")

// AltDADisabled is a noop AltDA implementation for stubbing.
type AltDADisabled struct{}

func (d *AltDADisabled) GetInput(ctx context.Context, l1 L1Fetcher, commitment CommitmentData, blockId eth.L1BlockRef) (eth.Data, error) {
	return nil, ErrNotEnabled
}

func (d *AltDADisabled) Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error {
	return io.EOF
}

func (d *AltDADisabled) Finalize(ref eth.L1BlockRef) {
}

func (d *AltDADisabled) OnFinalizedHeadSignal(f HeadSignalFn) {
}

func (d *AltDADisabled) AdvanceL1Origin(ctx context.Context, l1 L1Fetcher, blockId eth.BlockID) error {
	return ErrNotEnabled
}
