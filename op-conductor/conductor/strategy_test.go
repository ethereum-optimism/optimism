package conductor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
)

func voters(ids ...string) *consensus.ClusterMembership {
	m := &consensus.ClusterMembership{}
	for _, id := range ids {
		m.Servers = append(m.Servers, consensus.ServerInfo{
			ID:       id,
			Addr:     id + ":50050",
			Suffrage: consensus.Voter,
		})
	}
	return m
}

func targetIDs(t *testing.T, servers []consensus.ServerInfo) []string {
	t.Helper()
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		out = append(out, s.ID)
	}
	return out
}

// Every member must derive the same ring, each starting from itself, so that
// leadership walks the whole cluster instead of bouncing between two members.
func TestRoundRobinTransferStrategy_RingOrder(t *testing.T) {
	m := voters("seq1", "seq2", "seq3")
	s := RoundRobinTransferStrategy{}

	for _, tc := range []struct {
		self string
		want []string
	}{
		{self: "seq1", want: []string{"seq2", "seq3"}},
		{self: "seq2", want: []string{"seq3", "seq1"}},
		{self: "seq3", want: []string{"seq1", "seq2"}},
	} {
		t.Run(tc.self, func(t *testing.T) {
			got, err := s.SelectTargets(context.Background(), tc.self, m)
			require.NoError(t, err)
			require.Equal(t, tc.want, targetIDs(t, got))
		})
	}
}

// Ordering must come from the membership contents, not the order raft happens
// to return them in.
func TestRoundRobinTransferStrategy_OrderIsIndependentOfMembershipOrder(t *testing.T) {
	s := RoundRobinTransferStrategy{}

	forward, err := s.SelectTargets(context.Background(), "seq2", voters("seq1", "seq2", "seq3"))
	require.NoError(t, err)
	reversed, err := s.SelectTargets(context.Background(), "seq2", voters("seq3", "seq2", "seq1"))
	require.NoError(t, err)

	require.Equal(t, targetIDs(t, forward), targetIDs(t, reversed))
}

func TestRoundRobinTransferStrategy_SkipsNonVoters(t *testing.T) {
	m := voters("seq1", "seq2")
	m.Servers = append(m.Servers, consensus.ServerInfo{
		ID:       "observer",
		Addr:     "observer:50050",
		Suffrage: consensus.Nonvoter,
	})

	got, err := RoundRobinTransferStrategy{}.SelectTargets(context.Background(), "seq1", m)
	require.NoError(t, err)
	require.Equal(t, []string{"seq2"}, targetIDs(t, got))
}

// With nobody else to hand off to, returning no targets lets the caller fall
// back to the consensus layer rather than failing.
func TestRoundRobinTransferStrategy_SingleVoterHasNoTargets(t *testing.T) {
	got, err := RoundRobinTransferStrategy{}.SelectTargets(context.Background(), "seq1", voters("seq1"))
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestRoundRobinTransferStrategy_ErrorsWhenSelfMissing(t *testing.T) {
	_, err := RoundRobinTransferStrategy{}.SelectTargets(context.Background(), "seq9", voters("seq1", "seq2"))
	require.ErrorContains(t, err, "not found in voter list")
}

func TestRoundRobinTransferStrategy_ErrorsOnNilMembership(t *testing.T) {
	_, err := RoundRobinTransferStrategy{}.SelectTargets(context.Background(), "seq1", nil)
	require.Error(t, err)
}
