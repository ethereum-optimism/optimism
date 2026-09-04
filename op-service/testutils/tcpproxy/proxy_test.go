package tcpproxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
)

// startEchoServer returns a listener that echoes one byte back on each accepted
// connection, and a channel that receives a signal per accepted connection.
func startEchoServer(t *testing.T, addr string) (net.Listener, chan struct{}) {
	lis, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	accepted := make(chan struct{}, 16)
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			accepted <- struct{}{}
			go func() {
				defer conn.Close()
				buf := make([]byte, 1)
				if _, err := conn.Read(buf); err != nil {
					return
				}
				_, _ = conn.Write(buf)
			}()
		}
	}()
	return lis, accepted
}

func roundTrip(t *testing.T, addr string) error {
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
	if _, err := conn.Write([]byte{42}); err != nil {
		return err
	}
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	return err
}

// TestProxyClearUpstream checks that a cleared proxy refuses connections
// instead of forwarding to whatever rebinds the freed upstream port.
func TestProxyClearUpstream(t *testing.T) {
	lgr := testlog.Logger(t, log.LevelInfo)
	p := New(lgr)
	require.NoError(t, p.Start())
	defer p.Close()

	upstreamA, acceptedA := startEchoServer(t, "127.0.0.1:0")
	upstreamAddr := upstreamA.Addr().String()
	p.SetUpstream(upstreamAddr)

	require.NoError(t, roundTrip(t, p.Addr()), "proxy must pipe to the live upstream")
	<-acceptedA

	require.NoError(t, upstreamA.Close())
	p.ClearUpstream()

	// An unrelated server reuses the exact same port.
	upstreamB, acceptedB := startEchoServer(t, upstreamAddr)
	defer upstreamB.Close()

	require.Error(t, roundTrip(t, p.Addr()),
		"cleared proxy must refuse connections instead of piping to the port's new owner")
	select {
	case <-acceptedB:
		t.Fatal("stale proxy leaked a connection to the unrelated process that reused the port")
	case <-time.After(100 * time.Millisecond):
	}

	p.SetUpstream(upstreamB.Addr().String())
	require.NoError(t, roundTrip(t, p.Addr()), "proxy must pipe again after SetUpstream")
	<-acceptedB
}

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

func TestProxySwitchUpstreamClosesPersistentConnections(t *testing.T) {
	upstreamA := startEchoUpstream(t)
	upstreamB := startEchoUpstream(t)
	p := startProxy(t, upstreamA.Addr().String())

	oldConn, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer oldConn.Close()
	require.NoError(t, oldConn.SetDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, echoRoundTrip(oldConn, "before"))

	p.SwitchUpstream(upstreamB.Addr().String())
	_, err = oldConn.Read(make([]byte, 1))
	require.Error(t, err, "switch must tear down persistent connections to the old upstream")

	newConn, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer newConn.Close()
	require.NoError(t, newConn.SetDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, echoRoundTrip(newConn, "after"))
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

// heldDialer blocks every dial to the held address until release is closed or
// the dial context is done (matching net.Dialer.DialContext semantics); other
// addresses are dialed for real. Dial target addresses are recorded in order.
type heldDialer struct {
	heldAddr string
	started  chan string   // receives the target addr as each dial begins
	release  chan struct{} // close to unblock held dials
	real     net.Dialer
}

func newHeldDialer(heldAddr string) *heldDialer {
	return &heldDialer{
		heldAddr: heldAddr,
		started:  make(chan string, 16),
		release:  make(chan struct{}),
	}
}

func (d *heldDialer) dial(ctx context.Context, addr string) (net.Conn, error) {
	d.started <- addr
	if addr == d.heldAddr {
		select {
		case <-d.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return d.real.DialContext(ctx, "tcp", addr)
}

func waitDialStart(t *testing.T, d *heldDialer) string {
	t.Helper()
	select {
	case addr := <-d.started:
		return addr
	case <-time.After(10 * time.Second):
		t.Fatal("no upstream dial started")
		return ""
	}
}

// TestProxyStalledDialDoesNotBlockProxy holds one upstream dial in flight and
// asserts that SetUpstream, a second connection's data path, and Close all
// make progress before the held dial is released. On an implementation that
// holds the proxy mutex across the dial, each of these blocks behind it.
func TestProxyStalledDialDoesNotBlockProxy(t *testing.T) {
	stalled := startEchoUpstream(t)
	live := startEchoUpstream(t)

	d := newHeldDialer(stalled.Addr().String())
	defer close(d.release)

	p := New(log.NewLogger(log.DiscardHandler()))
	p.dial = d.dial
	require.NoError(t, p.Start())
	p.SetUpstream(stalled.Addr().String())

	// First connection: its upstream dial starts and is held in flight.
	conn1, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer conn1.Close()
	waitDialStart(t, d)

	// SetUpstream must not wait for the held dial.
	setDone := make(chan struct{})
	go func() {
		p.SetUpstream(live.Addr().String())
		close(setDone)
	}()
	select {
	case <-setDone:
	case <-time.After(2 * time.Second):
		t.Fatal("SetUpstream blocked behind an in-flight upstream dial")
	}

	// A second connection's data path must complete while the dial is held.
	conn2, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer conn2.Close()
	require.NoError(t, conn2.SetDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, echoRoundTrip(conn2, "not-stalled"))

	// Close must return while the dial is still held.
	closeDone := make(chan error, 1)
	go func() { closeDone <- p.Close() }()
	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Close blocked behind an in-flight upstream dial")
	}
}

// TestProxySetUpstreamAffectsOnlyFutureConns verifies the SetUpstream
// contract: a connection in flight keeps the upstream it captured; the update
// applies to connections accepted afterwards.
func TestProxySetUpstreamAffectsOnlyFutureConns(t *testing.T) {
	upstreamA := startEchoUpstream(t)
	upstreamB := startEchoUpstream(t)

	d := newHeldDialer(upstreamA.Addr().String())

	p := New(log.NewLogger(log.DiscardHandler()))
	p.dial = d.dial
	require.NoError(t, p.Start())
	t.Cleanup(func() { _ = p.Close() })
	p.SetUpstream(upstreamA.Addr().String())

	conn1, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer conn1.Close()
	addr1 := waitDialStart(t, d)

	// Update the upstream while conn1's dial is still in flight, with a bound
	// so a mutex held across the dial fails rather than hangs the test.
	setDone := make(chan struct{})
	go func() {
		p.SetUpstream(upstreamB.Addr().String())
		close(setDone)
	}()
	select {
	case <-setDone:
	case <-time.After(2 * time.Second):
		t.Fatal("SetUpstream blocked behind an in-flight upstream dial")
	}

	conn2, err := net.DialTimeout("tcp", p.Addr(), 5*time.Second)
	require.NoError(t, err)
	defer conn2.Close()
	addr2 := waitDialStart(t, d)

	close(d.release)

	require.Equal(t, upstreamA.Addr().String(), addr1, "in-flight connection must keep the upstream it captured")
	require.Equal(t, upstreamB.Addr().String(), addr2, "connections accepted after SetUpstream must use the new upstream")

	// Both connections proxy to their respective upstreams.
	for _, conn := range []net.Conn{conn1, conn2} {
		require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
		require.NoError(t, echoRoundTrip(conn, "captured-upstream"))
	}
}
