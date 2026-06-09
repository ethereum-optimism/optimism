package monitor

import (
	"context"
	"errors"
	"math/big"
	"testing"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockSupernodeClient struct {
	status  eth.SuperNodeSyncStatusResponse
	syncErr error
}

func (m *mockSupernodeClient) SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	return m.status, m.syncErr
}
func (m *mockSupernodeClient) Close() {}

// mockEL returns a fixed canonical block info (or error) for InfoByNumber.
type mockEL struct {
	info eth.BlockInfo
	err  error
}

func (m *mockEL) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	return m.info, m.err
}

// canonicalEL builds an EL-client map whose chain reports the given hash as
// canonical at the given block number.
func canonicalEL(chain eth.ChainID, number uint64, hash common.Hash) map[eth.ChainID]CanonicalBlockSource {
	info := eth.HeaderBlockInfoTrusted(hash, &types.Header{Number: new(big.Int).SetUint64(number)})
	return map[eth.ChainID]CanonicalBlockSource{chain: &mockEL{info: info}}
}

func TestSupernodeObserverCrossSafetyViolation(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	hash := common.HexToHash("0xaa")
	st := eth.SuperNodeSyncStatusResponse{
		Chains: map[eth.ChainID]eth.SyncStatus{
			execChain: {SafeL2: eth.L2BlockRef{Number: 250}},
		},
	}
	badJob := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1)},
		executingChain: execChain,
		executingBlock: eth.BlockID{Number: 200, Hash: hash}, // <= cross-safe head 250 => violation
	}
	badJob.UpdateStatus(jobStatusInvalid)

	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, canonicalEL(execChain, 200, hash), mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{badJob.ID(): badJob})
	require.Len(t, mm.actualCrossSafetyViolations, 1)

	// A second pass must not double-count the same violating job.
	obs.Observe(context.Background(), map[JobID]*Job{badJob.ID(): badJob})
	require.Len(t, mm.actualCrossSafetyViolations, 1)
}

func TestSupernodeObserverNoViolationAboveHead(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	st := eth.SuperNodeSyncStatusResponse{
		Chains: map[eth.ChainID]eth.SyncStatus{
			execChain: {SafeL2: eth.L2BlockRef{Number: 100}},
		},
	}
	badJob := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1)},
		executingChain: execChain,
		executingBlock: eth.BlockID{Number: 200, Hash: common.HexToHash("0xaa")}, // > cross-safe head 100
	}
	badJob.UpdateStatus(jobStatusInvalid)

	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, map[eth.ChainID]CanonicalBlockSource{}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{badJob.ID(): badJob})
	require.Empty(t, mm.actualCrossSafetyViolations)
}

// A bad job whose executing block was reorged out (the canonical block at that
// height now has a different hash) must not be flagged: the supernode validated
// the replacement block, not the orphaned one.
func TestSupernodeObserverNoViolationOnReorg(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	st := eth.SuperNodeSyncStatusResponse{
		Chains: map[eth.ChainID]eth.SyncStatus{
			execChain: {SafeL2: eth.L2BlockRef{Number: 250}},
		},
	}
	badJob := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1)},
		executingChain: execChain,
		executingBlock: eth.BlockID{Number: 200, Hash: common.HexToHash("0xaa")}, // orphaned hash
	}
	badJob.UpdateStatus(jobStatusInvalid)

	mm := &mockMetrics{}
	// Canonical block at height 200 now has a different hash than the job's block.
	els := canonicalEL(execChain, 200, common.HexToHash("0xbb"))
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, els, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{badJob.ID(): badJob})
	require.Empty(t, mm.actualCrossSafetyViolations)
}

func TestSupernodeObserverDown(t *testing.T) {
	mm := &mockMetrics{lastSupernodeUp: true}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{syncErr: errors.New("down")}, nil, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{})
	require.False(t, mm.lastSupernodeUp)
}

func TestSupernodeObserverRecordsHeads(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	st := eth.SuperNodeSyncStatusResponse{
		Chains: map[eth.ChainID]eth.SyncStatus{
			execChain: {SafeL2: eth.L2BlockRef{Number: 250}, FinalizedL2: eth.L2BlockRef{Number: 100}},
		},
	}
	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, nil, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{})
	require.True(t, mm.lastSupernodeUp)
	// one cross_safe + one finalized head per chain
	require.Len(t, mm.actualSupernodeSafeHeads, 2)
}
