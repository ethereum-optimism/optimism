package derive

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type fakeAttributesQueue struct {
	origin       eth.L1BlockRef
	attrs        *eth.PayloadAttributes
	islastInSpan bool
}

func (f *fakeAttributesQueue) Origin() eth.L1BlockRef {
	return f.origin
}

func (f *fakeAttributesQueue) NextAttributes(_ context.Context, safeHead eth.L2BlockRef) (*AttributesWithParent, error) {
	if f.attrs == nil {
		return nil, io.EOF
	}
	return &AttributesWithParent{f.attrs, safeHead, f.islastInSpan}, nil
}

var _ NextAttributesProvider = (*fakeAttributesQueue)(nil)

type noopFinality struct {
}

func (n noopFinality) OnDerivationL1End(ctx context.Context, derivedFrom eth.L1BlockRef) error {
	return nil
}

func (n noopFinality) PostProcessSafeL2(l2Safe eth.L2BlockRef, derivedFrom eth.L1BlockRef) {
}

func (n noopFinality) Reset() {
}

var _ FinalizerHooks = (*noopFinality)(nil)

type fakeAttributesHandler struct {
	attributes *AttributesWithParent
	err        error
}

func (f *fakeAttributesHandler) HasAttributes() bool {
	return f.attributes != nil
}

func (f *fakeAttributesHandler) SetAttributes(attributes *AttributesWithParent) {
	f.attributes = attributes
}

func (f *fakeAttributesHandler) Proceed(ctx context.Context) error {
	if f.err != nil {
		return f.err
	}
	f.attributes = nil
	return io.EOF
}

var _ AttributesHandler = (*fakeAttributesHandler)(nil)

