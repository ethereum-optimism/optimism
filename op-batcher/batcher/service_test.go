package batcher

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestBatcherService_MultiBlockRequiresSpanBatches checks that op-batcher refuses to start against
// a multi-block chain with a batch format that cannot express siblings, instead of quietly
// submitting batches that push the safe chain onto a side branch.
func TestBatcherService_MultiBlockRequiresSpanBatches(t *testing.T) {
	rollupCfg := *defaultTestRollupConfig
	rollupCfg.MultiBlockTime = ptr.New(uint64(0))

	initWithBatchType := func(t *testing.T, batchType uint) error {
		bs := &BatcherService{Log: testlog.Logger(t, log.LevelCrit), RollupConfig: &rollupCfg}
		return bs.initChannelConfig(&CLIConfig{
			MaxL1TxSize:          120_000,
			TargetNumFrames:      1,
			ApproxComprRatio:     0.4,
			Compressor:           compressor.RatioKind,
			CompressionAlgo:      derive.Zlib,
			DataAvailabilityType: flags.CalldataType,
			BatchType:            batchType,
		})
	}

	err := initWithBatchType(t, derive.SingularBatchType)
	require.ErrorContains(t, err, "multi_block_time")
	require.NoError(t, initWithBatchType(t, derive.SpanBatchType))
}
