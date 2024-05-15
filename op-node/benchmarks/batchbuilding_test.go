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

const (
	// a really large target output size to ensure that the compressors are never full
	targetOutput_huge = uint64(100_000_000_000)
	// this target size was determiend by the devnet sepolia batcher's configuration
	targetOuput_real = uint64(780120)
)

var (
	// compressors used in the benchmark
	rc, _ = compressor.NewRatioCompressor(compressor.Config{
		TargetOutputSize: targetOutput_huge,
		ApproxComprRatio: 0.4,
	})
	sc, _ = compressor.NewShadowCompressor(compressor.Config{
		TargetOutputSize: targetOutput_huge,
	})
	nc, _ = compressor.NewNonCompressor(compressor.Config{
		TargetOutputSize: targetOutput_huge,
	})
	realsc, _ = compressor.NewShadowCompressor(compressor.Config{
		TargetOutputSize: targetOuput_real,
	})

	// compressors used in the benchmark mapped by their name
	// they come paired with a target output size so span batches can use the target size directly
	compressors = map[string]compressorAndTarget{
		"NonCompressor":        {nc, targetOutput_huge},
		"RatioCompressor":      {rc, targetOutput_huge},
		"ShadowCompressor":     {sc, targetOutput_huge},
		"RealShadowCompressor": {realsc, targetOuput_real},
	}
	// batch types used in the benchmark
	batchTypes = []uint{
		derive.SpanBatchType,
		// uncomment to include singular batches in the benchmark
		// singular batches are not included by default because they are not the target of the benchmark
		//derive.SingularBatchType,
	}
)

type compressorAndTarget struct {
	compressor   derive.Compressor
	targetOutput uint64
}

// channelOutByType returns a channel out of the given type as a helper for the benchmarks
func channelOutByType(batchType uint, compKey string, algo derive.CompressionAlgo) (derive.ChannelOut, error) {
	chainID := big.NewInt(333)
	if batchType == derive.SingularBatchType {
		return derive.NewSingularChannelOut(compressors[compKey].compressor)
	}
	if batchType == derive.SpanBatchType {
		return derive.NewSpanChannelOut(0, chainID, compressors[compKey].targetOutput, algo)
	}
	return nil, fmt.Errorf("unsupported batch type: %d", batchType)
}

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
// Hint: Raise the rollup.MaxRLPBytesPerChannel to 10_000_000_000 to avoid hitting limits if adding larger test cases
func BenchmarkFinalBatchChannelOut(b *testing.B) {
	// Targets define the number of batches and transactions per batch to test
	type target struct{ bs, tpb int }
	targets := []target{
		{10, 1},
		{100, 1},
		{1000, 1},

		{10, 100},
		{100, 100},
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
		for _, algo := range derive.CompressionAlgoTypes {
			b.Run(tc.String()+"_"+algo.String(), func(b *testing.B) {
				// reset the compressor used in the test case
				for bn := 0; bn < b.N; bn++ {
					// don't measure the setup time
					b.StopTimer()
					compressors[tc.compKey].compressor.Reset()
					cout, _ := channelOutByType(tc.BatchType, tc.compKey, algo)
					// add all but the final batch to the channel out
					for i := 0; i < tc.BatchCount-1; i++ {
						err := cout.AddSingularBatch(batches[i], 0)
						require.NoError(b, err)
					}
					// measure the time to add the final batch
					b.StartTimer()
					// add the final batch to the channel out
					err := cout.AddSingularBatch(batches[tc.BatchCount-1], 0)
					require.NoError(b, err)
				}
			})
		}

	}
}

// BenchmarkIncremental fills a channel out incrementally with batches
// each increment is counted as its own benchmark
// Hint: use -benchtime=1x to run the benchmarks for a single iteration
// it is not currently designed to use b.N
func BenchmarkIncremental(b *testing.B) {
	chainID := big.NewInt(333)
	rng := rand.New(rand.NewSource(0x543331))
	// use the real compressor for this benchmark
	// use batchCount as the number of batches to add in each benchmark iteration
	// and use txPerBatch as the number of transactions per batch
	tcs := []BatchingBenchmarkTC{
		{derive.SpanBatchType, 5, 1, "RealBlindCompressor"},
		//{derive.SingularBatchType, 100, 1, "RealShadowCompressor"},
	}
	for _, algo := range derive.CompressionAlgoTypes {
		for _, tc := range tcs {
			cout, err := channelOutByType(tc.BatchType, tc.compKey, algo)
			if err != nil {
				b.Fatal(err)
			}
			done := false
			for base := 0; !done; base += tc.BatchCount {
				rangeName := fmt.Sprintf("Incremental %s-%s: %d-%d", algo, tc.String(), base, base+tc.BatchCount)
				b.Run(rangeName+"_"+algo.String(), func(b *testing.B) {
					b.StopTimer()
					// prepare the batches
					t := time.Now()
					batches := make([]*derive.SingularBatch, tc.BatchCount)
					for i := 0; i < tc.BatchCount; i++ {
						t := t.Add(time.Second)
						batches[i] = derive.RandomSingularBatch(rng, tc.txPerBatch, chainID)
						// set the timestamp to increase with each batch
						// to leverage optimizations in the Batch Linked List
						batches[i].Timestamp = uint64(t.Unix())
					}
					b.StartTimer()
					for i := 0; i < tc.BatchCount; i++ {
						err := cout.AddSingularBatch(batches[i], 0)
						if err != nil {
							done = true
							return
						}
					}
				})
			}
		}
	}
}

// BenchmarkAllBatchesChannelOut benchmarks the performance of adding singular batches to a channel out
// this exercises the compression and batching logic, as well as any batch-building logic
// Every Compressor in the compressor map is benchmarked for each test case
// The results of the Benchmark measure the time to add the *all batches* to the channel out,
// not the time to send all the batches through the channel out
// Hint: Raise the rollup.MaxRLPBytesPerChannel to 10_000_000_000 to avoid hitting limits
func BenchmarkAllBatchesChannelOut(b *testing.B) {
	// Targets define the number of batches and transactions per batch to test
	type target struct{ bs, tpb int }
	targets := []target{
		{10, 1},
		{100, 1},
		{1000, 1},

		{10, 100},
		{100, 100},
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

	for _, algo := range derive.CompressionAlgoTypes {
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
			b.Run(tc.String()+"_"+algo.String(), func(b *testing.B) {
				// reset the compressor used in the test case
				for bn := 0; bn < b.N; bn++ {
					// don't measure the setup time
					b.StopTimer()
					compressors[tc.compKey].compressor.Reset()
					cout, _ := channelOutByType(tc.BatchType, tc.compKey, algo)
					b.StartTimer()
					// add all batches to the channel out
					for i := 0; i < tc.BatchCount; i++ {
						err := cout.AddSingularBatch(batches[i], 0)
						require.NoError(b, err)
					}
				}
			})
		}
	}
}

// BenchmarkGetRawSpanBatch benchmarks the performance of building a span batch from singular batches
// this exercises the span batch building logic directly
// The adding of batches to the span batch builder is not included in the benchmark, only the final build to RawSpanBatch
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
				spanBatch := derive.NewSpanBatch(uint64(0), chainID)
				for i := 0; i < tc.BatchCount; i++ {
					err := spanBatch.AppendSingularBatch(batches[i], 0)
					require.NoError(b, err)
				}
				b.StartTimer()
				_, err := spanBatch.ToRawSpanBatch()
				require.NoError(b, err)
			}
		})
	}
}