func TestEngineQueue_ResetWhenUnsafeOriginNotCanonical(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	rng := rand.New(rand.NewSource(1234))

	l1Time := uint64(2)
	refA := testutils.RandomBlockRef(rng)

	refB := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refA.Number + 1,
		ParentHash: refA.Hash,
		Time:       refA.Time + l1Time,
	}
	refC := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refB.Number + 1,
		ParentHash: refB.Hash,
		Time:       refB.Time + l1Time,
	}
	refD := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refC.Number + 1,
		ParentHash: refC.Hash,
		Time:       refC.Time + l1Time,
	}
	refE := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refD.Number + 1,
		ParentHash: refD.Hash,
		Time:       refD.Time + l1Time,
	}
	refF := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refE.Number + 1,
		ParentHash: refE.Hash,
		Time:       refE.Time + l1Time,
	}

	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}
	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}
	refB0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA1.Number + 1,
		ParentHash:     refA1.Hash,
		Time:           refA1.Time + cfg.BlockTime,
		L1Origin:       refB.ID(),
		SequenceNumber: 0,
	}
	refB1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refB0.Number + 1,
		ParentHash:     refB0.Hash,
		Time:           refB0.Time + cfg.BlockTime,
		L1Origin:       refB.ID(),
		SequenceNumber: 1,
	}
	refC0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refB1.Number + 1,
		ParentHash:     refB1.Hash,
		Time:           refB1.Time + cfg.BlockTime,
		L1Origin:       refC.ID(),
		SequenceNumber: 0,
	}
	refC1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refC0.Number + 1,
		ParentHash:     refC0.Hash,
		Time:           refC0.Time + cfg.BlockTime,
		L1Origin:       refC.ID(),
		SequenceNumber: 1,
	}
	refD0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refC1.Number + 1,
		ParentHash:     refC1.Hash,
		Time:           refC1.Time + cfg.BlockTime,
		L1Origin:       refD.ID(),
		SequenceNumber: 0,
	}
	refD1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refD0.Number + 1,
		ParentHash:     refD0.Hash,
		Time:           refD0.Time + cfg.BlockTime,
		L1Origin:       refD.ID(),
		SequenceNumber: 1,
	}
	refE0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refD1.Number + 1,
		ParentHash:     refD1.Hash,
		Time:           refD1.Time + cfg.BlockTime,
		L1Origin:       refE.ID(),
		SequenceNumber: 0,
	}
	refE1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refE0.Number + 1,
		ParentHash:     refE0.Hash,
		Time:           refE0.Time + cfg.BlockTime,
		L1Origin:       refE.ID(),
		SequenceNumber: 1,
	}
	refF0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refE1.Number + 1,
		ParentHash:     refE1.Hash,
		Time:           refE1.Time + cfg.BlockTime,
		L1Origin:       refF.ID(),
		SequenceNumber: 0,
	}
	refF1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refF0.Number + 1,
		ParentHash:     refF0.Hash,
		Time:           refF0.Time + cfg.BlockTime,
		L1Origin:       refF.ID(),
		SequenceNumber: 1,
	}
	t.Log("refA", refA.Hash)
	t.Log("refB", refB.Hash)
	t.Log("refC", refC.Hash)
	t.Log("refD", refD.Hash)
	t.Log("refE", refE.Hash)
	t.Log("refF", refF.Hash)
	t.Log("refA0", refA0.Hash)
	t.Log("refA1", refA1.Hash)
	t.Log("refB0", refB0.Hash)
	t.Log("refB1", refB1.Hash)
	t.Log("refC0", refC0.Hash)
	t.Log("refC1", refC1.Hash)
	t.Log("refD0", refD0.Hash)
	t.Log("refD1", refD1.Hash)
	t.Log("refE0", refE0.Hash)
	t.Log("refE1", refE1.Hash)
	t.Log("refF0", refF0.Hash)
	t.Log("refF1", refF1.Hash)

	metrics := &testutils.TestDerivationMetrics{}
	eng := &testutils.MockEngine{}
	// we find the common point to initialize to by comparing the L1 origins in the L2 chain with the L1 chain
	l1F := &testutils.MockL1Source{}

	eng.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	eng.ExpectL2BlockRefByLabel(eth.Safe, refE0, nil)
	eng.ExpectL2BlockRefByLabel(eth.Unsafe, refF1, nil)

	// unsafe
	l1F.ExpectL1BlockRefByNumber(refF.Number, refF, nil)
	eng.ExpectL2BlockRefByHash(refF1.ParentHash, refF0, nil)
	eng.ExpectL2BlockRefByHash(refF0.ParentHash, refE1, nil)

	// meet previous safe, counts 1/2
	l1F.ExpectL1BlockRefByHash(refE.Hash, refE, nil)
	eng.ExpectL2BlockRefByHash(refE1.ParentHash, refE0, nil)
	eng.ExpectL2BlockRefByHash(refE0.ParentHash, refD1, nil)

	// now full seq window, inclusive
	l1F.ExpectL1BlockRefByHash(refD.Hash, refD, nil)
	eng.ExpectL2BlockRefByHash(refD1.ParentHash, refD0, nil)
	eng.ExpectL2BlockRefByHash(refD0.ParentHash, refC1, nil)

	// now one more L1 origin
	l1F.ExpectL1BlockRefByHash(refC.Hash, refC, nil)
	eng.ExpectL2BlockRefByHash(refC1.ParentHash, refC0, nil)
	// parent of that origin will be considered safe
	eng.ExpectL2BlockRefByHash(refC0.ParentHash, refB1, nil)

	// and we fetch the L1 origin of that as starting point for engine queue
	l1F.ExpectL1BlockRefByHash(refB.Hash, refB, nil)
	l1F.ExpectL1BlockRefByHash(refB.Hash, refB, nil)

	// and mock a L1 config for the last L2 block that references the L1 starting point
	eng.ExpectSystemConfigByL2Hash(refB1.Hash, eth.SystemConfig{
		BatcherAddr: common.Address{42},
		Overhead:    [32]byte{123},
		Scalar:      [32]byte{42},
		GasLimit:    20_000_000,
	}, nil)

	prev := &fakeAttributesQueue{origin: refE}

	ec := NewEngineController(eng, logger, metrics, &rollup.Config{}, sync.CLSync)
	eq := NewEngineQueue(logger, cfg, eng, ec, metrics, prev, l1F, &sync.Config{}, safedb.Disabled, noopFinality{}, &fakeAttributesHandler{})
	require.ErrorIs(t, eq.Reset(context.Background(), eth.L1BlockRef{}, eth.SystemConfig{}), io.EOF)

	require.Equal(t, refB1, ec.SafeL2Head(), "L2 reset should go back to sequence window ago: blocks with origin E and D are not safe until we reconcile, C is extra, and B1 is the end we look for")
	require.Equal(t, refB, eq.Origin(), "Expecting to be set back derivation L1 progress to B")
	require.Equal(t, refA1, ec.Finalized(), "A1 is recognized as finalized before we run any steps")

	// First step after reset will do a fork choice update
	eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
		HeadBlockHash:      eq.ec.UnsafeL2Head().Hash,
		SafeBlockHash:      eq.ec.SafeL2Head().Hash,
		FinalizedBlockHash: eq.ec.Finalized().Hash,
	}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
	err := eq.Step(context.Background())
	require.NoError(t, err)

	require.Equal(t, refF.ID(), eq.ec.UnsafeL2Head().L1Origin, "should have refF as unsafe head origin")

	// L1 chain reorgs so new origin is at same slot as refF but on a different fork
	prev.origin = eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refF.Number,
		ParentHash: refE.Hash,
		Time:       refF.Time,
	}
	err = eq.Step(context.Background())
	require.ErrorIs(t, err, ErrReset, "should reset pipeline due to mismatched origin")

	l1F.AssertExpectations(t)
	eng.AssertExpectations(t)
}

