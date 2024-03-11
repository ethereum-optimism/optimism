package plasma

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDAClient(t *testing.T) {
	store := memorydb.New()
	logger := testlog.Logger(t, log.LevelDebug)

	ctx := context.Background()

	server := NewDAServer("127.0.0.1", 0, store, WithLogger(logger))

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

	require.Equal(t, comm, Keccak256(input))

	stored, err := client.GetInput(ctx, comm)
	require.NoError(t, err)

	require.Equal(t, input, stored)

	// set a bad commitment in the store
	require.NoError(t, store.Put(comm.Encode(), []byte("bad data")))

	_, err = client.GetInput(ctx, comm)
	require.ErrorIs(t, err, ErrCommitmentMismatch)

	// test not found error
	comm = Keccak256(RandomData(rng, 32))
	_, err = client.GetInput(ctx, comm)
	require.ErrorIs(t, err, ErrNotFound)

	// test storing bad data
	_, err = client.SetInput(ctx, []byte{})
	require.ErrorIs(t, err, ErrInvalidInput)

	// server not responsive
	require.NoError(t, server.Stop())
	_, err = client.SetInput(ctx, input)
	require.Error(t, err)

	_, err = client.GetInput(ctx, Keccak256(input))
	require.Error(t, err)
}
