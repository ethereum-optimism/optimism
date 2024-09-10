package interop

import (
	"context"
	"math/big"
	"math/rand" // nosemgrep
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestInteropDeriver(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	l2Source := &testutils.MockL2Client{}
	emitter := &testutils.MockEmitter{}
	interopBackend := &testutils.MockInteropBackend{}
	cfg := &rollup.Config{
		InteropTime: new(uint64),
		L2ChainID:   big.NewInt(42),
	}
	chainID := supervisortypes.ChainIDFromBig(cfg.L2ChainID)
	interopDeriver := NewInteropDeriver(logger, cfg, context.Background(), interopBackend, l2Source)
	interopDeriver.AttachEmitter(emitter)
	rng := rand.New(rand.NewSource(123))

	t.Run("unsafe blocks trigger cross-unsafe check attempts", func(t *testing.T) {
		emitter.ExpectOnce(engine.RequestCrossUnsafeEvent{})
		interopDeriver.OnEvent(engine.UnsafeUpdateEvent{
			Ref: testutils.RandomL2BlockRef(rng),
		})
		emitter.AssertExpectations(t)
	})
	t.Run("establish cross-unsafe", func(t *testing.T) {
		crossUnsafe := testutils.RandomL2BlockRef(rng)
		firstLocalUnsafe := testutils.NextRandomL2Ref(rng, 2, crossUnsafe, crossUnsafe.L1Origin)
		lastLocalUnsafe := testutils.NextRandomL2Ref(rng, 2, firstLocalUnsafe, firstLocalUnsafe.L1Origin)
		interopBackend.ExpectCheckBlock(
			chainID, firstLocalUnsafe.Number, supervisortypes.CrossUnsafe, nil)
		emitter.ExpectOnce(engine.PromoteCrossUnsafeEvent{
			Ref: firstLocalUnsafe,
		})
		l2Source.ExpectL2BlockRefByNumber(firstLocalUnsafe.Number, firstLocalUnsafe, nil)
		interopDeriver.OnEvent(engine.CrossUnsafeUpdateEvent{
			CrossUnsafe: crossUnsafe,
			LocalUnsafe: lastLocalUnsafe,
		})
		interopBackend.AssertExpectations(t)
		emitter.AssertExpectations(t)
		l2Source.AssertExpectations(t)
	})
	t.Run("deny cross-unsafe", func(t *testing.T) {
		crossUnsafe := testutils.RandomL2BlockRef(rng)
		firstLocalUnsafe := testutils.NextRandomL2Ref(rng, 2, crossUnsafe, crossUnsafe.L1Origin)
		lastLocalUnsafe := testutils.NextRandomL2Ref(rng, 2, firstLocalUnsafe, firstLocalUnsafe.L1Origin)
		interopBackend.ExpectCheckBlock(
			chainID, firstLocalUnsafe.Number, supervisortypes.Unsafe, nil)
		l2Source.ExpectL2BlockRefByNumber(firstLocalUnsafe.Number, firstLocalUnsafe, nil)
		interopDeriver.OnEvent(engine.CrossUnsafeUpdateEvent{
			CrossUnsafe: crossUnsafe,
			LocalUnsafe: lastLocalUnsafe,
		})
		interopBackend.AssertExpectations(t)
		// no cross-unsafe promote event is expected
		emitter.AssertExpectations(t)
		l2Source.AssertExpectations(t)
	})
	t.Run("register local-safe", func(t *testing.T) {
		derivedFrom := testutils.RandomBlockRef(rng)
		localSafe := testutils.RandomL2BlockRef(rng)
		emitter.ExpectOnce(engine.RequestCrossSafeEvent{})
		interopDeriver.OnEvent(engine.LocalSafeUpdateEvent{
			Ref:         localSafe,
			DerivedFrom: derivedFrom,
		})
		require.Contains(t, interopDeriver.derivedFrom, localSafe.Hash)
		require.Equal(t, derivedFrom, interopDeriver.derivedFrom[localSafe.Hash])
		emitter.AssertExpectations(t)
	})
	t.Run("establish cross-safe", func(t *testing.T) {
		derivedFrom := testutils.RandomBlockRef(rng)
		crossSafe := testutils.RandomL2BlockRef(rng)
		firstLocalSafe := testutils.NextRandomL2Ref(rng, 2, crossSafe, crossSafe.L1Origin)
		lastLocalSafe := testutils.NextRandomL2Ref(rng, 2, firstLocalSafe, firstLocalSafe.L1Origin)
		emitter.ExpectOnce(engine.RequestCrossSafeEvent{})
		// The local safe block must be known, for the derived-from mapping to work
		interopDeriver.OnEvent(engine.LocalSafeUpdateEvent{
			Ref:         firstLocalSafe,
			DerivedFrom: derivedFrom,
		})
		interopBackend.ExpectCheckBlock(
			chainID, firstLocalSafe.Number, supervisortypes.CrossSafe, nil)
		emitter.ExpectOnce(engine.PromoteSafeEvent{
			Ref:         firstLocalSafe,
			DerivedFrom: derivedFrom,
		})
		l2Source.ExpectL2BlockRefByNumber(firstLocalSafe.Number, firstLocalSafe, nil)
		interopDeriver.OnEvent(engine.CrossSafeUpdateEvent{
			CrossSafe: crossSafe,
			LocalSafe: lastLocalSafe,
		})
		interopBackend.AssertExpectations(t)
		emitter.AssertExpectations(t)
		l2Source.AssertExpectations(t)
	})
	t.Run("deny cross-safe", func(t *testing.T) {
		derivedFrom := testutils.RandomBlockRef(rng)
		crossSafe := testutils.RandomL2BlockRef(rng)
		firstLocalSafe := testutils.NextRandomL2Ref(rng, 2, crossSafe, crossSafe.L1Origin)
		lastLocalSafe := testutils.NextRandomL2Ref(rng, 2, firstLocalSafe, firstLocalSafe.L1Origin)
		emitter.ExpectOnce(engine.RequestCrossSafeEvent{})
		// The local safe block must be known, for the derived-from mapping to work
		interopDeriver.OnEvent(engine.LocalSafeUpdateEvent{
			Ref:         firstLocalSafe,
			DerivedFrom: derivedFrom,
		})
		interopBackend.ExpectCheckBlock(
			chainID, firstLocalSafe.Number, supervisortypes.Safe, nil)
		l2Source.ExpectL2BlockRefByNumber(firstLocalSafe.Number, firstLocalSafe, nil)
		interopDeriver.OnEvent(engine.CrossSafeUpdateEvent{
			CrossSafe: crossSafe,
			LocalSafe: lastLocalSafe,
		})
		interopBackend.AssertExpectations(t)
		// no cross-safe promote event is expected
		emitter.AssertExpectations(t)
		l2Source.AssertExpectations(t)
	})
}