func TestVerifyNewL1Origin(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	rng := rand.New(rand.NewSource(1234))

	l1Time := uint64(2)
	refA := testutils.RandomBlockRef(rng)

	refB := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refA.Number + 1,
		ParentHash: refA.Hash,
		Time:       refA.Time + l1Time,
	}
	refC := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refB.Number + 1,
		ParentHash: refB.Hash,
		Time:       refB.Time + l1Time,
	}
	refD := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refC.Number + 1,
		ParentHash: refC.Hash,
		Time:       refC.Time + l1Time,
	}
	refE := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refD.Number + 1,
		ParentHash: refD.Hash,
		Time:       refD.Time + l1Time,
	}
	refF := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refE.Number + 1,
		ParentHash: refE.Hash,
		Time:       refE.Time + l1Time,
	}
	refG := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refF.Number + 1,
		ParentHash: refF.Hash,
		Time:       refF.Time + l1Time,
	}
	refH := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refG.Number + 1,
		ParentHash: refG.Hash,
		Time:       refG.Time + l1Time,
	}

	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}
	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}
	refB0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA1.Number + 1,
		ParentHash:     refA1.Hash,
		Time:           refA1.Time + cfg.BlockTime,
		L1Origin:       refB.ID(),
		SequenceNumber: 0,
	}
	refB1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refB0.Number + 1,
		ParentHash:     refB0.Hash,
		Time:           refB0.Time + cfg.BlockTime,
		L1Origin:       refB.ID(),
		SequenceNumber: 1,
	}
	refC0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refB1.Number + 1,
		ParentHash:     refB1.Hash,
		Time:           refB1.Time + cfg.BlockTime,
		L1Origin:       refC.ID(),
		SequenceNumber: 0,
	}
	refC1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refC0.Number + 1,
		ParentHash:     refC0.Hash,
		Time:           refC0.Time + cfg.BlockTime,
		L1Origin:       refC.ID(),
		SequenceNumber: 1,
	}
	refD0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refC1.Number + 1,
		ParentHash:     refC1.Hash,
		Time:           refC1.Time + cfg.BlockTime,
		L1Origin:       refD.ID(),
		SequenceNumber: 0,
	}
	refD1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refD0.Number + 1,
		ParentHash:     refD0.Hash,
		Time:           refD0.Time + cfg.BlockTime,
		L1Origin:       refD.ID(),
		SequenceNumber: 1,
	}
	refE0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refD1.Number + 1,
		ParentHash:     refD1.Hash,
		Time:           refD1.Time + cfg.BlockTime,
		L1Origin:       refE.ID(),
		SequenceNumber: 0,
	}
	refE1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refE0.Number + 1,
		ParentHash:     refE0.Hash,
		Time:           refE0.Time + cfg.BlockTime,
		L1Origin:       refE.ID(),
		SequenceNumber: 1,
	}
	refF0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refE1.Number + 1,
		ParentHash:     refE1.Hash,
		Time:           refE1.Time + cfg.BlockTime,
		L1Origin:       refF.ID(),
		SequenceNumber: 0,
	}
	refF1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refF0.Number + 1,
		ParentHash:     refF0.Hash,
		Time:           refF0.Time + cfg.BlockTime,
		L1Origin:       refF.ID(),
		SequenceNumber: 1,
	}
	t.Log("refA", refA.Hash)
	t.Log("refB", refB.Hash)
	t.Log("refC", refC.Hash)
	t.Log("refD", refD.Hash)
	t.Log("refE", refE.Hash)
	t.Log("refF", refF.Hash)
	t.Log("refG", refG.Hash)
	t.Log("refH", refH.Hash)
	t.Log("refA0", refA0.Hash)
	t.Log("refA1", refA1.Hash)
	t.Log("refB0", refB0.Hash)
	t.Log("refB1", refB1.Hash)
	t.Log("refC0", refC0.Hash)
	t.Log("refC1", refC1.Hash)
	t.Log("refD0", refD0.Hash)
	t.Log("refD1", refD1.Hash)
	t.Log("refE0", refE0.Hash)
	t.Log("refE1", refE1.Hash)
	t.Log("refF0", refF0.Hash)
	t.Log("refF1", refF1.Hash)

	metrics := &testutils.TestDerivationMetrics{}

	tests := []struct {
		name                string
		newOrigin           eth.L1BlockRef
		expectReset         bool
		expectedFetchBlocks map[uint64]eth.L1BlockRef
	}{
		{
			name:        "L1OriginBeforeUnsafeOrigin",
			newOrigin:   refD,
			expectReset: false,
		},
		{
			name:        "Matching",
			newOrigin:   refF,
			expectReset: false,
		},
		{
			name: "BlockNumberEqualDifferentHash",
			newOrigin: eth.L1BlockRef{
				Hash:       testutils.RandomHash(rng),
				Number:     refF.Number,
				ParentHash: refE.Hash,
				Time:       refF.Time,
			},
			expectReset: true,
		},
		{
			name:        "UnsafeIsParent",
			newOrigin:   refG,
			expectReset: false,
		},
		{
			name: "UnsafeIsParentNumberDifferentHash",
			newOrigin: eth.L1BlockRef{
				Hash:       testutils.RandomHash(rng),
				Number:     refG.Number,
				ParentHash: testutils.RandomHash(rng),
				Time:       refG.Time,
			},
			expectReset: true,
		},
		{
			name:        "UnsafeIsOlderCanonical",
			newOrigin:   refH,
			expectReset: false,
			expectedFetchBlocks: map[uint64]eth.L1BlockRef{
				refF.Number: refF,
			},
		},
		{
			name: "UnsafeIsOlderNonCanonical",
			newOrigin: eth.L1BlockRef{
				Hash:       testutils.RandomHash(rng),
				Number:     refH.Number,
				ParentHash: testutils.RandomHash(rng),
				Time:       refH.Time,
			},
			expectReset: true,
			expectedFetchBlocks: map[uint64]eth.L1BlockRef{
				// Second look up gets a different block in F's block number due to a reorg
				refF.Number: {
					Hash:       testutils.RandomHash(rng),
					Number:     refF.Number,
					ParentHash: refE.Hash,
					Time:       refF.Time,
				},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			eng := &testutils.MockEngine{}
			// we find the common point to initialize to by comparing the L1 origins in the L2 chain with the L1 chain
			l1F := &testutils.MockL1Source{}

			eng.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
			eng.ExpectL2BlockRefByLabel(eth.Safe, refE0, nil)
			eng.ExpectL2BlockRefByLabel(eth.Unsafe, refF1, nil)

			// unsafe
			l1F.ExpectL1BlockRefByNumber(refF.Number, refF, nil)
			eng.ExpectL2BlockRefByHash(refF1.ParentHash, refF0, nil)
			eng.ExpectL2BlockRefByHash(refF0.ParentHash, refE1, nil)

			for blockNum, block := range test.expectedFetchBlocks {
				l1F.ExpectL1BlockRefByNumber(blockNum, block, nil)
			}

			// meet previous safe, counts 1/2
			l1F.ExpectL1BlockRefByHash(refE.Hash, refE, nil)
			eng.ExpectL2BlockRefByHash(refE1.ParentHash, refE0, nil)
			eng.ExpectL2BlockRefByHash(refE0.ParentHash, refD1, nil)

			// now full seq window, inclusive
			l1F.ExpectL1BlockRefByHash(refD.Hash, refD, nil)
			eng.ExpectL2BlockRefByHash(refD1.ParentHash, refD0, nil)
			eng.ExpectL2BlockRefByHash(refD0.ParentHash, refC1, nil)

			// now one more L1 origin
			l1F.ExpectL1BlockRefByHash(refC.Hash, refC, nil)
			eng.ExpectL2BlockRefByHash(refC1.ParentHash, refC0, nil)
			// parent of that origin will be considered safe
			eng.ExpectL2BlockRefByHash(refC0.ParentHash, refB1, nil)

			// and we fetch the L1 origin of that as starting point for engine queue
			l1F.ExpectL1BlockRefByHash(refB.Hash, refB, nil)
			l1F.ExpectL1BlockRefByHash(refB.Hash, refB, nil)

			// and mock a L1 config for the last L2 block that references the L1 starting point
			eng.ExpectSystemConfigByL2Hash(refB1.Hash, eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{123},
				Scalar:      [32]byte{42},
				GasLimit:    20_000_000,
			}, nil)

			prev := &fakeAttributesQueue{origin: refE}
			ec := NewEngineController(eng, logger, metrics, &rollup.Config{}, sync.CLSync)
			eq := NewEngineQueue(logger, cfg, eng, ec, metrics, prev, l1F, &sync.Config{}, safedb.Disabled, noopFinality{}, &fakeAttributesHandler{})
			require.ErrorIs(t, eq.Reset(context.Background(), eth.L1BlockRef{}, eth.SystemConfig{}), io.EOF)

			require.Equal(t, refB1, ec.SafeL2Head(), "L2 reset should go back to sequence window ago: blocks with origin E and D are not safe until we reconcile, C is extra, and B1 is the end we look for")
			require.Equal(t, refB, eq.Origin(), "Expecting to be set back derivation L1 progress to B")
			require.Equal(t, refA1, ec.Finalized(), "A1 is recognized as finalized before we run any steps")

			// First step after reset will do a fork choice update
			eng.ExpectForkchoiceUpdate(&eth.ForkchoiceState{
				HeadBlockHash:      eq.ec.UnsafeL2Head().Hash,
				SafeBlockHash:      eq.ec.SafeL2Head().Hash,
				FinalizedBlockHash: eq.ec.Finalized().Hash,
			}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
			err := eq.Step(context.Background())
			require.NoError(t, err)

			require.Equal(t, refF.ID(), eq.ec.UnsafeL2Head().L1Origin, "should have refF as unsafe head origin")

			// L1 chain reorgs so new origin is at same slot as refF but on a different fork
			prev.origin = test.newOrigin
			err = eq.Step(context.Background())
			if test.expectReset {
				require.ErrorIs(t, err, ErrReset, "should reset pipeline due to mismatched origin")
			} else {
				require.ErrorIs(t, err, io.EOF, "should not reset pipeline")
			}

			l1F.AssertExpectations(t)
			eng.AssertExpectations(t)
		})
	}
}

