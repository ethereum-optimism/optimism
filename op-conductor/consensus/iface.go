package consensus

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ServerSuffrage determines whether a Server in a Configuration gets a vote.
type ServerSuffrage int

const (
	// Voter is a server whose vote is counted in elections.
	Voter ServerSuffrage = iota
	// Nonvoter is a server that receives log entries but is not considered for
	// elections or commitment purposes.
	Nonvoter
)

func (s ServerSuffrage) String() string {
	switch s {
	case Voter:
		return "Voter"
	case Nonvoter:
		return "Nonvoter"
	}
	return "ServerSuffrage"
}

// ServerInfo defines the server information.
type ServerInfo struct {
	ID       string         `json:"id"`
	Addr     string         `json:"addr"`
	Suffrage ServerSuffrage `json:"suffrage"`
}

// Consensus defines the consensus interface for leadership election.
//
//go:generate mockery --name Consensus --output mocks/ --with-expecter=true
type Consensus interface {
	// AddVoter adds a voting member into the cluster, voter is eligible to become leader.
	AddVoter(id, addr string) error
	// AddNonVoter adds a non-voting member into the cluster, non-voter is not eligible to become leader.
	AddNonVoter(id, addr string) error
	// DemoteVoter demotes a voting member into a non-voting member, if leader is being demoted, it will cause a new leader election.
	DemoteVoter(id string) error
	// RemoveServer removes a member (both voter or non-voter) from the cluster, if leader is being removed, it will cause a new leader election.
	RemoveServer(id string) error
	// LeaderCh returns a channel that will be notified when leadership status changes (true = leader, false = follower)
	LeaderCh() <-chan bool
	// Leader returns if it is the leader of the cluster.
	Leader() bool
	// LeaderWithID returns the leader's server ID and address.
	LeaderWithID() *ServerInfo
	// ServerID returns the server ID of the consensus.
	ServerID() string
	// TransferLeader triggers leadership transfer to another member in the cluster.
	TransferLeader() error
	// TransferLeaderTo triggers leadership transfer to a specific member in the cluster.
	TransferLeaderTo(id, addr string) error
	// ClusterMembership returns the current cluster membership configuration.
	ClusterMembership() ([]*ServerInfo, error)

	// CommitPayload commits latest unsafe payload to the FSM in a strongly consistent fashion.
	CommitUnsafePayload(payload *eth.ExecutionPayloadEnvelope) error
	// LatestUnsafeBlock returns the latest unsafe payload from FSM in a strongly consistent fashion.
	LatestUnsafePayload() (*eth.ExecutionPayloadEnvelope, error)

	// Shutdown shuts down the consensus protocol client.
	Shutdown() error
}
