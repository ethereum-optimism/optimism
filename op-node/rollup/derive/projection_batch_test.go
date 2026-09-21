package derive

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// The malformed transaction is in the final block: no valid prefix may escape
// into attributes/execution. Use the exact same bytes as the Go/Kona pure tests.
func TestProjectionSpanAdmissionBeforeFirstBlock(t *testing.T) {
	raw, err := os.ReadFile("../../../op-private-interop/projection/testdata/ranges.json")
	require.NoError(t, err)
	var vectors []struct {
		Schedule string            `json:"-"`
		Name     string            `json:"name"`
		Accept   bool              `json:"accept"`
		Config   projection.Config `json:"config"`
		Blocks   []struct {
			Timestamp    uint64          `json:"timestamp"`
			Epoch        uint64          `json:"epoch"`
			Transactions []hexutil.Bytes `json:"transactions"`
		} `json:"blocks"`
	}
	require.NoError(t, json.Unmarshal(raw, &vectors))
	for _, name := range []string{"late_origin", "late_fork", "late_drift"} {
		v := vectors[0]
		v.Name, v.Schedule, v.Accept = name, name, false
		vectors = append(vectors, v)
	}
	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			l1 := []eth.L1BlockRef{{Hash: common.Hash{5}, Number: 5, Time: 1000}, {Hash: common.Hash{6}, ParentHash: common.Hash{5}, Number: 6, Time: 1012}}
			parent := eth.L2BlockRef{Hash: common.Hash{1}, Number: 9, Time: 1018, L1Origin: l1[0].ID()}
			cfg := &rollup.Config{Genesis: rollup.Genesis{L2Time: 1000}, BlockTime: 2, L2ChainID: big.NewInt(901), SeqWindowSize: 100, MaxSequencerDrift: 600, HoloceneTime: &zero64, DeltaTime: &zero64, PrivateProjection: &v.Config}
			switch v.Schedule {
			case "late_origin":
				l1[1].Time = 1025
			case "late_fork":
				fork := uint64(1024)
				cfg.JovianTime = &fork
			case "late_drift":
				cfg.FjordTime = &zero64
				cfg.Genesis.L2Time += 800
				parent.Time += 800
				v.Blocks = append(v.Blocks[:0:0], v.Blocks...)
				for i := range v.Blocks {
					v.Blocks[i].Timestamp += 800
				}
				l1[0].Time = 20
				l1[1].Time = 21
			}
			singles := make([]*SingularBatch, len(v.Blocks))
			for i, b := range v.Blocks {
				origin := l1[0]
				if b.Epoch == 6 {
					origin = l1[1]
				}
				singles[i] = &SingularBatch{ParentHash: parent.Hash, Timestamp: b.Timestamp, EpochNum: rollup.Epoch(b.Epoch), EpochHash: origin.Hash, Transactions: b.Transactions}
			}
			// The span wire derives block timestamps from its start and the configured interval.
			// Noncontiguous timestamps cannot survive that wire, and are tested by the pure validator.
			if v.Name == "noncontiguous" {
				return
			}
			span := initializedSpanBatch(singles, 0, cfg.L2ChainID)
			input := &fakeBatchQueueInput{batches: []Batch{span}, errors: []error{nil}, origin: l1[1]}
			fetcher := newFakeSafeBlockFetcher()
			to := predeploys.ClaimRegistryAddr
			output := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(901), To: &to, Data: wire.EncodeOutput(common.Hash{9})})
			raw, err := output.MarshalBinary()
			require.NoError(t, err)
			fetcher.addBlock(parent, &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{BlockHash: parent.Hash, BlockNumber: hexutil.Uint64(parent.Number), Transactions: []hexutil.Bytes{raw}}})
			stage := NewBatchStage(testlog.Logger(t, log.LevelDebug), cfg, input, fetcher)
			stage.l1Blocks = l1
			stage.origin = l1[1]
			next, _, err := stage.NextBatch(context.Background(), parent)
			if v.Accept {
				require.NoError(t, err)
				require.NotNil(t, next)
				require.Equal(t, singles[0].Transactions, next.Transactions)
			} else {
				require.ErrorIs(t, err, NotEnoughData)
				require.Nil(t, next)
				require.Empty(t, stage.nextSpan)
				require.Empty(t, input.batches)
			}
		})
	}
}

func TestProjectionRejectsSubmittedSingularButPreservesOrdinaryChains(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		cfg := &rollup.Config{}
		if enabled {
			cfg.PrivateProjection = &projection.Config{Verifier: projection.InsecureStub}
		}
		submitted := &SingularBatch{Timestamp: 2}
		input := &fakeBatchQueueInput{batches: []Batch{submitted}, errors: []error{nil}}
		stage := NewBatchStage(testlog.Logger(t, log.LevelDebug), cfg, input, newFakeSafeBlockFetcher())
		next, err := stage.nextSingularBatchCandidate(context.Background(), eth.L2BlockRef{})
		if enabled {
			require.ErrorIs(t, err, NotEnoughData)
			require.Nil(t, next)
			require.Empty(t, input.batches)
		} else {
			require.NoError(t, err)
			require.Same(t, submitted, next)
		}
	}
}

