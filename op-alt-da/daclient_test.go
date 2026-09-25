package altda

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDAClientPrecomputed(t *testing.T) {
	store := NewMemStore()
	logger := testlog.Logger(t, log.LevelDebug)

	ctx := context.Background()

	server := NewDAServer("127.0.0.1", 0, store, logger, false)

	require.NoError(t, server.Start())

	cfg := CLIConfig{
		Enabled:      true,
		DAServerURL:  server.HttpEndpoint(),
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
		DAServerURL:  server.HttpEndpoint(),
		VerifyOnRead: false,
		GenericDA:    false,
	}
	require.NoError(t, cfg.Check())

	client := cfg.NewDAClient()

	rng := rand.New(rand.NewSource(1234))

	input := RandomData(rng, 2000)

	comm, err := client.SetInput(ctx, input)
	require.NoError(t, err)

	require.Equal(t, comm.String(), NewKeccak256Commitment(input).String())

	stored, err := client.GetInput(ctx, comm)
	require.NoError(t, err)

	require.Equal(t, input, stored)

	// set a bad commitment in the store
	require.NoError(t, store.Put(ctx, comm.Encode(), []byte("bad data")))

	// assert no error as generic commitments cannot be verified client side
	_, err = client.GetInput(ctx, comm)
	require.NoError(t, err)

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

func TestDAClientGetInputClosesBodyOnError(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		var open atomic.Int64
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(status), status)
		}))
		srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
			switch s {
			case http.StateNew:
				open.Add(1)
			case http.StateClosed, http.StateHijacked:
				open.Add(-1)
			}
		}
		srv.Start()
		t.Cleanup(srv.Close)

		client := NewDAClient(srv.URL, false, true)
		comm := NewKeccak256Commitment([]byte("data"))
		for i := 0; i < 10; i++ {
			_, err := client.GetInput(context.Background(), comm)
			require.Error(t, err)
		}
		require.Eventually(t, func() bool { return open.Load() <= 1 }, 5*time.Second, 10*time.Millisecond,
			"status %d: %d connections still open", status, open.Load())
	}
}
