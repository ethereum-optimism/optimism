package client

import (
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
	require.True(t, IsURLAvailable(addr))
	require.False(t, IsURLAvailable("http://localhost:0"))

	// Fail open if we don't recognize the scheme
	require.True(t, IsURLAvailable("mailto://example.com"))

}

func TestIsURLAvailableNonLocal(t *testing.T) {
	if !IsURLAvailable("http://example.com") {
		t.Skip("No internet connection found, skipping this test")
	}

	// True without ports. http & https
	require.True(t, IsURLAvailable("http://example.com"))
	require.True(t, IsURLAvailable("http://example.com/hello"))
	require.True(t, IsURLAvailable("https://example.com"))
	require.True(t, IsURLAvailable("https://example.com/hello"))

	// True without ports. ws & wss
	require.True(t, IsURLAvailable("ws://example.com"))
	require.True(t, IsURLAvailable("ws://example.com/hello"))
	require.True(t, IsURLAvailable("wss://example.com"))
	require.True(t, IsURLAvailable("wss://example.com/hello"))

	// False without ports
	require.False(t, IsURLAvailable("http://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable("http://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
	require.False(t, IsURLAvailable("https://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable("https://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
	require.False(t, IsURLAvailable("ws://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable("ws://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
	require.False(t, IsURLAvailable("wss://fakedomainnamethatdoesnotexistandshouldneverexist.com"))
	require.False(t, IsURLAvailable("wss://fakedomainnamethatdoesnotexistandshouldneverexist.com/hello"))
}
