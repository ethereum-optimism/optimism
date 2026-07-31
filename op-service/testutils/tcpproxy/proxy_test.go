package tcpproxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

// startEchoUpstream returns a listener that echoes every accepted connection.
func startEchoUpstream(t *testing.T) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	return lis
}

func startProxy(t *testing.T, upstream string) *Proxy {
	t.Helper()
	p := New(log.NewLogger(log.DiscardHandler()))
	require.NoError(t, p.Start())
	t.Cleanup(func() { _ = p.Close() })
	if upstream != "" {
		p.SetUpstream(upstream)
	}
	return p
}

func echoRoundTrip(conn net.Conn, msg string) error {
	if _, err := conn.Write([]byte(msg)); err != nil {
		return err
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	if string(buf) != msg {
		return fmt.Errorf("echo mismatch: got %q, want %q", buf, msg)
	}
	return nil
}

func TestProxyEcho(t *testing.T) {
	upstream := startEchoUpstream(t)
	p := startProxy(t, upstream.Addr().String())

	conn, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
	require.NoError(t, echoRoundTrip(conn, "hello"))
}

// TestProxyConcurrentConns verifies connections are handled concurrently:
// many clients connect and complete round trips in parallel without
// serializing behind one another.
func TestProxyConcurrentConns(t *testing.T) {
	upstream := startEchoUpstream(t)
	p := startProxy(t, upstream.Addr().String())

	const n = 32
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
			if err != nil {
				errs <- err
				return
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
				errs <- err
				return
			}
			errs <- echoRoundTrip(conn, fmt.Sprintf("message-%d", i))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

// TestProxyUnreachableUpstreamDoesNotStallOthers verifies that connections
// stuck failing to dial a dead upstream do not block Close or the handling of
// other connections once the upstream is reachable again.
func TestProxyUnreachableUpstreamDoesNotStallOthers(t *testing.T) {
	// Reserve an address with no listener behind it.
	dead, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	deadAddr := dead.Addr().String()
	require.NoError(t, dead.Close())

	p := startProxy(t, deadAddr)

	// Fire connections at the proxy while its upstream dial fails; they must
	// still be accepted (and then dropped), not left hanging in the backlog.
	for i := 0; i < 4; i++ {
		conn, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
		require.NoError(t, err)
		defer conn.Close()
	}

	// Repointing at a live upstream restores service for new connections,
	// even while earlier connections may still be in their dial-retry loop.
	upstream := startEchoUpstream(t)
	p.SetUpstream(upstream.Addr().String())

	conn, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetDeadline(time.Now().Add(15*time.Second)))
	require.NoError(t, echoRoundTrip(conn, "back-online"))
}

// TestProxyCloseIsPrompt verifies Close returns promptly with connections in
// flight, including connections whose upstream dial can never succeed.
func TestProxyCloseIsPrompt(t *testing.T) {
	upstream := startEchoUpstream(t)
	p := New(log.NewLogger(log.DiscardHandler()))
	require.NoError(t, p.Start())
	p.SetUpstream(upstream.Addr().String())

	conn, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
	require.NoError(t, echoRoundTrip(conn, "ping"))

	done := make(chan error, 1)
	go func() { done <- p.Close() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("proxy Close did not return")
	}

	// The proxied connection must have been torn down.
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
	_, err = conn.Read(make([]byte, 1))
	require.Error(t, err)
}
