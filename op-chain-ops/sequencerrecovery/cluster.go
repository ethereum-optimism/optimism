package sequencerrecovery

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Conductor is the part of the op-conductor RPC this package uses.
type Conductor interface {
	Leader(ctx context.Context) (bool, error)
	LeaderWithID(ctx context.Context) (*consensus.ServerInfo, error)
	Paused(ctx context.Context) (bool, error)
	SequencerHealthy(ctx context.Context) (bool, error)
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
	TransferLeaderToServer(ctx context.Context, id string, addr string) error
}

// Node is the part of the op-node rollup RPC this package uses. The supernode
// (or the proxyd-cl in front of it) serves the same methods per chain.
type Node interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
	SequencerActive(ctx context.Context) (bool, error)
	StartSequencer(ctx context.Context, unsafeHead common.Hash) error
	StopSequencer(ctx context.Context) (common.Hash, error)
}

// Member is a sequencer with its RPC clients.
type Member struct {
	Sequencer
	Conductor Conductor
	Node      Node
}

// Cluster is a conductor cluster plus the supernode the sequencers follow.
type Cluster struct {
	Members []Member
	// Supernode answers for this chain only, e.g. the chain's proxyd-cl URL.
	Supernode Node
	// StallAfter is how old the newest unsafe head may be before the chain
	// counts as stalled.
	StallAfter time.Duration
	Now        func() time.Time
}

// MemberStatus is what one sequencer reports. Err is set when any call failed;
// the other fields then hold whatever was read before the failure.
type MemberStatus struct {
	ID               string
	Leader           bool
	Paused           bool
	SequencerHealthy bool
	SequencerActive  bool
	Sync             *eth.SyncStatus
	// Canonical is true when this op-node's safe head matches the supernode's
	// block at the same height, so its unsafe chain descends from the
	// supernode's view of the chain.
	Canonical bool
	// Err is set when this sequencer's own conductor or op-node could not be read.
	Err error
	// CompareErr is set when the supernode comparison failed (e.g. the supernode
	// RPC is briefly unavailable during a rewind); the rest of the status is valid.
	CompareErr error
}

// Status is a snapshot of the whole cluster.
type Status struct {
	Supernode *eth.SyncStatus
	Members   []MemberStatus
	// LeaderID is the raft leader as reported by any reachable conductor.
	LeaderID string
	// Stalled is true when no op-node has an unsafe head newer than StallAfter.
	Stalled bool
}

// SequencingProblems lists what stops the chain from being sequenced: one
// active sequencer, the raft leader, on the supernode's chain, with its unsafe
// head advancing and its conductor running. Recovery acts on these and waits
// for them to clear.
func (s *Status) SequencingProblems() []string {
	var out []string
	if s.LeaderID == "" {
		out = append(out, "no raft leader")
	} else if !s.hasMember(s.LeaderID) {
		out = append(out, fmt.Sprintf("raft leader %q is not a sequencer in the config; config sequencer IDs must be the raft server IDs", s.LeaderID))
	}
	if s.Stalled {
		out = append(out, "unsafe head is not advancing")
	}
	var active []MemberStatus
	for _, m := range s.Members {
		if m.Err == nil && m.SequencerActive {
			active = append(active, m)
		}
	}
	switch {
	case len(active) == 0:
		out = append(out, "no sequencer is active")
	case len(active) > 1:
		ids := make([]string, len(active))
		for i, m := range active {
			ids[i] = m.ID
		}
		out = append(out, fmt.Sprintf("more than one sequencer is active: %v", ids))
	default:
		a := active[0]
		if s.LeaderID != "" && a.ID != s.LeaderID {
			out = append(out, fmt.Sprintf("active sequencer %s is not the raft leader %s", a.ID, s.LeaderID))
		}
		if a.CompareErr == nil && !a.Canonical {
			out = append(out, fmt.Sprintf("active sequencer %s is not on the supernode's chain", a.ID))
		}
		if a.Paused {
			out = append(out, fmt.Sprintf("conductor of active sequencer %s is paused", a.ID))
		}
	}
	return out
}

func (s *Status) hasMember(id string) bool {
	for _, m := range s.Members {
		if m.ID == id {
			return true
		}
	}
	return false
}

