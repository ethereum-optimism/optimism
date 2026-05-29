package consensus

import (
	"os"
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

// v3Payload builds a minimal BlockV3 ExecutionPayloadEnvelope at the given number.
func v3Payload(number uint64) *eth.ExecutionPayloadEnvelope {
	one := hexutil.Uint64(1)
	hash := common.HexToHash("0x12345")
	return &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &hash,
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber:   hexutil.Uint64(number),
			Timestamp:     hexutil.Uint64(time.Now().Unix()),
			Transactions:  []eth.Data{},
			ExtraData:     []byte{},
			Withdrawals:   &types.Withdrawals{},
			ExcessBlobGas: &one,
			BlobGasUsed:   &one,
		},
	}
}

// newBootstrapConsensus spins up a single-node bootstrapped raft cluster and waits
// for leadership. storageDir is wiped first.
func newBootstrapConsensus(t *testing.T, serverID, storageDir string, allowNonMonotonic bool) *RaftConsensus {
	now := uint64(time.Now().Unix())
	rollupCfg := &rollup.Config{CanyonTime: &now}
	require.NoError(t, os.RemoveAll(storageDir))
	cfg := &RaftConsensusConfig{
		ServerID:                    serverID,
		ListenPort:                  0,
		ListenAddr:                  "127.0.0.1",
		AdvertisedAddr:              "",
		StorageDir:                  storageDir,
		Bootstrap:                   true,
		RollupCfg:                   rollupCfg,
		SnapshotInterval:            120 * time.Second,
		SnapshotThreshold:           10240,
		TrailingLogs:                8192,
		HeartbeatTimeout:            1000 * time.Millisecond,
		LeaderLeaseTimeout:          500 * time.Millisecond,
		AllowNonMonotonicUnsafeHead: allowNonMonotonic,
	}
	cons, err := NewRaftConsensus(testlog.Logger(t, log.LevelInfo), cfg)
	require.NoError(t, err)
	<-cons.LeaderCh()
	return cons
}

// TestCommitNonMonotonicWhenUnguarded proves a backward CommitUnsafePayload is
// recorded through the real raft Apply path when the flag is on (the reorg case).
// Multi-node follower convergence is covered by the Phase 3 acceptance test; the
// FSM Apply that followers replay is identical, exercised in raft_fsm_test.go.
func TestCommitNonMonotonicWhenUnguarded(t *testing.T) {
	cons := newBootstrapConsensus(t, "SequencerReorgOn", "/tmp/sequencer-reorg-on", true)

	require.NoError(t, cons.CommitUnsafePayload(v3Payload(5)))
	require.NoError(t, cons.CommitUnsafePayload(v3Payload(2))) // backward (reorg)

	unsafeHead, err := cons.LatestUnsafePayload()
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(2), unsafeHead.ExecutionPayload.BlockNumber,
		"with the guard bypassed, a backward commit must move the recorded head back")
}

// TestCommitForwardOnlyWhenGuarded proves the default (flag off) path drops a
// backward commit through the real raft Apply path — byte-for-byte today's behavior.
func TestCommitForwardOnlyWhenGuarded(t *testing.T) {
	cons := newBootstrapConsensus(t, "SequencerReorgOff", "/tmp/sequencer-reorg-off", false)

	require.NoError(t, cons.CommitUnsafePayload(v3Payload(5)))
	require.NoError(t, cons.CommitUnsafePayload(v3Payload(2))) // backward — must be dropped

	unsafeHead, err := cons.LatestUnsafePayload()
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5), unsafeHead.ExecutionPayload.BlockNumber,
		"with the guard on, a backward commit must be ignored")
}
