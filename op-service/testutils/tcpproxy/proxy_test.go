package tcpproxy

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestPauseAndResume(t *testing.T) {
	upstream, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, upstream.Close()) })

	go func() {
		for {
			conn, err := upstream.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()

	proxy := New(log.New())
	require.NoError(t, proxy.Start())
	t.Cleanup(func() { require.NoError(t, proxy.Close()) })
	proxy.SetUpstream(upstream.Addr().String())

	active := dialAndEcho(t, proxy.Addr(), "before pause")
	proxy.Pause()
	require.NoError(t, active.SetDeadline(time.Now().Add(time.Second)))
	_, err = active.Write([]byte("closed"))
	if err == nil {
		_, err = active.Read(make([]byte, len("closed")))
	}
	require.Error(t, err, "pausing must close active connections")
	require.NoError(t, active.Close())

	paused, err := net.Dial("tcp", proxy.Addr())
	require.NoError(t, err, "proxy listener must remain available while paused")
	require.NoError(t, paused.SetDeadline(time.Now().Add(time.Second)))
	_, err = paused.Write([]byte("rejected"))
	if err == nil {
		_, err = paused.Read(make([]byte, len("rejected")))
	}
	require.Error(t, err, "paused proxy must reject new upstream connections")
	require.NoError(t, paused.Close())

	proxy.Resume()
	resumed := dialAndEcho(t, proxy.Addr(), "after resume")
	require.NoError(t, resumed.Close())
}

func dialAndEcho(t *testing.T, addr, message string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second)))
	_, err = conn.Write([]byte(message))
	require.NoError(t, err)
	response := make([]byte, len(message))
	_, err = io.ReadFull(conn, response)
	require.NoError(t, err)
	require.Equal(t, message, string(response))
	return conn
}