// MemberProblems lists per-sequencer problems that do not stop sequencing but
// leave the cluster without a healthy standby, such as an op-node on a fork the
// supernode replaced. Recovery reports these; fixing them (for example by
// resyncing an op-node) is a separate operation.
func (s *Status) MemberProblems() []string {
	var out []string
	for _, m := range s.Members {
		switch {
		case m.Err != nil:
			out = append(out, fmt.Sprintf("%s: %v", m.ID, m.Err))
		case m.CompareErr != nil:
			out = append(out, fmt.Sprintf("%s: %v", m.ID, m.CompareErr))
		case !m.Canonical:
			out = append(out, fmt.Sprintf("%s: safe head does not match the supernode", m.ID))
		case m.Paused:
			out = append(out, fmt.Sprintf("%s: conductor paused", m.ID))
		case !m.SequencerHealthy:
			out = append(out, fmt.Sprintf("%s: conductor reports sequencer unhealthy", m.ID))
		}
	}
	return out
}

// Status reads the cluster. It returns an error only if the supernode cannot be
// read; per-member failures are recorded on the member.
func (c *Cluster) Status(ctx context.Context) (*Status, error) {
	sn, err := c.Supernode.SyncStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("supernode sync status: %w", err)
	}
	st := &Status{Supernode: sn, Stalled: true}
	for _, m := range c.Members {
		ms := c.memberStatus(ctx, m, sn)
		if ms.Sync != nil && c.Now().Sub(time.Unix(int64(ms.Sync.UnsafeL2.Time), 0)) < c.StallAfter {
			st.Stalled = false
		}
		st.Members = append(st.Members, ms)
	}
	for _, m := range c.Members {
		info, err := m.Conductor.LeaderWithID(ctx)
		if err == nil && info != nil {
			st.LeaderID = info.ID
			break
		}
	}
	return st, nil
}

func (c *Cluster) memberStatus(ctx context.Context, m Member, sn *eth.SyncStatus) MemberStatus {
	ms := MemberStatus{ID: m.ID}
	var err error
	if ms.Leader, err = m.Conductor.Leader(ctx); err != nil {
		ms.Err = fmt.Errorf("conductor leader: %w", err)
		return ms
	}
	if ms.Paused, err = m.Conductor.Paused(ctx); err != nil {
		ms.Err = fmt.Errorf("conductor paused: %w", err)
		return ms
	}
	if ms.SequencerHealthy, err = m.Conductor.SequencerHealthy(ctx); err != nil {
		ms.Err = fmt.Errorf("conductor sequencer healthy: %w", err)
		return ms
	}
	if ms.SequencerActive, err = m.Node.SequencerActive(ctx); err != nil {
		ms.Err = fmt.Errorf("op-node sequencer active: %w", err)
		return ms
	}
	if ms.Sync, err = m.Node.SyncStatus(ctx); err != nil {
		ms.Err = fmt.Errorf("op-node sync status: %w", err)
		return ms
	}
	if ms.Canonical, err = c.matchesSupernode(ctx, m.Node, ms.Sync, sn); err != nil {
		ms.CompareErr = fmt.Errorf("compare with supernode: %w", err)
	}
	return ms
}

// matchesSupernode compares the op-node's safe chain with the supernode's at
// the lower of the two safe heights. The op-node's unsafe chain always extends
// its own safe head, so a match means the unsafe head is on the canonical chain
// as far as the supernode has verified it.
func (c *Cluster) matchesSupernode(ctx context.Context, node Node, ns, sn *eth.SyncStatus) (bool, error) {
	n := min(ns.SafeL2.Number, sn.SafeL2.Number)
	if n == 0 {
		return true, nil
	}
	nodeOut, err := node.OutputAtBlock(ctx, n)
	if err != nil {
		return false, fmt.Errorf("op-node output at %d: %w", n, err)
	}
	snOut, err := c.Supernode.OutputAtBlock(ctx, n)
	if err != nil {
		return false, fmt.Errorf("supernode output at %d: %w", n, err)
	}
	return nodeOut.BlockRef.Hash == snOut.BlockRef.Hash, nil
}
