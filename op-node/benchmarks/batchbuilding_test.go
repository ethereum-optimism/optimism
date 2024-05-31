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

// compressorDetails is a helper struct to create compressors or supply the configuration for span batches
type compressorDetails struct {
	name         string
	compressorFn func(compressor.Config) (derive.Compressor, error)
	config       compressor.Config
}

func (cd compressorDetails) String() string {
	return fmt.Sprintf("%s-%s-%d", cd.name, cd.config.CompressionAlgo, cd.config.TargetOutputSize)
}
func (cd compressorDetails) Compressor() (derive.Compressor, error) {
	return cd.compressorFn(cd.config)
}

var (
	// batch types used in the benchmark
	batchTypes = []uint{
		derive.SpanBatchType,
	}

	compAlgos = []derive.CompressionAlgo{
		derive.Zlib,
		derive.Brotli,
		derive.Brotli9,
		derive.Brotli11,
	}

	// compressors used in the benchmark
	// they are all configured to Zlib compression, which may be overridden in the test cases
	compressors = map[string]compressorDetails{
		"NonCompressor": {
			name:         "NonCompressor",
			compressorFn: compressor.NewNonCompressor,
			config: compressor.Config{
				TargetOutputSize: targetOutput_huge,
				CompressionAlgo:  derive.Zlib,
			},
		},
		"RatioCompressor": {
			name:         "RatioCompressor",
			compressorFn: compressor.NewRatioCompressor,
			config: compressor.Config{
				TargetOutputSize: targetOutput_huge,
				CompressionAlgo:  derive.Zlib,
			},
		},
		"ShadowCompressor": {
			name:         "ShadowCompressor",
			compressorFn: compressor.NewShadowCompressor,
			config: compressor.Config{
				TargetOutputSize: targetOutput_huge,
				CompressionAlgo:  derive.Zlib,
			},
		},
		"RealShadowCompressor": {
			name:         "ShadowCompressor",
			compressorFn: compressor.NewShadowCompressor,
			config: compressor.Config{
				TargetOutputSize: targetOuput_real,
				CompressionAlgo:  derive.Zlib,
			},
		},
	}
)

// channelOutByType returns a channel out of the given type as a helper for the benchmarks
func channelOutByType(b *testing.B, batchType uint, cd compressorDetails) (derive.ChannelOut, error) {
	chainID := big.NewInt(333)
	if batchType == derive.SingularBatchType {
		compressor, err := cd.Compressor()
		require.NoError(b, err)
		return derive.NewSingularChannelOut(compressor)
	}
	if batchType == derive.SpanBatchType {
		return derive.NewSpanChannelOut(0, chainID, cd.config.TargetOutputSize, cd.config.CompressionAlgo)
	}
	return nil, fmt.Errorf("unsupported batch type: %d", batchType)
}

// a test case for the benchmark controls the number of batches and transactions per batch,
// as well as the batch type and compressor used
type BatchingBenchmarkTC struct {
	BatchType  uint
	BatchCount int
	txPerBatch int
	cd         compressorDetails
}

func (t BatchingBenchmarkTC) String() string {
	var btype string
	if t.BatchType == derive.SingularBatchType {
		btype = "Singular"
	}
	if t.BatchType == derive.SpanBatchType {
		btype = "Span"
	}
	return fmt.Sprintf("BatchType=%s, txPerBatch=%d, BatchCount=%d, Compressor=%s", btype, t.txPerBatch, t.BatchCount, t.cd.String())
}

// BenchmarkChannelOut benchmarks the performance of adding singular batches to a channel out
// this exercises the compression and batching logic, as well as any batch-building logic
// Every Compressor in the compressor map is benchmarked for each test case
// The results of the Benchmark measure *only* the time to add the final batch to the channel out,
// not the time to send all the batches through the channel out
// Hint: Raise the rollup.MaxRLPBytesPerChannel to 10_000_000_000 to avoid hitting limits if adding larger test cases
func BenchmarkFinalBatchChannelOut(b *testing.B) {
	// Targets define the number of batches and transactions per batch to test
	// they will be multiplied by various compressors
	type target struct{ bs, tpb int }
	targets := []target{
		{10, 1},
		{100, 1},
		{1000, 1},

		{10, 100},
		{100, 100},
	}

	// make test-cases for every batch type, compressor, compressorAlgo, and target-pair
	tests := []BatchingBenchmarkTC{}
	for _, bt := range batchTypes {
		for _, compDetails := range compressors {
			for _, algo := range compAlgos {
				for _, t := range targets {
					cd := compDetails
					cd.config.CompressionAlgo = algo
					tests = append(tests, BatchingBenchmarkTC{bt, t.bs, t.tpb, cd})
				}
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
				cout, _ := channelOutByType(b, tc.BatchType, tc.cd)
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
		{derive.SpanBatchType, 5, 1, compressorDetails{
			name: "RealThreshold",
			config: compressor.Config{
				TargetOutputSize: targetOuput_real,
				CompressionAlgo:  derive.Zlib,
			},
		}},
		{derive.SpanBatchType, 5, 1, compressorDetails{
			name: "RealThreshold",
			config: compressor.Config{
				TargetOutputSize: targetOuput_real,
				CompressionAlgo:  derive.Brotli10,
			},
		}},
	}
	for _, tc := range tcs {
		cout, err := channelOutByType(b, tc.BatchType, tc.cd)
		if err != nil {
			b.Fatal(err)
		}
		done := false
		for base := 0; !done; base += tc.BatchCount {
			rangeName := fmt.Sprintf("Incremental %s: %d-%d", tc.String(), base, base+tc.BatchCount)
			b.Run(rangeName, func(b *testing.B) {
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
		for _, compDetails := range compressors {
			for _, algo := range compAlgos {
				for _, t := range targets {
					cd := compDetails
					cd.config.CompressionAlgo = algo
					tests = append(tests, BatchingBenchmarkTC{bt, t.bs, t.tpb, cd})
				}
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
				cout, _ := channelOutByType(b, tc.BatchType, tc.cd)
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
		tests = append(tests, BatchingBenchmarkTC{derive.SpanBatchType, t.bs, t.tpb, compressors["NonCompressor"]})
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
