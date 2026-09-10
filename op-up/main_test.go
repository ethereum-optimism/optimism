package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(errCh)
		if err := run(ctx, []string{
			"op-up", "--dir", t.TempDir(), "--l2a-rpc-port", strconv.Itoa(port),
		}, io.Discard, io.Discard); err != nil {
			errCh <- err
		}
	}()

	client, err := ethclient.DialContext(ctx, fmt.Sprintf("http://127.0.0.1:%d", port))
	require.NoError(t, err)
	ticker := time.NewTicker(time.Millisecond * 250)
	for {
		select {
		case e := <-errCh:
			require.NoError(t, e)
		case <-ticker.C:
			chainID, err := client.ChainID(ctx)
			if err != nil {
				t.Logf("error while querying chain ID, will retry: %s", err)
				continue
			}
			require.Equal(t, sysgo.DefaultL2AID.ToBig(), chainID)
			return
		}
	}
}
