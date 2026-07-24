package tcpproxy

import (
	"net"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
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

// TestProxyClearUpstream covers the port-reuse cross-wiring hazard: a proxy
// whose upstream process has stopped must not keep piping connections to the
// stale address, because the OS may reassign that port to an unrelated
// process. Clearing the upstream on stop makes the proxy refuse connections
// until the owning process restarts and re-points it.
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

	// Upstream stops; its port is freed and may be rebound by anyone.
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

	// Re-pointing the proxy restores service.
	p.SetUpstream(upstreamB.Addr().String())
	require.NoError(t, roundTrip(t, p.Addr()), "proxy must pipe again after SetUpstream")
	<-acceptedB
}
