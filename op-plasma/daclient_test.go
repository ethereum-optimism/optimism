package plasma

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type MemStore struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemStore() *MemStore {
	return &MemStore{
		db: make(map[string][]byte),
	}
}

// Get retrieves the given key if it's present in the key-value store.
func (s *MemStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if entry, ok := s.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, ErrNotFound
}

// Put inserts the given value into the key-value store.
func (s *MemStore) Put(ctx context.Context, key []byte, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.db[string(key)] = common.CopyBytes(value)
	return nil
}

func TestDAClientPrecomputed(t *testing.T) {
	store := NewMemStore()
	logger := testlog.Logger(t, log.LevelDebug)

	ctx := context.Background()

	server := NewDAServer("127.0.0.1", 0, store, logger, false)

	require.NoError(t, server.Start())

	cfg := CLIConfig{
		Enabled:      true,
		DAServerURL:  fmt.Sprintf("http://%s", server.Endpoint()),
		VerifyOnRead: true,
	}
	require.NoError(t, cfg.Check())

	client := cfg.NewDAClient()

	rng := rand.New(rand.NewSource(1234))

	input := RandomData(rng, 2000)

	comm, err := client.SetInput(ctx, input)
	require.NoError(t, err)

	require.Equal(t, comm, NewKeccak256Commitment(input))

	stored, err := client.GetInput(ctx, comm)
	require.NoError(t, err)

	require.Equal(t, input, stored)

	// set a bad commitment in the store
	require.NoError(t, store.Put(ctx, comm.Encode(), []byte("bad data")))

	_, err = client.GetInput(ctx, comm)
	require.ErrorIs(t, err, ErrCommitmentMismatch)

	// test not found error
	comm = NewKeccak256Commitment(RandomData(rng, 32))
	_, err = client.GetInput(ctx, comm)
	require.ErrorIs(t, err, ErrNotFound)

	// test storing bad data
	_, err = client.SetInput(ctx, []byte{})
	require.ErrorIs(t, err, ErrInvalidInput)

	// server not responsive
	require.NoError(t, server.Stop())
	_, err = client.SetInput(ctx, input)
	require.Error(t, err)

	_, err = client.GetInput(ctx, NewKeccak256Commitment(input))
	require.Error(t, err)
}

func TestDAClientService(t *testing.T) {
	store := NewMemStore()
	logger := testlog.Logger(t, log.LevelDebug)

	ctx := context.Background()

	server := NewDAServer("127.0.0.1", 0, store, logger, false)

	require.NoError(t, server.Start())

	cfg := CLIConfig{
		Enabled:      true,
		DAServerURL:  fmt.Sprintf("http://%s", server.Endpoint()),
		VerifyOnRead: false,
		GenericDA:    true,
	}
	require.NoError(t, cfg.Check())

	client := cfg.NewDAClient()

	rng := rand.New(rand.NewSource(1234))

	input := RandomData(rng, 2000)

	comm, err := client.SetInput(ctx, input)
	require.NoError(t, err)

	require.Equal(t, comm, NewGenericCommitment(crypto.Keccak256(input)))

	stored, err := client.GetInput(ctx, comm)
	require.NoError(t, err)

	require.Equal(t, input, stored)

	// set a bad commitment in the store
	require.NoError(t, store.Put(ctx, comm.Encode(), []byte("bad data")))

	// assert no error as generic commitments cannot be verified client side
	_, err = client.GetInput(ctx, comm)
	require.NoError(t, err)

	// test not found error
	comm = NewGenericCommitment(RandomData(rng, 32))
	_, err = client.GetInput(ctx, comm)
	require.ErrorIs(t, err, ErrNotFound)

	// test storing bad data
	_, err = client.SetInput(ctx, []byte{})
	require.ErrorIs(t, err, ErrInvalidInput)

	// server not responsive
	require.NoError(t, server.Stop())
	_, err = client.SetInput(ctx, input)
	require.Error(t, err)

	_, err = client.GetInput(ctx, NewGenericCommitment(input))
	require.Error(t, err)
}
