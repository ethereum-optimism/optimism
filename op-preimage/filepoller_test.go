package preimage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFilePoller_Read(t *testing.T) {
	chanA, chanB, err := CreateBidirectionalChannel()
	require.NoError(t, err)
	ctx := context.Background()
	chanAPoller := NewFilePoller(ctx, chanA, time.Millisecond*100)

	go func() {
		_, _ = chanB.Write([]byte("hello"))
		time.Sleep(time.Second * 1)
		_, _ = chanB.Write([]byte("world"))
	}()
	var buf [10]byte
	n, err := chanAPoller.Read(buf[:])
	require.Equal(t, 10, n)
	require.NoError(t, err)
}

func TestFilePoller_Write(t *testing.T) {
	chanA, chanB, err := CreateBidirectionalChannel()
	require.NoError(t, err)
	ctx := context.Background()
	chanAPoller := NewFilePoller(ctx, chanA, time.Millisecond*100)

	bufch := make(chan []byte, 1)
	go func() {
		var buf [10]byte
		_, _ = chanB.Read(buf[:5])
		time.Sleep(time.Second * 1)
		_, _ = chanB.Read(buf[5:])
		bufch <- buf[:]
		close(bufch)
	}()
	buf := []byte("helloworld")
	n, err := chanAPoller.Write(buf)
	require.Equal(t, 10, n)
	require.NoError(t, err)
	select {
	case <-time.After(time.Second * 60):
		t.Fatal("timed out waiting for read")
	case readbuf := <-bufch:
		require.Equal(t, buf, readbuf)
	}
}

func TestFilePoller_ReadCancel(t *testing.T) {
	chanA, chanB, err := CreateBidirectionalChannel()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	chanAPoller := NewFilePoller(ctx, chanA, time.Millisecond*100)

	go func() {
		_, _ = chanB.Write([]byte("hello"))
		cancel()
	}()
	var buf [10]byte
	n, err := chanAPoller.Read(buf[:])
	require.Equal(t, 5, n)
	require.ErrorIs(t, err, context.Canceled)
}

func TestFilePoller_WriteCancel(t *testing.T) {
	chanA, chanB, err := CreateBidirectionalChannel()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	chanAPoller := NewFilePoller(ctx, chanA, time.Millisecond*100)

	go func() {
		var buf [5]byte
		_, _ = chanB.Read(buf[:])
		cancel()
	}()
	// use a large buffer to overflow the kernel buffer provided to pipe(2) so the write actually blocks
	buf := make([]byte, 1024*1024)
	_, err = chanAPoller.Write(buf)
	require.ErrorIs(t, err, context.Canceled)
}

func TestFilePoller_ContextDeadline(t *testing.T) {
	for _, expired := range []bool{false, true} {
		for _, write := range []bool{false, true} {
			name := "read"
			if write {
				name = "write"
			}
			if expired {
				name += " expired"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				client, host, err := CreateBidirectionalChannel()
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = client.Close()
					_ = host.Close()
				})
				timeout := 100 * time.Millisecond
				if expired {
					timeout = -time.Second
				}
				ctx, cancel := context.WithTimeout(t.Context(), timeout)
				defer cancel()
				poller := NewFilePoller(ctx, client, time.Hour)
				type result struct {
					n   int
					err error
				}
				done := make(chan result, 1)
				go func() {
					if write {
						n, err := poller.Write(make([]byte, 1024*1024))
						done <- result{n, err}
					} else {
						n, err := poller.Read(make([]byte, 1))
						done <- result{n, err}
					}
				}()
				select {
				case got := <-done:
					require.ErrorIs(t, got.err, context.DeadlineExceeded)
					if expired || !write {
						require.Zero(t, got.n)
					} else {
						require.GreaterOrEqual(t, got.n, 0)
						require.Less(t, got.n, 1024*1024)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("context deadline did not interrupt file IO")
				}
			})
		}
	}
}
