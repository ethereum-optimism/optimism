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

// projectionVector is the shared admission vector schema v2 (op-private-interop/projection
// testdata): each vector carries its consensus config and derivation context.
type projectionVector struct {
	Schedule string            `json:"-"`
	Name     string            `json:"name"`
	Accept   bool              `json:"accept"`
	Config   projection.Config `json:"config"`
	Context  struct {
		ChainID       uint64      `json:"chain_id"`
		GenesisNumber uint64      `json:"genesis_number"`
		GenesisHash   common.Hash `json:"genesis_hash"`
		GenesisTime   uint64      `json:"genesis_time"`
		BlockTime     uint64      `json:"block_time"`
		ParentHash    common.Hash `json:"parent_hash"`
		L1Head        common.Hash `json:"l1_head"`
		Continuation  struct {
			Anchor       eth.BlockID `json:"anchor"`
			OutputRoot   common.Hash `json:"output_root"`
			RecoveryHash common.Hash `json:"recovery_hash"`
		} `json:"continuation"`
	} `json:"context"`
	Blocks []struct {
		Timestamp    uint64          `json:"timestamp"`
		Epoch        uint64          `json:"epoch"`
		Transactions []hexutil.Bytes `json:"transactions"`
	} `json:"blocks"`
}

func loadProjectionVectors(t *testing.T, names ...string) []projectionVector {
	var out []projectionVector
	for _, name := range names {
		raw, err := os.ReadFile("../../../op-private-interop/projection/testdata/" + name)
		require.NoError(t, err)
		var vs []projectionVector
		require.NoError(t, json.Unmarshal(raw, &vs))
		out = append(out, vs...)
	}
	return out
}

// rollupConfig is the projection rollup config the vector's context describes.
func (v *projectionVector) rollupConfig(seqWindow uint64) *rollup.Config {
	return &rollup.Config{
		Genesis:   rollup.Genesis{L2: eth.BlockID{Number: v.Context.GenesisNumber, Hash: v.Context.GenesisHash}, L2Time: v.Context.GenesisTime},
		BlockTime: v.Context.BlockTime, L2ChainID: new(big.Int).SetUint64(v.Context.ChainID), SeqWindowSize: seqWindow,
		MaxSequencerDrift: 600, HoloceneTime: &zero64, DeltaTime: &zero64, PrivateProjection: &v.Config,
	}
}

// pureAdmission is the verdict of the pure admission function under the vector's configured
// verifier, with the context the batch stage resolves in these tests.
func (v *projectionVector) pureAdmission(t *testing.T, cfg *rollup.Config, l1Head common.Hash) bool {
	verifier, err := projection.VerifierFor(cfg.PrivateProjection, cfg.L2ChainID)
	if err != nil {
		return false
	}
	blocks := make(testProjectionSpan, len(v.Blocks))
	for i, b := range v.Blocks {
		blocks[i] = testProjectionBlock{b.Timestamp, b.Epoch, b.Transactions}
	}
	_, err = projection.ValidateProjectionRange(cfg.PrivateProjection, projection.Context{
		ChainID: cfg.L2ChainID, GenesisNumber: cfg.Genesis.L2.Number, GenesisTime: cfg.Genesis.L2Time, BlockTime: cfg.BlockTime,
		GenesisHash: cfg.Genesis.L2.Hash, ParentHash: common.Hash{1}, L1Head: l1Head,
		Continuation: projection.Continuation{Anchor: eth.BlockID{Number: 9, Hash: common.Hash{1}}, OutputRoot: common.Hash{9}},
	}, blocks, verifier)
	return err == nil
}

type testProjectionBlock struct {
	timestamp, epoch uint64
	txs              []hexutil.Bytes
}
type testProjectionSpan []testProjectionBlock

func (s testProjectionSpan) GetBlockCount() int                         { return len(s) }
func (s testProjectionSpan) GetBlockTimestamp(i int) uint64             { return s[i].timestamp }
func (s testProjectionSpan) GetBlockEpochNum(i int) uint64              { return s[i].epoch }
func (s testProjectionSpan) GetBlockTransactions(i int) []hexutil.Bytes { return s[i].txs }

// The malformed transaction is in the final block: no valid prefix may escape
// into attributes/execution. Use the exact same bytes as the Go/Kona pure tests.
// The batch stage must agree with pure admission under the configured verifier, which accepts
// every vector that carries a valid envelope for its config.
func TestProjectionSpanAdmissionBeforeFirstBlock(t *testing.T) {
	vectors := loadProjectionVectors(t, "ranges.json", "proofs.json")
	for _, name := range []string{"late_origin", "late_fork", "late_drift"} {
		v := vectors[0]
		v.Name, v.Schedule, v.Accept = name, name, false
		vectors = append(vectors, v)
	}
	accepted := 0
	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			// The fake fetcher below serves the normal-mode continuation only; recovery-mode
			// vectors are exercised by the pure admission tests.
			if c := v.Context.Continuation; c.Anchor != (eth.BlockID{Number: 9, Hash: common.Hash{1}}) || c.OutputRoot != (common.Hash{9}) || c.RecoveryHash != (common.Hash{}) {
				t.Skip("recovery-mode context")
			}
			l1 := []eth.L1BlockRef{{Hash: common.Hash{5}, Number: 5, Time: 1000}, {Hash: common.Hash{6}, ParentHash: common.Hash{5}, Number: 6, Time: 1012}}
			parent := eth.L2BlockRef{Hash: common.Hash{1}, Number: 9, Time: 1018, L1Origin: l1[0].ID()}
			cfg := v.rollupConfig(100)
			pure := v.pureAdmission(t, cfg, l1[1].Hash)
			if !v.Accept && v.Schedule == "" {
				require.False(t, pure, "a structurally rejected vector cannot be admitted")
			}
			if v.Schedule != "" {
				require.True(t, pure, "schedule variants are rejected only by the schedule")
			}
			want := v.Schedule == "" && pure
			switch v.Schedule {
			case "late_origin":
				l1[1].Time = 1025
			case "late_fork":
				fork := uint64(1024)
				cfg.JovianTime = &fork
			case "late_drift":
				// Pre-Fjord drift (600s) is exceeded by the replay block: 1024 - 21 > 600. The
				// vector geometry is kept, so the claim's rollupConfigHash still matches and the
				// schedule is the only reason for rejection.
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
			if want {
				accepted++
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
	// mixed, empty_messages, wide_chain_id, events_enabled, execution_mock_valid, sp1_mock_valid.
	require.GreaterOrEqual(t, accepted, 6)
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
	v := loadProjectionVectors(t, "ranges.json")[0]
	for _, reset := range []bool{false, true} {
		l1 := []eth.L1BlockRef{{Hash: common.Hash{5}, Number: 5, Time: 1000}, {Hash: common.Hash{6}, ParentHash: common.Hash{5}, Number: 6, Time: 1012}}
		parent := eth.L2BlockRef{Hash: common.Hash{1}, Number: 9, Time: 1018, L1Origin: l1[0].ID()}
		cfg := v.rollupConfig(2)
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
