package safedb

import (
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

type testMetrics struct {
	head    uint64
	present bool
	errors  map[string]int
}

func (m *testMetrics) RecordSafeDBLastL2(number uint64)        { m.head = number; m.present = true }
func (m *testMetrics) ClearSafeDBLastL2()                      { m.present = false }
func (m *testMetrics) RecordSafeDBWriteError(operation string) { m.errors[operation]++ }

func TestPersistedHeadMetric(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	metrics := &testMetrics{errors: make(map[string]int)}
	db, err := NewSafeDBWithMetrics(logger, dir, metrics)
	require.NoError(t, err)
	require.False(t, metrics.present)
	for _, entry := range []struct{ l1, l2 uint64 }{{100, 20}, {150, 30}, {50, 10}} {
		require.NoError(t, db.SafeHeadUpdated(eth.L2BlockRef{Number: entry.l2}, eth.BlockID{Number: entry.l1}))
	}
	// An out-of-order key does not replace the highest-L1 entry.
	require.True(t, metrics.present)
	require.Equal(t, uint64(30), metrics.head)
	require.NoError(t, db.SafeHeadReset(eth.L2BlockRef{Number: 25, L1Origin: eth.BlockID{Number: 100}}))
	require.Equal(t, uint64(25), metrics.head)
	require.NoError(t, db.Close())
	metrics.present = false
	db, err = NewSafeDBWithMetrics(logger, dir, metrics)
	require.NoError(t, err)
	defer db.Close()
	require.True(t, metrics.present)
	require.Equal(t, uint64(25), metrics.head)
	// A reset beyond recorded history leaves the persisted tip unchanged.
	require.NoError(t, db.SafeHeadReset(eth.L2BlockRef{Number: 99, L1Origin: eth.BlockID{Number: 200}}))
	require.Equal(t, uint64(25), metrics.head)
	// Reset before the first entry removes history and its metric.
	require.NoError(t, db.SafeHeadReset(eth.L2BlockRef{Number: 5}))
	require.False(t, metrics.present)
	require.Empty(t, metrics.errors)
	// Genesis at zero is a real recorded head, unlike an empty database.
	require.NoError(t, db.SafeHeadUpdated(eth.L2BlockRef{}, eth.BlockID{}))
	require.True(t, metrics.present)
	require.Zero(t, metrics.head)
}

func TestFailedCommitsLeaveHeadMetricUnchanged(t *testing.T) {
	for _, operation := range []string{"update", "reset"} {
		t.Run(operation, func(t *testing.T) {
			store, err := pebble.Open(t.TempDir(), &pebble.Options{DisableWAL: true})
			require.NoError(t, err)
			metrics := &testMetrics{errors: make(map[string]int)}
			db := &SafeDB{db: store, log: testlog.Logger(t, log.LevelInfo), writeOpts: pebble.Sync, metrics: metrics}
			defer db.Close()
			for _, entry := range []struct{ l1, l2 uint64 }{{100, 10}, {200, 20}} {
				require.NoError(t, store.Set(safeByL1BlockNumKey.Of(entry.l1), safeByL1BlockNumValue(eth.BlockID{Number: entry.l1}, eth.BlockID{Number: entry.l2}), pebble.NoSync))
			}
			db.recordHead()
			if operation == "update" {
				err = db.SafeHeadUpdated(eth.L2BlockRef{Number: 30}, eth.BlockID{Number: 300})
			} else {
				err = db.SafeHeadReset(eth.L2BlockRef{Number: 15, L1Origin: eth.BlockID{Number: 100}})
			}
			require.Error(t, err) // A synchronous commit cannot succeed with WAL disabled.
			require.Equal(t, map[string]int{operation: 1}, metrics.errors)
			require.True(t, metrics.present)
			require.Equal(t, uint64(20), metrics.head)
		})
	}
}

func TestUnreadableHeadClearsMetric(t *testing.T) {
	metrics := &testMetrics{errors: make(map[string]int)}
	db, err := NewSafeDBWithMetrics(testlog.Logger(t, log.LevelInfo), t.TempDir(), metrics)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.SafeHeadUpdated(eth.L2BlockRef{Number: 10}, eth.BlockID{Number: 100}))
	require.True(t, metrics.present)
	require.NoError(t, db.db.Set(safeByL1BlockNumKey.Of(100), []byte("invalid"), pebble.Sync))
	db.recordHead()
	require.False(t, metrics.present)
	require.Empty(t, metrics.errors) // A metric read failure is not a failed write.
}
