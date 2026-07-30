package consensus

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestCommitAndRead(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)
	now := uint64(time.Now().Unix())
	rollupCfg := &rollup.Config{
		CanyonTime: &now,
	}
	storageDir := "/tmp/sequencerA"
	if err := os.RemoveAll(storageDir); err != nil {
		t.Fatal(err)
	}
	raftConsensusConfig := &RaftConsensusConfig{
		ServerID:           "SequencerA",
		ListenPort:         0,
		ListenAddr:         "127.0.0.1", // local test, don't bind to external interface
		AdvertisedAddr:     "",          // use local address that the server binds to
		StorageDir:         storageDir,
		Bootstrap:          true,
		RollupCfg:          rollupCfg,
		SnapshotInterval:   120 * time.Second,
		SnapshotThreshold:  10240,
		TrailingLogs:       8192,
		HeartbeatTimeout:   1000 * time.Millisecond,
		LeaderLeaseTimeout: 500 * time.Millisecond,
	}

	cons, err := NewRaftConsensus(log, raftConsensusConfig)
	require.NoError(t, err)

	// wait till it became leader
	<-cons.LeaderCh()

	// eth.BlockV1
	payload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber:  1,
			Timestamp:    hexutil.Uint64(now - 20),
			Transactions: []eth.Data{},
			ExtraData:    []byte{},
		},
	}

	err = cons.CommitUnsafePayload(payload)
	// ExecutionPayloadEnvelope is expected to fail when unmarshalling a blockV1
	require.Error(t, err)

	// eth.BlockV3
	one := hexutil.Uint64(1)
	hash := common.HexToHash("0x12345")
	payload = &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &hash,
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber:   2,
			Timestamp:     hexutil.Uint64(time.Now().Unix()),
			Transactions:  []eth.Data{},
			ExtraData:     []byte{},
			Withdrawals:   &types.Withdrawals{},
			ExcessBlobGas: &one,
			BlobGasUsed:   &one,
		},
	}

	err = cons.CommitUnsafePayload(payload)
	// ExecutionPayloadEnvelope is expected to succeed when unmarshalling a blockV3
	require.NoError(t, err)

	unsafeHead, err := cons.LatestUnsafePayload()
	require.NoError(t, err)
	require.Equal(t, payload, unsafeHead)
}

// blockedFuture stands in for the future raft.Shutdown() returns when raft's internal
// goroutines never exit.
type blockedFuture struct {
	released chan struct{}
}

func (f *blockedFuture) Error() error {
	<-f.released
	return nil
}

func newRaftConsensusForTest(t *testing.T) *RaftConsensus {
	now := uint64(time.Now().Unix())
	cons, err := NewRaftConsensus(testlog.Logger(t, log.LevelInfo), &RaftConsensusConfig{
		ServerID:           "SequencerA",
		ListenPort:         0,
		ListenAddr:         "127.0.0.1",
		StorageDir:         t.TempDir(),
		Bootstrap:          true,
		RollupCfg:          &rollup.Config{CanyonTime: &now},
		SnapshotInterval:   120 * time.Second,
		SnapshotThreshold:  10240,
		TrailingLogs:       8192,
		HeartbeatTimeout:   1000 * time.Millisecond,
		LeaderLeaseTimeout: 500 * time.Millisecond,
	})
	require.NoError(t, err)
	return cons
}

// Shutdown must not outlive its budget when raft never reports that it has stopped, and it must
// close the transport itself in that case — raft only closes it after the wait we gave up on.
func TestShutdownGivesUpOnRaft(t *testing.T) {
	cons := newRaftConsensusForTest(t)
	addr := string(cons.transport.LocalAddr())

	released := make(chan struct{})
	t.Cleanup(func() { close(released) })
	cons.shutdownRaft = func() raft.Future { return &blockedFuture{released: released} }
	cons.shutdownTimeout = 50 * time.Millisecond

	done := make(chan error, 1)
	go func() { done <- cons.Shutdown() }()

	select {
	case err := <-done:
		require.ErrorContains(t, err, "raft did not shut down within 100ms")
	case <-time.After(30 * time.Second):
		t.Fatal("Shutdown blocked on raft instead of giving up")
	}

	// The transport is closed even though raft never finished, so nothing is left listening.
	_, err := net.DialTimeout("tcp", addr, time.Second)
	require.Error(t, err, "raft transport should no longer be listening on %s", addr)
}

func TestAwaitShutdown(t *testing.T) {
	t.Run("reports the error raft returned", func(t *testing.T) {
		stopped := make(chan error, 1)
		stopped <- errors.New("boom")
		err, ok := awaitShutdown(stopped, time.Minute)
		require.True(t, ok)
		require.ErrorContains(t, err, "boom")
	})

	t.Run("gives up when nothing arrives", func(t *testing.T) {
		err, ok := awaitShutdown(make(chan error), time.Millisecond)
		require.False(t, ok)
		require.NoError(t, err)
	})
}
