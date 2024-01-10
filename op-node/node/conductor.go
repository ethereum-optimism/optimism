package node

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/log"

	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
)

// ConductorClient is a client for the op-conductor RPC service.
type ConductorClient struct {
	cfg       *Config
	metrics   *metrics.Metrics
	log       log.Logger
	apiClient conductorRpc.API
}

// NoOpConductor is a no-op conductor that assumes this node is the leader sequencer.
type NoOpConductor struct{}

var _ driver.SequencerConductor = &NoOpConductor{}
var _ driver.SequencerConductor = &ConductorClient{}

// NewNoOpConductor returns a new conductor client for the op-conductor RPC service.
func NewConductorClient(cfg *Config, log log.Logger, metrics *metrics.Metrics) *ConductorClient {
	return &ConductorClient{cfg: cfg, metrics: metrics, log: log}
}

// Initialize initializes the conductor client.
func (c *ConductorClient) Initialize() error {
	conductorRpcClient, err := dial.DialRPCClientWithTimeout(context.Background(), time.Minute*1, c.log, c.cfg.ConductorEndpoint())
	if err != nil {
		return fmt.Errorf("failed to dial conductor RPC: %w", err)
	}
	c.apiClient = conductorRpc.NewAPIClient(conductorRpcClient)
	return nil
}

// Leader returns true if this node is the leader sequencer.
func (c *ConductorClient) Leader(ctx context.Context) (bool, error) {
	if c.apiClient == nil {
		return false, fmt.Errorf("conductor client not initialized")
	}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.ConductorRpcTimeout)
	defer cancel()

	isLeader, err := retry.Do(ctx, 2, retry.Fixed(50*time.Millisecond), func() (bool, error) {
		record := c.metrics.RecordRPCClientRequest("conductor_leader")
		result, err := c.apiClient.Leader(ctx)
		record(err)
		return result, err
	})
	if err != nil {
		return false, err
	}
	return isLeader, nil
}

// CommitUnsafePayload commits an unsafe payload to the conductor log.
func (c *ConductorClient) CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayload) error {
	if c.apiClient == nil {
		return fmt.Errorf("conductor client not initialized")
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

// Initialize initializes the conductor client. In this case its a no-op.
func (c *NoOpConductor) Initialize() error {
	return nil
}

// Leader returns true if this node is the leader sequencer.
func (c *NoOpConductor) Leader(ctx context.Context) (bool, error) {
	return true, nil
}

// CommitUnsafePayload commits an unsafe payload to the conductor log.
func (c *NoOpConductor) CommitUnsafePayload(ctx context.Context, payload *eth.ExecutionPayload) error {
	return nil
}
