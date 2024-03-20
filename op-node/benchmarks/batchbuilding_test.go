package benchmarks

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func RandomSingularBatch(rng *rand.Rand, txCount int, chainID *big.Int) *derive.SingularBatch {
	signer := types.NewLondonSigner(chainID)
	baseFee := big.NewInt(rng.Int63n(300_000_000_000))
	txsEncoded := make([]hexutil.Bytes, 0, txCount)
	// force each tx to have equal chainID
	for i := 0; i < txCount; i++ {
		tx := testutils.RandomTx(rng, baseFee, signer)
		txEncoded, err := tx.MarshalBinary()
		if err != nil {
			panic("tx Marshal binary" + err.Error())
		}
		txsEncoded = append(txsEncoded, hexutil.Bytes(txEncoded))
	}
	return &derive.SingularBatch{
		ParentHash:   testutils.RandomHash(rng),
		EpochNum:     rollup.Epoch(1 + rng.Int63n(100_000_000)),
		EpochHash:    testutils.RandomHash(rng),
		Timestamp:    uint64(rng.Int63n(2_000_000_000)),
		Transactions: txsEncoded,
	}
}

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
				batches[i] = RandomSingularBatch(rng, tc.txPerBatch, chainID)
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
