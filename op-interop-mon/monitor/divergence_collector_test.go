package monitor

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// fakeReplica is a scripted ReplicaClient for divergence-collector tests.
type fakeReplica struct {
	endpoint  string
	finalized uint64
	syncErr   error
	// rootAt returns the super root the replica reports at a timestamp; a zero
	// Bytes32 with present=false models "no data at ts".
	rootAt   map[uint64]eth.Bytes32
	chainIDs []eth.ChainID // dependency set reported with every super root
	dataErr  error
}

func (f *fakeReplica) SyncStatus(context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	if f.syncErr != nil {
		return eth.SuperNodeSyncStatusResponse{}, f.syncErr
	}
	return eth.SuperNodeSyncStatusResponse{FinalizedTimestamp: f.finalized}, nil
}

func (f *fakeReplica) SuperRootAtTimestamp(_ context.Context, ts uint64) (eth.SuperRootAtTimestampResponse, error) {
	if f.dataErr != nil {
		return eth.SuperRootAtTimestampResponse{}, f.dataErr
	}
	root, ok := f.rootAt[ts]
	if !ok {
		return eth.SuperRootAtTimestampResponse{Data: nil}, nil
	}
	return eth.SuperRootAtTimestampResponse{
		ChainIDs: f.chainIDs,
		Data:     &eth.SuperRootResponseData{SuperRoot: root},
	}, nil
}

func (f *fakeReplica) Endpoint() string { return f.endpoint }
func (f *fakeReplica) Close()           {}

func b32(b byte) eth.Bytes32 {
	var out eth.Bytes32
	out[31] = b
	return out
}

// recordingDivergenceMetrics captures the last RecordReplicaDivergence call.
type recordingDivergenceMetrics struct {
	mu       sync.Mutex
	calls    int
	diverged bool
	compared int
	ts       uint64
}

func (m *recordingDivergenceMetrics) RecordReplicaDivergence(diverged bool, compared int, ts uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.diverged, m.compared, m.ts = diverged, compared, ts
}

func newCollector(t *testing.T, clients []ReplicaClient, m DivergenceMetrics, fs []FailsafeClient, trigger bool) *ReplicaDivergenceCollector {
	t.Helper()
	return NewReplicaDivergenceCollector(clients, 0, log.NewLogger(log.DiscardHandler()), m, fs, trigger)
}

func TestCompareSuperRoots(t *testing.T) {
	t.Run("agreement", func(t *testing.T) {
		r := compareSuperRoots(100, []replicaSuperRoot{
			{endpoint: "a", superRoot: b32(1)},
			{endpoint: "b", superRoot: b32(1)},
			{endpoint: "c", superRoot: b32(1)},
		})
		require.False(t, r.diverged)
		require.Equal(t, 3, r.compared)
		require.Len(t, r.groups, 1)
	})
	t.Run("divergence", func(t *testing.T) {
		r := compareSuperRoots(100, []replicaSuperRoot{
			{endpoint: "a", superRoot: b32(1)},
			{endpoint: "b", superRoot: b32(2)},
		})
		require.True(t, r.diverged)
		require.Len(t, r.groups, 2)
	})
}

func TestCollectOnce_Agreement(t *testing.T) {
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 500, rootAt: map[uint64]eth.Bytes32{500: b32(7)}},
		&fakeReplica{endpoint: "b", finalized: 600, rootAt: map[uint64]eth.Bytes32{500: b32(7)}},
	}
	m := &recordingDivergenceMetrics{}
	c := newCollector(t, clients, m, nil, false)

	c.collectOnce(context.Background())

	require.Equal(t, 1, m.calls)
	require.False(t, m.diverged)
	require.Equal(t, 2, m.compared)
	require.Equal(t, uint64(500), m.ts, "comparison must use the min finalized timestamp")
}

func TestCollectOnce_DivergenceTriggersFailsafe(t *testing.T) {
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 500, rootAt: map[uint64]eth.Bytes32{500: b32(7)}},
		&fakeReplica{endpoint: "b", finalized: 500, rootAt: map[uint64]eth.Bytes32{500: b32(9)}},
	}
	m := &recordingDivergenceMetrics{}
	fs := &fakeFailsafe{}
	c := newCollector(t, clients, m, []FailsafeClient{fs}, true)

	c.collectOnce(context.Background())

	require.Equal(t, 1, m.calls)
	require.True(t, m.diverged)
	require.True(t, fs.enabled, "divergence with trigger-failsafe must enable failsafe")
}

