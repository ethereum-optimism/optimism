package client

import (
	"io"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBootstrapOracle(t *testing.T) {
	r, w := io.Pipe()
	br := NewBootstrapOracleReader(r)
	bw := NewBootstrapOracleWriter(w)

	bootInfo := BootInfo{
		Rollup:             new(rollup.Config),
		L2ChainConfig:      new(params.ChainConfig),
		L1Head:             common.HexToHash("0xffffa"),
		L2Head:             common.HexToHash("0xffffb"),
		L2Claim:            common.HexToHash("0xffffc"),
		L2ClaimBlockNumber: 1,
	}

	go func() {
		err := bw.WriteBootInfo(&bootInfo)
		require.NoError(t, err)
	}()

	type result struct {
		bootInnfo *BootInfo
		err       error
	}
	read := make(chan result)
	go func() {
		readBootInfo, err := br.BootInfo()
		read <- result{readBootInfo, err}
		close(read)
	}()

	select {
	case <-time.After(time.Second * 30):
		t.Error("timeout waiting for bootstrap oracle")
	case r := <-read:
		require.NoError(t, r.err)
		require.Equal(t, bootInfo, *r.bootInnfo)
	}
}