func TestProjectionRetainsCandidateAndOriginalInclusion(t *testing.T) {
	raw, err := os.ReadFile("../../../op-private-interop/projection/testdata/ranges.json")
	require.NoError(t, err)
	var vectors []struct {
		Config projection.Config `json:"config"`
		Blocks []struct {
			Timestamp    uint64          `json:"timestamp"`
			Epoch        uint64          `json:"epoch"`
			Transactions []hexutil.Bytes `json:"transactions"`
		} `json:"blocks"`
	}
	require.NoError(t, json.Unmarshal(raw, &vectors))
	v := vectors[0]
	for _, reset := range []bool{false, true} {
		l1 := []eth.L1BlockRef{{Hash: common.Hash{5}, Number: 5, Time: 1000}, {Hash: common.Hash{6}, ParentHash: common.Hash{5}, Number: 6, Time: 1012}}
		parent := eth.L2BlockRef{Hash: common.Hash{1}, Number: 9, Time: 1018, L1Origin: l1[0].ID()}
		cfg := &rollup.Config{Genesis: rollup.Genesis{L2Time: 1000}, BlockTime: 2, L2ChainID: big.NewInt(901), SeqWindowSize: 2, MaxSequencerDrift: 600, HoloceneTime: &zero64, DeltaTime: &zero64, PrivateProjection: &v.Config}
		singles := make([]*SingularBatch, len(v.Blocks))
		for i, b := range v.Blocks {
			origin := l1[b.Epoch-5]
			singles[i] = &SingularBatch{ParentHash: parent.Hash, Timestamp: b.Timestamp, EpochNum: rollup.Epoch(b.Epoch), EpochHash: origin.Hash, Transactions: b.Transactions}
		}
		span := initializedSpanBatch(singles, 0, cfg.L2ChainID)
		input := &fakeBatchQueueInput{batches: []Batch{span}, errors: []error{nil}, origin: l1[1]}
		fetcher := newFakeSafeBlockFetcher()
		stage := NewBatchStage(testlog.Logger(t, log.LevelDebug), cfg, input, fetcher)
		stage.l1Blocks, stage.origin = l1, l1[1]
		next, _, err := stage.NextBatch(t.Context(), parent)
		require.ErrorIs(t, err, projection.ErrContextUnavailable)
		require.Nil(t, next)
		require.NotNil(t, stage.pending)
		require.Equal(t, 1, input.i)
		if reset {
			stage.FlushChannel()
			require.Nil(t, stage.pending)
			continue
		}
		to := predeploys.ClaimRegistryAddr
		output := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(901), To: &to, Data: wire.EncodeOutput(common.Hash{9})})
		encoded, err := output.MarshalBinary()
		require.NoError(t, err)
		fetcher.addBlock(parent, &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{BlockHash: parent.Hash, BlockNumber: hexutil.Uint64(parent.Number), Transactions: []hexutil.Bytes{encoded}}})
		// Move traversal beyond the inclusion deadline while context is being retried.
		input.origin = eth.L1BlockRef{Hash: common.Hash{20}, Number: 20, Time: 1180}
		for i, want := range singles {
			next, _, err = stage.NextBatch(t.Context(), parent)
			require.NoError(t, err, "block %d retains the original timely inclusion", i)
			require.Equal(t, want.Transactions, next.Transactions)
			parent = eth.L2BlockRef{Hash: common.Hash{byte(100 + i)}, Number: parent.Number + 1, Time: next.Timestamp, L1Origin: eth.BlockID{Hash: next.EpochHash, Number: uint64(next.EpochNum)}}
		}
		require.Equal(t, 1, input.i, "retained candidate was not read twice")
		require.Nil(t, stage.pending)
	}
}

func TestProjectionMetadataPreservesDriftScheduling(t *testing.T) {
	vectorBytes, err := os.ReadFile("../../../op-private-interop/projection/testdata/ranges.json")
	require.NoError(t, err)
	var vectors []struct {
		Blocks []struct {
			Transactions []hexutil.Bytes `json:"transactions"`
		} `json:"blocks"`
	}
	require.NoError(t, json.Unmarshal(vectorBytes, &vectors))
	replay := vectors[0].Blocks[2].Transactions[1]
	to := predeploys.ClaimRegistryAddr
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(901), To: &to, Data: wire.EncodeOutput(common.Hash{9})})
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	for _, scenario := range []string{"late L1", "next origin available", "missing origin", "with replay", "ordinary chain", "activation"} {
		t.Run(scenario, func(t *testing.T) {
			l1 := []eth.L1BlockRef{{Hash: common.Hash{5}, Number: 5, Time: 1000}, {Hash: common.Hash{6}, Number: 6, Time: 3006}}
			parent := eth.L2BlockRef{Hash: common.Hash{1}, Number: 1, Time: 3002, L1Origin: l1[0].ID()}
			cfg := &rollup.Config{BlockTime: 2, SeqWindowSize: 100, MaxSequencerDrift: 1, HoloceneTime: &zero64, FjordTime: &zero64, PrivateProjection: &projection.Config{Verifier: projection.InsecureStub}}
			batch := &SingularBatch{ParentHash: parent.Hash, Timestamp: 3004, EpochNum: 5, EpochHash: l1[0].Hash, Transactions: []hexutil.Bytes{raw}}
			want := BatchValidity(BatchAccept)
			switch scenario {
			case "next origin available":
				l1[1].Time = batch.Timestamp
				want = BatchDrop
			case "missing origin":
				l1 = l1[:1]
				want = BatchUndecided
			case "with replay":
				batch.Transactions = append(batch.Transactions, replay)
				want = BatchDrop
			case "ordinary chain":
				cfg.PrivateProjection = nil
				want = BatchDrop
			case "activation":
				cfg.JovianTime = &batch.Timestamp
				want = BatchDrop
			}
			require.Equal(t, want, checkSingularBatch(cfg, testlog.Logger(t, log.LevelDebug), l1, parent, batch, eth.L1BlockRef{Number: 6}))
		})
	}
}
