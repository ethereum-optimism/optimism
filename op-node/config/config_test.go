package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestCheckL1BlockTagPolling(t *testing.T) {
	require.NoError(t, checkL1BlockTagPolling(&driver.Config{}, 0))
	require.NoError(t, checkL1BlockTagPolling(&driver.Config{VerifierL1BlockTag: eth.Unsafe}, 0))
	require.NoError(t, checkL1BlockTagPolling(&driver.Config{VerifierL1BlockTag: eth.Safe}, time.Second))
	require.ErrorContains(t, checkL1BlockTagPolling(&driver.Config{VerifierL1BlockTag: eth.Finalized}, 0), "poll interval")
	require.ErrorContains(t, checkL1BlockTagPolling(&driver.Config{SequencerL1BlockTag: eth.Safe}, 0), "poll interval")
}