func TestCollectOnce_LaggingReplicaNotFlagged(t *testing.T) {
	// Replica b is behind: finalized 300, and has no data at the min timestamp
	// 300 for replica a's chain... here min is 300, both must report data at 300.
	// b reports data at 300 too (agreeing); a is just further ahead. No divergence.
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 800, rootAt: map[uint64]eth.Bytes32{300: b32(5)}},
		&fakeReplica{endpoint: "b", finalized: 300, rootAt: map[uint64]eth.Bytes32{300: b32(5)}},
	}
	m := &recordingDivergenceMetrics{}
	c := newCollector(t, clients, m, nil, false)

	c.collectOnce(context.Background())

	require.Equal(t, 1, m.calls)
	require.False(t, m.diverged)
	require.Equal(t, uint64(300), m.ts)
}

func TestCollectOnce_SkipsWhenReplicaUnreachable(t *testing.T) {
	// Only one replica responds to SyncStatus → fewer than two comparable → skip.
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 500, rootAt: map[uint64]eth.Bytes32{500: b32(7)}},
		&fakeReplica{endpoint: "b", syncErr: errors.New("down")},
	}
	m := &recordingDivergenceMetrics{}
	c := newCollector(t, clients, m, nil, false)

	c.collectOnce(context.Background())

	// minFinalized still resolves from replica a (=500), but only a returns a
	// super root, so collectSuperRoots yields <2 and the round is skipped.
	require.Equal(t, 0, m.calls)
}

func TestCollectOnce_SkipsWhenNoDataAtTimestamp(t *testing.T) {
	// Both finalized at 500 but neither has data at 500 → transient, skip.
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 500, rootAt: map[uint64]eth.Bytes32{}},
		&fakeReplica{endpoint: "b", finalized: 500, rootAt: map[uint64]eth.Bytes32{}},
	}
	m := &recordingDivergenceMetrics{}
	c := newCollector(t, clients, m, nil, false)

	c.collectOnce(context.Background())
	require.Equal(t, 0, m.calls)
}

// A booting/desynced replica reporting finalized=0 must not drag the min to 0
// and blind detection of a real divergence between the healthy replicas.
func TestCollectOnce_ZeroFinalizedReplicaDoesNotBlind(t *testing.T) {
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 1000, rootAt: map[uint64]eth.Bytes32{1000: b32(7)}},
		&fakeReplica{endpoint: "b", finalized: 1000, rootAt: map[uint64]eth.Bytes32{1000: b32(9)}},
		&fakeReplica{endpoint: "c", finalized: 0, rootAt: map[uint64]eth.Bytes32{}}, // booting
	}
	m := &recordingDivergenceMetrics{}
	fs := &fakeFailsafe{}
	c := newCollector(t, clients, m, []FailsafeClient{fs}, true)

	c.collectOnce(context.Background())

	require.Equal(t, 1, m.calls, "the booting replica must not suppress the round")
	require.True(t, m.diverged, "divergence between healthy replicas must still be detected")
	require.Equal(t, uint64(1000), m.ts)
	require.True(t, fs.enabled)
}

// Replicas configured with different dependency sets legitimately compute
// different super roots; that is a config difference, not a consensus
// divergence, and must NOT trip the failsafe.
func TestCollectOnce_DifferentChainSetsNotFlagged(t *testing.T) {
	chainsA := []eth.ChainID{eth.ChainIDFromUInt64(10), eth.ChainIDFromUInt64(20)}
	chainsB := []eth.ChainID{eth.ChainIDFromUInt64(10)} // different dep set
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 500, rootAt: map[uint64]eth.Bytes32{500: b32(7)}, chainIDs: chainsA},
		&fakeReplica{endpoint: "b", finalized: 500, rootAt: map[uint64]eth.Bytes32{500: b32(9)}, chainIDs: chainsB},
	}
	m := &recordingDivergenceMetrics{}
	fs := &fakeFailsafe{}
	c := newCollector(t, clients, m, []FailsafeClient{fs}, true)

	c.collectOnce(context.Background())

	require.Equal(t, 0, m.calls, "mismatched dependency sets must not be compared")
	require.False(t, fs.enabled, "a config difference must not trip the failsafe")
}

func TestCollectOnce_NoOpBelowTwoReplicas(t *testing.T) {
	clients := []ReplicaClient{
		&fakeReplica{endpoint: "a", finalized: 500, rootAt: map[uint64]eth.Bytes32{500: b32(7)}},
	}
	m := &recordingDivergenceMetrics{}
	c := newCollector(t, clients, m, nil, false)
	c.collectOnce(context.Background())
	require.Equal(t, 0, m.calls)
}

// fakeFailsafe records whether failsafe was enabled.
type fakeFailsafe struct{ enabled bool }

func (f *fakeFailsafe) SetFailsafeEnabled(_ context.Context, enabled bool) error {
	f.enabled = enabled
	return nil
}
func (f *fakeFailsafe) GetFailsafeEnabled(context.Context) (bool, error) { return f.enabled, nil }
