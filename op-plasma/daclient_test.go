package plasma

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDAClient(t *testing.T) {
	store := memorydb.New()
	logger := testlog.Logger(t, log.LevelDebug)

	ctx := context.Background()

	mux := http.NewServeMux()
	mux.Handle("/get/", http.StripPrefix("/get/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("GET", "url", r.URL)

		comm, err := hexutil.Decode(r.URL.String())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		input, err := store.Get(comm)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := w.Write(input); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})))
	mux.Handle("/put/", http.StripPrefix("/put/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("PUT", "url", r.URL)

		input, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		comm, err := hexutil.Decode(r.URL.String())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := store.Put(comm, input); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(comm); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})))

	tsrv := httptest.NewServer(mux)

	cfg := CLIConfig{
		Enabled:      true,
		DAServerURL:  tsrv.URL,
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
	tsrv.Close()
	_, err = client.SetInput(ctx, input)
	require.Error(t, err)

	_, err = client.GetInput(ctx, Keccak256(input))
	require.Error(t, err)
}
