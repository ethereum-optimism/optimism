package benchmarks

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/stretchr/testify/require"
)

type BatchingBenchmarkTC struct {
	BatchType  uint
	BatchCount int
	txPerBatch int
}

func (t BatchingBenchmarkTC) String(cName string) string {
	var btype string
	if t.BatchType == derive.SingularBatchType {
		btype = "Singular"
	}
	if t.BatchType == derive.SpanBatchType {
		btype = "Span"
	}
	return fmt.Sprintf("BatchType=%s, txPerBatch=%d, BatchCount=%d, Compressor=%s", btype, t.txPerBatch, t.BatchCount, cName)
}

// BenchmarkChannelOut benchmarks the performance of adding singular batches to a channel out
// this exercises the compression and batching logic, as well as any batch-building logic
// Every Compressor in the compressor map is benchmarked for each test case
func BenchmarkChannelOut(b *testing.B) {
	rc, _ := compressor.NewRatioCompressor(compressor.Config{
		TargetOutputSize: 100_000_000_000,
		ApproxComprRatio: 0.4,
	})
	sc, _ := compressor.NewShadowCompressor(compressor.Config{
		TargetOutputSize: 100_000_000_000,
	})
	nc, _ := compressor.NewNonCompressor(compressor.Config{
		TargetOutputSize: 100_000_000_000,
	})

	compressors := map[string]derive.Compressor{
		"NonCompressor":    nc,
		"RatioCompressor":  rc,
		"ShadowCompressor": sc,
	}

	tests := []BatchingBenchmarkTC{
		// Singular Batch Tests
		// low-throughput chains
		{derive.SingularBatchType, 10, 1},
		{derive.SingularBatchType, 100, 1},
		{derive.SingularBatchType, 1000, 1},
		{derive.SingularBatchType, 10000, 1},

		// higher-throughput chains
		{derive.SingularBatchType, 10, 100},
		{derive.SingularBatchType, 100, 100},
		{derive.SingularBatchType, 1000, 100},

		// Span Batch Tests
		// low-throughput chains
		{derive.SpanBatchType, 10, 1},
		{derive.SpanBatchType, 100, 1},
		{derive.SpanBatchType, 1000, 1},
		{derive.SpanBatchType, 10000, 1},

		// higher-throughput chains
		{derive.SpanBatchType, 10, 100},
		{derive.SpanBatchType, 100, 100},
		{derive.SpanBatchType, 1000, 100},
	}

	// for each compressor, run each the tests
	for cName, c := range compressors {
		for _, tc := range tests {
			chainID := big.NewInt(333)
			spanBatchBuilder := derive.NewSpanBatchBuilder(0, chainID)
			rng := rand.New(rand.NewSource(0x543331))
			c.Reset()
			// pre-generate batches to keep the benchmark from including the random generation
			batches := make([]*derive.SingularBatch, tc.BatchCount)
			for i := 0; i < tc.BatchCount; i++ {
				batches[i] = derive.RandomSingularBatch(rng, tc.txPerBatch, chainID)
			}
			b.Run(tc.String(cName), func(b *testing.B) {
				for bn := 0; bn < b.N; bn++ {
					cout, _ := derive.NewChannelOut(tc.BatchType, c, spanBatchBuilder)
					for i := 0; i < tc.BatchCount; i++ {
						_, err := cout.AddSingularBatch(batches[i], 0)
						require.NoError(b, err)
					}
				}
			})
		}
	}
}
