package main

import (
	"context"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(errCh)
		if err := run(ctx, []string{"op-up", "--dir", t.TempDir()}, io.Discard, io.Discard); err != nil {
			errCh <- err
		}
	}()

	client, err := ethclient.DialContext(ctx, "http://localhost:8545")
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

func TestRunWithIntent(t *testing.T) {
	// Load intent to get expected chain ID
	intent, err := jsonutil.LoadTOML[state.Intent]("example.intent.toml")
	require.NoError(t, err)
	require.NotEmpty(t, intent.Chains)
	expectedChainID := new(big.Int).SetBytes(intent.Chains[0].ID[:])

	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(errCh)
		if err := run(ctx, []string{"op-up", "--dir", t.TempDir(), "--intent", "example.intent.toml"}, io.Discard, io.Discard); err != nil {
			errCh <- err
		}
	}()

	client, err := ethclient.DialContext(ctx, "http://localhost:8545")
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
			require.Equal(t, expectedChainID, chainID)
			return
		}
	}
}
