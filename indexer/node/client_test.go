package node

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDialEthClientUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp4", ":0")
	require.NoError(t, err)
	defer listener.Close()

	a := listener.Addr().String()
	parts := strings.Split(a, ":")
	addr := fmt.Sprintf("http://localhost:%s", parts[1])

	metrics := &clientMetrics{}

	// available
	_, err = DialEthClient(context.Background(), addr, metrics)
	require.NoError(t, err)

	// :0 requests a new unbound port
	_, err = DialEthClient(context.Background(), "http://localhost:0", metrics)
	require.Error(t, err)

	// Fail open if we don't recognize the scheme
	_, err = DialEthClient(context.Background(), "mailto://example.com", metrics)
	require.Error(t, err)
}
