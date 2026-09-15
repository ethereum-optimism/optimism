package consensus

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
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

// TestSnapshotBeforeFirstCommit takes Raft snapshots of a live single-node cluster before and after the first
// unsafe head is committed, and restarts from each snapshot. Before snapshot.Persist supported an empty
// snapshot, the first snapshot dereferenced the nil payload and crashed op-conductor (#13363).
func TestSnapshotBeforeFirstCommit(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	now := uint64(time.Now().Unix())
	rollupCfg := &rollup.Config{
		CanyonTime: &now,
	}
	const serverID = "SequencerA"

	newConsensus := func(storageDir string, bootstrap bool) *RaftConsensus {
		cons, err := NewRaftConsensus(logger, &RaftConsensusConfig{
			ServerID:           serverID,
			ListenPort:         0,
			ListenAddr:         "127.0.0.1", // local test, don't bind to external interface
			AdvertisedAddr:     "",          // use local address that the server binds to
			StorageDir:         storageDir,
			Bootstrap:          bootstrap,
			RollupCfg:          rollupCfg,
			SnapshotInterval:   120 * time.Second,
			SnapshotThreshold:  10240,
			TrailingLogs:       8192,
			HeartbeatTimeout:   1000 * time.Millisecond,
			LeaderLeaseTimeout: 500 * time.Millisecond,
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = cons.Shutdown() })
		select {
		case leader := <-cons.LeaderCh():
			require.True(t, leader)
		case <-time.After(30 * time.Second):
			t.Fatal("timed out waiting for leadership")
		}
		return cons
	}

	// Shutdown leaves the bolt stores open in this process, so a restart copies the storage dir of the previous
	// instance to a fresh location. Raft then restores the latest snapshot and replays the log on startup, as a
	// restarted node would.
	restart := func(from string) (*RaftConsensus, string) {
		storageDir := t.TempDir()
		require.NoError(t, os.CopyFS(filepath.Join(storageDir, serverID), os.DirFS(filepath.Join(from, serverID))))
		return newConsensus(storageDir, false), storageDir
	}

	takeSnapshot := func(cons *RaftConsensus) int64 {
		future := cons.r.Snapshot()
		require.NoError(t, future.Error())
		meta, rc, err := future.Open()
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		return meta.Size
	}

	storageDir := t.TempDir()
	cons := newConsensus(storageDir, true)

	// The barrier issued by LatestUnsafePayload is what makes the FSM snapshot-able: raft only snapshots once a
	// barrier or command has been applied, and the conductor's health checks issue these barriers in production.
	unsafeHead, err := cons.LatestUnsafePayload()
	require.NoError(t, err)
	require.Nil(t, unsafeHead)

	// Snapshot before any unsafe head has been committed. This used to panic in snapshot.Persist.
	require.Zero(t, takeSnapshot(cons))
	require.NoError(t, cons.Shutdown())

	// Restart from the empty snapshot: the node starts with no unsafe head and accepts commits.
	cons, storageDir = restart(storageDir)
	unsafeHead, err = cons.LatestUnsafePayload()
	require.NoError(t, err)
	require.Nil(t, unsafeHead)

	one := hexutil.Uint64(1)
	hash := common.HexToHash("0x12345")
	payload := &eth.ExecutionPayloadEnvelope{
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
	require.NoError(t, cons.CommitUnsafePayload(payload))

	// Snapshot after the first commit is non-empty and restores the committed unsafe head.
	require.Positive(t, takeSnapshot(cons))
	require.NoError(t, cons.Shutdown())

	cons, _ = restart(storageDir)
	unsafeHead, err = cons.LatestUnsafePayload()
	require.NoError(t, err)
	require.Equal(t, payload, unsafeHead)
}
