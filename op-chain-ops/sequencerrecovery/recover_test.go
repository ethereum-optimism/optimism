package sequencerrecovery

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// chain is a toy L2 chain: block hashes per height on the canonical chain and
// on a stale fork that an invalidated block started.
type chain struct {
	canonical map[uint64]common.Hash
	fork      map[uint64]common.Hash
}

func newChain(n uint64, forkFrom uint64) *chain {
	c := &chain{canonical: map[uint64]common.Hash{}, fork: map[uint64]common.Hash{}}
	for i := uint64(0); i <= n+100; i++ {
		c.canonical[i] = common.BigToHash(new(big.Int).SetUint64(i))
		if i >= forkFrom {
			c.fork[i] = common.BigToHash(new(big.Int).SetUint64(i + 1_000_000))
		} else {
			c.fork[i] = c.canonical[i]
		}
	}
	return c
}

type fakeCluster struct {
	leader  string
	members map[string]*fakeMember
	now     time.Time
}

type fakeConductor struct {
	cl     *fakeCluster
	id     string
	paused bool
}

func (f *fakeConductor) Leader(context.Context) (bool, error) { return f.cl.leader == f.id, nil }
func (f *fakeConductor) LeaderWithID(context.Context) (*consensus.ServerInfo, error) {
	if f.cl.leader == "" {
		return nil, errors.New("no leader")
	}
	return &consensus.ServerInfo{ID: f.cl.leader}, nil
}
func (f *fakeConductor) Paused(context.Context) (bool, error)           { return f.paused, nil }
func (f *fakeConductor) SequencerHealthy(context.Context) (bool, error) { return true, nil }
func (f *fakeConductor) Pause(context.Context) error                    { f.paused = true; return nil }
func (f *fakeConductor) Resume(context.Context) error                   { f.paused = false; return nil }
func (f *fakeConductor) TransferLeaderToServer(_ context.Context, id, _ string) error {
	if f.cl.leader != f.id {
		return errors.New("not leader")
	}
	f.cl.leader = id
	return nil
}

// fakeNode is an op-node. While active it builds one block per SyncStatus call.
type fakeNode struct {
	cl     *fakeCluster
	chain  *chain
	onFork bool
	unsafe uint64
	safe   uint64
	active bool
	// startedFrom records the hash passed to StartSequencer.
	startedFrom common.Hash
}

func (f *fakeNode) hash(n uint64) common.Hash {
	if f.onFork {
		return f.chain.fork[n]
	}
	return f.chain.canonical[n]
}

func (f *fakeNode) SyncStatus(context.Context) (*eth.SyncStatus, error) {
	if f.active {
		f.unsafe++
	}
	ts := uint64(f.cl.now.Unix())
	if !f.active {
		ts -= 600
	}
	return &eth.SyncStatus{
		UnsafeL2: eth.L2BlockRef{Number: f.unsafe, Hash: f.hash(f.unsafe), Time: ts},
		SafeL2:   eth.L2BlockRef{Number: f.safe, Hash: f.hash(f.safe)},
	}, nil
}

func (f *fakeNode) OutputAtBlock(_ context.Context, n uint64) (*eth.OutputResponse, error) {
	return &eth.OutputResponse{BlockRef: eth.L2BlockRef{Number: n, Hash: f.hash(n)}}, nil
}
func (f *fakeNode) SequencerActive(context.Context) (bool, error) { return f.active, nil }
func (f *fakeNode) StartSequencer(_ context.Context, h common.Hash) error {
	if h != f.hash(f.unsafe) {
		return errors.New("block hash does not match")
	}
	f.startedFrom = h
	f.active = true
	return nil
}
func (f *fakeNode) StopSequencer(context.Context) (common.Hash, error) {
	f.active = false
	return f.hash(f.unsafe), nil
}

type fakeMember struct {
	cond *fakeConductor
	node *fakeNode
}

func newFake(t *testing.T, leader string) (*fakeCluster, *Cluster) {
	ch := newChain(200, 150)
	fc := &fakeCluster{leader: leader, members: map[string]*fakeMember{}, now: time.Unix(1_800_000_000, 0)}
	c := &Cluster{
		Supernode:  &fakeNode{cl: fc, chain: ch, unsafe: 160, safe: 140},
		StallAfter: 30 * time.Second,
		Now:        func() time.Time { return fc.now },
	}
	for _, id := range []string{"seq-0", "seq-1", "seq-2"} {
		fm := &fakeMember{
			cond: &fakeConductor{cl: fc, id: id},
			node: &fakeNode{cl: fc, chain: ch, unsafe: 155, safe: 140},
		}
		fc.members[id] = fm
		c.Members = append(c.Members, Member{
			Sequencer: Sequencer{ID: id, RaftAddr: id + ":50050", Voting: true},
			Conductor: fm.cond,
			Node:      fm.node,
		})
	}
	return fc, c
}

