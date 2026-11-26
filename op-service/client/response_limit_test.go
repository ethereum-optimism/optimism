package client

import (
	"bytes"
	"io"
	"math"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitResponseBody(t *testing.T) {
	const limit = 100
	limitedBody := func(status int, size int) io.Reader {
		resp := &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(make([]byte, size)))}
		limitResponseBody(resp, limit)
		return resp.Body
	}

	t.Run("at limit", func(t *testing.T) {
		body, err := io.ReadAll(limitedBody(http.StatusOK, limit))
		require.NoError(t, err)
		require.Equal(t, limit, len(body))
	})
	t.Run("above limit", func(t *testing.T) {
		r := limitedBody(http.StatusOK, limit+1)
		body, err := io.ReadAll(r)
		require.ErrorIs(t, err, ErrResponseTooLarge)
		require.ErrorContains(t, err, "exceeds 100 bytes")
		require.Equal(t, limit, len(body))
		_, err = r.Read(make([]byte, 1))
		require.ErrorIs(t, err, ErrResponseTooLarge, "reads after the limit keep failing")
	})
	t.Run("max limit", func(t *testing.T) {
		resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(make([]byte, limit)))}
		limitResponseBody(resp, math.MaxInt64)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, limit, len(body))
	})
	t.Run("error response truncated", func(t *testing.T) {
		body, err := io.ReadAll(limitedBody(http.StatusInternalServerError, 4*maxErrorBodySize))
		require.NoError(t, err)
		require.Equal(t, maxErrorBodySize, len(body))
	})
}
