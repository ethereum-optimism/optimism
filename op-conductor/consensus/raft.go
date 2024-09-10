package consensus

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/raft"
	boltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/pkg/errors"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const defaultTimeout = 5 * time.Second

var _ Consensus = (*RaftConsensus)(nil)

// RaftConsensus implements Consensus using raft protocol.
type RaftConsensus struct {
	log       log.Logger
	rollupCfg *rollup.Config

	serverID raft.ServerID
	r        *raft.Raft

	unsafeTracker *unsafeHeadTracker
}

type RaftConsensusConfig struct {
	ServerID          string
	ServerAddr        string
	StorageDir        string
	Bootstrap         bool
	RollupCfg         *rollup.Config
	SnapshotInterval  time.Duration
	SnapshotThreshold uint64
	TrailingLogs      uint64
}

// checkTCPPortOpen attempts to connect to the specified address and returns an error if the connection fails.
func checkTCPPortOpen(address string) error {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

// NewRaftConsensus creates a new RaftConsensus instance.
func NewRaftConsensus(log log.Logger, cfg *RaftConsensusConfig) (*RaftConsensus, error) {
	rc := raft.DefaultConfig()
	rc.SnapshotInterval = cfg.SnapshotInterval
	rc.TrailingLogs = cfg.TrailingLogs
	rc.SnapshotThreshold = cfg.SnapshotThreshold
	rc.LocalID = raft.ServerID(cfg.ServerID)

	baseDir := filepath.Join(cfg.StorageDir, cfg.ServerID)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		if err := os.MkdirAll(baseDir, 0o755); err != nil {
			return nil, fmt.Errorf("error creating storage dir: %w", err)
		}
	}

	var err error
	logStorePath := filepath.Join(baseDir, "raft-log.db")
	logStore, err := boltdb.NewBoltStore(logStorePath)
	if err != nil {
		return nil, fmt.Errorf(`boltdb.NewBoltStore(%q): %w`, logStorePath, err)
	}

	stableStorePath := filepath.Join(baseDir, "raft-stable.db")
	stableStore, err := boltdb.NewBoltStore(stableStorePath)
	if err != nil {
		return nil, fmt.Errorf(`boltdb.NewBoltStore(%q): %w`, stableStorePath, err)
	}

	snapshotStore, err := raft.NewFileSnapshotStoreWithLogger(baseDir, 1, rc.Logger)
	if err != nil {
		return nil, fmt.Errorf(`raft.NewFileSnapshotStore(%q): %w`, baseDir, err)
	}

	addr, err := net.ResolveTCPAddr("tcp", cfg.ServerAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve tcp address")
	}

	maxConnPool := 10
	timeout := 5 * time.Second
	bindAddr := fmt.Sprintf("0.0.0.0:%d", addr.Port)
	transport, err := raft.NewTCPTransportWithLogger(bindAddr, addr, maxConnPool, timeout, rc.Logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create raft tcp transport")
	}

	fsm := NewUnsafeHeadTracker(log)

	r, err := raft.NewRaft(rc, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		log.Error("failed to create raft", "err", err)
		return nil, errors.Wrap(err, "failed to create raft")
	}

	// If bootstrap = true, start raft in bootstrap mode, this will allow the current node to elect itself as leader when there's no other participants
	// and allow other nodes to join the cluster.
	if cfg.Bootstrap {
		raftCfg := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:       rc.LocalID,
					Address:  raft.ServerAddress(cfg.ServerAddr),
					Suffrage: raft.Voter,
				},
			},
		}

		f := r.BootstrapCluster(raftCfg)
		if err := f.Error(); err != nil {
			return nil, errors.Wrap(err, "failed to bootstrap raft cluster")
		}
	}

	return &RaftConsensus{
		log:           log,
		r:             r,
		serverID:      raft.ServerID(cfg.ServerID),
		unsafeTracker: fsm,
		rollupCfg:     cfg.RollupCfg,
	}, nil
}

// AddNonVoter implements Consensus, it tries to add a non-voting member into the cluster.
func (rc *RaftConsensus) AddNonVoter(id string, addr string, version uint64) error {
	if err := checkTCPPortOpen(addr); err != nil {
		rc.log.Error("connection test to member addr failed", "id", id, "addr", addr, "err", err)
		return err
	}
	if err := rc.r.AddNonvoter(raft.ServerID(id), raft.ServerAddress(addr), version, defaultTimeout).Error(); err != nil {
		rc.log.Error("failed to add non-voter", "id", id, "addr", addr, "version", version, "err", err)
		return err
	}
	return nil
}

// AddVoter implements Consensus, it tries to add a voting member into the cluster.
func (rc *RaftConsensus) AddVoter(id string, addr string, version uint64) error {
	if err := checkTCPPortOpen(addr); err != nil {
		rc.log.Error("connection test to member addr failed", "id", id, "addr", addr, "err", err)
		return err
	}
	if err := rc.r.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), version, defaultTimeout).Error(); err != nil {
		rc.log.Error("failed to add voter", "id", id, "addr", addr, "version", version, "err", err)
		return err
	}
	return nil
}

