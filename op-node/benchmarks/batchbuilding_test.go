package benchmarks

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/stretchr/testify/require"
)

var (

	// compressors used in the benchmark
	rc, _ = compressor.NewRatioCompressor(compressor.Config{
		TargetOutputSize: 100_000_000_000,
		ApproxComprRatio: 0.4,
	})
	sc, _ = compressor.NewShadowCompressor(compressor.Config{
		TargetOutputSize: 100_000_000_000,
	})
	nc, _ = compressor.NewNonCompressor(compressor.Config{
		TargetOutputSize: 100_000_000_000,
	})

	compressors = map[string]derive.Compressor{
		"NonCompressor":    nc,
		"RatioCompressor":  rc,
		"ShadowCompressor": sc,
	}

	// batch types used in the benchmark
	batchTypes = []uint{
		derive.SingularBatchType,
		derive.SpanBatchType,
	}
)

// a test case for the benchmark controls the number of batches and transactions per batch,
// as well as the batch type and compressor used
type BatchingBenchmarkTC struct {
	BatchType  uint
	BatchCount int
	txPerBatch int
	compKey    string
}

func (t BatchingBenchmarkTC) String() string {
	var btype string
	if t.BatchType == derive.SingularBatchType {
		btype = "Singular"
	}
	if t.BatchType == derive.SpanBatchType {
		btype = "Span"
	}
	return fmt.Sprintf("BatchType=%s, txPerBatch=%d, BatchCount=%d, Compressor=%s", btype, t.txPerBatch, t.BatchCount, t.compKey)
}

// BenchmarkChannelOut benchmarks the performance of adding singular batches to a channel out
// this exercises the compression and batching logic, as well as any batch-building logic
// Every Compressor in the compressor map is benchmarked for each test case
// The results of the Benchmark measure *only* the time to add the final batch to the channel out,
// not the time to send all the batches through the channel out
// Hint: Remove the Start/Stop timers to measure the time to send all the batches through the channel out
// Hint: Raise the derive.MaxRLPBytesPerChannel to 10_000_000_000 to avoid hitting limits
func BenchmarkChannelOut(b *testing.B) {
	// Targets define the number of batches and transactions per batch to test
	type target struct{ bs, tpb int }
	targets := []target{
		{10, 1},
		{100, 1},
		{1000, 1},
		{10000, 1},

		{10, 100},
		{100, 100},
		{1000, 100},
	}

	// build a set of test cases for each batch type, compressor, and target-pair
	tests := []BatchingBenchmarkTC{}
	for _, bt := range batchTypes {
		for compkey := range compressors {
			for _, t := range targets {
				tests = append(tests, BatchingBenchmarkTC{bt, t.bs, t.tpb, compkey})
			}
		}
	}

	for _, tc := range tests {
		chainID := big.NewInt(333)
		rng := rand.New(rand.NewSource(0x543331))
		// pre-generate batches to keep the benchmark from including the random generation
		batches := make([]*derive.SingularBatch, tc.BatchCount)
		t := time.Now()
		for i := 0; i < tc.BatchCount; i++ {
			batches[i] = derive.RandomSingularBatch(rng, tc.txPerBatch, chainID)
			// set the timestamp to increase with each batch
			// to leverage optimizations in the Batch Linked List
			batches[i].Timestamp = uint64(t.Add(time.Duration(i) * time.Second).Unix())
		}
		b.Run(tc.String(), func(b *testing.B) {
			// reset the compressor used in the test case
			for bn := 0; bn < b.N; bn++ {
				// don't measure the setup time
				b.StopTimer()
				compressors[tc.compKey].Reset()
				spanBatchBuilder := derive.NewSpanBatchBuilder(0, chainID)
				cout, _ := derive.NewChannelOut(tc.BatchType, compressors[tc.compKey], spanBatchBuilder)
				// add all but the final batche to the channel out
				for i := 0; i < tc.BatchCount-1; i++ {
					_, err := cout.AddSingularBatch(batches[i], 0)
					require.NoError(b, err)
				}
				// measure the time to add the final batch
				b.StartTimer()
				// add the final batch to the channel out
				_, err := cout.AddSingularBatch(batches[tc.BatchCount-1], 0)
				require.NoError(b, err)
			}
		})
	}
}
func BenchmarkGetRawSpanBatch(b *testing.B) {
	// Targets define the number of batches and transactions per batch to test
	type target struct{ bs, tpb int }
	targets := []target{
		{10, 1},
		{100, 1},
		{1000, 1},
		{10000, 1},

		{10, 100},
		{100, 100},
		{1000, 100},
	}

	tests := []BatchingBenchmarkTC{}
	for _, t := range targets {
		tests = append(tests, BatchingBenchmarkTC{derive.SpanBatchType, t.bs, t.tpb, "NonCompressor"})
	}

	for _, tc := range tests {
		chainID := big.NewInt(333)
		rng := rand.New(rand.NewSource(0x543331))
		// pre-generate batches to keep the benchmark from including the random generation
		batches := make([]*derive.SingularBatch, tc.BatchCount)
		t := time.Now()
		for i := 0; i < tc.BatchCount; i++ {
			batches[i] = derive.RandomSingularBatch(rng, tc.txPerBatch, chainID)
			batches[i].Timestamp = uint64(t.Add(time.Duration(i) * time.Second).Unix())
		}
		b.Run(tc.String(), func(b *testing.B) {
			for bn := 0; bn < b.N; bn++ {
				// don't measure the setup time
				b.StopTimer()
				spanBatchBuilder := derive.NewSpanBatchBuilder(0, chainID)
				for i := 0; i < tc.BatchCount; i++ {
					spanBatchBuilder.AppendSingularBatch(batches[i], 0)
				}
				b.StartTimer()
				_, err := spanBatchBuilder.GetRawSpanBatch()
				require.NoError(b, err)
			}
		})
	}
}