func TestBlockBuildingRace(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	eng := &testutils.MockEngine{}

	rng := rand.New(rand.NewSource(1234))

	refA := testutils.RandomBlockRef(rng)
	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{123},
				Scalar:      [32]byte{42},
				GasLimit:    20_000_000,
			},
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}
	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}

	l1F := &testutils.MockL1Source{}

	eng.ExpectL2BlockRefByLabel(eth.Finalized, refA0, nil)
	eng.ExpectL2BlockRefByLabel(eth.Safe, refA0, nil)
	eng.ExpectL2BlockRefByLabel(eth.Unsafe, refA0, nil)
	l1F.ExpectL1BlockRefByNumber(refA.Number, refA, nil)
	l1F.ExpectL1BlockRefByHash(refA.Hash, refA, nil)
	l1F.ExpectL1BlockRefByHash(refA.Hash, refA, nil)

	eng.ExpectSystemConfigByL2Hash(refA0.Hash, cfg.Genesis.SystemConfig, nil)

	metrics := &testutils.TestDerivationMetrics{}

	gasLimit := eth.Uint64Quantity(20_000_000)
	attrs := &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(refA1.Time),
		PrevRandao:            eth.Bytes32{},
		SuggestedFeeRecipient: common.Address{},
		Transactions:          nil,
		NoTxPool:              false,
		GasLimit:              &gasLimit,
	}

	prev := &fakeAttributesQueue{origin: refA, attrs: attrs, islastInSpan: true}
	ec := NewEngineController(eng, logger, metrics, &rollup.Config{}, sync.CLSync)
	attribHandler := &fakeAttributesHandler{}
	eq := NewEngineQueue(logger, cfg, eng, ec, metrics, prev, l1F, &sync.Config{}, safedb.Disabled, noopFinality{}, attribHandler)
	require.ErrorIs(t, eq.Reset(context.Background(), eth.L1BlockRef{}, eth.SystemConfig{}), io.EOF)

	id := eth.PayloadID{0xff}

	preFc := &eth.ForkchoiceState{
		HeadBlockHash:      refA0.Hash,
		SafeBlockHash:      refA0.Hash,
		FinalizedBlockHash: refA0.Hash,
	}
	preFcRes := &eth.ForkchoiceUpdatedResult{
		PayloadStatus: eth.PayloadStatusV1{
			Status:          eth.ExecutionValid,
			LatestValidHash: &refA0.Hash,
			ValidationError: nil,
		},
		PayloadID: &id,
	}

	// Expect initial forkchoice update
	eng.ExpectForkchoiceUpdate(preFc, nil, preFcRes, nil)
	require.NoError(t, eq.Step(context.Background()), "clean forkchoice state after reset")

	// Expect initial building update, to process the attributes we queued up. Attributes get in
	require.ErrorIs(t, eq.Step(context.Background()), NotEnoughData, "queue up attributes")
	require.True(t, eq.attributesHandler.HasAttributes())

	// Don't let the payload be confirmed straight away
	// The job will be not be cancelled, the untyped error is a temporary error
	mockErr := fmt.Errorf("mock error")
	attribHandler.err = mockErr
	require.ErrorIs(t, eq.Step(context.Background()), mockErr, "expecting to fail to process attributes")
	require.True(t, eq.attributesHandler.HasAttributes(), "still have attributes")

	// Now allow the building to complete
	attribHandler.err = nil

	require.ErrorIs(t, eq.Step(context.Background()), NotEnoughData, "next attributes")

	l1F.AssertExpectations(t)
	eng.AssertExpectations(t)
}

