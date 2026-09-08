package conductor

import (
	"context"
	"errors"
	"sort"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
)

// TransferStrategy decides which cluster members a departing leader should hand
// leadership to, and in what order.
//
// Implementations may perform I/O (e.g. querying peers) and must respect ctx.
// Returning an empty target list defers to the consensus layer's own selection,
// which picks the voter with the most up-to-date log. A conductor with no
// strategy configured at all does the same, without fetching membership.
type TransferStrategy interface {
	SelectTargets(ctx context.Context, self string, membership *consensus.ClusterMembership) ([]consensus.ServerInfo, error)
}

// RoundRobinTransferStrategy walks the voters in sorted ServerID order, starting
// from the member after the current one. This guarantees every member gets a
// turn, so a cluster where the most up-to-date members are unhealthy still
// converges on one that can sequence.
//
// It must be used by every member of the cluster: a mixed configuration can
// bounce leadership between round-robin and log-based selection.
type RoundRobinTransferStrategy struct{}

var _ TransferStrategy = (*RoundRobinTransferStrategy)(nil)

func (RoundRobinTransferStrategy) SelectTargets(_ context.Context, self string, membership *consensus.ClusterMembership) ([]consensus.ServerInfo, error) {
	if membership == nil {
		return nil, errors.New("nil cluster membership")
	}

	var voters []consensus.ServerInfo
	for _, server := range membership.Servers {
		if server.Suffrage == consensus.Voter {
			voters = append(voters, server)
		}
	}
	if len(voters) <= 1 {
		return nil, nil
	}

	// Sort by ServerID so every member derives the same ring.
	sort.Slice(voters, func(i, j int) bool {
		return voters[i].ID < voters[j].ID
	})

	selfIdx := -1
	for i, server := range voters {
		if server.ID == self {
			selfIdx = i
			break
		}
	}
	if selfIdx == -1 {
		return nil, errors.New("current server not found in voter list")
	}

	targets := make([]consensus.ServerInfo, 0, len(voters)-1)
	for i := 1; i < len(voters); i++ {
		targets = append(targets, voters[(selfIdx+i)%len(voters)])
	}
	return targets, nil
}
