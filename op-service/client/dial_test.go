package client

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsURLAvailable(t *testing.T) {
	listener, err := net.Listen("tcp4", ":0")
	require.NoError(t, err)
	defer listener.Close()

	a := listener.Addr().String()
	parts := strings.Split(a, ":")
	addr := fmt.Sprintf("http://localhost:%s", parts[1])

	require.True(t, IsURLAvailable(addr))
	require.False(t, IsURLAvailable("http://localhost:0"))
}