func TestResetLoop(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	eng := &testutils.MockEngine{}
	l1F := &testutils.MockL1Source{}

	rng := rand.New(rand.NewSource(1234))

	refA := testutils.RandomBlockRef(rng)
	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}
	gasLimit := eth.Uint64Quantity(20_000_000)
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{123},
				Scalar:      [32]byte{42},
				GasLimit:    20_000_000,
			},
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}
	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}
	refA2 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA1.Number + 1,
		ParentHash:     refA1.Hash,
		Time:           refA1.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 2,
	}

	attrs := &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(refA2.Time),
		PrevRandao:            eth.Bytes32{},
		SuggestedFeeRecipient: common.Address{},
		Transactions:          nil,
		NoTxPool:              false,
		GasLimit:              &gasLimit,
	}

	eng.ExpectL2BlockRefByLabel(eth.Finalized, refA0, nil)
	eng.ExpectL2BlockRefByLabel(eth.Safe, refA1, nil)
	eng.ExpectL2BlockRefByLabel(eth.Unsafe, refA2, nil)
	eng.ExpectL2BlockRefByHash(refA1.Hash, refA1, nil)
	eng.ExpectL2BlockRefByHash(refA0.Hash, refA0, nil)
	eng.ExpectSystemConfigByL2Hash(refA0.Hash, cfg.Genesis.SystemConfig, nil)
	l1F.ExpectL1BlockRefByNumber(refA.Number, refA, nil)
	l1F.ExpectL1BlockRefByHash(refA.Hash, refA, nil)
	l1F.ExpectL1BlockRefByHash(refA.Hash, refA, nil)

	prev := &fakeAttributesQueue{origin: refA, attrs: attrs, islastInSpan: true}

	ec := NewEngineController(eng, logger, metrics.NoopMetrics, &rollup.Config{}, sync.CLSync)
	eq := NewEngineQueue(logger, cfg, eng, ec, metrics.NoopMetrics, prev, l1F, &sync.Config{}, safedb.Disabled, noopFinality{}, &fakeAttributesHandler{})
	eq.ec.SetUnsafeHead(refA2)
	eq.ec.SetSafeHead(refA1)
	eq.ec.SetFinalizedHead(refA0)

	// Queue up the safe attributes
	// Expect a FCU after during the first step
	preFc := &eth.ForkchoiceState{
		HeadBlockHash:      refA2.Hash,
		SafeBlockHash:      refA1.Hash,
		FinalizedBlockHash: refA0.Hash,
	}
	eng.ExpectForkchoiceUpdate(preFc, nil, nil, nil)
	require.False(t, eq.attributesHandler.HasAttributes())
	require.ErrorIs(t, eq.Step(context.Background()), nil)
	require.ErrorIs(t, eq.Step(context.Background()), NotEnoughData)
	require.True(t, eq.attributesHandler.HasAttributes())

	// Perform the reset
	require.ErrorIs(t, eq.Reset(context.Background(), eth.L1BlockRef{}, eth.SystemConfig{}), io.EOF)

	// Expect a FCU after the reset
	postFc := &eth.ForkchoiceState{
		HeadBlockHash:      refA2.Hash,
		SafeBlockHash:      refA0.Hash,
		FinalizedBlockHash: refA0.Hash,
	}
	eng.ExpectForkchoiceUpdate(postFc, nil, nil, nil)
	require.NoError(t, eq.Step(context.Background()), "clean forkchoice state after reset")

	// Crux of the test. Should be in a valid state after the reset.
	require.ErrorIs(t, eq.Step(context.Background()), NotEnoughData, "Should be able to step after a reset")

	l1F.AssertExpectations(t)
	eng.AssertExpectations(t)
}
