package devnet

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestRetryProxy(t *testing.T) {
	// proxy returns the status and body that the proxy answers to one request, and the number of
	// upstream calls. The upstream answers with the given statuses, one per call.
	proxy := func(t *testing.T, statuses ...int) (int, string, int32) {
		var calls atomic.Int32
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			status := statuses[min(int(calls.Add(1)), len(statuses))-1]
			w.WriteHeader(status)
			_, _ = io.WriteString(w, http.StatusText(status))
		}))
		t.Cleanup(upstream.Close)
		prox := NewRetryProxy(testlog.Logger(t, slog.LevelInfo), upstream.URL)
		require.NoError(t, prox.Start())
		t.Cleanup(func() { require.NoError(t, prox.Stop()) })

		res, err := http.Post(prox.Endpoint(), "application/json", strings.NewReader(`{"method":"eth_chainId"}`))
		require.NoError(t, err)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		return res.StatusCode, string(body), calls.Load()
	}

	t.Run("retries a rate limit", func(t *testing.T) {
		status, body, calls := proxy(t, http.StatusTooManyRequests, http.StatusOK)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, http.StatusText(http.StatusOK), body)
		require.EqualValues(t, 2, calls)
	})

	t.Run("retries a server error", func(t *testing.T) {
		status, body, calls := proxy(t, http.StatusBadGateway, http.StatusOK)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, http.StatusText(http.StatusOK), body)
		require.EqualValues(t, 2, calls)
	})

	t.Run("forwards a client error without retries", func(t *testing.T) {
		// A node that does not serve a method, such as proxyd, answers 403. The client must get it at
		// once, as it does from the node itself.
		status, body, calls := proxy(t, http.StatusForbidden, http.StatusOK)
		require.Equal(t, http.StatusForbidden, status)
		require.Equal(t, http.StatusText(http.StatusForbidden), body)
		require.EqualValues(t, 1, calls)
	})
}
