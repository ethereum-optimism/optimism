package plasma

import (
	"context"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDAClient(t *testing.T) {
	store := memorydb.New()
	logger := testlog.Logger(t, log.LvlDebug)

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
		w.Write(input)
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
		store.Put(comm, input)

		w.Write(comm)
	})))

	tsrv := httptest.NewServer(mux)

	client := NewDAClient(tsrv.URL)
	client.VerifyOnRead(true)

	rng := rand.New(rand.NewSource(1234))

	input := testutils.RandomData(rng, 2000)

	comm, err := client.SetInput(ctx, input)
	require.NoError(t, err)

	require.Equal(t, comm, crypto.Keccak256(input))

	stored, err := client.GetInput(ctx, comm)
	require.NoError(t, err)

	require.Equal(t, input, stored)

	// set a bad commitment in the store
	store.Put(comm, []byte("bad data"))

	stored, err = client.GetInput(ctx, comm)
	require.ErrorIs(t, err, ErrCommitmentMismatch)

	// test not found error
	comm = crypto.Keccak256(testutils.RandomData(rng, 32))
	stored, err = client.GetInput(ctx, comm)
	require.ErrorIs(t, err, ErrNotFound)

	// test storing bad data
	comm, err = client.SetInput(ctx, []byte{})
	require.ErrorIs(t, err, ErrInvalidInput)

	// server not responsive
	tsrv.Close()
	comm, err = client.SetInput(ctx, input)
	require.Error(t, err)

	stored, err = client.GetInput(ctx, crypto.Keccak256(input))
	require.Error(t, err)
}
