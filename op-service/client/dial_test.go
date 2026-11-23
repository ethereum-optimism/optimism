package client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const defaultConnectTimeout = 5 * time.Second

func TestIsURLAvailableLocal(t *testing.T) {
	listener, err := net.Listen("tcp4", ":0")
	require.NoError(t, err)
	defer listener.Close()

	a := listener.Addr().String()
	parts := strings.Split(a, ":")
	addr := fmt.Sprintf("http://localhost:%s", parts[1])

	// True & False with ports
	require.True(t, IsURLAvailable(context.Background(), addr, defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "http://localhost:0", defaultConnectTimeout))

	// Fail open if we don't recognize the scheme
	require.True(t, IsURLAvailable(context.Background(), "mailto://example.com", defaultConnectTimeout))

}

func TestIsURLAvailableNonLocal(t *testing.T) {
	if !IsURLAvailable(context.Background(), "http://example.com", defaultConnectTimeout) {
		t.Skip("No internet connection found, skipping this test")
	}

	// True without ports. http & https
	require.True(t, IsURLAvailable(context.Background(), "http://example.com", defaultConnectTimeout))
	require.True(t, IsURLAvailable(context.Background(), "http://example.com/hello", defaultConnectTimeout))
	require.True(t, IsURLAvailable(context.Background(), "https://example.com", defaultConnectTimeout))
	require.True(t, IsURLAvailable(context.Background(), "https://example.com/hello", defaultConnectTimeout))

	// True without ports. ws & wss
	require.True(t, IsURLAvailable(context.Background(), "ws://example.com", defaultConnectTimeout))
	require.True(t, IsURLAvailable(context.Background(), "ws://example.com/hello", defaultConnectTimeout))
	require.True(t, IsURLAvailable(context.Background(), "wss://example.com", defaultConnectTimeout))
	require.True(t, IsURLAvailable(context.Background(), "wss://example.com/hello", defaultConnectTimeout))

	// False without ports
	require.False(t, IsURLAvailable(context.Background(), "http://fakedomainnamethatdoesnotexistandshouldneverexist.com", defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "http://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello", defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "https://fakedomainnamethatdoesnotexistandshouldneverexist.com", defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "https://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello", defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "ws://fakedomainnamethatdoesnotexistandshouldneverexist.com", defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "ws://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello", defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "wss://fakedomainnamethatdoesnotexistandshouldneverexist.com", defaultConnectTimeout))
	require.False(t, IsURLAvailable(context.Background(), "wss://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello", defaultConnectTimeout))
}
