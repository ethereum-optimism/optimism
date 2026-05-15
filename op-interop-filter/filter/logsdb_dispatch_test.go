package filter

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// fakeLogsDB satisfies the LogsDB interface and lets tests inject specific
// sentinels into write-path methods that the real DB cannot produce under
// correct ingester control flow (parent-hash and block-number pre-checks in
// writeFetchedBlock pre-empt them).
type fakeLogsDB struct {
	addLogErr    error
	sealBlockErr error

	hasSealed bool
	latest    eth.BlockID
}

func (f *fakeLogsDB) Close() error { return nil }
func (f *fakeLogsDB) Contains(types.ContainsQuery) (types.BlockSeal, error) {
	return types.BlockSeal{}, nil
}
func (f *fakeLogsDB) FindSealedBlock(uint64) (types.BlockSeal, error) { return types.BlockSeal{}, nil }
func (f *fakeLogsDB) FirstSealedBlock() (types.BlockSeal, error)      { return types.BlockSeal{}, nil }
func (f *fakeLogsDB) Rewind(eth.BlockID) error                        { return nil }
func (f *fakeLogsDB) OpenBlock(uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error) {
	return eth.BlockRef{}, 0, nil, nil
}
func (f *fakeLogsDB) LatestSealedBlock() (eth.BlockID, bool) { return f.latest, f.hasSealed }

func (f *fakeLogsDB) AddLog(common.Hash, eth.BlockID, uint32, *types.ExecutingMessage) error {
	return f.addLogErr
}

func (f *fakeLogsDB) SealBlock(common.Hash, eth.BlockID, uint64) error {
	return f.sealBlockErr
}

func newIngesterWithFakeDB(t *testing.T, db *fakeLogsDB) *LogsDBChainIngester {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &LogsDBChainIngester{
		log:              testlog.Logger(t, log.LevelError),
		metrics:          metrics.NoopMetrics,
		chainID:          eth.ChainIDFromUInt64(901),
		logsDB:           db,
		rollupCfg:        testRollupConfig(901, 0, 1000),
		fetchConcurrency: 1,
		ctx:              ctx,
		cancel:           cancel,
	}
}

func fetchedBlock(num, ts uint64, parent common.Hash) blockFetch {
	info := makeBlockInfo(num, ts, parent)
	return blockFetch{
		blockNum:  num,
		blockInfo: info,
		receipts:  gethTypes.Receipts{{TxHash: common.Hash{byte(num)}, Logs: []*gethTypes.Log{plainLog(0, num)}}},
	}
}

// TestWriteFetchedBlock_WriteErrorDispatch pins the sentinel-dispatch contract
// for paths the integration suite cannot reach: AddLog/SealBlock returning
// ErrConflict / ErrDataCorruption / ErrInvalidLog. These can't be produced
// by a real on-disk logsdb through op-interop-filter's normal control flow
// because writeFetchedBlock pre-checks block number and parent hash before
// calling either method.
func TestWriteFetchedBlock_WriteErrorDispatch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		setup      func(*fakeLogsDB)
		wantReason IngesterErrorReason
	}{
		{
			name:       "AddLog_ErrConflict",
			setup:      func(f *fakeLogsDB) { f.addLogErr = fmt.Errorf("add: %w", types.ErrConflict) },
			wantReason: ErrorConflict,
		},
		{
			name:       "AddLog_ErrDataCorruption",
			setup:      func(f *fakeLogsDB) { f.addLogErr = fmt.Errorf("add: %w", types.ErrDataCorruption) },
			wantReason: ErrorDataCorruption,
		},
		{
			name:       "SealBlock_ErrConflict",
			setup:      func(f *fakeLogsDB) { f.sealBlockErr = fmt.Errorf("seal: %w", types.ErrConflict) },
			wantReason: ErrorConflict,
		},
		{
			name:       "SealBlock_ErrDataCorruption",
			setup:      func(f *fakeLogsDB) { f.sealBlockErr = fmt.Errorf("seal: %w", types.ErrDataCorruption) },
			wantReason: ErrorDataCorruption,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db := &fakeLogsDB{}
			tc.setup(db)
			ing := newIngesterWithFakeDB(t, db)

			err := ing.writeFetchedBlock(fetchedBlock(100, 1200, common.Hash{}))
			require.Error(t, err)
			ingErr := ing.Error()
			require.NotNil(t, ingErr)
			require.Equal(t, tc.wantReason, ingErr.Reason)
		})
	}
}

// TestWriteFetchedBlock_WriteErrorPassthrough pins the dispatch *negative* —
// sentinels the ingester does not classify (ErrFuture, ErrSkipped,
// ErrOutOfOrder, plain errors) must bubble up wrapped without setting the
// ingester into a permanent error state. This matches today's behaviour and
// is the regression guard for any future dispatch refactor.
func TestWriteFetchedBlock_WriteErrorPassthrough(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{"AddLog_ErrFuture", fmt.Errorf("add: %w", types.ErrFuture)},
		{"AddLog_ErrSkipped", fmt.Errorf("add: %w", types.ErrSkipped)},
		{"AddLog_ErrOutOfOrder", fmt.Errorf("add: %w", types.ErrOutOfOrder)},
		{"AddLog_Generic", errors.New("transient i/o")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db := &fakeLogsDB{addLogErr: tc.err}
			ing := newIngesterWithFakeDB(t, db)

			err := ing.writeFetchedBlock(fetchedBlock(100, 1200, common.Hash{}))
			require.Error(t, err)
			require.Nil(t, ing.Error(),
				"non-classified sentinel must not set IngesterError")
		})
	}
}
