package derive

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
			stage := NewBatchStage(testlog.Logger(t, log.LevelDebug), cfg, input, newFakeSafeBlockFetcher())
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
