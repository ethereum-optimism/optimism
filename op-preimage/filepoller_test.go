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