var testRecoverConfig = RecoverConfig{PollInterval: time.Millisecond, Timeout: time.Second, AdvanceBlocks: 2, CaughtUpWithin: 10 * time.Second}

func runPlan(t *testing.T, c *Cluster) []Step {
	ctx := context.Background()
	st, err := c.Status(ctx)
	require.NoError(t, err)
	steps, err := c.Plan(st, testRecoverConfig)
	require.NoError(t, err)
	require.NoError(t, Run(ctx, testlog.Logger(t, slog.LevelInfo), steps))
	return steps
}

func TestHealthyClusterNeedsNoSteps(t *testing.T) {
	fc, c := newFake(t, "seq-0")
	fc.members["seq-0"].node.active = true

	st, err := c.Status(context.Background())
	require.NoError(t, err)
	require.Empty(t, st.SequencingProblems())
	require.Empty(t, st.MemberProblems())
	steps, err := c.Plan(st, testRecoverConfig)
	require.NoError(t, err)
	require.Empty(t, steps)
}

// The stall after an interop reorg: the raft leader's conductor will not start
// its op-node, so no sequencer is active and the head stops advancing.
func TestRecoverStalledLeaderStartsFromOwnHead(t *testing.T) {
	fc, c := newFake(t, "seq-0")
	leaderNode := fc.members["seq-0"].node
	head := leaderNode.hash(leaderNode.unsafe)

	st, err := c.Status(context.Background())
	require.NoError(t, err)
	require.True(t, st.Stalled)
	require.Contains(t, st.SequencingProblems(), "no sequencer is active")

	runPlan(t, c)
	require.Equal(t, "seq-0", fc.leader, "the canonical leader keeps leadership")
	require.Equal(t, head, leaderNode.startedFrom)
	for id, m := range fc.members {
		require.False(t, m.cond.paused, "conductor %s must be resumed", id)
		require.Equal(t, id == "seq-0", m.node.active, "only the leader sequences")
	}
	st, err = c.Status(context.Background())
	require.NoError(t, err)
	require.Empty(t, st.SequencingProblems())
	require.Empty(t, st.MemberProblems())
}

// A leader whose op-node stayed on the invalidated fork (e.g. a sequencer that
// derives from L1 itself) must not be restarted; leadership moves to the
// canonical sequencer with the highest head.
func TestRecoverMovesLeadershipOffDivergedLeader(t *testing.T) {
	fc, c := newFake(t, "seq-2")
	diverged := fc.members["seq-2"].node
	diverged.onFork = true
	diverged.safe = 152 // past the fork point at 150
	diverged.unsafe = 158
	fc.members["seq-1"].node.unsafe = 157
	c.Supernode.(*fakeNode).safe = 155

	st, err := c.Status(context.Background())
	require.NoError(t, err)
	require.Contains(t, st.MemberProblems(), "seq-2: safe head does not match the supernode")

	runPlan(t, c)
	require.Equal(t, "seq-1", fc.leader)
	require.True(t, fc.members["seq-1"].node.active)
	require.False(t, diverged.active)
	require.Equal(t, common.Hash{}, diverged.startedFrom)

	// Sequencing is restored; the diverged op-node is still reported.
	st, err = c.Status(context.Background())
	require.NoError(t, err)
	require.Empty(t, st.SequencingProblems())
	require.Equal(t, []string{"seq-2: safe head does not match the supernode"}, st.MemberProblems())
}

// A diverged standby alone does not trigger recovery.
func TestPlanIgnoresDivergedStandby(t *testing.T) {
	fc, c := newFake(t, "seq-0")
	fc.members["seq-0"].node.active = true
	fc.members["seq-2"].node.onFork = true
	fc.members["seq-2"].node.safe = 152
	c.Supernode.(*fakeNode).safe = 155

	st, err := c.Status(context.Background())
	require.NoError(t, err)
	require.Empty(t, st.SequencingProblems())
	require.NotEmpty(t, st.MemberProblems())
	steps, err := c.Plan(st, testRecoverConfig)
	require.NoError(t, err)
	require.Empty(t, steps)
}

func TestRecoverStopsExtraActiveSequencer(t *testing.T) {
	fc, c := newFake(t, "seq-0")
	fc.members["seq-1"].node.active = true

	runPlan(t, c)
	require.True(t, fc.members["seq-0"].node.active)
	require.False(t, fc.members["seq-1"].node.active)
}

