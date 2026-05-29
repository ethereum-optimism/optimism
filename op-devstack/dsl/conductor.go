package dsl

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

type ConductorSet []*Conductor

func NewConductorSet(inner []stack.Conductor) ConductorSet {
	conductors := make([]*Conductor, len(inner))
	for i, c := range inner {
		conductors[i] = NewConductor(c)
	}
	return conductors
}

type Conductor struct {
	commonImpl
	inner stack.Conductor
}

func NewConductor(inner stack.Conductor) *Conductor {
	return &Conductor{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *Conductor) String() string {
	return c.inner.Name()
}

func (c *Conductor) Escape() stack.Conductor {
	return c.inner
}

func (c *Conductor) FetchClusterMembership() *consensus.ClusterMembership {
	c.log.Debug("Fetching cluster membership")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	clusterMembership, err := retry.Do(ctx, 2, retry.Fixed(500*time.Millisecond), func() (*consensus.ClusterMembership, error) {
		clusterMembership, err := c.inner.RpcAPI().ClusterMembership(c.ctx)
		return clusterMembership, err
	})
	c.require.NoError(err, "Failed to fetch cluster membership")
	c.log.Info("Fetched cluster membership",
		"clusterMembership", clusterMembership)
	return clusterMembership
}

func (c *Conductor) FetchLeader() *consensus.ServerInfo {
	c.log.Debug("Fetching leader information")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	leaderInfo, err := retry.Do[*consensus.ServerInfo](ctx, 2, retry.Fixed(500*time.Millisecond), func() (*consensus.ServerInfo, error) {
		leaderInfo, err := c.inner.RpcAPI().LeaderWithID(c.ctx)
		return leaderInfo, err
	})
	c.require.NoError(err, "Failed to fetch leader information")
	c.log.Info("Fetched leader information",
		"leaderInfo", leaderInfo)
	return leaderInfo
}

// FetchLatestUnsafePayload returns the latest unsafe payload recorded in the
// conductor's consensus layer. With reorg-recovery enabled this value is
// non-monotonic (it can move backward on a reorg).
func (c *Conductor) FetchLatestUnsafePayload() *eth.ExecutionPayloadEnvelope {
	c.log.Debug("Fetching latest unsafe payload")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	payload, err := retry.Do(ctx, 2, retry.Fixed(500*time.Millisecond), func() (*eth.ExecutionPayloadEnvelope, error) {
		return c.inner.RpcAPI().LatestUnsafePayload(ctx)
	})
	c.require.NoError(err, "Failed to fetch latest unsafe payload")
	return payload
}

func (c *Conductor) FetchSequencerHealthy() bool {
	c.log.Debug("Fetching sequencer healthy status")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	healthy, err := c.inner.RpcAPI().SequencerHealthy(ctx)
	c.require.NoError(err, "Failed to fetch sequencer healthy status")
	c.log.Info("Fetched sequencer healthy status", "healthy", healthy)
	return healthy
}

func (c *Conductor) FetchPaused() bool {
	c.log.Debug("Fetching paused status")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	paused, err := c.inner.RpcAPI().Paused(ctx)
	c.require.NoError(err, "Failed to fetch paused status")
	c.log.Info("Fetched paused status", "paused", paused)
	return paused
}

func (c *Conductor) IsLeader() bool {
	c.log.Debug("Checking if conductor is leader")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	leader, err := c.inner.RpcAPI().Leader(ctx)
	c.require.NoError(err, "Failed to check if conductor is leader")
	c.log.Info("Checked if conductor is leader", "leader", leader)
	return leader
}

// ScrapeCounter fetches the conductor's Prometheus metrics endpoint and returns the
// value of the named counter (e.g. "op_conductor_reorgs_committed_count"). Returns 0
// when the metric is absent, which is the correct reading for a counter that has never
// been incremented.
func (c *Conductor) ScrapeCounter(name string) float64 {
	endpoint := c.inner.MetricsEndpoint()
	c.require.NotEmpty(endpoint, "conductor %s does not expose a metrics endpoint", c.inner.Name())
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	value, err := retry.Do(ctx, 3, retry.Fixed(250*time.Millisecond), func() (float64, error) {
		return scrapeCounter(ctx, endpoint, name)
	})
	c.require.NoErrorf(err, "failed to scrape metric %s from conductor %s", name, c.inner.Name())
	return value
}

func scrapeCounter(ctx context.Context, endpoint string, name string) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Prometheus exposition: "<metric>[{labels}] <value>". The reorg counters are
		// label-less, so an exact name prefix followed by a space is sufficient.
		if !strings.HasPrefix(line, name+" ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		return strconv.ParseFloat(fields[len(fields)-1], 64)
	}
	// Metric not yet emitted — a never-incremented counter reads as 0.
	return 0, nil
}

func (c *Conductor) TransferLeadershipTo(targetLeaderInfo consensus.ServerInfo) {
	c.log.Debug("Transferring leadership to target leader", "targetLeaderID", targetLeaderInfo.ID, "targetLeaderAddr", targetLeaderInfo.Addr)
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().TransferLeaderToServer(ctx, targetLeaderInfo.ID, targetLeaderInfo.Addr)
	c.require.NoError(err, "Failed to transfer leadership to target leader", "targetLeaderID", targetLeaderInfo.ID)
	c.log.Info("Transferred leadership to target leader", "targetLeaderID", targetLeaderInfo.ID)
}
