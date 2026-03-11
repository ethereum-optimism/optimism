package consensus

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	snappy "github.com/golang/snappy"
)

var payloadSizes = []struct {
	name string
	size int
}{
	{"10KB", 10_000},
	{"100KB", 100_000},
	{"500KB", 500_000},
	{"1MB", 1_000_000},
	{"2MB", 2_000_000},
}

func BenchmarkCommitPayload_SingleNode(b *testing.B) {
	for _, ps := range payloadSizes {
		b.Run(ps.name, func(b *testing.B) {
			cluster := newBenchCluster(b, 1, 0)
			defer cluster.shutdown()

			leader := cluster.leader()
			rng := rand.New(rand.NewSource(42))

			b.ReportAllocs()
			cluster.metrics.reset()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				payload := generatePayloadEnvelope(rng, uint64(i+1), ps.size)
				if err := leader.CommitUnsafePayload(payload); err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			cluster.metrics.reportTo(b)
		})
	}
}

func BenchmarkCommitPayload_ThreeNode(b *testing.B) {
	for _, ps := range payloadSizes {
		b.Run(ps.name, func(b *testing.B) {
			cluster := newBenchCluster(b, 3, 0)
			defer cluster.shutdown()

			leader := cluster.leader()
			rng := rand.New(rand.NewSource(42))

			b.ReportAllocs()
			cluster.metrics.reset()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				payload := generatePayloadEnvelope(rng, uint64(i+1), ps.size)
				if err := leader.CommitUnsafePayload(payload); err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			cluster.metrics.reportTo(b)
		})
	}
}

func BenchmarkCommitPayload_ThreeNode_1msLatency(b *testing.B) {
	for _, ps := range payloadSizes {
		b.Run(ps.name, func(b *testing.B) {
			cluster := newBenchCluster(b, 3, 1*time.Millisecond)
			defer cluster.shutdown()

			leader := cluster.leader()
			rng := rand.New(rand.NewSource(42))

			b.ReportAllocs()
			cluster.metrics.reset()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				payload := generatePayloadEnvelope(rng, uint64(i+1), ps.size)
				if err := leader.CommitUnsafePayload(payload); err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			cluster.metrics.reportTo(b)
		})
	}
}

func withCompression(cfg *RaftConsensusConfig) {
	cfg.CompressPayload = true
}

func BenchmarkCommitPayload_SingleNode_Compressed(b *testing.B) {
	for _, ps := range payloadSizes {
		b.Run(ps.name, func(b *testing.B) {
			cluster := newBenchCluster(b, 1, 0, withCompression)
			defer cluster.shutdown()

			leader := cluster.leader()
			rng := rand.New(rand.NewSource(42))

			b.ReportAllocs()
			cluster.metrics.reset()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				payload := generatePayloadEnvelope(rng, uint64(i+1), ps.size)
				if err := leader.CommitUnsafePayload(payload); err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			cluster.metrics.reportTo(b)
		})
	}
}

func BenchmarkCommitPayload_ThreeNode_Compressed(b *testing.B) {
	for _, ps := range payloadSizes {
		b.Run(ps.name, func(b *testing.B) {
			cluster := newBenchCluster(b, 3, 0, withCompression)
			defer cluster.shutdown()

			leader := cluster.leader()
			rng := rand.New(rand.NewSource(42))

			b.ReportAllocs()
			cluster.metrics.reset()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				payload := generatePayloadEnvelope(rng, uint64(i+1), ps.size)
				if err := leader.CommitUnsafePayload(payload); err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			cluster.metrics.reportTo(b)
		})
	}
}

func BenchmarkSSZMarshalRoundtrip(b *testing.B) {
	for _, ps := range payloadSizes {
		b.Run(ps.name, func(b *testing.B) {
			rng := rand.New(rand.NewSource(42))
			payload := generatePayloadEnvelope(rng, 1, ps.size)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				if _, err := payload.MarshalSSZ(&buf); err != nil {
					b.Fatal(err)
				}

				data := &eth.ExecutionPayloadEnvelope{}
				if err := data.UnmarshalSSZ(eth.BlockV3, uint32(buf.Len()), bytes.NewReader(buf.Bytes())); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// --- Compression micro-benchmarks ---

func BenchmarkCompression(b *testing.B) {
	type compressor struct {
		name       string
		compress   func([]byte) []byte
		decompress func([]byte) ([]byte, error)
	}

	enc, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	dec, _ := zstd.NewReader(nil)

	compressors := []compressor{
		{
			name:       "snappy",
			compress:   func(b []byte) []byte { return snappy.Encode(nil, b) },
			decompress: func(b []byte) ([]byte, error) { return snappy.Decode(nil, b) },
		},
		{
			name:       "s2",
			compress:   func(b []byte) []byte { return s2.Encode(nil, b) },
			decompress: func(b []byte) ([]byte, error) { return s2.Decode(nil, b) },
		},
		{
			name:       "zstd1",
			compress:   func(b []byte) []byte { return enc.EncodeAll(b, nil) },
			decompress: func(b []byte) ([]byte, error) { dst, err := dec.DecodeAll(b, nil); return dst, err },
		},
	}

	for _, c := range compressors {
		for _, ps := range payloadSizes {
			b.Run(fmt.Sprintf("%s/%s", c.name, ps.name), func(b *testing.B) {
				rng := rand.New(rand.NewSource(42))
				payload := generatePayloadEnvelope(rng, 1, ps.size)

				var buf bytes.Buffer
				if _, err := payload.MarshalSSZ(&buf); err != nil {
					b.Fatal(err)
				}
				raw := buf.Bytes()
				compressed := c.compress(raw)
				ratio := float64(len(compressed)) / float64(len(raw))

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					comp := c.compress(raw)
					if _, err := c.decompress(comp); err != nil {
						b.Fatal(err)
					}
				}

				b.StopTimer()
				b.ReportMetric(ratio, "ratio")
			})
		}
	}
}