func TestPlanWithoutCanonicalSequencer(t *testing.T) {
	fc, c := newFake(t, "seq-0")
	for _, m := range fc.members {
		m.node.onFork = true
		m.node.safe = 152
	}
	c.Supernode.(*fakeNode).safe = 155

	st, err := c.Status(context.Background())
	require.NoError(t, err)
	_, err = c.Plan(st, testRecoverConfig)
	require.ErrorIs(t, err, ErrNoCanonicalSequencer)
}

// Leadership flapping between conductors (each hands it off after failing to
// start) must not stop recovery: the plan waits for a leader after pausing.
func TestRecoverWhileLeadershipFlaps(t *testing.T) {
	fc, c := newFake(t, "")
	ctx := context.Background()
	st, err := c.Status(ctx)
	require.NoError(t, err)
	steps, err := c.Plan(st, testRecoverConfig)
	require.NoError(t, err)
	fc.leader = "seq-2" // a leader appears after the snapshot
	require.NoError(t, Run(ctx, testlog.Logger(t, slog.LevelInfo), steps))
	active := 0
	for _, m := range fc.members {
		if m.node.active {
			active++
		}
	}
	require.Equal(t, 1, active)
	require.True(t, fc.members[fc.leader].node.active)
}

// The format netchef renders from netchef/pkg/templates/op-conductor-ops/config.toml.tmpl.
const netchefConfig = `cert_path = "./combined-cacert.pem"

[networks]

[networks.int-reorg-0-0]
sequencers = [
  "int-reorg-0-0-seq-0",
  "int-reorg-0-0-seq-1",
]

[sequencers]

[sequencers.int-reorg-0-0-seq-0]
raft_addr = "seq-0-raft:50050"
conductor_rpc_url = "https://seq-0-op-conductor"
node_rpc_url = "https://seq-0-op-node"
voting = true

[sequencers.int-reorg-0-0-seq-1]
raft_addr = "seq-1-raft:50050"
conductor_rpc_url = "https://seq-1-op-conductor"
node_rpc_url = "https://seq-1-op-node"
voting = false
`

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "int-reorg-0-0.toml")
	require.NoError(t, os.WriteFile(path, []byte(netchefConfig), 0o600))

	cfg, err := LoadConfig(path, "int-reorg-0-0")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "combined-cacert.pem"), cfg.CertPath)
	require.Equal(t, []Sequencer{
		{ID: "int-reorg-0-0-seq-0", RaftAddr: "seq-0-raft:50050", ConductorURL: "https://seq-0-op-conductor", NodeURL: "https://seq-0-op-node", Voting: true},
		{ID: "int-reorg-0-0-seq-1", RaftAddr: "seq-1-raft:50050", ConductorURL: "https://seq-1-op-conductor", NodeURL: "https://seq-1-op-node", Voting: false},
	}, cfg.Sequencers)

	_, err = LoadConfig(path, "other")
	require.ErrorContains(t, err, `network "other" not in config`)
}

func TestStatusFlagsConfigIDsThatAreNotRaftIDs(t *testing.T) {
	fc, c := newFake(t, "an-int-reorg-0-0-opn-reth-f-seq-0")
	fc.members["seq-0"].node.active = true
	st, err := c.Status(context.Background())
	require.NoError(t, err)
	require.Contains(t, st.SequencingProblems()[0], "is not a sequencer in the config")
}

// A failure after the conductors are paused must still resume them.
func TestFailedRecoveryResumesConductors(t *testing.T) {
	fc, c := newFake(t, "seq-0")
	ctx := context.Background()
	st, err := c.Status(ctx)
	require.NoError(t, err)
	steps, err := c.Plan(st, testRecoverConfig)
	require.NoError(t, err)
	// Make the start step fail.
	for i := range steps {
		if strings.HasPrefix(steps[i].Desc, "start sequencer") {
			steps[i].Run = func(context.Context) error { return errors.New("boom") }
		}
	}
	require.ErrorContains(t, Run(ctx, testlog.Logger(t, slog.LevelInfo), steps), "boom")
	for id, m := range fc.members {
		require.False(t, m.cond.paused, "conductor %s left paused", id)
	}
}

// A supernode RPC outage must not make an active sequencer look inactive.
func TestSupernodeOutageDoesNotHideActiveSequencer(t *testing.T) {
	fc, c := newFake(t, "seq-0")
	fc.members["seq-0"].node.active = true
	c.Supernode = &flakySupernode{fakeNode: c.Supernode.(*fakeNode)}
	st, err := c.Status(context.Background())
	require.NoError(t, err)
	require.Empty(t, st.SequencingProblems())
	require.NotEmpty(t, st.MemberProblems())
}

type flakySupernode struct{ *fakeNode }

func (f *flakySupernode) OutputAtBlock(context.Context, uint64) (*eth.OutputResponse, error) {
	return nil, errors.New("503 Service Unavailable")
}
