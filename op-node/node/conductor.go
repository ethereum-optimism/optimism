package node

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// ConductorClient is a client for the op-conductor RPC service.
type ConductorClient struct {
	cfg       *Config
	metrics   *metrics.Metrics
	log       log.Logger
	apiClient *conductorRpc.APIClient

	// overrideLeader is used to override the leader check for disaster recovery purposes.
	// During disaster situations where the cluster is unhealthy (no leader, only 1 or less nodes up),
	// set this to true to allow the node to assume sequencing responsibilities without being the leader.
	overrideLeader atomic.Bool
}

var _ conductor.SequencerConductor = &ConductorClient{}

// NewConductorClient returns a new conductor client for the op-conductor RPC service.
func NewConductorClient(cfg *Config, log log.Logger, metrics *metrics.Metrics) conductor.SequencerConductor {
	return &ConductorClient{
		cfg:     cfg,
		metrics: metrics,
		log:     log,
	}
}

// Initialize initializes the conductor client, make sure the remote service is reachable.
func (c *ConductorClient) Initialize(ctx context.Context) error {
	apiClient, err := retry.Do(ctx, 60, retry.Fixed(5*time.Second), func() (*conductorRpc.APIClient, error) {
		conductorRpcClient, err := dial.DialRPCClientWithTimeout(ctx, c.cfg.ConductorRpcTimeout, c.log, c.cfg.ConductorRpc)
		if err != nil {
			log.Warn("failed to dial conductor RPC", "err", err)
			return nil, fmt.Errorf("failed to dial conductor RPC: %w", err)
		}

		log.Info("conductor connected")
		return conductorRpc.NewAPIClient(conductorRpcClient), nil
	})
	if err != nil {
		return err
	}

	c.apiClient = apiClient
	return nil
}

// Enabled returns true if conductor is enabled.
func (c *ConductorClient) Enabled(ctx context.Context) bool {
	return true
}

// Leader returns true if this node is the leader sequencer.
func (c *ConductorClient) Leader(ctx context.Context) (bool, error) {
	if c.overrideLeader.Load() {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.cfg.ConductorRpcTimeout)
	defer cancel()

	isLeader, err := retry.Do(ctx, 2, retry.Fixed(50*time.Millisecond), func() (bool, error) {
		record := c.metrics.RecordRPCClientRequest("conductor_leader")
		result, err := c.apiClient.Leader(ctx)
		record(err)
		return result, err
	})
	return isLeader, err
}

// CommitUnsafePayload commits an unsafe payload to the conductor log.
func (c *ConductorClient) CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	if c.overrideLeader.Load() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.cfg.ConductorRpcTimeout)
	defer cancel()

	// extra bool return value is required for the generic, can be ignored.
	_, err := retry.Do(ctx, 2, retry.Fixed(50*time.Millisecond), func() (bool, error) {
		record := c.metrics.RecordRPCClientRequest("conductor_commitUnsafePayload")
		err := c.apiClient.CommitUnsafePayload(ctx, payload)
		record(err)
		return true, err
	})
	return err
}

// OverrideLeader implements conductor.SequencerConductor.
func (c *ConductorClient) OverrideLeader(ctx context.Context) error {
	c.overrideLeader.Store(true)
	return nil
}

func (c *ConductorClient) Close() {
	if c.apiClient == nil {
		return
	}
	c.apiClient.Close()
	c.apiClient = nil
}
