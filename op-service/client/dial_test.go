package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsURLAvailable(t *testing.T) {
	go func() {
		_ = http.ListenAndServe(":8989", nil)
	}()

	require.True(t, IsURLAvailable("http://localhost:8989"))
	require.False(t, IsURLAvailable("http://localhost:9898"))
}
