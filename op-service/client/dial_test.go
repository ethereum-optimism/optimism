package client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsURLAvailableLocal(t *testing.T) {
	listener, err := net.Listen("tcp4", ":0")
	require.NoError(t, err)
	defer listener.Close()

	a := listener.Addr().String()
	parts := strings.Split(a, ":")
	addr := fmt.Sprintf("http://localhost:%s", parts[1])

	// True & False with ports
	require.True(t, IsURLAvailable(context.Background(), addr))
	require.False(t, IsURLAvailable(context.Background(), "http://localhost:0"))

	// Fail open if we don't recognize the scheme
	require.True(t, IsURLAvailable(context.Background(), "mailto://example.com"))

}

func TestIsURLAvailableNonLocal(t *testing.T) {
	if !IsURLAvailable(context.Background(), "http://example.com") {
		t.Skip("No internet connection found, skipping this test")
	}

	// True without ports. http & https
	require.True(t, IsURLAvailable(context.Background(), "http://example.com"))
	require.True(t, IsURLAvailable(context.Background(), "http://example.com/hello"))
	require.True(t, IsURLAvailable(context.Background(), "https://example.com"))
	require.True(t, IsURLAvailable(context.Background(), "https://example.com/hello"))

	// True without ports. ws & wss
	require.True(t, IsURLAvailable(context.Background(), "ws://example.com"))
	require.True(t, IsURLAvailable(context.Background(), "ws://example.com/hello"))
	require.True(t, IsURLAvailable(context.Background(), "wss://example.com"))
	require.True(t, IsURLAvailable(context.Background(), "wss://example.com/hello"))

	// False without ports
	require.False(t, IsURLAvailable(context.Background(), "http://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable(context.Background(), "http://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
	require.False(t, IsURLAvailable(context.Background(), "https://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable(context.Background(), "https://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
	require.False(t, IsURLAvailable(context.Background(), "ws://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable(context.Background(), "ws://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
	require.False(t, IsURLAvailable(context.Background(), "wss://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable(context.Background(), "wss://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
}