// DemoteVoter implements Consensus, it tries to demote a voting member into a non-voting member in the cluster.
func (rc *RaftConsensus) DemoteVoter(id string, version uint64) error {
	if err := rc.r.DemoteVoter(raft.ServerID(id), version, defaultTimeout).Error(); err != nil {
		rc.log.Error("failed to demote voter", "id", id, "version", version, "err", err)
		return err
	}
	return nil
}

// RemoveServer implements Consensus, it tries to remove a member (both voter or non-voter) from the cluster, if leader is being removed, it will cause a new leader election.
func (rc *RaftConsensus) RemoveServer(id string, version uint64) error {
	if err := rc.r.RemoveServer(raft.ServerID(id), version, defaultTimeout).Error(); err != nil {
		rc.log.Error("failed to remove voter", "id", id, "version", version, "err", err)
		return err
	}
	return nil
}

// Leader implements Consensus, it returns true if it is the leader of the cluster.
func (rc *RaftConsensus) Leader() bool {
	return rc.r.State() == raft.Leader
}

// LeaderWithID implements Consensus, it returns the leader's server ID and address.
func (rc *RaftConsensus) LeaderWithID() *ServerInfo {
	addr, id := rc.r.LeaderWithID()
	return &ServerInfo{
		ID:       string(id),
		Addr:     string(addr),
		Suffrage: Voter, // leader will always be Voter
	}
}

// LeaderCh implements Consensus, it returns a channel that will be notified when leadership status changes (true = leader, false = follower).
func (rc *RaftConsensus) LeaderCh() <-chan bool {
	return rc.r.LeaderCh()
}

// ServerID implements Consensus, it returns the server ID of the current server.
func (rc *RaftConsensus) ServerID() string {
	return string(rc.serverID)
}

// TransferLeader implements Consensus, it triggers leadership transfer to another member in the cluster.
func (rc *RaftConsensus) TransferLeader() error {
	if err := rc.r.LeadershipTransfer().Error(); err != nil {
		// Expected error if not leader
		if errors.Is(err, raft.ErrNotLeader) {
			return nil
		}

		rc.log.Error("failed to transfer leadership", "err", err)
		return err
	}
	return nil
}

// TransferLeaderTo implements Consensus, it triggers leadership transfer to a specific member in the cluster.
func (rc *RaftConsensus) TransferLeaderTo(id string, addr string) error {
	if err := rc.r.LeadershipTransferToServer(raft.ServerID(id), raft.ServerAddress(addr)).Error(); err != nil {
		rc.log.Error("failed to transfer leadership to server", "id", id, "addr", addr, "err", err)
		return err
	}
	return nil
}

// Shutdown implements Consensus, it shuts down the consensus protocol client.
func (rc *RaftConsensus) Shutdown() error {
	if err := rc.r.Shutdown().Error(); err != nil {
		rc.log.Error("failed to shutdown raft", "err", err)
		return err
	}
	return nil
}

// CommitUnsafePayload implements Consensus, it commits latest unsafe payload to the cluster FSM in a strongly consistent fashion.
func (rc *RaftConsensus) CommitUnsafePayload(payload *eth.ExecutionPayloadEnvelope) error {
	rc.log.Debug("committing unsafe payload", "number", uint64(payload.ExecutionPayload.BlockNumber), "hash", payload.ExecutionPayload.BlockHash.Hex())

	var buf bytes.Buffer
	if _, err := payload.MarshalSSZ(&buf); err != nil {
		return errors.Wrap(err, "failed to marshal payload envelope")
	}

	f := rc.r.Apply(buf.Bytes(), defaultTimeout)
	if err := f.Error(); err != nil {
		return errors.Wrap(err, "failed to apply payload envelope")
	}
	rc.log.Debug("unsafe payload committed", "number", uint64(payload.ExecutionPayload.BlockNumber), "hash", payload.ExecutionPayload.BlockHash.Hex())

	return nil
}

// LatestUnsafePayload implements Consensus, it returns the latest unsafe payload from FSM in a strongly consistent fashion.
func (rc *RaftConsensus) LatestUnsafePayload() (*eth.ExecutionPayloadEnvelope, error) {
	if err := rc.r.Barrier(defaultTimeout).Error(); err != nil {
		return nil, errors.Wrap(err, "failed to apply barrier")
	}

	return rc.unsafeTracker.UnsafeHead(), nil
}

// ClusterMembership implements Consensus, it returns the current cluster membership configuration.
func (rc *RaftConsensus) ClusterMembership() (*ClusterMembership, error) {
	var future raft.ConfigurationFuture
	if future = rc.r.GetConfiguration(); future.Error() != nil {
		return nil, future.Error()
	}

	var servers []ServerInfo
	for _, srv := range future.Configuration().Servers {
		servers = append(servers, ServerInfo{
			ID:       string(srv.ID),
			Addr:     string(srv.Address),
			Suffrage: ServerSuffrage(srv.Suffrage),
		})
	}
	return &ClusterMembership{
		Servers: servers,
		Version: future.Index(),
	}, nil
}
