package host

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-program/client"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/io"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestServerMode(t *testing.T) {
	dir := t.TempDir()

	l1Head := common.Hash{0x11}
	cfg := config.NewConfig(&chaincfg.Goerli, config.OPGoerliChainConfig, l1Head, common.Hash{0x22}, common.Hash{0x33}, 1000)
	cfg.DataDir = dir
	cfg.ServerMode = true

	preimageServer, preimageClient, err := io.CreateBidirectionalChannel()
	require.NoError(t, err)
	defer preimageClient.Close()
	hintServer, hintClient, err := io.CreateBidirectionalChannel()
	require.NoError(t, err)
	defer hintClient.Close()
	logger := testlog.Logger(t, log.LvlTrace)
	result := make(chan error)
	go func() {
		result <- PreimageServer(context.Background(), logger, cfg, preimageServer, hintServer)
	}()

	pClient := preimage.NewOracleClient(preimageClient)
	hClient := preimage.NewHintWriter(hintClient)
	l1PreimageOracle := l1.NewPreimageOracle(pClient, hClient)

	require.Equal(t, l1Head.Bytes(), pClient.Get(client.L1HeadLocalIndex), "Should get preimages")

	// Should exit when a preimage is unavailable
	require.Panics(t, func() {
		l1PreimageOracle.HeaderByBlockHash(common.HexToHash("0x1234"))
	}, "Preimage should not be available")
	require.ErrorIs(t, waitFor(result), kvstore.ErrNotFound)
}

func waitFor(ch chan error) error {
	timeout := time.After(30 * time.Second)
	select {
	case err := <-ch:
		return err
	case <-timeout:
		return errors.New("timed out")
	}
}
